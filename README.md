A simple AI code editing agent

## Use

1. Change `env` to `.env`
2. Add your Anthropic API key to `ANTHROPIC_API_KEY="your-api-key-here"` in the `.env` file
3. Install the [Go programming language](https://go.dev/doc/install)
4. Execute `go run ./main.go` in the terminal (Make sure to `cd` into the folder holding the file)

## TODOs

- [>] Add prompting
- [ ] Options for local models via Ollama?
- [ ] Switching between different inference engines (OpenAI/Anthropic/etc.)
- [ ] Make the output coming out smoother (streaming?)
- [ ] Add option to judge code?
- [ ] Send image content block (For what? We dont know-keep going!)
- [ ] Add thinking option (Start from Sonnet 3.7+)
- [ ] MCP server?

Tools

- [ ] Data parser
- [ ] Statistical Analysis
- [ ] Web search
- [ ] Knowledge base query
- [ ] Document generator
- [ ] Code generator from natural language
- [ ] Form builder
- [ ] Scheduling assistant
- [ ] Code base search
- [ ] Propose & run terminal commands
- [ ] Grep search
- [ ] Fuzzy file search
- [ ] Delete files
- [ ] Call smarter models
- [ ] Retrieve the history of recent changes

## Refs

- [anthropic-sdk-go (Official package from Anthropic)](https://github.com/anthropics/anthropic-sdk-go)
- [openai-sdk-go](https://github.com/openai/openai-go)
- [anthropic-sdk-go (Unofficial package)](https://github.com/unfunco/anthropic-sdk-go)
- [cline](https://github.com/cline/cline)
- [serena](https://github.com/oraios/serena)
- [code-judger](https://github.com/mrnugget/code-judger)
- [jan](https://github.com/menloresearch/jan/blob/dev/core/src/types/model/modelEntity.ts#L16)
- [mcp-go](https://github.com/mark3labs/mcp-go/tree/main)
