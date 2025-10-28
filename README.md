[![justforfunnoreally.dev badge](https://img.shields.io/badge/justforfunnoreally-dev-9ff)](https://justforfunnoreally.dev)

<div class="title-block" style="text-align: center;" align="center">

# Clue - Simple AI Coding Agent in Go

<p><img title="clue logo" src="assets/images/clue-logo.svg" width="320" height="320"></p>

</div>

If this proves to be helpful to anyone, consider it my thanks to the open-source community :)

(Important) Read through this wonderful article on [how to build an agent by Thorsten Ball](https://ampcode.com/how-to-build-an-agent) and follow along if possible

## Dependencies

[ripgrep](https://github.com/BurntSushi/ripgrep)

## Installation

1. Add Anthropic API key as an environment variable with `export ANTHROPIC_API_KEY="your-api-key-here"`
2. Run the installation script for the latest version (Linux only at the moment):

```bash
curl -fsSL https://raw.githubusercontent.com/honganh1206/clue/main/scripts/install.sh | sudo -E bash
```

## MCP

To add MCP servers to clue:

```sh
clue mcp --server-cmd "my-server:npx @modelcontextprotocol/server-everything"
```

## Breaking Changes

> **⚠️ WARNING**: If you have a running clue daemon from a previous version, you must purge it before installing the new version:

1. Disable the systemd service:

```bash
sudo systemctl disable clue
sudo systemctl stop clue
```

2. Identify the clue process:

```bash
ps aux | grep clue
```

3. Kill the process entirely (replace `<PID>` with the actual process ID):

```bash
kill -9 <PID>
```

4. Remove the service file:

```bash
sudo rm /etc/systemd/system/clue.service
sudo systemctl daemon-reload
```

5. Move the existing `conversation.db` from `~/.local/.clue` to `~/.clue` and rename the database to `clue.db`

## Development

```bash
make serve # Run the server
make # Run the agent
```

[References](./docs/References.md)

