package deps

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Check verifies that the minimum required tools are in PATH.
func Check(engine string) error {
	if _, err := exec.LookPath("uvx"); err != nil {
		return fmt.Errorf("uvx not found — install uv: https://docs.astral.sh/uv/")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found — install via: apt install ffmpeg")
	}
	return nil
}

type DepStatus struct {
	Name    string
	OK      bool
	Version string
	Note    string
}

// Report probes all dependencies and returns their status.
func Report(engine string) []DepStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var results []DepStatus

	// uvx
	results = append(results, probe(ctx, "uvx", "uvx", "--version"))

	// ffmpeg
	results = append(results, probe(ctx, "ffmpeg", "ffmpeg", "-version"))

	// yt-dlp via uvx
	results = append(results, probeUvx(ctx, "yt-dlp", "yt-dlp", "--version"))

	// whisper engine
	if engine == "openai" {
		results = append(results, probeUvxFrom(ctx, "openai-whisper", "openai-whisper", "whisper", "--help"))
	} else {
		results = append(results, probeUvxFrom(ctx, "faster-whisper", "faster-whisper", "faster-whisper", "--help"))
	}

	return results
}

func probe(ctx context.Context, name, cmd string, args ...string) DepStatus {
	out, err := exec.CommandContext(ctx, cmd, args...).Output()
	if err != nil {
		return DepStatus{Name: name, OK: false, Note: err.Error()}
	}
	ver := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return DepStatus{Name: name, OK: true, Version: ver}
}

func probeUvx(ctx context.Context, name, tool string, args ...string) DepStatus {
	cmdArgs := append([]string{tool}, args...)
	out, err := exec.CommandContext(ctx, "uvx", cmdArgs...).Output()
	if err != nil {
		return DepStatus{Name: name, OK: false, Note: "not available via uvx: " + err.Error()}
	}
	ver := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return DepStatus{Name: name, OK: true, Version: ver}
}

func probeUvxFrom(ctx context.Context, name, pkg, tool string, args ...string) DepStatus {
	cmdArgs := append([]string{"--from", pkg, tool}, args...)
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "uvx", cmdArgs...)
	cmd.Stderr = &stderr
	err := cmd.Run()
	// whisper --help exits with non-zero, so check stderr for meaningful output
	if err != nil && !strings.Contains(stderr.String(), "usage") && !strings.Contains(stderr.String(), "Usage") {
		return DepStatus{Name: name, OK: false, Note: "not available via uvx: " + err.Error()}
	}
	return DepStatus{Name: name, OK: true, Note: "available via uvx"}
}
