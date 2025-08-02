package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
)

// io.Closer that does nothing
// to implement io.Closer interface only
// so that Connection always has something to call Close() on
type noopCloser struct{}

func (noopCloser) Close() error { return nil }

// Cleanly close multiple underlying resources e.g., pipes, files or sockets
// if both reader and writer need to be closed
type multiCloser []io.Closer

func (mc multiCloser) Close() error {
	var err error
	for _, c := range mc {
		if e := c.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

// incomingMessage is used to determine if a message is a Response or Notification
type incomingMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
}

func Connect(reader io.Reader, writer io.Writer) *Connection {
	var closer io.Closer
	closers := []io.Closer{}
	if rCloser, ok := reader.(io.Closer); ok {
		closers = append(closers, rCloser)
	}
	// Single net.Conn object will detect and add the closer only once,
	// since we only close the underlying resource one
	if wCloser, ok := writer.(io.Closer); ok {
		// Check if reader and writer are the same closable entity
		// to avoid double adding/closing?
		isSame := false
		if len(closers) > 0 && wCloser != nil {
			if closers[0] == wCloser {
				isSame = true
			}
		}
		if !isSame {
			closers = append(closers, wCloser)
		}
	}
	if len(closers) == 0 {
		closer = noopCloser{}
	} else if len(closers) == 1 {
		closer = closers[0]
	} else {
		closer = multiCloser(closers)
	}

	c := &Connection{
		reader:       reader,
		writer:       writer,
		encoder:      json.NewEncoder(writer),
		decoder:      json.NewDecoder(reader),
		closer:       closer,
		pending:      make(map[string]chan *Response),
		notification: make(chan *Notification, 10),
		closing:      make(chan struct{}),
		shutdown:     make(chan struct{}),
	}

	go c.readLoop()
	return c
}

func (c *Connection) Call(ctx context.Context, method string, params any, result any) error {
	reqID := c.newRequestID()
	req := NewRequest(method, params, reqID)
	respChan := make(chan *Response, 1)

	c.pendingMu.Lock()
	// Check if client is closing
	select {
	case <-c.closing:
		c.pendingMu.Unlock()
		if c.connErr != nil {
			return c.connErr
		}
		return fmt.Errorf("jsonrpc: client is closing")
	default:
		// Continue if not closing
	}
	c.pending[reqID] = respChan
	c.pendingMu.Unlock()

	// Defer cleanup for pending channel
	defer func() {
		c.pendingMu.Lock()
		if ch, ok := c.pending[reqID]; ok && ch == respChan {
			delete(c.pending, reqID)
		}
		c.pendingMu.Unlock()
	}()

	c.encodeMu.Lock()
	err := c.encoder.Encode(req)
	c.encodeMu.Unlock()
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err() // Then run deferred cleanup
	case resp, ok := <-respChan:
		if !ok {
			// Depending on the timing:
			// If readLoop() closes respChan before Call() receives anything,
			// then ok == false
			// If Call() reads from respChan after it is closed by readLoop(),
			// we will enter this block
			return c.Err() // Return error set by readLoop()
		}
		if resp.Error != nil {
			return fmt.Errorf("jsonrpc: server error (code: %d): %s", resp.Error.Code, resp.Error.Message)
		}

		// TODO: resp len 0 cap 0 here
		if result != nil && resp.Result != nil && len(resp.Result) > 0 && string(resp.Result) != "null" {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return err
			}
		}
		return nil
	case <-c.closing:
		return c.Err()

	}
}

// Process notifications sent by servers to clients to inform tool updates or additions,
// as required by MCP specifications
func (c *Connection) Notify(ctx context.Context, method string, params any) error {
	req := &Request{
		Method: method,
		Params: params,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:

	}
	c.encodeMu.Lock()
	err := c.encoder.Encode(req)
	c.encodeMu.Unlock()
	return err
}

func (c *Connection) Close() error {
	close(c.closing)
	<-c.shutdown
	return c.closer.Close()
}

func (c *Connection) Err() error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.connErr
}

// Return a channel that receives unsolicited server notification
// i.e., an event or change to the tools
func (c *Connection) Subscribe() <-chan *Notification {
	return c.notification
}

func (c *Connection) newRequestID() string {
	// Increment address of next request ID by delta
	return strconv.FormatUint(atomic.AddUint64(&c.nextID, 1), 10)
}

// Continuously read responses from connections
// then match them to pending requests and deliver them via channels
func (c *Connection) readLoop() {
	defer close(c.shutdown)
	defer func() {
		c.pendingMu.Lock()
		for id, ch := range c.pending {
			close(ch)
			delete(c.pending, id)
		}
		c.pendingMu.Unlock()
		// Close noti chan if it has not been closed,
		// but we try to avoid double closing here
		// Per Go's recommendation, only the sender should close a channel
		// or the channel should be closed by a mux (select) on a done/closing channel.
		// Since readLoop() is the sender to the noti chan, it should close it on defer
		select {
		case <-c.notification:
		// Already closed or has items?
		default:
			close(c.notification)
		}
	}()

	for {
		var raw json.RawMessage

		// Check decoding
		if err := c.decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				err = fmt.Errorf("connection closed by remote")
			} else {
				err = fmt.Errorf("decode error: %w", err)
			}
			c.errMu.Lock()
			if c.connErr == nil {
				c.connErr = err
			}
			c.errMu.Unlock()

			select {
			case <-c.closing: // Already closing or closed?
			default:
				close(c.closing)
			}
			return
		}

		// Check malformed message
		var checker incomingMessage
		if err := json.Unmarshal(raw, &checker); err != nil {
			err = fmt.Errorf("malformed message envelope: %w, raw: %s", err, string(raw))
			c.errMu.Lock()
			if c.connErr == nil {
				c.connErr = err
			}
			c.errMu.Unlock()

			select {
			case <-c.closing: // Already closing or closed?
			default:
				close(c.closing)
			}
			return
		}

		// Check unmarshal to types
		if checker.ID != nil && string(checker.ID) != "null" {
			// Response case
			var resp Response
			if err := json.Unmarshal(raw, &resp); err != nil {
				err = fmt.Errorf("malformed message body: %w, raw: %s", err, string(raw))
				c.errMu.Lock()
				if c.connErr == nil {
					c.connErr = err
				}
				c.errMu.Unlock()

				select {
				case <-c.closing: // Already closing or closed?
				default:
					close(c.closing)
				}
				return
			}
			var id string

			switch v := resp.ID.(type) {
			case string:
				id = v
			case float64:
				// Could happen?
				// encoding/json unmarshals numbers into float64 BY DEFAULT
				id = strconv.FormatFloat(v, 'f', -1, 64)
			case json.Number:
				// If decoder is configured to use this
				id = v.String()
			default:
				id = fmt.Sprintf("%v", v)
			}

			c.pendingMu.Lock()
			respChan, ok := c.pending[id]
			if ok {
				delete(c.pending, id)
			}
			c.pendingMu.Unlock()

			if ok {
				select {
				case respChan <- &resp:
				case <-c.closing:
					// If connection is closing, don't block?
				}
				close(respChan)
			}
		} else if checker.Method != "" {
			// Handle notification if method is present

			// Tempo struct for full unmarshalling
			type tempNoti struct {
				JSONRPC string          `json:"jsonrpc"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params,omitempty"`
			}

			var noti tempNoti
			if err := json.Unmarshal(raw, &noti); err != nil {
				err = fmt.Errorf("malformed notification body: %w, raw: %s", err, string(raw))
				c.errMu.Lock()
				if c.connErr == nil {
					c.connErr = err
				}
				c.errMu.Unlock()

				select {
				case <-c.closing: // Already closing or closed?
				default:
					close(c.closing)
				}
				return
			}

			notiToSend := &Notification{
				Method: noti.Method,
				Params: noti.Params,
			}

			select {
			case c.notification <- notiToSend:
			case <-c.closing:
				return
			}
		} else {

		}

		select {
		case <-c.closing:
			return
		default:
			// Continue reading loop
		}
	}
}
