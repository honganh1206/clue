package main

import (
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"os/exec"

	"github.com/honganh1206/clue/mcp/jsonrpc"
)

// SubprocessReadWriteCloser wraps the stdin and stdout of an exec.Cmd
// to be used as an io.ReadWriteCloser for RPC.
// It also manages the lifecycle of the command.
type SubprocessReadWriteCloser struct {
	io.WriteCloser // Stdin of the subprocess
	io.ReadCloser  // Stdout of the subprocess
	cmd            *exec.Cmd
}

// NewSubprocessReadWriteCloser creates a new SubprocessReadWriteCloser.
// It requires the caller to have already started the command.
// TeeReadCloser wraps an io.Reader (the TeeReader) and an io.Closer (the original stdout pipe)
// to satisfy the io.ReadCloser interface.
type TeeReadCloser struct {
	reader io.Reader
	closer io.Closer
}

// Read reads from the TeeReader.
func (trc *TeeReadCloser) Read(p []byte) (n int, err error) {
	return trc.reader.Read(p)
}

// Close closes the underlying stdout pipe.
func (trc *TeeReadCloser) Close() error {
	return trc.closer.Close()
}

func NewSubprocessReadWriteCloser(cmd *exec.Cmd, stdin io.WriteCloser, stdout io.ReadCloser) *SubprocessReadWriteCloser {
	return &SubprocessReadWriteCloser{
		WriteCloser: stdin,
		ReadCloser:  stdout,
		cmd:         cmd,
	}
}

// Close closes the stdin and stdout pipes and waits for the subprocess to exit.
func (s *SubprocessReadWriteCloser) Close() error {
	var errs []error

	// Close stdin first to signal EOF to the subprocess
	if err := s.WriteCloser.Close(); err != nil {
		errString := fmt.Sprintf("failed to close subprocess stdin: %v", err)
		log.Println(errString)
		errs = append(errs, fmt.Errorf(errString))
	}

	// Closing stdout isn't strictly necessary for the subprocess to terminate based on stdin EOF,
	// but good practice for resource cleanup on our side.
	if err := s.ReadCloser.Close(); err != nil {
		errString := fmt.Sprintf("failed to close subprocess stdout: %v", err)
		log.Println(errString)
		errs = append(errs, fmt.Errorf(errString))
	}

	// Wait for the command to exit and release its resources.
	if err := s.cmd.Wait(); err != nil {
		// log.Printf("Subprocess wait error: %v (ExitCode: %d)", err, s.cmd.ProcessState.ExitCode())
		// Differentiate between non-zero exit and other errors if needed
		if exitErr, ok := err.(*exec.ExitError); ok {
			errString := fmt.Sprintf("subprocess exited with error: %v, stderr: %s", exitErr, string(exitErr.Stderr))
			log.Println(errString)
			errs = append(errs, fmt.Errorf(errString))
		} else {
			errString := fmt.Sprintf("failed to wait for subprocess: %v", err)
			log.Println(errString)
			errs = append(errs, fmt.Errorf(errString))
		}
	} else {
		log.Printf("Subprocess exited cleanly (PID: %d).", s.cmd.ProcessState.Pid())
	}

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d error(s) during close: %v", len(errs), errs)
	}
	return nil
}

func main() {
	log.Println("Starting MCP tester...")

	// 1. Prepare the command
	cmd := exec.Command("uvx", "mcp-server-fetch")

	// 2. Get pipes for stdin and stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout pipe: %v", err)
	}

	// Create a TeeReader to simultaneously write to os.Stdout and be read by the RPC client.
	teeStdoutReader := io.TeeReader(stdoutPipe, os.Stdout)
	// Wrap the TeeReader and the original stdoutPipe.Close method in our TeeReadCloser.
	teedStdout := &TeeReadCloser{reader: teeStdoutReader, closer: stdoutPipe}

	// Optional: Capture stderr for debugging the subprocess
	cmd.Stderr = os.Stderr // Or a bytes.Buffer if you want to capture it

	// 3. Start the command
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}
	log.Printf("Subprocess started (PID: %d)", cmd.Process.Pid)

	// 4. Create the ReadWriteCloser using the teed stdout
	spRwc := NewSubprocessReadWriteCloser(cmd, stdin, teedStdout)

	// 5. Create the JSON-RPC client
	// The mcp.NewJSONRPC2ClientCodec comes from the mcp package we wrote earlier.
	// Ensure github.com/dhamidi/smolcode/mcp is correct in imports.
	codec := jsonrpc.NewJSONRPC2ClientCodec(spRwc)
	codec.Debug = true // Enable debug logging in the codec
	client := rpc.NewClientWithCodec(codec)
	defer func() {
		log.Println("Closing RPC client and subprocess ReadWriteCloser...")
		if err := client.Close(); err != nil {
			log.Printf("Error closing client/subprocess: %v", err)
		}
		log.Println("Client closed.")
	}()

	// 6. Prepare and send the initialization request
	log.Println("Sending RPC request to 'initialize'...")
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "smolcode",
			"version": "1.0.0",
		},
	}
	var initReply interface{}
	err = client.Call("initialize", initParams, &initReply)
	if err != nil {
		log.Fatalf("RPC call 'initialize' failed: %v", err)
	}
	// NOT HERE YET
	log.Println("RPC call 'initialize' successful.")
	fmt.Printf("Response from 'initialize': %+v\n", initReply)

	// Send notifications/initialized notification
	// as required by MCP specification
	log.Println("Sending RPC notification to 'notifications/initialized'...")
	// For notifications, params can be nil if no params are expected,
	// or an empty map. The JSON-RPC spec for this notification shows no params.
	var initializedReply interface{}                                       // For Call, a reply arg is needed. Server should not send a JSON-RPC response body for a notification.
	err = client.Call("notifications/initialized", nil, &initializedReply) // Using nil for params
	if err != nil {
		// net/rpc's Call expects a response. If the server sends no JSON content back for a notification (correct behavior),
		// the decoder might return EOF or similar. This may not be a fatal error for the notification itself.
		log.Printf("RPC notification 'notifications/initialized' completed; err (may be expected for notifications): %v", err)
	} else {
		log.Println("RPC notification 'notifications/initialized' sent successfully (and a response was unexpectedly received).")
	}

	// 7. Prepare the 'tools/list' request
	// As per spec: id 1, method: "tools/list", and an empty params object.
	// The net/rpc client handles ID generation. We just provide method and params.
	listParams := make(map[string]interface{}) // Empty params object
	var listReply interface{}                  // To store the generic JSON response

	log.Println("Sending RPC request to 'tools/list'...")
	// 8. Perform the 'tools/list' RPC call
	err = client.Call("tools/list", listParams, &listReply)
	if err != nil {
		// If the error is rpc.ErrShutdown, it means the client was closed, possibly due to subprocess exit.
		// The deferred Close() also calls cmd.Wait() which might log more details about subprocess exit.
		log.Fatalf("RPC call 'tools/list' failed: %v", err)
	}

	// 9. Output the 'tools/list' response
	log.Println("RPC call 'tools/list' successful.")
	fmt.Printf("Response from 'tools/list': %+v\n", listReply)

	// 10. Prepare and send the 'tools/call' request for the 'fetch' tool
	log.Println("Sending RPC request to 'tools/call' for 'fetch' tool...")
	toolCallParams := map[string]interface{}{
		"name": "fetch",
		"arguments": map[string]interface{}{
			"url": "https://news.ycombinator.com",
			// Omitting other args to use defaults (max_length, start_index, raw)
		},
	}
	var toolCallReply interface{}
	err = client.Call("tools/call", toolCallParams, &toolCallReply)
	if err != nil {
		log.Fatalf("RPC call 'tools/call' for 'fetch' failed: %v", err)
	}
	log.Println("RPC call 'tools/call' for 'fetch' successful.")
	fmt.Printf("Response from 'tools/call' (fetch):\n%+v\n", toolCallReply)
	log.Println("MCP tester finished successfully.")
}
