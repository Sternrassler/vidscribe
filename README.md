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
| `url` | required | Video URL |
| `model` | `small` | Whisper model: tiny\|base\|small\|medium\|large |
| `language` | `auto` | Language code or `auto` |
| `output_dir` | `./transcripts` | Output directory |
| `cookies_browser` | — | Browser for cookie auth: chrome\|firefox\|safari\|edge |
| `cookies_file` | — | Netscape cookie file |
| `js_runtime` | auto | JS runtime: `node:/path/to/node` or `deno:/path/to/deno` |
| `engine` | `faster` | Whisper engine: `faster` or `openai` |
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
| `--engine` | `faster` | Whisper engine: `faster` or `openai` |
| `--device` | `auto` | Compute device: `auto`, `cpu`, `cuda`. `auto` selects CUDA if available, otherwise CPU |
| `--compute-type` | `int8` | Quantization: `int8`, `int8_float16`, `float16`, `float32`. Defaults to `float16` when `--device` is `auto` or `cuda` |
| `--mcp` | — | Start as MCP server (stdio) |
| `--verbose` | — | Verbose output |
