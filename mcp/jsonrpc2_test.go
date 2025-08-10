package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCallSuccess(t *testing.T) {
	clientReadFromServer := new(bytes.Buffer) // Server writes to this, client reads from it
	clientWriteToServer := new(bytes.Buffer)  // Client writes to this, server reads from it

	// Create a client connection that reads from clientReadFromServer and writes to clientWriteToServer
	c := Connect(clientReadFromServer, clientWriteToServer)

	go func() {
		t.Logf("Server: Goroutine started")
		// Client Connects to (reader, writer)
		// Client writes its requests to 'writer'. Data flows writer -> reader.
		// Client reads responses from 'reader'.
		// Server goroutine must decode from 'reader' and encode to 'writer'.

		serverDecoder := json.NewDecoder(clientWriteToServer)  // Server reads from the buffer client writes to
		serverEncoder := json.NewEncoder(clientReadFromServer) // Server writes to the buffer client reads from

		t.Logf("Server: Decoding request...")
		var clientReq Request
		if err := serverDecoder.Decode(&clientReq); err != nil {
			// If server fails to decode, client call will likely fail/timeout.
			// Consider t.Log or similar if debugging hangs.
			t.Logf("Server: Error decoding request: %v", err)
			return
		}
		t.Logf("Server: Request decoded: %+v", clientReq)

		// Simulate successful response
		respToSend := &Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`{"result": "success"}`),
			ID:      clientReq.ID,
		}
		t.Logf("Server: Encoding response: %+v", respToSend)
		if errEncode := serverEncoder.Encode(respToSend); errEncode != nil {
			t.Logf("Server: Error encoding response: %v", errEncode)
		} else {
			t.Logf("Server: Response encoded and sent")
		}
		// Send a notification to keep the readLoop engaged
		notification := &Request{
			JSONRPC: "2.0",
			Method:  "server.event",
			Params:  map[string]string{"data": "keepalive"},
			// ID is omitted for notifications, making it a Notification
		}
		t.Logf("Server: Encoding notification: %+v", notification)
		if errNotify := serverEncoder.Encode(notification); errNotify != nil {
			t.Logf("Server: Error encoding notification: %v", errNotify)
		} else {
			t.Logf("Server: Notification encoded and sent")
		}

		t.Logf("Server: Goroutine blocking to keep connection alive for test")
		select {} // Block forever
	}()

	var result map[string]string
	err := c.Call(context.Background(), "testMethod", map[string]interface{}{"param1": "value1"}, &result)
	assert.NoError(t, err, "c.Call should succeed without error")
	assert.NotNil(t, result, "Result map should not be nil after successful call")
	assert.Equal(t, "success", result["result"], "Result field did not match")
}

func TestCallJSONRPCError(t *testing.T) {
	reader, writer := io.Pipe()
	c := Connect(reader, writer)

	go func() {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "testMethod",
			Params:  map[string]interface{}{"param1": "value1"},
			ID:      "1",
		}
		resp := &Response{
			JSONRPC: "2.0",
			Error:   &Error{Code: -32601, Message: "Method not found"},
			ID:      "1",
		}
		json.NewEncoder(writer).Encode(req)
		json.NewEncoder(writer).Encode(resp)
	}()

	var result map[string]string
	err := c.Call(context.Background(), "testMethod", map[string]interface{}{"param1": "value1"}, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Method not found")
}

func TestCallContextCancelled(t *testing.T) {
	reader, writer := io.Pipe()
	c := Connect(reader, writer)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var result map[string]string
	err := c.Call(ctx, "testMethod", map[string]interface{}{"param1": "value1"}, &result)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestCallConnectionClosed(t *testing.T) {
	reader, writer := io.Pipe()
	c := Connect(reader, writer)

	writer.Close()

	var result map[string]string
	err := c.Call(context.Background(), "testMethod", map[string]interface{}{"param1": "value1"}, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection closed by remote")
}

func TestNotifySuccess(t *testing.T) {
	reader, writer := io.Pipe()
	c := Connect(reader, writer)

	go func() {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "testMethod",
			Params:  map[string]interface{}{"param1": "value1"},
		}
		json.NewEncoder(writer).Encode(req)
	}()

	err := c.Notify(context.Background(), "testMethod", map[string]interface{}{"param1": "value1"})
	assert.NoError(t, err)
}

func TestNotifyConnectionClosed(t *testing.T) {
	reader, writer := io.Pipe()
	c := Connect(reader, writer)

	writer.Close()

	err := c.Notify(context.Background(), "testMethod", map[string]interface{}{"param1": "value1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection closed by remote")
}

func TestSubscribe(t *testing.T) {
	reader, writer := io.Pipe()
	c := Connect(reader, writer)

	subChan := c.Subscribe()

	go func() {
		notification := &Notification{
			Method: "testMethod",
			Params: json.RawMessage(`{"param1": "value1"}`),
		}
		json.NewEncoder(writer).Encode(notification)
	}()

	select {
	case notification := <-subChan:
		assert.Equal(t, "testMethod", notification.Method)
		assert.Equal(t, `{"param1": "value1"}`, string(notification.Params))
	case <-time.After(1 * time.Second):
		t.Errorf("did not receive notification in time")
	}
}
