A simple AI code editing agent

## Use

1. Change `env` to `.env`
2. Add your Anthropic API key to `ANTHROPIC_API_KEY="your-api-key-here"` in the `.env` file
3. Install the [Go programming language](https://go.dev/doc/install)
4. Execute `go run ./main.go` in the terminal (Make sure to `cd` into the folder holding the file)

## TODOs

- [ ] Options for local models via Ollama?
- [ ] Switching between different inference engines (OpenAI/Anthropic/etc.)
- [ ] Make the output coming out smoother (streaming?)
- [ ] Add option to judge code?

## Refs

- [anthropic-sdk-go (Official package from Anthropic)](https://github.com/anthropics/anthropic-sdk-go)
- [anthropic-sdk-go (Unofficial package)](https://github.com/unfunco/anthropic-sdk-go)
- [cline](https://github.com/cline/cline)
- [serena](https://github.com/oraios/serena)
- [code-judger](https://github.com/mrnugget/code-judger)
- [jan](https://github.com/menloresearch/jan/blob/dev/core/src/types/model/modelEntity.ts#L16)
