A simple AI code editing agent

## Use

1. Change `env` to `.env`
2. Add your Anthropic API key to `ANTHROPIC_API_KEY="your-api-key-here"` in the `.env` file
3. Install the [Go programming language](https://go.dev/doc/install)
4. Execute `go run ./main.go` in the terminal (Make sure to `cd` into the folder holding the file)

## TODOs

- [>] Add prompting
- [ ] Options for local models via Ollama?
- [>] Switching between different inference engines (OpenAI/Anthropic/etc.)
- [ ] Make the output coming out smoother (streaming?)
- [ ] Send image content block (For what? We dont know-keep going!)
- [ ] Add thinking option (Start from Sonnet 3.7+)
- [ ] MCP server?

Profiles

- [ ] Write
- [ ] Ask
- [ ] Manual

Tools

- [ ] `copy_path`
- [ ] `create_directory`
- [ ] `web_search`
- [ ] `knowledge_query`
- [ ] `gen_doc`
- [ ] `gen_code`
- [ ] `judge_code`
- [ ] `create_file`
- [ ] `schedule`
- [ ] `grep_search`
- [ ] `propose_cmd`
- [ ] `fuzzy_search`
- [ ] `call_smarter_models`
- [ ] `thinking`

## Refs

- [anthropic-sdk-go (Official package from Anthropic)](https://github.com/anthropics/anthropic-sdk-go)
- [openai-sdk-go](https://github.com/openai/openai-go)
- [anthropic-sdk-go (Unofficial package)](https://github.com/unfunco/anthropic-sdk-go)
- [cline](https://github.com/cline/cline)
- [serena](https://github.com/oraios/serena)
- [code-judger](https://github.com/mrnugget/code-judger)
- [jan](https://github.com/menloresearch/jan/blob/dev/core/src/types/model/modelEntity.ts#L16)
- [mcp-go](https://github.com/mark3labs/mcp-go/tree/main)
- [go-grep](https://github.com/rastasheep/go-grep)
- [deepseek-go (client for DeepSeek and Ollama)](https://github.com/cohesion-org/deepseek-go)
