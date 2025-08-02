package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func (t *Transport) Send(ctx context.Context, payload []byte) error {
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

func (t *Transport) Receive(ctx context.Context) ([]byte, error) {
	errChan := make(chan error, 1)
	byteChan := make(chan []byte, 1)

	go func() {
		var raw json.RawMessage
		if err := t.decoder.Decode(&raw); err != nil {
			errChan <- err
			// Break out of goroutine immediately?
			// Do we have to wait for it?
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

func (t *Transport) Close() error {
	if t.closer != nil {
		err := t.closer.Close()
		// Specifically check for os.ErrClosed? Why?
		if strings.Contains(err.Error(), "file already closed") || err == os.ErrClosed {
			return nil
		}
		return err
	}
	return nil
}
