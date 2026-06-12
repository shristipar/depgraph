package verify

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/depgraph/internal/analyze"
)

func TestRun_DepgraphRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	result, cleanup, err := analyze.Run(context.Background(), analyze.Options{
		Path: repoRoot,
		Lang: "go",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()

	report := Run(result, DefaultOptions())
	if !report.OK {
		t.Fatalf("expected depgraph repo to pass verify, violations: %+v", report.Violations)
	}
	if report.Stats.Files == 0 {
		t.Fatal("expected at least one Go source file")
	}
}

func TestRun_GinRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gin integration test in short mode")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	result, cleanup, err := analyze.Run(context.Background(), analyze.Options{
		URL:  "https://github.com/gin-gonic/gin",
		Lang: "go",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()

	report := Run(result, DefaultOptions())
	if !report.OK {
		t.Fatalf("expected gin to pass no_cycles, violations: %+v", report.Violations)
	}
}
