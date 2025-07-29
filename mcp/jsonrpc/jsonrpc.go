package jsonrpc

import "encoding/json"

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
	Error   *ErrorObject    `json:"error,omitempty"`
	ID      any             `json:"id"`
}

type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
