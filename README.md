<p align="center">
  <img src="logo.png" alt="Potacast" width="200"/>
</p>

<h1 align="center">Potacast</h1>

<p align="center">
  <strong>Run GGUF models locally with llama.cpp</strong>
</p>

<p align="center">
  Download models from Hugging Face, serve them with an OpenAI-compatible API—no compilation required.
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#usage">Usage</a> •
  <a href="#api">API</a> •
  <a href="#configuration">Configuration</a>
</p>

---

## Features

- **One-command install** — No compilation, no CUDA setup
- **Hugging Face integration** — Pull GGUF models directly from the Hub
- **OpenAI-compatible API** — Drop-in replacement for `openai` Python/JS clients
- **Interactive chat** — Terminal chat with `/clear`, `/model`, multi-line input
- **Resumable downloads** — Breakpoint resume and skip-if-exists
- **Model management** — Search, export, import, list with size and sort

## Installation

### Linux & macOS

```bash
curl -fsSL https://raw.githubusercontent.com/potacast/potacast/main/scripts/install.sh | sh
```

Install a specific version:

```bash
POTACAST_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/potacast/potacast/main/scripts/install.sh | sh
```

### Build from Source

Requires [Go 1.21+](https://go.dev/dl/).

```bash
git clone https://github.com/potacast/potacast.git
cd potacast
go build -o potacast ./cmd/potacast
```

## Quick Start

```bash
# 1. Download a model from Hugging Face
potacast pull bartowski/Llama-3.2-1B-Instruct-GGUF

# 2. Start interactive chat (server starts automatically)
potacast run

# 3. Or run with a specific model
potacast run bartowski/Llama-3.2-1B-Instruct-GGUF
```

The server runs at `http://127.0.0.1:8080` with an OpenAI-compatible API.

## Usage

### Models

| Command | Description |
|---------|-------------|
| `potacast pull <model-id>` | Download a GGUF model from Hugging Face (supports resume) |
| `potacast list` | List downloaded models with size |
| `potacast list --sort size` | Sort by size (or `name`, `mtime`) |
| `potacast search <query>` | Search GGUF models on Hugging Face |
| `potacast search <query> --limit 10` | Limit search results |
| `potacast info <model>` | Show model path, size, quantization |
| `potacast rm <model>` | Remove a downloaded model |
| `potacast export <model>` | Export model to `.tar.gz` |
| `potacast import <file>` | Import from `.tar.gz` or `.zip` |

### Chat & Server

| Command | Description |
|---------|-------------|
| `potacast run [model]` | Interactive chat (starts server if needed) |
| `potacast run --no-interactive` | Read prompt from stdin, output to stdout |
| `potacast server start` | Start API server in background |
| `potacast server stop` | Stop the server |
| `potacast server status` | Show status, port, PID |
| `potacast ps` | List models currently loaded (like `ollama ps`) |

### Configuration

| Command | Description |
|---------|-------------|
| `potacast config init` | Create default config file |
| `potacast config path` | Print config file path |

### Model ID Format

| Format | Example |
|--------|---------|
| `org/model` | `bartowski/Llama-3.2-1B-Instruct-GGUF` (defaults to Q4_K_M) |
| `org/model:Q4_K_M` | Specify quantization |
| `org/model:branch` | Use a specific branch |
| `org/model:branch:Q4_K_M` | Branch and quantization |

### Chat Commands (interactive mode)

| Command | Description |
|---------|-------------|
| `/bye`, `/exit`, `/quit` | Exit chat |
| `/clear` | Clear conversation history |
| `/model` | List models and switch |
| `/model <name>` | Switch to model directly |
| `\` at end of line | Continue multi-line input |
| Empty line | Submit multi-line input |
| Ctrl+C | Interrupt generation |

### Server Flags

```bash
potacast server start [flags]
```

| Flag | Description |
|------|-------------|
| `--parallel N` | Concurrent slots (-1 = auto) |
| `--ctx N` | Context window size |
| `--threads N` | CPU threads (-1 = auto) |
| `--batch-size N` | Max batch size |
| `--n-predict N` | Max tokens (-1 = unlimited) |
| `--cache-ram N` | KV cache size in MiB |
| `--embeddings` | Enable embeddings (default: true) |
| `--log-file` | Write logs to file (default on non-Linux) |

## API

Potacast serves an OpenAI-compatible API at `http://127.0.0.1:8080/v1`.

### Chat Completions

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "bartowski_Llama-3.2-1B-Instruct-GGUF",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'
```

### Python

```python
from openai import OpenAI

client = OpenAI(base_url="http://127.0.0.1:8080/v1", api_key="not-needed")

response = client.chat.completions.create(
    model="bartowski_Llama-3.2-1B-Instruct-GGUF",
    messages=[{"role": "user", "content": "Hello!"}],
)
print(response.choices[0].message.content)
```

### Non-interactive (pipeline)

```bash
echo "Summarize this: ..." | potacast run --no-interactive
```

## Configuration

Config file: `~/.config/potacast/config.yaml` (or `$XDG_CONFIG_HOME/potacast/config.yaml`)

```yaml
host: "127.0.0.1"
port: 8080
ctx: 4096
parallel: -1      # concurrent slots, -1 = auto
threads: -1       # CPU threads, -1 = auto
batch_size: 2048
n_predict: -1     # max tokens, -1 = unlimited
cache_ram: 8192   # KV cache MiB
embeddings: true  # enable chat + embedding models
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `POTACAST_HOST` | Override config host |
| `POTACAST_PORT` | Override config port |
| `HF_TOKEN` | Hugging Face token (for gated models) |

### Logs

**Linux (systemd):**

```bash
journalctl --user -u potacast-server -f
```

**macOS / non-Linux:** Logs are written to `~/.local/share/potacast/logs/llama-server.log` by default.

## Gated Models

For gated models on Hugging Face:

```bash
export HF_TOKEN=your_token_here
potacast pull meta-llama/Llama-3.2-1B-Instruct-GGUF
```

## Backend

Potacast uses [llama.cpp](https://github.com/ggml-org/llama.cpp) under the hood. The `llama-server` binary is downloaded automatically on first use.

## License

MIT
