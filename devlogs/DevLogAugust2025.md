## MCP implementation:

We are going to write a client that can talk to multiple MCP servers. For that we will use the package `rpc` to implement JSON-RPC 2.0 protocol.

## JSON-RPC 2.0

Stateless, light-weight, transport-agnostic RPC protocol. It can be used within the same process, over sockets/HTTP or in different message passing environments ([Link to specification](https://github.com/dhamidi/smolcode/blob/main/mcp/jsonrpc2/spec.md))

Our request would look something like this ([official documentation](https://modelcontextprotocol.io/docs/learn/architecture#data-layer-2))

The core methods for our JSON-RPC client are:

- `Call()`: Format the request to the server and send it via the transport. Each request ID will be mapped to a response channel, and `Call()` will unmarshal values from that response channel
- `Listen()`: The most complex one. It works as a goroutine listening to messages from the transport via `Receive()`. The incoming message then goes through some type assertion to see whether it is a request/notification from the server, or response to a `Call()`. The valid request will be sent to the response channel mapped with a specific request ID as the key.
- `Close()`: Shut down the goroutine in `Listen()` and close the transport
- `Notify()`: Use the transport to send a request to the server _without expecting a response_

What our transport will do:

- `Send()`: Encode/Write the payload into the stream/buffer
- `Receive()`: A `while` loop that decodes/reads the raw message from the stream/buffer and return the data of type `[]byte`
- `Close()`: Signal transport closing
