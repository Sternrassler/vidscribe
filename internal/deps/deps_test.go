//go:build smoke

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
	if err := Check("faster"); err != nil {
		t.Errorf("Check with all deps present: %v", err)
	}
}

func TestReport_Structure(t *testing.T) {
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

func TestReport_Parallel(t *testing.T) {
	// Verify that parallelized probes still return deterministic order.
	s1 := Report("faster")
	s2 := Report("faster")
	if len(s1) != len(s2) {
		t.Fatalf("Report lengths differ: %d vs %d", len(s1), len(s2))
	}
	for i := range s1 {
		if s1[i].Name != s2[i].Name {
			t.Errorf("Report[%d] order differs: %q vs %q", i, s1[i].Name, s2[i].Name)
		}
	}
}
