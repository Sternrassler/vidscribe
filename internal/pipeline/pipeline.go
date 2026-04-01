package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Run executes the full download → transcribe → format pipeline.
func Run(ctx context.Context, cfg *Config, logw io.Writer) ([]string, error) {
	if logw == nil {
		logw = io.Discard
	}

	// Step 1: download audio.
	audioPath, meta, err := Download(ctx, cfg, logw)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(audioPath))

	if cfg.Verbose {
		fmt.Fprintf(logw, "[vidscribe] downloaded: %s (%s)\n", meta.Title, meta.ID)
	}

	// Step 2: transcribe.
	tx, err := Transcribe(ctx, cfg, audioPath, logw)
	if err != nil {
		return nil, fmt.Errorf("transcribe: %w", err)
	}
	defer os.RemoveAll(tx.TempDir)

	if cfg.Verbose {
		fmt.Fprintf(logw, "[vidscribe] transcription complete\n")
	}

	// Step 3: write output files.
	paths, err := WriteOutputs(cfg, tx, meta, logw)
	if err != nil {
		return paths, fmt.Errorf("write outputs: %w", err)
	}

	return paths, nil
}
