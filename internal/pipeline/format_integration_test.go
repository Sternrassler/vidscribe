package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteMD(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.md")

	meta := &Metadata{
		Title:      "Test Video",
		WebpageURL: "https://youtube.com/watch?v=test",
		Channel:    "Test Channel",
		Duration:   3661,
		UploadDate: "20240315",
	}

	if err := writeMD(dst, meta, "Hello world transcript."); err != nil {
		t.Fatalf("writeMD: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	content := string(data)

	checks := []string{
		"# Test Video",
		"https://youtube.com/watch?v=test",
		"Test Channel",
		"1:01:01",
		"2024-03-15",
		"Hello world transcript.",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("writeMD output missing %q\n\nFull output:\n%s", want, content)
		}
	}
}

func TestWriteMDFallbackToUploader(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.md")

	meta := &Metadata{
		Title:    "Video",
		Uploader: "SomeUploader",
		Duration: 60,
	}

	if err := writeMD(dst, meta, "text"); err != nil {
		t.Fatalf("writeMD: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if !strings.Contains(string(data), "SomeUploader") {
		t.Errorf("expected Uploader as fallback, got:\n%s", data)
	}
}

func TestWriteOutputs(t *testing.T) {
	// Prepare a fake transcription temp dir with whisper-like output files.
	txDir := t.TempDir()
	baseName := "testvideo"

	for _, ext := range []string{"txt", "srt", "vtt", "json"} {
		if err := os.WriteFile(
			filepath.Join(txDir, baseName+"."+ext),
			[]byte("content of "+ext),
			0o644,
		); err != nil {
			t.Fatalf("setup: write %s: %v", ext, err)
		}
	}

	outDir := t.TempDir()
	cfg := &Config{
		OutputDir: outDir,
		Formats:   []string{"txt", "srt", "md"},
	}
	tx := &TranscribeResult{TempDir: txDir, BaseName: baseName}
	meta := &Metadata{
		ID:         "abc",
		Title:      "My Video",
		Duration:   120,
		UploadDate: "20240101",
	}

	paths, err := WriteOutputs(cfg, tx, meta, nil)
	if err != nil {
		t.Fatalf("WriteOutputs: %v", err)
	}

	// Expect txt, srt, md — not vtt or json (not requested).
	wantExts := map[string]bool{"txt": true, "srt": true, "md": true}
	gotExts := map[string]bool{}
	for _, p := range paths {
		gotExts[filepath.Ext(p)[1:]] = true
	}

	for ext := range wantExts {
		if !gotExts[ext] {
			t.Errorf("expected .%s in output, got %v", ext, paths)
		}
	}
	for ext := range gotExts {
		if !wantExts[ext] {
			t.Errorf("unexpected .%s in output", ext)
		}
	}

	// Verify files actually exist.
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("output file missing: %s", p)
		}
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "hello" {
		t.Errorf("copyFile result = %q, %v", data, err)
	}
}
