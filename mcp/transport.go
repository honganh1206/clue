package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Handle sending and receiving of byte payloads
// over stdin/stdout
type stdioTransport struct {
	// TODO: Use io.Writer as low-level interface for raw byte streams.
	// since we might be dealing with formats other than JSON
	// when convert Go data structures,
	// and we are able to check error when writing?
	encoder *json.Encoder
	decoder *json.Decoder
	closer  io.Closer
}

func NewStdioTransport(rwc io.ReadWriteCloser) *stdioTransport {
	return &stdioTransport{
		encoder: json.NewEncoder(rwc),
		decoder: json.NewDecoder(rwc),
		closer:  rwc,
	}
}

// Send pre-formatted JSON-RPC payload with a newline delimiter
func (t *stdioTransport) Send(ctx context.Context, payload []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if err := t.encoder.Encode(payload); err != nil {
			return fmt.Errorf("failed to write payload: %w", err)
		}
		if err := t.encoder.Encode([]byte{'\n'}); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}

		return nil
	}
}

func (t *stdioTransport) Receive(ctx context.Context) ([]byte, error) {
	errChan := make(chan error, 1)
	byteChan := make(chan []byte, 1)

	go func() {
		var raw json.RawMessage
		if err := t.decoder.Decode(&raw); err != nil {
			errChan <- err
			// Break out of goroutine immediately
			return
		}
		// Cleaner way to convert to []byte?
		byteChan <- []byte(raw)
	}()

	select {
	case <-ctx.Done():
		if t.closer != nil {
			_ = t.closer.Close()
		}
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case data := <-byteChan:
		return data, nil
	}
}

func (t *stdioTransport) Close() error {
	if t.closer != nil {
		err := t.closer.Close()
		if strings.Contains(err.Error(), "file already closed") || err == os.ErrClosed {
			return nil
		}
		return err
	}
	return nil
}
