package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

func NewClient(transport Transport) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		transport: transport,
		nextID:    1, // Start from 1
		// Mutexes are zero-value when constructed i.e., unlocked state
		notiHandlers: make(map[string]func(params *json.RawMessage) error),
		pendingCalls: make(map[any]chan *Response),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Make RPC calls and handle responses
func (c *Client) Call(ctx context.Context, args *ClientCallArgs, resultDest any) error {
	c.idMu.Lock()
	currentID := c.nextID
	c.nextID++
	c.idMu.Unlock()

	reqArgs := &RequestArgs{
		Method: args.Method,
		Params: args.Params,
		ID:     currentID,
	}

	reqBytes, err := FormatRequest(reqArgs)
	if err != nil {
		return fmt.Errorf("jsonrpc: failed to format request: %w", err)
	}

	respChan := make(chan *Response, 1)
	c.pendingCallsMu.Lock()
	// Check if client is closing or has closed
	select {
	case <-c.ctx.Done():
		c.pendingCallsMu.Unlock()
		return fmt.Errorf("jsonrpc: client is closed: %w", c.ctx.Err())
	default:
		// Proceed if client is not closed
	}
	c.pendingCalls[currentID] = respChan
	c.pendingCallsMu.Unlock()

	// Ensure pending call is cleaned up if Call returns
	// before response is received or processed,
	// so that we are ready to handle a new request?
	defer func() {
		c.pendingCallsMu.Lock()
		delete(c.pendingCalls, currentID)
		c.pendingCallsMu.Unlock()
	}()

	// TODO: There should be a way for the transport to send back data
	// and that might be communicating over respChan
	if err := c.transport.Send(ctx, reqBytes); err != nil {
		return fmt.Errorf("jsonrpc: transport failed to send request: %w", err)
	}

	select {
	case <-ctx.Done(): // Context for this specific call
		return fmt.Errorf("jsonrpc: call timed out or was cancelled: %w", ctx.Err())
	case <-c.ctx.Done(): // Client's main context, indicates listener might be shutting down
		return fmt.Errorf("jsonrpc: client is closing: %w", c.ctx.Err())
	case resp := <-respChan:
		if resp == nil {
			// Listen loop might close the channel during shutdown
			// without sending a response
			return fmt.Errorf("jsonrpc: call for ID %v aborted due to client shutdown or an issue in listener", currentID)
		}

		if resp.Error != nil {
			return fmt.Errorf("jsonrpc: server error (code: %d): %s", resp.Error.Code, resp.Error.Message)
		}

		if resp.Result == nil && resultDest != nil {
			// The caller expects a value?
			// resultDest remains unchanged here?
			return nil
		}

		if resultDest != nil && resp.Result != nil {
			if err := json.Unmarshal(*resp.Result, resultDest); err != nil {
				return fmt.Errorf("jsonrpc: failed to unmarshal result: %w", err)
			}
		}
	}
	return nil

}

// Register a handler function for a given server notification method.
// Overwrite the existing handler if there is a new one.
func (c *Client) OnNotification(method string, handler func(params *json.RawMessage) error) {
	c.notiMu.Lock()
	defer c.notiMu.Unlock()
	c.notiHandlers[method] = handler
}

// Send notifications without expecting a response
func (c *Client) Notify(ctx context.Context, args *ClientNotifyArgs) error {
	// ID is nil for notifications
	reqArgs := &RequestArgs{
		Method: args.Method,
		Params: args.Params,
	}
	reqBytes, err := FormatRequest(reqArgs)
	if err != nil {
		return fmt.Errorf("jsonrpc: failed to format notification: %w", err)
	}

	// We still send a request but not expect a response
	// since per JSON-RPC, no response is sent for notifications.
	// However, we still expect a payload if the transport is HTTP-based for example.
	err = c.transport.Send(ctx, reqBytes)
	if err != nil {
		return fmt.Errorf("jsonrpc: transport error during notify: %w", err)
	}

	return nil
}

// Start the client's listening goroutine.
// This method will block until the client's context is canceled (e.g., by calling Close)
// or an unrecoverable error occurs in the transport's Receive method.
// All server-to-client notifications and responses to client calls are processed here.
func (c *Client) Listen() error {
	c.wg.Add(1)
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			// Client closing
			return c.ctx.Err()
		default:
			// Non-blocking: Check if the context is canceled
			// and exit immediately if so
		}

		// A blocking function call.
		// This call is blocking since it waits for a message from the transport
		// and only returns once a message is received or an error occurs.
		payload, err := c.transport.Receive(c.ctx)
		if err != nil {
			// TODO: Need string comparison ordinal case for final condition?
			if c.ctx.Err() != nil && (err == context.Canceled || err == context.DeadlineExceeded || err.Error() == "context canceled") {
				// Expected shutdown
				c.cleanupPendingCalls()
				return c.ctx.Err()
			}
			// Unexpected transport error
			fmt.Printf("jsonrpc: error receiving message from transport: %v\n", err)
			c.cleanupPendingCalls()
			return fmt.Errorf("jsonrpc: transport receive error:: %w", err)
		}

		if len(payload) == 0 {
			// Connection closed cleanly?
			continue
		}

		var incomingMsg IncomingMessage
		if err := json.Unmarshal(payload, &incomingMsg); err != nil {
			fmt.Printf("jsonrpc: error unmarshalling incoming message %v: %s\n", err, string(payload))
			continue
		}

		// Dispatch the message
		if incomingMsg.Method != "" {
			// Either a request or notification from server
			c.notiMu.Lock()
			handler, ok := c.notiHandlers[incomingMsg.Method]
			c.notiMu.Unlock()

			if ok {
				go func(p *json.RawMessage) {
					if hErr := handler(p); hErr != nil {
						fmt.Printf("jsonprc: notification handler for method '%s' failed: %v", incomingMsg.Method, hErr)
					}
				}(incomingMsg.Params)
			} else {
				fmt.Printf("jsonrpc: no notification handler method: '%s'\n", incomingMsg.Method)
			}
		} else if incomingMsg.ID != nil {
			// Response to a client call
			if incomingMsg.Error != nil && incomingMsg.Result != nil {
				// Invalid response
				fmt.Printf("jsonrpc: received response with ID %v that has both result and error fields\n", incomingMsg.ID)
				continue
			}
			if incomingMsg.Error == nil && incomingMsg.Result == nil && incomingMsg.JSONRPC == "2.0" {
				// Invalid response
				fmt.Printf("jsonrpc: received response with ID %v that has neither error nor result\n", incomingMsg.ID)
				continue
			}
			if incomingMsg.Error == nil && incomingMsg.Result == nil && incomingMsg.JSONRPC == "2.0" { // ID is present, JSONRPC is present, but no result/error
				fmt.Printf("jsonrpc: received response with ID %v that has neither result nor error field\n", incomingMsg.ID)
				continue // Invalid response, skip
			}

			// TODO: This could be a separate function
			var mapKey any
			switch idVal := incomingMsg.ID.(type) {
			case float64:
				mapKey = uint64(idVal)
			case string:
				mapKey = idVal
			default:
				// Use as is, assuming consistent types or Call side handles it?
				mapKey = incomingMsg.ID
			}

			c.pendingCallsMu.Lock()
			ch, ok := c.pendingCalls[mapKey]
			c.pendingCallsMu.Unlock()

			// Handle valid responses
			if ok && ch != nil {
				respForCall := &Response{
					JSONRPC: incomingMsg.JSONRPC,
					Result:  incomingMsg.Result,
					Error:   incomingMsg.Error,
					ID:      incomingMsg.ID,
				}
				select {
				case ch <- respForCall:
				// Why is there no handling here?
				case <-c.ctx.Done():
				}
			} else {
				fmt.Printf("jsonrpc: received response for unknown or already handled ID: %v\n", incomingMsg.ID)
			}

		} else {
			// Neither response for call nor notification/request to client
			fmt.Printf("jsonrpc: received ill-formed message (no method and no/null ID for dispatch): %s\n", string(payload))
		}
	}
}

// Shutdown the client's listener goroutine and clean up resources
// by closing the clients main context, which signals the listener to stop.
func (c *Client) Close() error {
	c.cancel()  // Sinal listener goroutine to stop via context cancellation
	c.wg.Wait() // Wait for listener goroutine to finish

	// At this point, the loop in Listen() has exited and cleanupPendingCalls() has been invoked due to context cancellation,
	// cleanupPendingCalls() will have closed pendingCalls map (request ID and response channel)
	// but we still clear the map just to make sure
	c.pendingCallsMu.Lock()
	for id, ch := range c.pendingCalls {
		if ch != nil {
			close(ch)
		}
		delete(c.pendingCalls, id)
	}
	// Re-initialize just to be safe?
	// The client might be re-used after closing,
	// and it's a good practice to reset resouces after cleanup?
	c.pendingCalls = make(map[any]chan *Response)
	c.pendingCallsMu.Unlock()

	// Close transport
	if closer, ok := c.transport.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			// This does not prevent other cleanup or shadow client context errors.
			fmt.Printf("jsonrpc: error closing transport: %v\n", err)
			return fmt.Errorf("jsonrpc: error closing transport: %w", err)
		}
	}
	return nil
}

// Unblock any pending Call,
// allowing them to exit gracefully instead of timing out
func (c *Client) cleanupPendingCalls() {
	c.pendingCallsMu.Lock()
	defer c.pendingCallsMu.Unlock()
	for id, ch := range c.pendingCalls {
		if ch != nil {
			// Send a nil as a signal for the waiting goroutine know shutdown is happening.
			// This prevents panic or deadlock
			select {
			case ch <- nil:
			// Channel is ready, the send will happen immediately
			// Unblock any goroutine that might be stuck waiting to receive from ch
			default:
				// Channel is not ready, the send is skipped
				// This means no goroutine is waiting we can move on with the cleaning process
			}
			close(ch)
		}
		delete(c.pendingCalls, id)
	}
}

// FormatRequest creates a JSON-RPC request object and marshals it to JSON.
// The id can be a string, number, or null. If id is nil, it will be omitted (for notifications).
func FormatRequest(args *RequestArgs) ([]byte, error) {
	req := Request{
		JSONRPC: "2.0",
		Method:  args.Method,
		Params:  args.Params,
		ID:      args.ID,
	}
	return json.Marshal(req)
}

// ParseResponse unmarshals a JSON response and separates the id, result (as json.RawMessage), and error fields.
func ParseResponse(jsonResponse []byte) (id any, result *json.RawMessage, errResp *Error, parseErr error) {
	var resp Response
	parseErr = json.Unmarshal(jsonResponse, &resp)
	if parseErr != nil {
		return nil, nil, nil, parseErr
	}
	// Note: The JSON-RPC 2.0 spec says the id field in a response MUST match the id field in the request,
	// or be null if there was an error parsing the request id (which this client-side parser doesn't deal with directly).
	// It's also possible for id to be null for notifications that illicit an error.
	// The jsonrpc field is optional in responses according to some interpretations, but we check if present.
	if resp.JSONRPC != "" && resp.JSONRPC != "2.0" {
		// This is a stricter check than the spec absolutely requires for responses, but good practice.
		return resp.ID, nil, &Error{Code: -32600, Message: "Invalid JSON-RPC version"}, nil
	}

	// Validate mutual exclusivity of result and error
	if resp.Error != nil && resp.Result != nil {
		return resp.ID, nil, nil, fmt.Errorf("jsonrpc: response contains both result and error fields")
	}

	if resp.Error == nil && resp.Result == nil {
		// A valid success response must have a "result" field (even if null), and an error response must have an "error" field.
		return resp.ID, nil, nil, fmt.Errorf("jsonrpc: response contains neither result nor error field")
	}

	return resp.ID, resp.Result, resp.Error, nil
}
