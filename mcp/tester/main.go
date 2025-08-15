package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/honganh1206/clue/mcp"
)

// TeeReadCloser wraps an io.Reader (the TeeReader) and an io.Closer (the original stdout pipe)
// to satisfy the io.ReadCloser interface.
// The "tee" prefix comes from the Unix tee command
// to read data from stdin and write it in two places
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

// Transport to communicate with a subprocess (separate OS process my program communicates with) via stdin/stdout
type SubprocessTransport struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader // For ReadBytes?
	stdoutPipe io.ReadCloser // Original pipe for closing?
	stderrPipe io.ReadCloser // For closing and separate reading goroutine

	closed   chan struct{}
	closeErr error
	// Guarantee a specific function/operation is executed once only,
	// even if called concurrently by multiple goroutines
	once sync.Once
}

func NewSubprocessTransport(command string, args ...string) (*SubprocessTransport, error) {
	cmd := exec.Command(command, args...)

	// A pipe is a communicational channel with two endpoints: Writer (sned bytes to pipe) and Reader (read bytes from pipe)
	// Pipes connect Go program to subprocess's stdin, stdout and stderr
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		stdinPipe.Close()
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Tee stdout to os.Stdout without a buffer inbetween
	teeStdoutReader := io.TeeReader(stdoutPipe, os.Stdout)

	stderrPipeForCmd, err := cmd.StderrPipe()
	if err != nil {
		stdinPipe.Close()
		stdoutPipe.Close()
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdinPipe.Close()
		stdoutPipe.Close()
		stderrPipeForCmd.Close()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	log.Printf("Subprocess started (PID: %d): %s %v", cmd.Process.Pid, command, args)

	st := &SubprocessTransport{
		cmd:    cmd,
		stdin:  stdinPipe,
		stdout: bufio.NewReader(teeStdoutReader),
	}

	// Capture and log stderr from stderrPipe
	go func(stderrToRead io.ReadCloser) {
		stderrScanner := bufio.NewScanner(stderrToRead)
		for stderrScanner.Scan() {
			// Could be multiple errors
			// Should we handle the errors on our client-side?
			log.Printf("MCP Server STDERR: %s", stderrScanner.Text())
		}

		if scanErr := stderrScanner.Err(); scanErr != nil {
			log.Printf("error reading subprocess stderr: %v", scanErr)
		}
	}(stderrPipeForCmd)

	return st, nil

}

// Send a pre-formatted JSON-RPC message payload,
// adding a newline delimiter \n as well
func (st *SubprocessTransport) Send(ctx context.Context, payload []byte) error {
	select {
	case <-st.closed:
		// Error with transport
		return st.closeErr
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Nothing here, move on to execute the outer code block
	}

	if _, err := st.stdin.Write(payload); err != nil {
		st.Close()
		return fmt.Errorf("transport: failed to write payload: %w", err)
	}

	if _, err := st.stdin.Write([]byte{'\n'}); err != nil {
		st.Close()
		return fmt.Errorf("transport: failed to write payload: %w", err)
	}

	return nil
}

// Wait for and return the next JSON-RPC message payload from the underlying connection
func (st *SubprocessTransport) Receive(ctx context.Context) ([]byte, error) {
	// Signal completion or timeout
	done := make(chan struct{})
	var readErr error
	var line []byte

	go func() {
		// Read till reaching delimiter
		line, readErr = st.stdout.ReadBytes('\n')
		close(done)
	}()

	select {
	// TODO: Where is send operation to closed chan?
	case <-st.closed:
		// Transport is explicitly closed
		return nil, st.closeErr
	case <-ctx.Done():
		// Context cancelled
		return nil, ctx.Err()
	case <-done:
		// operation completed
	}

	if readErr != nil {
		if readErr == io.EOF {
			if len(line) > 0 {
				log.Printf("Transport: Received EOF with partial data: %s", string(line))
				return trimSpace(line), nil // Return the trimmed partial line
			} else {
				log.Println("Transport: Received EOF from subprocess stdout.")
				st.Close()         // Subprocess exited
				return nil, io.EOF // Signal clean EOF
			}
		}
		select {
		case <-st.closed:
			return nil, st.closeErr
		default:
			log.Printf("Transport: Error reading from subprocess stdout: %v", readErr)
			return nil, fmt.Errorf("transport: receive error: %w", readErr)
		}
	}

	trimmedLine := trimSpace(line)
	if len(trimmedLine) == 0 {
		return nil, fmt.Errorf("transport: received empty line")
	}
	return trimmedLine, nil

}

// Remove leading and trailing ASCII white space
func trimSpace(s []byte) []byte {
	start := 0
	// Determine the white space indexes
	for start < len(s) && isSpace(s[start]) {
		start++
	}
	end := len(s)
	for end > start && isSpace(s[end-1]) {
		end--
	}

	if start == end {
		return s[0:0]
	}
	return s[start:end]

}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// Implement io.Closer for transport.
// Close order: stdin -> stdout -> stderr and wait for it to exit,
// then clean up resources
func (st *SubprocessTransport) Close() error {
	st.once.Do(func() {
		log.Println("subprocess transport: closing...")

		if st.stdin != nil {
			if err := st.stdin.Close(); err != nil {
				log.Printf("SubprocessTransport: failed to close subprocess stdin: %v", err)
				if st.closeErr == nil {
					st.closeErr = err
				}
			}
		}

		if st.stdoutPipe != nil {
			if err := st.stdoutPipe.Close(); err != nil {
				log.Printf("SubprocessTransport: error closing stdout pipe: %v", err)
				if st.closeErr == nil {
					st.closeErr = err
				}
			}
		}

		// Close the stderr pipe that the transport was using.
		if st.stderrPipe != nil {
			if err := st.stderrPipe.Close(); err != nil {
				log.Printf("SubprocessTransport: error closing stderr pipe: %v", err)
				if st.closeErr == nil {
					st.closeErr = err
				}
			}
		}

		waitErrChan := make(chan error, 1)
		go func() {
			// Other than wait for the command to exit
			// we wait for copying to stdin done
			// and copying from stdout and stderr done
			// Must be preceded with cmd.Start()
			waitErrChan <- st.cmd.Wait()
		}()

		select {
		case err := <-waitErrChan:
			if err != nil {
				// How likely this error could occur?
				if exitErr, ok := err.(*exec.ExitError); ok {
					errStr := fmt.Sprintf("subprocess transport: subprocess exited with error: %v. stderr may have more.", exitErr)
					log.Println(errStr)
					if st.closeErr == nil {
						st.closeErr = fmt.Errorf("%s", errStr)
					}
				} else {
					errStr := fmt.Sprintf("SubprocessTransport: failed to wait for subprocess: %v", err)
					log.Println(errStr)
					if st.closeErr == nil {
						st.closeErr = fmt.Errorf("%s", errStr)
					}
				}
			} else {
				log.Printf("subprocessor transport: subprocess with PID %d exited cleanly", st.cmd.Process.Pid)
			}
			// No st.closed closing?
		case <-time.After(5 * time.Second):
			log.Println("subprocess transport: timeout waiting for subprocess to exit. attempt to kill subprocess...")
			if err := st.cmd.Process.Kill(); err != nil {
				log.Printf("subprocess transport: failed to kill subprocess: %v", err)
				if st.closeErr == nil {
					st.closeErr = fmt.Errorf("failed to kill subprocess: %w", err)
				} else {
					log.Println("SubprocessTransport: Subprocess killed.")
					if st.closeErr == nil {
						st.closeErr = fmt.Errorf("subprocess timed out and was killed")
					}
				}
				// Send so channel is safe to be closed?
				<-waitErrChan
			}
			close(st.closed)
			log.Println("subprocess transport: closed")
		}
	})
	return st.closeErr
}

func main() {
	log.Println("Starting MCP tester with jsonrpc2 client...")

	transport, err := NewSubprocessTransport("uvx", "mcp-server-fetch")
	if err != nil {
		log.Fatalf("Failed to create subprocess transport: %v", err)
	}

	client := mcp.NewClient(transport)
	clientCtx, clientCancel := context.WithCancel(context.Background())
	defer clientCancel()

	go func() {
		log.Println("Client listener starting...")
		listenErr := client.Listen()
		if listenErr != nil && listenErr != context.Canceled && listenErr != io.EOF && listenErr.Error() != "context canceled" {
			log.Printf("Client Listen error: %v", listenErr)
		}
		log.Println("Client listener stopped.")
	}()

	defer func() {
		log.Println("Closing RPC client (which will also close transport)...")
		if err := client.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
		log.Println("Client and transport closed.")
	}()

	mainOpCtx, mainOpCancel := context.WithTimeout(clientCtx, 30*time.Second)
	defer mainOpCancel()

	log.Println("Dispatching RPC request to 'initialize'...")
	initParams := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "clue-tester-jsonrpc2",
			"version": "1.0.2",
		},
	}
	var initReply any
	callCtx, callCancel := context.WithTimeout(mainOpCtx, 10*time.Second)

	// Send init request
	err = client.Call(callCtx, &mcp.ClientCallArgs{Method: "initialize", Params: initParams}, &initReply)
	callCancel()
	if err != nil {
		log.Fatalf("RPC call 'initialize' failed: %v", err)
	}
	log.Printf("'initialize' call successful. Reply: %+v\n", initReply)

	log.Println("Sending RPC notification to 'notifications/initialized'...")
	notifyCtx, notifyCancel := context.WithTimeout(mainOpCtx, 5*time.Second)

	// Send noti for init request
	err = client.Notify(notifyCtx, &mcp.ClientNotifyArgs{Method: "notifications/initialized", Params: nil})
	notifyCancel()
	if err != nil {
		log.Printf("RPC notification 'notifications/initialized' failed: %v", err)
	} else {
		log.Println("RPC notification 'notifications/initialized' sent successfully.")
	}

	// Send request to fetch list of tools
	log.Println("Sending RPC request to 'tools/list'...")
	listParams := make(map[string]any)
	var listReply any
	callCtx, callCancel = context.WithTimeout(mainOpCtx, 10*time.Second)
	err = client.Call(callCtx, &mcp.ClientCallArgs{Method: "tools/list", Params: listParams}, &listReply)
	callCancel()
	if err != nil {
		log.Fatalf("RPC call 'tools/list' failed: %v", err)
	}
	log.Println("RPC call 'tools/list' successful.")
	fmt.Printf("Response from 'tools/list': %+v\n", listReply)

	// Send request to use fetch tool
	log.Println("Sending RPC request to 'tools/call' for 'fetch' tool...")
	// Response will be something like "This should be a valid URL"
	toolCallParams := map[string]any{
		"name": "fetch",
		"arguments": map[string]any{
			"url": "https_example_com_this_should_be_fetched",
		},
	}
	var toolCallReply any
	callCtx, callCancel = context.WithTimeout(mainOpCtx, 15*time.Second)
	err = client.Call(callCtx, &mcp.ClientCallArgs{Method: "tools/call", Params: toolCallParams}, &toolCallReply)
	callCancel()
	if err != nil {
		log.Fatalf("RPC call 'tools/call' for 'fetch' failed: %v", err)
	}
	log.Println("RPC call 'tools/call' for 'fetch' successful.")
	fmt.Printf("Response from 'tools/call' (fetch):\n%+v\n", toolCallReply)

	log.Println("MCP tester finished successfully using jsonrpc2 client.")
}
