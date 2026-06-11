package analyze

import (
	"context"
	"os/exec"
	"testing"
)

const ginRepoURL = "https://github.com/gin-gonic/gin"

func TestRun_GinRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gin integration test in short mode")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	result, cleanup, err := Run(context.Background(), Options{
		URL:  ginRepoURL,
		Lang: "go",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()

	if result.Language != "go" {
		t.Fatalf("Language = %q, want go", result.Language)
	}
	if result.Deps.FileCount < 50 {
		t.Fatalf("FileCount = %d, want at least 50", result.Deps.FileCount)
	}
	if result.Deps.NodeCount < 50 {
		t.Fatalf("NodeCount = %d, want at least 50", result.Deps.NodeCount)
	}
	if result.Deps.EdgeCount < 100 {
		t.Fatalf("EdgeCount = %d, want at least 100", result.Deps.EdgeCount)
	}

	const rootPkg = "github.com/gin-gonic/gin"
	if _, ok := result.Deps.Nodes[rootPkg]; !ok {
		t.Fatalf("missing node %q", rootPkg)
	}
	if _, ok := result.Deps.Nodes["encoding/json"]; !ok {
		t.Fatal("missing external dependency encoding/json")
	}

	report := result.Report()
	if report.Stats.Files != result.Deps.FileCount {
		t.Fatalf("report files = %d, deps files = %d", report.Stats.Files, result.Deps.FileCount)
	}
	if report.Language != "go" {
		t.Fatalf("report language = %q, want go", report.Language)
	}

	neighbors, err := result.NeighborsReport(rootPkg)
	if err != nil {
		t.Fatalf("NeighborsReport(%q): %v", rootPkg, err)
	}
	if neighbors.Stats.Downstream == 0 {
		t.Fatal("expected downstream neighbors for gin root package")
	}
	if neighbors.Stats.Upstream == 0 {
		t.Fatal("expected upstream neighbors for gin root package")
	}

	foundBinding := false
	for _, n := range neighbors.Downstream {
		if n.ID == "github.com/gin-gonic/gin/binding" {
			foundBinding = true
			break
		}
	}
	if !foundBinding {
		t.Fatalf("downstream = %+v, want github.com/gin-gonic/gin/binding", neighbors.Downstream)
	}
}
