package mcp

// JSON-RPC 2.0 client
type Client struct {
	transport Transport
	nextID    uint64
	// TODO: Enhance with async calls?
}
