//go:build smoke

package mcp

import (
	"context"
	"strings"
	"testing"
)

func TestTranscribeVideo_InvalidURL(t *testing.T) {
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

func TestListSupportedSites(t *testing.T) {
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
