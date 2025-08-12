package mcp

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockTransport struct {
	writeBuf *bytes.Buffer
	readBuf  *bytes.Buffer
	closed   chan struct{}
}

func (m *mockTransport) Send(ctx context.Context, payload []byte) error {
	select {
	case <-m.closed:
		return io.ErrClosedPipe
	default:
	}

	n, err := m.writeBuf.Write(payload)
	if err != nil {
		return err
	}

	if n != len(payload) {
		return io.ErrShortWrite
	}

	// Write the line break?
	_, errNL := m.writeBuf.Write([]byte{'\n'})
	if errNL != nil {
		return errNL
	}
	return nil
}

func (m *mockTransport) Receive(ctx context.Context) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-m.closed:
			return nil, io.ErrClosedPipe
		default:
			if m.readBuf.Len() > 0 {
				// Read until reaching linebreak?
				line, err := m.readBuf.ReadBytes('\n')
				trimmedLine := bytes.TrimSpace(line)
				if err != nil && err != io.EOF {
					return nil, err
				}

				if err == io.EOF && len(trimmedLine) == 0 {
					return nil, io.EOF
				}

				return trimmedLine, nil

			}

			time.Sleep(1 * time.Millisecond)
		}
	}
}

func (m *mockTransport) Close() error {
	close(m.closed)
	return nil
}

func TestCallSuccess(t *testing.T) {
	clientReadFromServer := new(bytes.Buffer)
	clientWriteToServer := new(bytes.Buffer)

	transport := &mockTransport{
		writeBuf: clientWriteToServer,
		readBuf:  clientReadFromServer,
		closed:   make(chan struct{}),
	}

	c := NewClient(transport)
	go func() {
		err := c.Listen()
		if err != nil && err != context.Canceled && err != io.ErrClosedPipe && err.Error() != "context canceled" {
			t.Logf("Client Listen error: %v", err)
		}
	}()
	defer func() {
		c.Close()
	}()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		// 1. Consume the request from the client.
		requestSink := make([]byte, 1024)
		_, err := clientWriteToServer.Read(requestSink)
		if err != nil && err != io.EOF {
			t.Logf("Server: Error reading client request: %v", err)
			return
		}

		// 2. Send a hardcoded response. Client's first call ID is 1.
		responseJSON := `{"jsonrpc": "2.0", "id": 1, "result": {"result":"success"}}` + "\n"
		_, err = clientReadFromServer.Write([]byte(responseJSON))
		if err != nil {
			t.Logf("Server: Failed to write hardcoded response: %v", err)
			return
		}
	}()

	var result map[string]string
	callCtx, callCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer callCancel()

	err := c.Call(callCtx, &ClientCallArgs{Method: "testMethod", Params: map[string]any{"param1": "value1"}}, &result)

	// Why is this here?
	// Assume to signal that the server is done and shutting down?
	<-serverDone

	assert.NoError(t, err, "c.Call should succeed without error")
	if err == nil {
		assert.NotNil(t, result, "Result map should not be nil after successful call")
		assert.Equal(t, "success", result["result"], "Result field did not match")
	}
}
