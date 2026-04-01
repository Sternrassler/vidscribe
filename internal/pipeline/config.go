package pipeline

import "strings"

// Config holds all parameters for a single transcription run.
type Config struct {
	URL            string
	Model          string
	Language       string
	OutputDir      string
	CookiesBrowser string
	CookiesFile    string
	JSRuntime      string
	Formats        []string
	Engine         string
	Verbose        bool
}

// Metadata holds video information retrieved from yt-dlp.
type Metadata struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Uploader   string  `json:"uploader"`
	Channel    string  `json:"channel"`
	Duration   float64 `json:"duration"`
	UploadDate string  `json:"upload_date"`
	WebpageURL string  `json:"webpage_url"`
}

// SafeTitle returns a filesystem-safe version of the title.
func (m *Metadata) SafeTitle() string {
	if m.Title == "" {
		return m.ID
	}
	r := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", "\n", "_", "\r", "_",
	)
	s := r.Replace(m.Title)
	if len(s) > 120 {
		s = s[:120]
	}
	return strings.TrimSpace(s)
}

// HasFormat reports whether the given format is requested.
func (c *Config) HasFormat(fmt string) bool {
	for _, f := range c.Formats {
		if strings.TrimSpace(f) == fmt {
			return true
		}
	}
	return false
}
