# clue

A simple CLI-based AI coding agent

If this proves to be helpful to anyone, consider it my thanks to the open-source community :)

## Use

0. (Important) Read through this wonderful article on [how to build an agent by Thorsten Ball](https://ampcode.com/how-to-build-an-agent) and follow along if possible
1. Execute `export ANTHROPIC_API_KEY="your-api-key-here"` in your favorite terminal
2. Install the [Go programming language](https://go.dev/doc/install)
3. Execute `go run ./main.go chat` or `make chat` if you have installed `make` in the terminal (Make sure to `cd` into the folder holding the file)

## Refs

- [anthropic-sdk-go (Official package from Anthropic)](https://github.com/anthropics/anthropic-sdk-go)
- [maestro](https://github.com/Doriandarko/maestro) - Orchestrate subagents
- [bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI
- [claude-engieer](https://github.com/Doriandarko/claude-engineer) - Self-explanatory, lots of tools
- [openai-sdk-go](https://github.com/openai/openai-go)
- [anthropic-sdk-go (Unofficial package)](https://github.com/unfunco/anthropic-sdk-go)
- [ollama APIs](https://pkg.go.dev/github.com/ollama/ollama@v0.6.8/api) - Interact with local models on Ollama
- [ollama-termui](https://github.com/mxyng/ollama-termui) - (Experimental) Terminal interface for Ollama

- [cobra](https://github.com/spf13/cobra) - CLI
- [cline](https://github.com/cline/cline) - Tool ideas, prompts and models?
- [serena](https://github.com/oraios/serena) - Tool ideas, prompts and models?
- [code-judger](https://github.com/mrnugget/code-judger) - Tool
- [jan](https://github.com/menloresearch/jan/blob/dev/core/src/types/model/modelEntity.ts#L16) - Connect to local models?
- [mcp-go](https://github.com/mark3labs/mcp-go/tree/main) - MCP (obv)
- [mcp-language-server](https://github.com/isaacphi/mcp-language-server) - Build on top of mco-go
- [go-grep](https://github.com/rastasheep/go-grep) - Tool
- [aider](https://github.com/Aider-AI/aider) - Terminal-based
- [deepseek-go](https://github.com/cohesion-org/deepseek-go) - Client for DeepSeek and Ollama
- [smolcode](https://github.com/dhamidi/smolcode) - Gemini integration, memory management
- [anthropic-go](https://github.com/madebywelch/anthropic-go) - Details on how to implement messages and message events
- [anthrogo](https://github.com/dleviminzi/anthrogo) Message decoder
- [stainless-api-cli](https://github.com/honganh1206/stainless-api-cli) - How to structure a CLI app professionally?
