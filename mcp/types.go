package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

const jsonrpcver = "2.0"

// We only need to provide the method and params
// net/rpc will handle ID generation
//
//	{
//	  "jsonrpc": "2.0",
//	  "method": "subtract",
//	  "params": [42, 23],
//	  "id": 1
//	}
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	// Specify the version, must be exactly 2.0
	// TODO: Why not make it a constant?
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
	ID     any    `json:"id,omitempty"`
}

type RequestArgs struct {
	Method string
	Params any
	ID     any
}

// Defines the parameters for the "initialize" request.
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// Defines the result for the "initialize" response.
// Based on typical JSON-RPC, but mcp/docs.md doesn't specify its structure.
// Assuming it might be an empty object or contain server capabilities.
type InitializeResult struct {
	Capabilities map[string]any `json:"capabilities,omitempty"`
}

// Either Result or Error not null
//
//	{
//	  "jsonrpc": "2.0",
//	  "result": 19,
//	  "id": 1
//	}
type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
	ID      any              `json:"id"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc: code: %d, message: %s", e.Code, e.Message)
}

// Message received from the server that is not a response to a request.
// Used to announce changes from the server e.g., new tools or tool updates
type Notification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Transport interface for different communication mechanisms
// e.g., HTTP, WebSocket, net.Conn?
type Transport interface {
	Send(ctx context.Context, payload []byte) error
	Receive(ctx context.Context) ([]byte, error)
	Close() error
}
