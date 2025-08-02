package mcp

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCallSuccess(t *testing.T) {
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
			Result:  json.RawMessage(`{"result": "success"}`),
			ID:      "1",
		}
		json.NewEncoder(writer).Encode(req)
		json.NewEncoder(writer).Encode(resp)
	}()

	var result map[string]string
	err := c.Call(context.Background(), "testMethod", map[string]interface{}{"param1": "value1"}, &result)
	assert.NoError(t, err)
	assert.Equal(t, "success", result["result"])
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
