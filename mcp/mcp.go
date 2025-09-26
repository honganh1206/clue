package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Config struct {
	ServerConfigs []ServerConfig
	ActiveServers []*Server
	Tools         []Tools
	ToolMap       map[string]ToolDetails
}

// Create a Server instance to manage server subprocesses and communication

// Bundle io.Reader, io.Writer and io.Closer(s) for stdio pipes
// to handle streaming data from one program to another
type stdioReadWriteCloser struct {
	io.Reader
	io.Writer
	stdinCloser  io.Closer
	stdoutCloser io.Closer
}

func (s *stdioReadWriteCloser) Close() error {
	stdinCloseErr := s.stdinCloser.Close()
	stdoutCloseErr := s.stdoutCloser.Close()
	if stdinCloseErr != nil {
		return stdinCloseErr
	} else if stdoutCloseErr != nil {
		return stdoutCloseErr
	} else {
		return nil
	}
}

// Represent an MCP server process and the client to communicate with it
type Server struct {
	id        string
	cmdPath   string
	cmdArgs   []string
	proc      *exec.Cmd
	rpcClient *Client
	// Close the subprocess' pipe
	closer io.Closer
	// Protect access to requestIDCounter
	requestIDLock sync.Mutex
	// Generate unique JSON-RPC request IDs
	requestIDCounter int64
}

func NewServer(id, cmd string) (*Server, error) {
	// For now, cmd is just an executable name with no args
	// TODO: Implement command splitting
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("mcp server: cmd cannot be empty")
	}

	cmdPath := parts[0]
	var cmdArgs []string
	if len(parts) > 1 {
		cmdArgs = parts[1:]
	}

	return &Server{
		id:               id,
		cmdPath:          cmdPath,
		cmdArgs:          cmdArgs,
		requestIDCounter: 0,
	}, nil
}

// Start the server subprocess and perform the initialization handshake
func (s *Server) Start(ctx context.Context) error {
	s.proc = exec.CommandContext(ctx, s.cmdPath, s.cmdArgs...)

	// Create file descriptors for stdin
	stdin, err := s.proc.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcp server: failed to get stdin pipe %w", err)
	}

	// Create file descriptors for stdout
	stdout, err := s.proc.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mcp server: failed to get stdout pipe %w", err)
	}

	rwc := &stdioReadWriteCloser{
		Reader: stdout,
		Writer: stdin,
		// We need to change this?
		stdinCloser:  stdin,
		stdoutCloser: stdout,
	}

	s.closer = rwc

	transport := NewStdioTransport(rwc)
	s.rpcClient = NewClient(transport)

	if err := s.proc.Start(); err != nil {
		return fmt.Errorf("mcp server: failed to start server process: %w", err)
	}

	go func() {
		err := s.rpcClient.Listen()
		// Check if file descriptors for stdin/stdout are closed
		if err != nil && err != io.EOF && err != context.Canceled && !strings.Contains(err.Error(), "file already closed") {
			fmt.Fprintf(os.Stderr, "MCP client listener error: %v\n", err)
		}
	}()

	initParams := &InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]any{},
		ClientInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name: "clue-mcp-client",
			// TODO: Update per clue version?
			Version: "0.1.0",
		},
	}

	var initResult InitializeResult
	callArgs := &ClientCallArgs{
		Method: "initialize",
		Params: initParams,
	}

	if err := s.rpcClient.Call(ctx, callArgs, &initResult); err != nil {
		// Clean up immediately if handshake fails
		// TODO: error is ignored here
		_ = s.Close()
		return fmt.Errorf("mcp server: jsonrpc call to 'initialize' failed: %w", err)
	}

	notifyArgs := &ClientNotifyArgs{
		Method: "notifications/initialized",
	}

	if err := s.rpcClient.Notify(ctx, notifyArgs); err != nil {
		_ = s.Close()
		return fmt.Errorf("mcp server: jsonrpc notify to 'notifications/initialized' failed: %w", err)
	}

	return nil
}

// Shutdown the server and clean up resources
func (s *Server) Close() error {
	var firstErr error

	if s.rpcClient != nil {
		if err := s.rpcClient.Close(); err != nil {
			firstErr = fmt.Errorf("mcp server: failed to close rpc client: %w", err)
		}
	}

	// Close the pipes (reader/writer/closer for the transport)
	if s.closer != nil {
		if err := s.closer.Close(); err != nil {
			if firstErr == nil {
				// TODO: Still error when close with SIGTERM
				firstErr = fmt.Errorf("mcp server: failed to close server pipes: %w", err)
			} else {
				fmt.Fprintf(os.Stderr, "additional error while closing server pipes: %v\n", err)
			}
		}
	}

	// Terminate the server subprocess
	if s.proc != nil && s.proc.Process != nil {
		// Send the process an interrupt
		if err := s.proc.Process.Signal(os.Interrupt); err != nil {
			// Interrupt fails, try to kill
			if killErr := s.proc.Process.Kill(); killErr != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("mcp server: failed to kill server pipes: %w", killErr)
				} else {
					fmt.Fprintf(os.Stderr, "additional error while closing server pipes: %v\n", err)
				}
			}
		}
	}

	// Wait for the process to exit to release resources.
	// We handle wait error when Signal/Kill causes unexpected erors
	_, waitErr := s.proc.Process.Wait()
	if waitErr != nil && !strings.Contains(waitErr.Error(), "signal: interrupt") && !strings.Contains(waitErr.Error(), "exit status 1") && !strings.Contains(waitErr.Error(), "killed") {
		if firstErr == nil {
			if !strings.Contains(waitErr.Error(), "Wait was already called") {
				firstErr = fmt.Errorf("mcp server: error waiting for server process to exit: %w", waitErr)
			} else {

				fmt.Fprintf(os.Stderr, "additional error while closing server pipes: %v\n", waitErr)
			}
		}
	}

	return firstErr
}

// Send a "tools/call" request to the server for the specified tool
func (s *Server) Call(ctx context.Context, toolName string, args map[string]any) ([]ToolResultContent, error) {
	callParams := &ToolsCallParams{
		Name:      toolName,
		Arguments: args,
	}

	var callResult ToolsCallResult

	callArgs := &ClientCallArgs{
		Method: "tools/call",
		Params: callParams,
	}

	if err := s.rpcClient.Call(ctx, callArgs, &callResult); err != nil {
		return nil, fmt.Errorf("mcp server: jsonrpc call to 'tools/call' (tool: %s) failed: %w", toolName, err)
	}

	if callResult.IsError {
		// For now, we return a generic error
		// and we can try a more sophisticated error handling method in the future
		// like extracting detail from callResult.Content
		if len(callResult.Content) > 0 && callResult.Content[0].Type == "text" {
			return callResult.Content, fmt.Errorf("mcp server: tool call for '%s' failed with server-side error: %s", toolName, callResult.Content[0].Text)
		}
		return callResult.Content, fmt.Errorf("mcp server: tool call for '%s' failed with server-side error", toolName)
	}

	return callResult.Content, nil
}

func (s *Server) ListTools(ctx context.Context) (Tools, error) {
	listParams := &ToolsListParams{}
	var listResult ToolsListResult

	callArgs := ClientCallArgs{
		Method: "tools/list",
		Params: listParams,
	}

	if err := s.rpcClient.Call(ctx, &callArgs, &listResult); err != nil {
		return nil, fmt.Errorf("mcp server: jsonrpc call to 'tools/list' failed: %w", err)
	}

	// TODO: Handle pagination using NextCursor
	return listResult.Tools, nil
}

func (s *Server) ID() string {
	return s.id
}
