package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	// Prevent test from premature exit,
	// since the test function could return before the goroutine finishes its work,
	// or Call() might return but the goroutine could still be in the middle of cleanup operations
	<-serverDone

	assert.NoError(t, err, "c.Call should succeed without error")
	if err == nil {
		assert.NotNil(t, result, "Result map should not be nil after successful call")
		assert.Equal(t, "success", result["result"], "Result field did not match")
	}
}

func TestClientHandlesNotification(t *testing.T) {
	clientReadFromServer := new(bytes.Buffer)
	clientWriteToServer := new(bytes.Buffer)

	transport := &mockTransport{
		writeBuf: clientWriteToServer,
		readBuf:  clientReadFromServer,
		closed:   make(chan struct{}),
	}

	c := NewClient(transport)

	notificationMethod := "test/notificationEvent"
	type NotificationParams struct {
		Message string `json:"message"`
		Value   int    `json:"value"`
	}
	expectedParams := NotificationParams{Message: "hello", Value: 123}

	notificationHandled := make(chan bool, 1)

	c.OnNotification(notificationMethod, func(params *json.RawMessage) error {
		if params == nil {
			t.Errorf("Notification handler received nil params for method %s", notificationMethod)
			notificationHandled <- false
			return errors.New("nil params")
		}
		var receivedParams NotificationParams
		if err := json.Unmarshal(*params, &receivedParams); err != nil {
			t.Errorf("Failed to unmarshal notification params: %v", err)
			notificationHandled <- false
			return err
		}

		if assert.Equal(t, expectedParams, receivedParams, "Notification params did not match expected") {
			notificationHandled <- true
		} else {
			notificationHandled <- false
		}
		return nil
	})

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
		paramsBytes, err := json.Marshal(expectedParams)
		if err != nil {
			t.Logf("Server: Failed to marshal notification params: %v", err)
			return
		}
		notificationJSON := `{"jsonrpc": "2.0", "method": "` + notificationMethod + `", "params": ` + string(paramsBytes) + `}` + "\n"

		_, err = clientReadFromServer.Write([]byte(notificationJSON))
		if err != nil {
			t.Logf("Server: Failed to write notification: %v", err)
			return
		}
	}()

	select {
	case success := <-notificationHandled:
		assert.True(t, success, "Notification handler reported failure or did not match params")
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for notification to be handled")
	}
	<-serverDone
}
