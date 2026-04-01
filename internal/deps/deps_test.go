package deps

import (
	"os/exec"
	"testing"
)

func TestCheck_MissingUvx(t *testing.T) {
	// Only meaningful if uvx is actually absent — skip otherwise.
	if _, err := exec.LookPath("uvx"); err == nil {
		t.Skip("uvx is present; skipping absence test")
	}
	if err := Check("faster"); err == nil {
		t.Error("Check should fail when uvx is missing")
	}
}

func TestCheck_WithDeps(t *testing.T) {
	if _, err := exec.LookPath("uvx"); err != nil {
		t.Skip("uvx not in PATH")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not in PATH")
	}
	if err := Check("faster"); err != nil {
		t.Errorf("Check with all deps present: %v", err)
	}
}

func TestReport_Structure(t *testing.T) {
	// Report must always return entries — even when tools are absent.
	for _, engine := range []string{"faster", "openai"} {
		statuses := Report(engine)
		if len(statuses) == 0 {
			t.Errorf("Report(%q) returned no entries", engine)
		}
		for _, s := range statuses {
			if s.Name == "" {
				t.Errorf("Report entry has empty Name: %+v", s)
			}
		}
	}
}
