package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxRetries    = 3
	retryBaseWait = 5 * time.Second
)

// Download retrieves the audio track for the given URL and returns the local
// path to the downloaded audio file together with video metadata.
// The caller is responsible for deleting the file when done.
func Download(ctx context.Context, cfg *Config, logw io.Writer) (audioPath string, meta *Metadata, err error) {
	// 1. Fetch metadata (fast, no download).
	meta, err = fetchMetadata(ctx, cfg, logw)
	if err != nil {
		return "", nil, fmt.Errorf("metadata: %w", err)
	}

	// 2. Download audio to a temp directory.
	tmpDir, err := os.MkdirTemp("", "vidscribe-dl-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	audioPath, err = downloadAudio(ctx, cfg, meta.ID, tmpDir, logw)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}

	return audioPath, meta, nil
}

// fetchMetadata runs yt-dlp --dump-json and parses the result.
func fetchMetadata(ctx context.Context, cfg *Config, logw io.Writer) (*Metadata, error) {
	args := buildBaseArgs(cfg)
	args = append(args, "--dump-json", "--no-download", cfg.URL)

	var stdout, stderr bytes.Buffer
	cmd := YtdlpCmd(ctx, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if cfg.Verbose {
		fmt.Fprintf(logw, "[vidscribe] yt-dlp metadata: %s\n", strings.Join(cmd.Args, " "))
	}

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if isKeyringError(errMsg) {
			return nil, fmt.Errorf("keyring/cookie-decryption error — export cookies to a file and use --cookies-file: %w", err)
		}
		return nil, fmt.Errorf("yt-dlp metadata failed: %s", firstLine(errMsg))
	}

	var meta Metadata
	if err := json.Unmarshal(stdout.Bytes(), &meta); err != nil {
		return nil, fmt.Errorf("parse metadata JSON: %w", err)
	}
	return &meta, nil
}

// downloadAudio downloads the audio track with retry logic for HTTP 403/429.
func downloadAudio(ctx context.Context, cfg *Config, videoID, destDir string, logw io.Writer) (string, error) {
	args := buildBaseArgs(cfg)
	args = append(args,
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--output", destDir+"/%(id)s.%(ext)s",
		cfg.URL,
	)

	var lastErr error
	secretstorageRetried := false

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt) * retryBaseWait
			if cfg.Verbose {
				fmt.Fprintf(logw, "[vidscribe] retry %d/%d in %s…\n", attempt, maxRetries-1, wait)
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}

		effectiveArgs := args
		// On secretstorage retry, switch from browser cookies to file cookies.
		if secretstorageRetried && cfg.CookiesFile != "" {
			effectiveArgs = removeBrowserCookies(args)
			effectiveArgs = append(effectiveArgs, "--cookies", cfg.CookiesFile)
		}

		var stderr bytes.Buffer
		cmd := YtdlpCmd(ctx, effectiveArgs...)
		cmd.Stderr = &stderr

		if cfg.Verbose {
			fmt.Fprintf(logw, "[vidscribe] yt-dlp download: %s\n", strings.Join(cmd.Args, " "))
		}

		err := cmd.Run()
		if err == nil {
			// Find the downloaded file.
			path := filepath.Join(destDir, videoID+".mp3")
			if _, statErr := os.Stat(path); statErr == nil {
				return path, nil
			}
			// Try fallback extensions (yt-dlp may keep original container).
			for _, ext := range []string{"m4a", "opus", "webm"} {
				p := filepath.Join(destDir, videoID+"."+ext)
				if _, statErr := os.Stat(p); statErr == nil {
					return p, nil
				}
			}
			return "", fmt.Errorf("download completed but audio file not found in %s", destDir)
		}

		errMsg := stderr.String()
		lastErr = fmt.Errorf("yt-dlp: %s", firstLine(errMsg))

		if isKeyringError(errMsg) && !secretstorageRetried {
			if cfg.CookiesFile != "" {
				fmt.Fprintf(logw, "[vidscribe] keyring error — retrying with cookie file %s\n", cfg.CookiesFile)
				secretstorageRetried = true
				attempt-- // don't count this as a retry
				continue
			}
			return "", fmt.Errorf("keyring unavailable — export browser cookies to a file and use --cookies-file")
		}

		if isHTTP429(errMsg) {
			fmt.Fprintf(logw, "[vidscribe] HTTP 429 (rate limited) — will retry\n")
			continue
		}

		if isHTTP403(errMsg) {
			if cfg.CookiesBrowser == "" && cfg.CookiesFile == "" {
				return "", fmt.Errorf("HTTP 403 — try --cookies-browser chrome (or --cookies-file)")
			}
			// Already have auth; one more retry.
			continue
		}

		// Non-retryable error.
		return "", lastErr
	}

	return "", fmt.Errorf("download failed after %d attempts: %w", maxRetries, lastErr)
}

// YtdlpCmd constructs the yt-dlp exec.Cmd using uvx.
func YtdlpCmd(ctx context.Context, args ...string) *exec.Cmd {
	uvxArgs := append([]string{"--with", "secretstorage", "yt-dlp"}, args...)
	return exec.CommandContext(ctx, "uvx", uvxArgs...)
}

// buildBaseArgs returns the common yt-dlp flags derived from cfg.
func buildBaseArgs(cfg *Config) []string {
	var args []string

	if cfg.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", cfg.CookiesBrowser)
	} else if cfg.CookiesFile != "" {
		args = append(args, "--cookies", cfg.CookiesFile)
	}

	jsRuntime := cfg.JSRuntime
	if jsRuntime == "" {
		if p, err := exec.LookPath("node"); err == nil {
			jsRuntime = "node:" + p
		}
	}
	if jsRuntime != "" {
		args = append(args, "--js-runtimes", jsRuntime, "--remote-components", "ejs:github")
	}

	return args
}

// removeBrowserCookies strips --cookies-from-browser from args.
func removeBrowserCookies(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--cookies-from-browser" {
			i++ // skip the value
			continue
		}
		out = append(out, args[i])
	}
	return out
}

// isKeyringError detects platform-specific keyring/cookie-decryption failures
// reported by yt-dlp across Linux (secretstorage), macOS (Keychain), and
// Windows (DPAPI / win32crypt).
func isKeyringError(s string) bool {
	patterns := []string{
		// Linux
		"secretstorage",
		"Failed to unlock keyring",
		"No module named 'secretstorage'",
		// macOS
		"Keychain",
		"OSStatus",
		"security: SecKeychainSearchCopyNext",
		"cannot be found in the keychain",
		// Windows
		"CryptUnprotectData",
		"win32crypt",
		"No module named 'win32crypt'",
	}
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func isHTTP403(s string) bool {
	return strings.Contains(s, "HTTP Error 403") || strings.Contains(s, "403 Forbidden")
}

func isHTTP429(s string) bool {
	return strings.Contains(s, "HTTP Error 429") || strings.Contains(s, "429 Too Many Requests")
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}
