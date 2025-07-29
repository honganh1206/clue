package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"strings"
	"sync"
)

// JSONRPC2Request represents a JSON-RPC 2.0 request object.
// As per net/rpc, the Params field is a single value (or a struct treated as one).
// The ID is uint64 to match rpc.Request.Seq.
type JSONRPC2Request struct {
	JSONRPC string  `json:"jsonrpc"`      // must be "2.0"
	Method  string  `json:"method"`       // the method to invoke on the server
	Params  any     `json:"params"`       // params used for invoking the method
	ID      *uint64 `json:"id,omitempty"` // request ID, omitempty for notifications
}

// JSONRPC2Response represents a JSON-RPC 2.0 response object.
// The ID is uint64 to match rpc.Response.Seq.
type JSONRPC2Response struct {
	JSONRPC string           `json:"jsonrpc"`          // must be "2.0"
	Result  *json.RawMessage `json:"result,omitempty"` // the result of calling the method
	Error   any              `json:"error,omitempty"`  // error object if an error occurred
	ID      uint64           `json:"id"`               // response ID, should match request ID
}

// JSONRPC2ClientCodec implements the rpc.ClientCodec interface for JSON-RPC 2.0.
type JSONRPC2ClientCodec struct {
	dec *json.Decoder // for reading JSON responses
	enc *json.Encoder // for writing JSON requests
	c   io.Closer     // to close the underlying connection

	reqMutex        sync.Mutex // Protects seq and pendingRequests
	seq             uint64
	pendingRequests map[uint64]string // Stores method for rpc.Response

	// bodyMutex protects lastResultForBody
	bodyMutex         sync.Mutex
	lastResultForBody *json.RawMessage

	// isNotificationCall is a buffered channel to signal if the last WriteRequest was a notification.
	// Used by ReadResponseHeader to determine if it should attempt to read a response.
	wasLastCallNotification bool
	Debug                   bool // Enable debug logging
}

var _ rpc.ClientCodec = (*JSONRPC2ClientCodec)(nil)

// NewJSONRPC2ClientCodec returns a new rpc.ClientCodec using JSON-RPC 2.0 on conn.
func NewJSONRPC2ClientCodec(conn io.ReadWriteCloser) *JSONRPC2ClientCodec {
	return &JSONRPC2ClientCodec{
		dec:             json.NewDecoder(conn),
		enc:             json.NewEncoder(conn),
		c:               conn,
		pendingRequests: make(map[uint64]string),
		// lastResultForBody is implicitly nil and bodyMutex is zero-valued
		// // wasLastCallNotification is implicitly false
	}
}

// WriteRequest writes a JSON-RPC request to the connection.
func (codec *JSONRPC2ClientCodec) WriteRequest(req *rpc.Request, params any) error {
	var jReq *JSONRPC2Request

	if strings.HasPrefix(req.ServiceMethod, "notifications/") {
		// This is a notification, ID should be nil (omitted)
		// Do not track in pendingRequests, do not increment seq for this.
		jReq = &JSONRPC2Request{
			JSONRPC: "2.0",
			Method:  req.ServiceMethod,
			Params:  params,
			ID:      nil, // ID is omitted for notifications
		}
		codec.wasLastCallNotification = true
	} else {
		// This is a regular request, assign an ID
		codec.reqMutex.Lock()
		codec.seq++
		id := codec.seq
		codec.pendingRequests[id] = req.ServiceMethod
		codec.reqMutex.Unlock()

		// Create a pointer to the id for the ID field
		reqID := id
		jReq = &JSONRPC2Request{
			JSONRPC: "2.0",
			Method:  req.ServiceMethod,
			Params:  params,
			ID:      &reqID,
		}
		codec.wasLastCallNotification = false
	}
	debugMsgBytes, _ := json.Marshal(jReq)
	codec.debug("WriteRequest - sending: %s\n", string(debugMsgBytes))
	if err := codec.enc.Encode(jReq); err != nil {
		return err
	}
	return codec.flush()
}

// jsonError represents a generic JSON-RPC error structure.
type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *jsonError) Error() string {
	return fmt.Sprintf("rpc: server error %d: %s", e.Code, e.Message)
}

// ReadResponseHeader reads the JSON-RPC response header.
// The header is the entire response object in JSON-RPC.
func (codec *JSONRPC2ClientCodec) ReadResponseHeader(resp *rpc.Response) error {
	isNotification := codec.wasLastCallNotification
	if isNotification {
		// For notifications, the server MUST NOT send a response.
		// net/rpc client still calls ReadResponseHeader.
		// We return nil immediately. The resp object passed by net/rpc
		// will have zero values for Seq, Error, ServiceMethod, which is acceptable.
		// ReadResponseBody will subsequently be called; our implementation of it
		// already handles the case where lastResultForBody is nil.
		return nil
	}
	codec.debug("ReadResponseHeader - attempting to decode response") // DEBUG
	var jResp JSONRPC2Response
	if err := codec.dec.Decode(&jResp); err != nil {
		fmt.Printf("DEBUG: ReadResponseHeader - decode error: %v\n", err) // DEBUG
		return err
	}
	codec.debug("ReadResponseHeader - decoded response: %+v\n", jResp) // DEBUG

	codec.reqMutex.Lock()
	resp.Seq = jResp.ID
	resp.ServiceMethod = codec.pendingRequests[jResp.ID]
	delete(codec.pendingRequests, jResp.ID)
	codec.reqMutex.Unlock()

	resp.Error = ""
	if jResp.Error != nil {
		errBytes, err := json.Marshal(jResp.Error)
		if err != nil {
			resp.Error = "rpc: failed to parse error object from server"
			return nil
		}
		var serverError jsonError
		if err := json.Unmarshal(errBytes, &serverError); err != nil {
			var strError string
			if umErr := json.Unmarshal(errBytes, &strError); umErr == nil {
				resp.Error = strError
				return nil
			}
			resp.Error = "rpc: failed to unmarshal error object from server"
			return nil
		}
		resp.Error = serverError.Error()
	}

	codec.bodyMutex.Lock() // New: Lock bodyMutex
	if jResp.Result != nil && jResp.Error == nil {
		codec.lastResultForBody = jResp.Result
	} else {
		codec.lastResultForBody = nil // Ensure it's cleared if no result or if error
	}
	codec.bodyMutex.Unlock() // New: Unlock bodyMutex

	// This condition needs to be outside the lock and after error processing
	if jResp.Result == nil && jResp.Error == nil {
		return fmt.Errorf("rpc: server_response: invalid response: missing result and error for id %d", jResp.ID)
	}

	return nil
}

// ReadResponseBody unmarshals the result from the response into the body.
// It uses the lastResultForBody field set by the preceding ReadResponseHeader call.
func (codec *JSONRPC2ClientCodec) ReadResponseBody(body any) error {
	codec.debug("ReadResponseBody - entered. Body type: %T\n", body) // DEBUG
	codec.bodyMutex.Lock()
	resultToUse := codec.lastResultForBody
	// Preventing a potential panic if `lastResultForBody` was nil.
	if resultToUse != nil { // Check before dereferencing for print
		codec.debug("ReadResponseBody - resultToUse: %s\n", string(*resultToUse)) // DEBUG
	} else {
		codec.debug("ReadResponseBody - resultToUse is nil") // DEBUG
	}
	codec.lastResultForBody = nil // Consume it, ensuring it's used only once
	codec.bodyMutex.Unlock()

	if resultToUse == nil { // No result was stored (e.g., error in response, or malformed response from header)
		// This can happen if ReadResponseHeader encountered an error or no result was set.
		// rpc.Client might call ReadResponseBody if ReadResponseHeader didn't return an error,
		// even if resp.Error was set (which means no actual result payload).
		// If body is also nil, nothing to do. If body is non-nil, it won't be populated.
		return nil // Nothing to unmarshal.
	}
	// Reliably consume the response data from the JSON decoder by unmarshaling
	//   into a dummy variable if the `body` (reply) argument is `nil` but a result payload (`lastResultForBody`) exists.
	//  This prevents the decoder from getting into an inconsistent state.
	if body == nil { // If rpc.Client Call/Go passes a nil reply value.
		// We still need to consume the result from the decoder stream.
		var dummy any
		err := json.Unmarshal(*resultToUse, &dummy) // Consume into dummy
		if err != nil {
			codec.debug("ReadResponseBody - unmarshal to dummy error: %v\n", err) // DEBUG
		}
		return err
	}

	// Normal case: body is not nil, resultToUse is not nil.
	err := json.Unmarshal(*resultToUse, body)
	if err != nil {
		codec.debug("ReadResponseBody - unmarshal error: %v\n", err) // DEBUG
	}
	return err // Return the original error
}

// flush checks if the underlying connection implements a Flush method and calls it.
// This is useful for buffered writers.
func (codec *JSONRPC2ClientCodec) flush() error {
	type flusher interface {
		Flush() error
	}
	if f, ok := codec.c.(flusher); ok {
		return f.Flush()
	}
	return nil
}

// Close closes the underlying connection.
func (codec *JSONRPC2ClientCodec) Close() error {
	return codec.c.Close()
}

// Dial connects to a JSON-RPC server at the specified network address.
func Dial(network, address string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	codec := NewJSONRPC2ClientCodec(conn)
	return rpc.NewClientWithCodec(codec), nil
}

// debug conditionally prints a debug message if the Debug field is true.
func (codec *JSONRPC2ClientCodec) debug(format string, args ...any) {
	if codec.Debug {
		log.Printf("DEBUG_CODEC: "+format, args...)
	}
}
