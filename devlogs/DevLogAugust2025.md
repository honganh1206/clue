## MCP implementation:

We are going to write a client that can talk to multiple MCP servers. For that we will use the package `rpc` to implement JSON-RPC 2.0 protocol.

## JSON-RPC 2.0

Stateless, light-weight, transport-agnostic RPC protocol. It can be used within the same process, over sockets/HTTP or in different message passing environments ([Link to specification](https://github.com/dhamidi/smolcode/blob/main/mcp/jsonrpc2/spec.md))

Our request would look something like this ([official documentation](https://modelcontextprotocol.io/docs/learn/architecture#data-layer-2))
