# Potacast

Run GGUF models locally with llama.cpp. Download models from Hugging Face, serve them with an OpenAI-compatible API—no compilation required.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/potacast/potacast/main/scripts/install.sh | sh
```

Or with a specific version:

```bash
POTACAST_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/potacast/potacast/main/scripts/install.sh | sh
```

## Quick Start

```bash
# Download a model from Hugging Face
potacast pull bartowski/Llama-3.2-1B-Instruct-GGUF

# Or specify quantization: org/model:Q4_K_M
potacast pull bartowski/Llama-3.2-1B-Instruct-GGUF:Q4_K_M

# List downloaded models
potacast list

# Start interactive chat (starts server in background if needed)
potacast run

# Or run with a specific model
potacast run bartowski/Llama-3.2-1B-Instruct-GGUF

# Start the server in background (OpenAI API at http://127.0.0.1:8080)
potacast server start

# Stop the server
potacast server stop
```

## Commands

| Command | Description |
|---------|-------------|
| `potacast pull <model-id>` | Download a GGUF model from Hugging Face |
| `potacast list` | List downloaded models |
| `potacast run [model]` | Interactive chat in terminal (like ollama run) |
| `potacast server start` | Start the API server in background |
| `potacast server stop` | Stop the server |
| `potacast rm <model>` | Remove a downloaded model |

## Model ID Format

- `org/model` — Defaults to Q4_K_M quantization
- `org/model:Q4_K_M` — Specify quantization (Q4_K_S, Q5_K_M, etc.)
- `org/model:filename.gguf` — Exact filename

## Configuration

Optional config at `~/.config/potacast/config.yaml`:

```yaml
host: "127.0.0.1"
port: 8080
ctx: 4096
parallel: -1      # concurrent slots, -1 = auto
threads: -1       # CPU threads, -1 = auto
batch_size: 2048
n_predict: -1     # max tokens to generate, -1 = unlimited
cache_ram: 8192   # KV cache MiB
embeddings: true  # enable chat + embedding models (default: true)
```

Server parameters can also be overridden via CLI flags:

```bash
potacast server start --parallel 4 --ctx 8192
```

Chat and embedding models are both supported by default. To disable embedding models, use `--embeddings=false` or set `embeddings: false` in config.yaml.

### Logs (journalctl)

On Linux, `potacast server start` runs via systemd so logs are available:

```bash
journalctl --user -u potacast-server -f
```

## Gated Models

For gated models on Hugging Face, set `HF_TOKEN`:

```bash
export HF_TOKEN=your_token_here
potacast pull meta-llama/Llama-3.2-1B-Instruct-GGUF
```

## Build from Source

```bash
go build -o potacast ./cmd/potacast
```

## License

MIT
