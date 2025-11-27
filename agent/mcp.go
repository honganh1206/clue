package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/honganh1206/tinker/mcp"
	"github.com/honganh1206/tinker/tools"
)

func (a *Agent) RegisterMCPServers() {
	// fmt.Printf("Initializing MCP servers based on %d configurations...\n", len(a.mcp.ServerConfigs))

	for _, serverCfg := range a.MCP.ServerConfigs {
		// fmt.Printf("Attempting to create MCP server instance for ID %s (command: %s)\n", serverCfg.ID, serverCfg.Command)
		server, err := mcp.NewServer(serverCfg.ID, serverCfg.Command)
		if err != nil {
			// TODO: Better error handling
			continue
		}

		if server == nil {
			fmt.Fprintf(os.Stderr, "Error creating MCP server instance for ID %s (command: %s): NewServer returned nil\\n", serverCfg.ID, serverCfg.Command)
			continue
		}

		if err := server.Start(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting MCP server %s (command: %s): %v\n", serverCfg.ID, serverCfg.Command, err)
			continue
		}

		// fmt.Printf("MCP Server %s started successfully.\n", serverCfg.ID)
		a.MCP.ActiveServers = append(a.MCP.ActiveServers, server)

		// fmt.Printf("Fetching tools from MCP server %s...\n", server.ID())
		tool, err := server.ListTools(context.Background()) // Using context.Background() for now
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing tools from MCP server %s: %v\\n", server.ID(), err)
			// We might still want to keep the server active even if listing tools fails initially.
			// Depending on desired robustness, could 'continue' here or allow agent to proceed.
			continue
			// return
		}
		// fmt.Printf("Fetched %d tools from MCP server %s\n", len(tool), server.ID())
		a.MCP.Tools = append(a.MCP.Tools, tool)

		for _, t := range tool {
			toolName := fmt.Sprintf("%s_%s", server.ID(), t.Name)

			decl := &tools.ToolDefinition{
				Name:        toolName,
				Description: t.Description,
				InputSchema: t.InputSchema,
			}

			a.ToolBox.Tools = append(a.ToolBox.Tools, decl)

			a.MCP.ToolMap[toolName] = mcp.ToolDetails{
				Server: server,
				Name:   t.Name,
			}
		}
	}

	// Print all MCP tools that were added
	if len(a.MCP.ToolMap) > 0 {
		var mcpToolNames []string
		for toolName := range a.MCP.ToolMap {
			mcpToolNames = append(mcpToolNames, toolName)
		}
		// fmt.Printf("Added MCP tools to agent toolbox: %v\n", mcpToolNames)
	}
}

func (a *Agent) ShutdownMCPServers() {
	fmt.Println("shutting down MCP servers...")
	for _, s := range a.MCP.ActiveServers {
		fmt.Printf("closing MCP server: %s\n", s.ID())
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing MCP server %s: %v\n", s.ID(), err)
		} else {
			fmt.Printf("MCP server %s closed successfully\n", s.ID())
		}
	}
}
