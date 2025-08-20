package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/honganh1206/clue/mcp"
)

func main() {
	log.Println("Starting MCP tester with mcp.Server API...")

	// Overall context with a timeout for all operations
	mainCtx, mainCancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer mainCancel()

	// 1. Instantiate mcp.NewServer
	// The command "uvx mcp-server-fetch" should be passed as a single string.
	// The NewServer function in mcp/mcp.go splits it.
	serverCmd := "uvx mcp-server-fetch"
	if len(os.Args) > 1 {
		// Allow overriding the server command via arguments for flexibility
		// e.g., go run ./mcp/tester/client.go path/to/your/server arg1 arg2
		serverCmd = os.Args[1]
		if len(os.Args) > 2 {
			for _, arg := range os.Args[2:] {
				serverCmd += " " + arg
			}
		}
		log.Printf("Using custom server command: %s", serverCmd)
	} else {
		log.Printf("Using default server command: %s", serverCmd)
	}

	server, _ := mcp.NewServer("fetchserver1", serverCmd)
	if server == nil {
		log.Fatalf("Failed to create mcp.Server instance (NewServer returned nil). Check server command format.")
	}

	// 2. Call server.Start(ctx)
	log.Println("Starting MCP server and performing handshake...")
	startCtx, startCancel := context.WithTimeout(mainCtx, 15*time.Second) // Timeout for start + handshake
	err := server.Start(startCtx)
	startCancel()
	if err != nil {
		log.Fatalf("Failed to start mcp.Server or complete handshake: %v", err)
	}
	log.Println("MCP server started and initialized successfully.")

	// Defer server.Close() to ensure it's called
	defer func() {
		log.Println("Closing MCP server...")
		if err := server.Close(); err != nil {
			log.Printf("Error closing mcp.Server: %v", err)
		}
		log.Println("MCP server closed.")
	}()

	// 3. Call server.ListTools(ctx)
	log.Println("Listing tools from MCP server...")
	listCtx, listCancel := context.WithTimeout(mainCtx, 10*time.Second)
	tools, err := server.ListTools(listCtx)
	listCancel()
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	log.Printf("Successfully listed %d tools:", len(tools))
	for i, tool := range tools {
		log.Printf("  Tool %d: Name: %s, Description: %s", i+1, tool.Name, tool.Description)
		// log.Printf("    Schema: %s", string(tool.RawInputSchema)) // Can be verbose
	}

	// 4. Try to find the "fetch" tool
	fetchToolName := "fetch"
	tool, found := tools.ByName(fetchToolName)
	if !found {
		log.Printf("Tool '%s' not found in the list. Skipping tool call.", fetchToolName)
	} else {
		log.Printf("Tool '%s' found. Description: %s", tool.Name, tool.Description)
		// log.Printf("Input schema for '%s': %s", tool.Name, string(tool.RawInputSchema))

		// 5. Call server.Call(ctx, "fetch", ...)
		log.Printf("Calling tool '%s' with url: https://example.com", fetchToolName)
		callCtx, callCancel := context.WithTimeout(mainCtx, 15*time.Second)
		callParams := map[string]any{
			"url": "https://example.com",
		}
		results, err := server.Call(callCtx, fetchToolName, callParams)
		callCancel()
		if err != nil {
			log.Fatalf("Failed to call tool '%s': %v", fetchToolName, err)
		}

		log.Printf("Successfully called tool '%s'. Results:", fetchToolName)
		for i, res := range results {
			log.Printf("  Result %d: Type: %s", i+1, res.Type)
			if res.Type == "text" {
				log.Printf("    Text: %s", res.Text)
			} else if res.Type == "image" {
				log.Printf("    MimeType: %s, Data: <base64_data_len:%d>", res.MimeType, len(res.Data))
			}
		}
	}

	// Test a non-existent tool call to see how server/client handles it
	log.Println("Attempting to call a non-existent tool 'nonexistent/tool'...")
	nonExistentToolName := "nonexistent/tool"
	callCtxNonExistent, callCancelNonExistent := context.WithTimeout(mainCtx, 10*time.Second)
	nonExistentParams := map[string]any{"param": "value"}
	_, err = server.Call(callCtxNonExistent, nonExistentToolName, nonExistentParams)
	callCancelNonExistent()
	if err != nil {
		log.Printf("Correctly received error for non-existent tool '%s': %v", nonExistentToolName, err)
	} else {
		log.Printf("Warning: Calling non-existent tool '%s' did NOT return an error as expected.", nonExistentToolName)
	}

	log.Println("MCP tester with mcp.Server API finished successfully.")
}
