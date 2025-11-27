[![justforfunnoreally.dev badge](https://img.shields.io/badge/justforfunnoreally-dev-9ff)](https://justforfunnoreally.dev)

# Tinker

## Dependencies

[ripgrep](https://github.com/BurntSushi/ripgrep)

## Installation

1. Add API keys as an environment variable

```bash
export ANTHROPIC_API_KEY="your-api-key-here"
export GOOGLE_API_KEY="your-api-key-here"
```

2. Run the installation script for the latest version (Linux only at the moment):

```bash
curl -fsSL https://raw.githubusercontent.com/honganh1206/tinker/main/scripts/install.sh | sudo -E bash
```

## MCP

To add MCP servers to tinker:

```sh
tinker mcp --server-cmd "my-server:npx @modelcontextprotocol/server-everything"
```

## Breaking Changes

> **⚠️ WARNING**: If you have a running tinker daemon from a previous version, you must purge it before installing the new version:

1. Disable the systemd service:

```bash
sudo systemctl disable tinker
sudo systemctl stop tinker
```

2. Identify the tinker process:

```bash
ps aux | grep tinker
```

3. Kill the process entirely (replace `<PID>` with the actual process ID):

```bash
kill -9 <PID>
```

4. Remove the service file:

```bash
sudo rm /etc/systemd/system/tinker.service
sudo systemctl daemon-reload
```

5. Move the existing `conversation.db` from `~/.local/.tinker` to `~/.tinker` and rename the database to `tinker.db`

## Development

```bash
make serve # Run the server
make # Run the agent
```

[References](./docs/References.md)
