# vidscribe

Video transcription via **yt-dlp** + **whisper-ctranslate2** (faster-whisper engine) — as CLI or MCP server for Claude.

## Inspiration

This project is based on the idea of [nhatvu148/video-transcriber-mcp](https://github.com/nhatvu148/video-transcriber-mcp) — a TypeScript MCP server that combines yt-dlp and OpenAI Whisper for video transcription. We took that concept and rebuilt it in Go with a few key differences:

- **Zero dependency install** via `uvx` — no manual yt-dlp, whisper or Python setup required
- **faster-whisper** (CTranslate2) instead of openai-whisper — significantly faster with lower memory usage
- **Single static binary** — no Node.js/npm runtime needed
- **CLI + MCP in one** — usable both as a standalone command and as an MCP server

## Requirements

- [uv](https://docs.astral.sh/uv/) (provides `uvx` — no separate yt-dlp or whisper install needed)
- `ffmpeg`

## Usage

```bash
# Transcribe a YouTube video
vidscribe "https://youtube.com/watch?v=XYZ"

# With browser cookie auth (for age-restricted / private videos)
vidscribe "https://youtube.com/watch?v=XYZ" --cookies-browser chrome

# Larger model, German language, all output formats
vidscribe "https://youtube.com/watch?v=XYZ" \
  --model medium \
  --language de \
  --format txt,md,json,srt,vtt

# GPU acceleration (float16 is selected automatically when device is auto or cuda)
vidscribe "https://youtube.com/watch?v=XYZ" --model large --device cuda

# Check dependencies
vidscribe --mcp  # then call check_dependencies via MCP
```

## MCP server (Claude integration)

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "vidscribe": {
      "command": "vidscribe",
      "args": ["--mcp"]
    }
  }
}
```

**Available MCP tools:**

| Tool | Description |
|------|-------------|
| `transcribe_video` | Download + transcribe a video URL |
| `check_dependencies` | Verify uvx, ffmpeg, yt-dlp, whisper-ctranslate2 |
| `list_supported_sites` | List all 1000+ yt-dlp platforms |

**`transcribe_video` parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `url` | required | Video URL (http/https only) |
| `model` | `small` | Whisper model: tiny\|base\|small\|medium\|large |
| `language` | `auto` | Language code or `auto` |
| `output_dir` | `./transcripts` | Output directory |
| `cookies_browser` | — | Browser for cookie auth: chrome\|firefox\|safari\|edge\|chromium\|brave\|opera\|vivaldi |
| `cookies_file` | — | Netscape cookie file |
| `js_runtime` | auto | JS runtime: `node:/path/to/node` or `deno:/path/to/deno` |
| `engine` | `faster` | Engine: `faster`, `openai` or `parakeet` |
| `device` | `auto` | Compute device: `auto`, `cpu`, `cuda`. `auto` selects CUDA if available |
| `compute_type` | `float16`/`int8` | Quantization: `int8`, `int8_float16`, `float16`, `float32`. Defaults to `float16` for CUDA, `int8` for CPU |
| `format` | `txt,md` | Output formats: txt, md, json, srt, vtt |

## Platform support

| Platform | Status | Notes |
|----------|--------|-------|
| Linux x86\_64 | ✅ Full | Tested |
| Linux arm64 | ✅ Full | Falls back to openai-whisper if CTranslate2 wheels unavailable |
| macOS x86\_64 | ✅ Full | Install ffmpeg via `brew install ffmpeg` |
| macOS arm64 | ✅ Full | Apple Silicon native; install ffmpeg via `brew install ffmpeg` |
| Windows x86\_64 | ✅ Full | Install ffmpeg via `winget install ffmpeg` |
| Windows arm64 | ⚠️ Partial | Transcription may fail (no CTranslate2/PyTorch arm64 Windows wheels); yt-dlp and MCP server work |

## YouTube compatibility

yt-dlp ≥ 2025 requires a JavaScript runtime for YouTube extraction. vidscribe auto-detects `node` from PATH and passes it automatically. If extraction fails, install Node.js or deno and ensure it is in PATH. On Linux, `secretstorage` is injected automatically via `uvx --with secretstorage` to enable browser cookie decryption without manual Python package installation.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `small` | Whisper model: tiny\|base\|small\|medium\|large |
| `--language` | `auto` | ISO 639-1 language code or `auto` |
| `--output-dir` | `./transcripts` | Output directory |
| `--cookies-browser` | — | Browser for cookie auth: chrome\|firefox\|safari\|edge |
| `--cookies-file` | — | Netscape cookie file (fallback if secretstorage unavailable) |
| `--js-runtime` | auto | JS runtime for yt-dlp YouTube extraction: `node:/path/to/node` or `deno:/path/to/deno`. Auto-detected from PATH if omitted |
| `--format` | `txt,md` | Output formats: txt, md, json, srt, vtt |
| `--engine` | `faster` | Engine: `faster`, `openai` or `parakeet` (see below) |
| `--device` | `auto` | Compute device: `auto`, `cpu`, `cuda`. `auto` selects CUDA if available, otherwise CPU |
| `--compute-type` | `int8` | Quantization: `int8`, `int8_float16`, `float16`, `float32`. Defaults to `float16` when `--device` is `auto` or `cuda` |
| `--mcp` | — | Start as MCP server (stdio) |
| `--verbose` | — | Verbose output |

### Engine `parakeet`

`--engine parakeet` transcribes with NVIDIA **parakeet-tdt-0.6b-v3** via
[onnx-asr](https://github.com/istupakov/onnx-asr) (installed on demand through `uvx`,
like the other engines). Accuracy sits between whisper-medium and whisper-large-v3
(FLEURS de WER 5.04 % vs. large-v3 4.30 %, far ahead of whisper-small) — without
touching the GPU, which matters when the GPU is busy (e.g. a running game).

Real-world benchmark (2026-07-12, 21.4 min German video, models cached;
Ryzen 7 5800X, RTX 5060 8 GB):

| Mode | Time | × realtime | Quality (de) |
|------|-----:|-----------:|--------------|
| whisper small CUDA fp16 | 33.5 s | 38× | lowest of these |
| whisper medium CUDA fp16 | 64 s | 20× | good |
| **parakeet CPU (VAD)** | **82 s** | **16×** | **between medium and large-v3** |
| whisper large-v3 CUDA fp16 | 100 s | 13× | best |
| whisper small CPU int8 | 281 s | 4.6× | lowest of these |

With the GPU occupied by a game (6.5/8 GB VRAM in use), every whisper CUDA run
crashed with OOM while parakeet still delivered 135 s on the contended CPU —
parakeet is the robust path on a machine that games.

Notes:
- CPU-only by design; `--model`, `--device` and `--compute-type` are ignored.
- Language is auto-detected (25 European languages) **per VAD segment**; `--language` is
  ignored. Known quirk: a very short trailing segment can flip language (observed: a 1 s
  splinter "beim nächsten Video" came back as "Next video.").
- Long audio is chunked with the built-in Silero VAD (required — the model runs out of
  memory beyond ~20 minutes otherwise); segment timestamps feed the `srt`/`vtt` outputs.
- On failure vidscribe falls back to the regular faster-whisper → openai-whisper chain.

## Testing

Tests are organized in three tiers using Go build tags:

```bash
# Unit tests — pure logic, no external deps, <1s
make test

# Smoke tests — requires uvx + ffmpeg in PATH
make test-smoke

# E2E tests — requires uvx + ffmpeg + network, full transcription
make test-e2e

# Performance benchmarks (CPU vs CUDA vs openai-whisper)
make test-bench
```

Override the test video (default: "Me at the zoo", 19s):

```bash
VIDSCRIBE_TEST_URL="https://youtube.com/watch?v=..." make test-e2e
VIDSCRIBE_TEST_BROWSER=firefox make test-e2e
```

| Tier | Build tag | CI | What |
|------|-----------|-----|------|
| Unit | — | Every push | Config, format, download helpers, input validation |
| Smoke | `smoke` | Weekly (integration.yml) | Dependency checks, MCP protocol, yt-dlp listing |
| E2E | `e2e` | Manual | Full pipeline, engine/device comparison, model comparison |
