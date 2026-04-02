package deps

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Check verifies that the minimum required tools are in PATH.
func Check(engine string) error {
	if _, err := exec.LookPath("uvx"); err != nil {
		return fmt.Errorf("uvx not found — install uv: https://docs.astral.sh/uv/")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found — %s", ffmpegInstallHint())
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
		results = append(results, probeUvxFrom(ctx, "whisper-ctranslate2", "whisper-ctranslate2", "whisper-ctranslate2", "--help"))
	}

	// CUDA (only when NVIDIA GPU is present)
	if gpuStatus := probeCUDA(ctx); gpuStatus != nil {
		results = append(results, *gpuStatus)
	}

	return results
}

// probeCUDA checks GPU availability and libcublas. Returns nil if no NVIDIA GPU is found.
func probeCUDA(ctx context.Context) *DepStatus {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name", "--format=csv,noheader").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return nil
	}
	gpuName := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]

	// Check if libcublas.so.12 is loadable via nvidia-cublas-cu12
	script := `import nvidia.cublas, pathlib, ctypes, sys
lib = pathlib.Path(nvidia.cublas.__spec__.submodule_search_locations[0]) / "lib" / "libcublas.so.12"
try:
    ctypes.CDLL(str(lib))
    print("ok")
except Exception as e:
    print("fail: " + str(e))
    sys.exit(1)
`
	cmd := exec.CommandContext(ctx, "uvx", "--with", "nvidia-cublas-cu12",
		"--from", "whisper-ctranslate2", "python3", "-c", script)
	cublasOut, cublasErr := cmd.Output()
	if cublasErr != nil || !strings.Contains(string(cublasOut), "ok") {
		return &DepStatus{
			Name: "cuda",
			OK:   false,
			Note: fmt.Sprintf("GPU: %s — libcublas.so.12 not loadable (install libcublas12 or nvidia-cublas-cu12)", gpuName),
		}
	}
	return &DepStatus{
		Name:    "cuda",
		OK:      true,
		Version: gpuName,
		Note:    "float16 available",
	}
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

func ffmpegInstallHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "install via: brew install ffmpeg"
	case "windows":
		return "install via: winget install ffmpeg  (or download from https://ffmpeg.org/download.html)"
	default:
		return "install via: apt install ffmpeg  (or your distro's package manager)"
	}
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
