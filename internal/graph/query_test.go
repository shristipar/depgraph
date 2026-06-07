package graph

import (
	"testing"

	"github.com/depgraph/internal/parser"
)

func TestCyclesAndNeighbors(t *testing.T) {
	deps := &parser.DependencyGraph{
		Nodes: map[string]*parser.Node{
			"a": {ID: "a", Label: "a"},
			"b": {ID: "b", Label: "b"},
			"c": {ID: "c", Label: "c"},
		},
		Edges: []*parser.Edge{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "a"},
		},
	}

	g := New(deps)

	cycles := g.Cycles()
	if len(cycles) != 1 {
		t.Fatalf("Cycles() = %d groups, want 1", len(cycles))
	}
	if len(cycles[0].Nodes) != 3 {
		t.Fatalf("cycle nodes = %v, want 3 nodes", cycles[0].Nodes)
	}

	neighbors, err := g.Neighbors("b")
	if err != nil {
		t.Fatalf("Neighbors(b): %v", err)
	}
	if len(neighbors.Upstream) != 1 || neighbors.Upstream[0] != "a" {
		t.Fatalf("upstream = %v, want [a]", neighbors.Upstream)
	}
	if len(neighbors.Downstream) != 1 || neighbors.Downstream[0] != "c" {
		t.Fatalf("downstream = %v, want [c]", neighbors.Downstream)
	}

	if _, err := g.Neighbors("missing"); err == nil {
		t.Fatal("expected error for missing node")
	}
}
