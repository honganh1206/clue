## MCP implementation:

We are going to write a client that can talk to multiple MCP servers. For that we will use the package `mcp/jsonrpc2` as a client implementation for the JSON-RPC 2.0 protocol. The MCP implementation will be inside the `server` package of ours

## JSON-RPC 2.0

Stateless, light-weight, transport-agnostic RPC protocol. It can be used within the same process, over sockets/HTTP or in different message passing environments ([Link to specification](https://github.com/dhamidi/smolcode/blob/main/mcp/jsonrpc2/spec.md))

Our request would look something like this (stole from smolcode)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "smolcode",
      "version": "1.0.0"
    }
  }
}
```

(Also copy the docs from smolcode)

The server then proceeds with a response to this message.

After receiving the response, the client then needs to confirm reception of the response with the following notification:

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

# Setup

1. Set up subprocess & create I/O pipe to send/receive JSON-RPC to the subprocess
2. Set up ReadWriteCloser as an adapter to write
3. Set up codec (transition layer between high-level RPC interface and low-level network protocol) `rpc.NewClientWithCodec` to convert Go RPC calls into JSON-RPC 2.0 format and parse it back
4. Prepare request (params, method to call, etc.)

# Data flow

```
Go Application Layer:     client.Call("tools/list", params, &reply)
                                    ↓
RPC Framework:           rpc.Request{ServiceMethod: "tools/list", Seq: 123}
                                    ↓
Custom Codec:            {"jsonrpc":"2.0","method":"tools/list","id":123}
                                    ↓
Transport Layer:         JSON bytes over stdin/stdout pipes
                                    ↓
MCP Server:              Processes JSON-RPC 2.0 message
```

Request path (Go -> Subprocess)

1. Client with codec invokes `Call()` (Go RPC layer)
2. Codec serializes the JSON-RPC 2.0 request
3. Sub-process writes the request to stdin
4. MCP server processes the request

Response path (Subprocess -> Go)

1. MCP server sends JSON-RPC response to stdout of subprocess
2. `ReadCloser()` display on console + process RPC
3. Codec deserializes the response

> We implement request & body mutexes to prevent race condition when multiple goroutines make RPC calls simultaneously

# Translation responsibilities

`WriteRequest()` converts Go structures to JSON-RPC 2.0 (invoked by `Call()`)

- Go method call `client.Call("tools/list", params, &reply)`
- **Into JSON-RPC 2.0:** `{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}

# Notifications

We need to send a notification for a request we've just successfully made as required my MCP specification
