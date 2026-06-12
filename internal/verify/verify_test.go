package verify

import (
	"testing"

	"github.com/depgraph/internal/analyze"
	"github.com/depgraph/internal/graph"
	"github.com/depgraph/internal/parser"
)

func TestRun_NoViolations(t *testing.T) {
	deps := &parser.DependencyGraph{
		Nodes: map[string]*parser.Node{
			"a": {ID: "a", Label: "a"},
			"b": {ID: "b", Label: "b"},
		},
		Edges: []*parser.Edge{{From: "a", To: "b"}},
	}
	result := &analyze.Result{
		Deps:     deps,
		Graph:    graph.New(deps),
		Language: "go",
		RepoPath: "/tmp/acyclic",
	}

	report := Run(result, DefaultOptions())
	if !report.OK {
		t.Fatalf("OK = false, violations = %+v", report.Violations)
	}
}

func TestRun_NoCyclesViolation(t *testing.T) {
	deps := &parser.DependencyGraph{
		Nodes: map[string]*parser.Node{
			"a": {ID: "a", Label: "a"},
			"b": {ID: "b", Label: "b"},
		},
		Edges: []*parser.Edge{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	}
	result := &analyze.Result{
		Deps:     deps,
		Graph:    graph.New(deps),
		Language: "go",
		RepoPath: "/tmp/cyclic",
	}

	report := Run(result, DefaultOptions())
	if report.OK {
		t.Fatal("expected verify to fail for cyclic graph")
	}
	if len(report.Violations) != 1 || report.Violations[0].Rule != "no_cycles" {
		t.Fatalf("violations = %+v", report.Violations)
	}
}
