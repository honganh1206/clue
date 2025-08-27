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
	// We use io.Writer to write directly to the underlying stream.
	// Using the encoder may cause the data to sit in an internal buffer
	// instead of being sent immediately
	writer  io.Writer
	decoder *json.Decoder
	closer  io.Closer
}

func NewStdioTransport(rwc io.ReadWriteCloser) *stdioTransport {
	return &stdioTransport{
		writer:  rwc,
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
		if _, err := t.writer.Write(payload); err != nil {
			return fmt.Errorf("failed to write payload: %w", err)
		}
		if _, err := t.writer.Write([]byte{'\n'}); err != nil {
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
