package mcp

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func makeReq(args map[string]any) mcplib.CallToolRequest {
	return mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Arguments: args},
	}
}

func uvxAvailable() bool {
	_, err := exec.LookPath("uvx")
	return err == nil
}

// ── transcribe_video ─────────────────────────────────────────────────────────

func TestTranscribeVideo_MissingURL(t *testing.T) {
	res, err := handleTranscribeVideo(context.Background(), makeReq(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for missing url")
	}
	text := toolResultText(t, res)
	if !strings.Contains(text, "url is required") {
		t.Errorf("expected 'url is required' in %q", text)
	}
}

func TestTranscribeVideo_InvalidURL(t *testing.T) {
	if !uvxAvailable() {
		t.Skip("uvx not in PATH")
	}
	res, err := handleTranscribeVideo(context.Background(), makeReq(map[string]any{
		"url": "https://invalid.invalid/notavideo",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Errorf("expected IsError=true for invalid URL, got text: %s", toolResultText(t, res))
	}
}

// ── check_dependencies ───────────────────────────────────────────────────────

func TestCheckDependencies_Faster(t *testing.T) {
	res, err := handleCheckDependencies(context.Background(), makeReq(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError=true: %s", toolResultText(t, res))
	}
	text := toolResultText(t, res)
	for _, want := range []string{"uvx", "ffmpeg", "yt-dlp", "whisper-ctranslate2"} {
		if !strings.Contains(text, want) {
			t.Errorf("check_dependencies output missing %q\n\nFull output:\n%s", want, text)
		}
	}
}

func TestCheckDependencies_OpenAI(t *testing.T) {
	res, err := handleCheckDependencies(context.Background(), makeReq(map[string]any{
		"engine": "openai",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := toolResultText(t, res)
	if !strings.Contains(text, "openai-whisper") {
		t.Errorf("expected 'openai-whisper' in output, got:\n%s", text)
	}
}

// ── list_supported_sites ─────────────────────────────────────────────────────

func TestListSupportedSites(t *testing.T) {
	if !uvxAvailable() {
		t.Skip("uvx not in PATH")
	}
	res, err := handleListSupportedSites(context.Background(), makeReq(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("IsError=true: %s", toolResultText(t, res))
	}
	text := toolResultText(t, res)
	for _, want := range []string{"youtube", "vimeo", "twitch"} {
		if !strings.Contains(strings.ToLower(text), want) {
			t.Errorf("list_supported_sites missing %q", want)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func toolResultText(t *testing.T, res *mcplib.CallToolResult) string {
	t.Helper()
	if res == nil {
		t.Fatal("result is nil")
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}
