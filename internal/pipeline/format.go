package pipeline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const mdTemplate = `# {{.Title}}

**URL:** {{.URL}}
**Kanal:** {{.Channel}}
**Dauer:** {{.Duration}}
**Datum:** {{.Date}}

---

{{.Transcript}}
`

// WriteOutputs copies / generates the requested format files into cfg.OutputDir.
// Returns the paths of the files that were written.
func WriteOutputs(cfg *Config, tx *TranscribeResult, meta *Metadata, logw io.Writer) ([]string, error) {
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	outBase := filepath.Join(cfg.OutputDir, meta.SafeTitle())
	var written []string

	// Whisper-native formats: txt, srt, vtt, json
	for _, ext := range []string{"txt", "srt", "vtt", "json"} {
		if !cfg.HasFormat(ext) {
			continue
		}
		src := filepath.Join(tx.TempDir, tx.BaseName+"."+ext)
		if _, err := os.Stat(src); err != nil {
			continue // not produced (e.g. old whisper version)
		}
		dst := outBase + "." + ext
		if err := copyFile(src, dst); err != nil {
			return written, fmt.Errorf("copy %s: %w", ext, err)
		}
		written = append(written, dst)
	}

	// Markdown: generated from txt + metadata.
	if cfg.HasFormat("md") {
		txtSrc := filepath.Join(tx.TempDir, tx.BaseName+".txt")
		transcript, err := os.ReadFile(txtSrc)
		if err != nil {
			return written, fmt.Errorf("read transcript for md: %w", err)
		}
		dst := outBase + ".md"
		if err := writeMD(dst, meta, string(transcript)); err != nil {
			return written, err
		}
		written = append(written, dst)
	}

	return written, nil
}

func writeMD(dst string, meta *Metadata, transcript string) error {
	tmpl, err := template.New("md").Parse(mdTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer f.Close()

	data := struct {
		Title      string
		URL        string
		Channel    string
		Duration   string
		Date       string
		Transcript string
	}{
		Title:      meta.Title,
		URL:        meta.WebpageURL,
		Channel:    channelName(meta),
		Duration:   formatDuration(meta.Duration),
		Date:       formatDate(meta.UploadDate),
		Transcript: strings.TrimSpace(transcript),
	}

	return tmpl.Execute(f, data)
}

func channelName(m *Metadata) string {
	if m.Channel != "" {
		return m.Channel
	}
	return m.Uploader
}

func formatDuration(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatDate(uploadDate string) string {
	// yt-dlp format: YYYYMMDD
	if len(uploadDate) == 8 {
		return uploadDate[:4] + "-" + uploadDate[4:6] + "-" + uploadDate[6:]
	}
	return uploadDate
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// FormatReport returns a human-readable list of written file paths.
func FormatReport(paths []string) string {
	var sb strings.Builder
	for _, p := range paths {
		sb.WriteString("  ")
		sb.WriteString(p)
		sb.WriteByte('\n')
	}
	return sb.String()
}
