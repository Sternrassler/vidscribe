package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sternrassler/vidscribe/internal/deps"
	"github.com/sternrassler/vidscribe/internal/pipeline"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const version = "0.1.0"

// Serve starts an MCP stdio server and blocks until the client disconnects.
func Serve() error {
	installClaudeCommands()

	s := server.NewMCPServer("vidscribe", version,
		server.WithToolCapabilities(false),
	)

	s.AddTool(transcribeVideoTool(), handleTranscribeVideo)
	s.AddTool(checkDependenciesTool(), handleCheckDependencies)
	s.AddTool(listSupportedSitesTool(), handleListSupportedSites)

	return server.ServeStdio(s)
}

// ── tool: transcribe_video ────────────────────────────────────────────────────

func transcribeVideoTool() mcplib.Tool {
	return mcplib.NewTool("transcribe_video",
		mcplib.WithDescription("Download and transcribe audio from a YouTube or other video URL using yt-dlp and faster-whisper."),
		mcplib.WithString("url",
			mcplib.Required(),
			mcplib.Description("Video URL (YouTube, Vimeo, or any yt-dlp-supported platform)"),
		),
		mcplib.WithString("model",
			mcplib.Description("Whisper model: tiny | base | small | medium | large (default: small)"),
		),
		mcplib.WithString("language",
			mcplib.Description("Language code (e.g. 'de', 'en') or 'auto' for auto-detection (default: auto)"),
		),
		mcplib.WithString("output_dir",
			mcplib.Description("Directory to write transcript files (default: ./transcripts)"),
		),
		mcplib.WithString("cookies_browser",
			mcplib.Description("Browser to load cookies from for authentication: chrome | firefox | safari | edge"),
		),
		mcplib.WithString("cookies_file",
			mcplib.Description("Path to Netscape-format cookie file (fallback when secretstorage is unavailable)"),
		),
		mcplib.WithString("engine",
			mcplib.Description("Whisper engine: faster | openai (default: faster)"),
		),
		mcplib.WithString("format",
			mcplib.Description("Comma-separated output formats: txt, md, json, srt, vtt (default: txt,md)"),
		),
		mcplib.WithString("js_runtime",
			mcplib.Description("JS runtime for yt-dlp YouTube extraction, e.g. 'node:/usr/bin/node' or 'deno:/usr/bin/deno' (auto-detected if omitted)"),
		),
		mcplib.WithString("device",
			mcplib.Description("Compute device: auto | cpu | cuda (default: auto — selects CUDA if available)"),
		),
		mcplib.WithString("compute_type",
			mcplib.Description("Quantization: int8 | int8_float16 | float16 | float32 (default: float16 for CUDA, int8 for CPU)"),
		),
	)
}

// allowedBrowsers is the set of browsers accepted for --cookies-from-browser.
var allowedBrowsers = map[string]bool{
	"chrome": true, "firefox": true, "safari": true, "edge": true,
	"chromium": true, "brave": true, "opera": true, "vivaldi": true,
}

func handleTranscribeVideo(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := req.GetArguments()

	url, _ := args["url"].(string)
	if url == "" {
		return mcplib.NewToolResultError("url is required"), nil
	}

	// URL scheme check
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return mcplib.NewToolResultError("only http:// and https:// URLs are supported"), nil
	}

	cfg := &pipeline.Config{
		URL:      url,
		Model:    stringArg(args, "model", "small"),
		Language: stringArg(args, "language", "auto"),
		Engine:   stringArg(args, "engine", "faster"),
		Formats:  strings.Split(stringArg(args, "format", "txt,md"), ","),
		Device:   stringArg(args, "device", "auto"),
	}
	cfg.CookiesBrowser, _ = args["cookies_browser"].(string)
	cfg.CookiesFile, _ = args["cookies_file"].(string)
	cfg.JSRuntime, _ = args["js_runtime"].(string)

	// Compute type: match CLI auto-selection logic
	cfg.ComputeType = stringArg(args, "compute_type", "")
	if cfg.ComputeType == "" {
		if cfg.Device == "auto" || cfg.Device == "cuda" {
			cfg.ComputeType = "float16"
		} else {
			cfg.ComputeType = "int8"
		}
	}

	// output_dir: resolve to absolute path
	outDir := stringArg(args, "output_dir", "./transcripts")
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return mcplib.NewToolResultError("invalid output_dir: " + err.Error()), nil
	}
	cfg.OutputDir = absOut

	// cookies_browser allowlist
	if cfg.CookiesBrowser != "" && !allowedBrowsers[strings.ToLower(cfg.CookiesBrowser)] {
		return mcplib.NewToolResultError("unsupported browser: " + cfg.CookiesBrowser), nil
	}

	// cookies_file: must be a regular file if specified
	if cfg.CookiesFile != "" {
		if fi, err := os.Stat(cfg.CookiesFile); err != nil || !fi.Mode().IsRegular() {
			return mcplib.NewToolResultError("cookies_file not found or not a regular file: " + cfg.CookiesFile), nil
		}
	}

	// Dependency check (same as CLI)
	if err := deps.Check(cfg.Engine); err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	var logBuf strings.Builder
	paths, err := pipeline.Run(ctx, cfg, &logBuf)
	if err != nil {
		msg := fmt.Sprintf("Transcription failed: %v\n\nLog:\n%s", err, logBuf.String())
		return mcplib.NewToolResultError(msg), nil
	}

	result := fmt.Sprintf("Transcription complete.\n\nFiles written:\n%s",
		pipeline.FormatReport(paths))
	if log := strings.TrimSpace(logBuf.String()); log != "" {
		result += "\nLog:\n" + log
	}
	return mcplib.NewToolResultText(result), nil
}

// ── tool: check_dependencies ─────────────────────────────────────────────────

func checkDependenciesTool() mcplib.Tool {
	return mcplib.NewTool("check_dependencies",
		mcplib.WithDescription("Check whether all required tools (uvx, ffmpeg, yt-dlp, faster-whisper) are available."),
		mcplib.WithString("engine",
			mcplib.Description("Whisper engine to check: faster | openai (default: faster)"),
		),
	)
}

func handleCheckDependencies(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := req.GetArguments()
	engine := stringArg(args, "engine", "faster")

	statuses := deps.Report(engine)

	var sb strings.Builder
	allOK := true
	for _, s := range statuses {
		if s.OK {
			sb.WriteString("✓ ")
		} else {
			sb.WriteString("✗ ")
			allOK = false
		}
		sb.WriteString(s.Name)
		if s.Version != "" {
			sb.WriteString(": ")
			sb.WriteString(s.Version)
		}
		if s.Note != "" {
			sb.WriteString(" (")
			sb.WriteString(s.Note)
			sb.WriteString(")")
		}
		sb.WriteByte('\n')
	}

	if !allOK {
		sb.WriteString("\nSome dependencies are missing. Install uv (https://docs.astral.sh/uv/) and ffmpeg.")
	} else {
		sb.WriteString("\nAll dependencies OK.")
	}

	return mcplib.NewToolResultText(sb.String()), nil
}

// ── tool: list_supported_sites ───────────────────────────────────────────────

func listSupportedSitesTool() mcplib.Tool {
	return mcplib.NewTool("list_supported_sites",
		mcplib.WithDescription("List all video platforms supported by yt-dlp (1000+)."),
	)
}

func handleListSupportedSites(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	cmd := pipeline.YtdlpCmd(ctx, "--list-extractors")
	out, err := cmd.Output()
	if err != nil {
		return mcplib.NewToolResultError("could not list extractors: " + err.Error()), nil
	}
	return mcplib.NewToolResultText(string(out)), nil
}

func stringArg(args map[string]any, key, def string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return def
}
