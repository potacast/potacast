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

# Start the server (OpenAI-compatible API at http://127.0.0.1:8080)
potacast run

# Stop the server
potacast stop
```

## Commands

| Command | Description |
|---------|-------------|
| `potacast pull <model-id>` | Download a GGUF model from Hugging Face |
| `potacast list` | List downloaded models |
| `potacast run` | Start llama-server (router mode, multi-model) |
| `potacast serve` | Alias for `run` |
| `potacast stop` | Stop the server |
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
