package mcp

import (
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

func NewRequest(method string, params, id any) *Request {
	return &Request{
		JSONRPC: jsonrpcver,
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

// Either Result or Error not null
//
//	{
//	  "jsonrpc": "2.0",
//	  "result": 19,
//	  "id": 1
//	}
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      any             `json:"id"`
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

type Connection struct {
	// Prevent concurrent writes to encoder
	encodeMu  sync.Mutex
	reader    io.Reader
	writer    io.Writer
	closer    io.Closer
	encoder   *json.Encoder
	decoder   *json.Decoder
	pendingMu sync.Mutex
	// Map request ID to response channel
	pending map[string]chan *Response
	nextID  uint64
	// Signal the start of graceful shutdown process
	closing chan struct{}
	// Signal the end of graceful shutdown process
	shutdown     chan struct{}
	notification chan *Notification
	errMu        sync.Mutex
	connErr      error
}

// Handle sending and receiving of byte payloads
// over stdin/stdout
// TODO: Transport interface for different communication mechanisms
// e.g., HTTP, WebSocket, net.Conn
type Transport struct {
	// TODO: Use io.Writer as low-level interface for raw byte streams.
	// since we might be dealing with formats other than JSON
	// when convert Go data structures,
	// and we are able to check error when writing?
	encoder *json.Encoder
	decoder *json.Decoder
	closer  io.Closer
}

// Transport for stdio communication
func NewTransport(rwc io.ReadWriteCloser) *Transport {
	// Assume messages are newline-separated JSON?
	return &Transport{
		encoder: json.NewEncoder(rwc),
		decoder: json.NewDecoder(rwc),
		closer:  rwc,
	}
}
