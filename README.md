<p align="center">
  <a href="https://github.com/blacktop/lifx"><img alt="lifx Logo" src="./docs/logo-light.svg" height="200" /></a>
  <h1 align="center">lifx</h1>
  <h4><p align="center">LIFX Light TUI and MCP Server</p></h4>
  <p align="center">
    <a href="https://github.com/blacktop/lifx/actions" alt="Actions">
          <img src="https://github.com/blacktop/lifx/actions/workflows/go.yml/badge.svg" /></a>
    <a href="https://github.com/blacktop/lifx/releases/latest" alt="Downloads">
          <img src="https://img.shields.io/github/downloads/blacktop/lifx/total.svg" /></a>
    <a href="https://github.com/blacktop/lifx/releases" alt="GitHub Release">
          <img src="https://img.shields.io/github/release/blacktop/lifx.svg" /></a>
    <a href="http://doge.mit-license.org" alt="LICENSE">
          <img src="https://img.shields.io/:license-mit-blue.svg" /></a>
</p>
<br>

A gorgeous terminal UI and MCP server for controlling LIFX lights. Built with Charm (Bubble Tea + Lip Gloss) with support for both LAN and Cloud API backends.

![demo](vhs.gif)

## Features

- **Dual Backend**: supports both LAN protocol and LIFX Cloud API.
- **Groups & Lights**: rooms/groups with per-light controls.
- **Brightness, Color, Kelvin**: keyboard-driven adjustments.
- **Scenes**: activate saved scenes via the API backend.
- **Search**: filter lights by name.
- **MCP Server**: integrate with AI agents via the Model Context Protocol.

## Install

Via Go:

```bash
go install github.com/blacktop/lifx@latest
```

Via [Homebrew](https://brew.sh)

```bash
brew install blacktop/tap/lifx
```

### Backend selection

- Default is **auto**: tries API if `LIFX_API_KEY` is set, otherwise uses LAN.
- Force backend with `--backend`:

```bash
lifx --backend lan
lifx --backend api --api-key "$LIFX_API_KEY"
```

### Environment

Get your API key from [cloud.lifx.com/settings](https://cloud.lifx.com/settings):

```bash
export LIFX_API_KEY=...  # Required for API backend and MCP server
```

## Keybindings

### Navigation

| Key       | Action              |
| --------- | ------------------- |
| `j` / `↓` | Move down           |
| `k` / `↑` | Move up             |
| `Tab`     | Toggle side panel   |

### Light Control

| Key     | Action                    |
| ------- | ------------------------- |
| `Space` | Toggle power              |
| `0-9`   | Set brightness to 10-100% |
| `h`/`l` | Decrease/increase brightness |
| `p`     | Cycle color preset         |
| `w`/`c` | Warmer/cooler temperature  |

### Group Control

| Key | Action                      |
| --- | --------------------------- |
| `a` | Turn all lights in group on |
| `x` | Turn all lights in group off|

### Other

| Key | Action            |
| --- | ----------------- |
| `/` | Search lights     |
| `s` | Scenes modal      |
| `r` | Refresh           |
| `q` | Quit              |

## MCP Server

Run lifx as an MCP (Model Context Protocol) server for AI agent integration:

```bash
lifx mcp
```

### Available Tools

| Tool | Description |
| ---- | ----------- |
| `list_lights` | List all lights with current state (power, brightness, color) |
| `set_power` | Turn a specific light on or off |
| `set_all_power` | Turn all lights on or off |
| `set_brightness` | Set brightness level (0-100) |
| `set_color` | Set hue, saturation, brightness, and/or kelvin |
| `list_scenes` | List all available scenes |
| `activate_scene` | Activate a scene by ID |

### Configuration

#### [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview)

```bash
claude mcp add lifx -e LIFX_API_KEY=your_api_key -- lifx mcp
```

#### [Codex CLI](https://platform.openai.com/docs/guides/mcp)

```bash
codex mcp add lifx --env LIFX_API_KEY=your_api_key -- lifx mcp
```

#### [Gemini CLI](https://github.com/google/gemini-cli)

```bash
gemini mcp add lifx lifx mcp -e LIFX_API_KEY=your_api_key
```

#### [Claude Desktop](https://claude.ai/download)

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "lifx": {
      "command": "lifx",
      "args": ["mcp"],
      "env": {
        "LIFX_API_KEY": "your_api_key"
      }
    }
  }
}
```

### Testing

```bash
make mcp-test
```

## Notes

- LAN protocol requires lights to be on the same network.
- MCP server and scenes require the LIFX Cloud API (`LIFX_API_KEY`).

## License

MIT Copyright (c) 2025 **blacktop**
