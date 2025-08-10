package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
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

// Handle sending and receiving of byte payloads
// over stdin/stdout
type StdioTransport struct {
	// TODO: Use io.Writer as low-level interface for raw byte streams.
	// since we might be dealing with formats other than JSON
	// when convert Go data structures,
	// and we are able to check error when writing?
	encoder *json.Encoder
	decoder *json.Decoder
	closer  io.Closer
}

type Client struct {
	transport StdioTransport
	nextID    uint64
	// Thread-safe request ID generation
	idMu sync.Mutex

	notiHandlers map[string]func(params *json.RawMessage) error
	notiMu       sync.Mutex

	// Map responses to calls from client
	pendingCalls   map[any]chan *Response
	pendingCallsMu sync.Mutex

	// Lifecycle management for listener goroutine
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Encapsulate the arguments for the Client.Call method
type ClientCallArgs struct {
	Method string
	Params any
}

// Encapsulate the arguments for the Client.Notify method
type ClientNotifyArgs struct {
	Method string
	Params any
}

// Used to unmarshal any incoming JSON-RPC message,
// this will then go through type-assertion
type IncomingMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  string           `json:"method,omitempty"` // Present in requests/notifications
	Params  *json.RawMessage `json:"params,omitempty"` // Present in requests/notifications
	ID      interface{}      `json:"id,omitempty"`     // Present in requests and responses (even if null for some responses)
	Result  *json.RawMessage `json:"result,omitempty"` // Present in successful responses
	Error   *Error           `json:"error,omitempty"`  // Present in error responses
}
