package analyze

import (
	"github.com/depgraph/internal/graph"
	"github.com/depgraph/internal/parser"
)

// CyclesReport summarizes circular dependencies in a repository.
type CyclesReport struct {
	Language   string             `json:"language"`
	RepoPath   string             `json:"repo_path"`
	Stats      ReportStats        `json:"stats"`
	Cycles     []graph.CycleGroup `json:"cycles"`
	CycleEdges []graph.CycleEdge  `json:"cycle_edges"`
}

// NeighborsReport lists upstream and downstream nodes for one module.
type NeighborsReport struct {
	Language   string           `json:"language"`
	RepoPath   string           `json:"repo_path"`
	Node       string           `json:"node"`
	Label      string           `json:"label"`
	External   bool             `json:"external"`
	Upstream   []ReportNode     `json:"upstream"`
	Downstream []ReportNode     `json:"downstream"`
	Stats      NeighborsStats   `json:"stats"`
}

// NeighborsStats counts direct neighbors for a node.
type NeighborsStats struct {
	Upstream   int `json:"upstream"`
	Downstream int `json:"downstream"`
}

// CyclesReport builds a cycle-focused report from the analysis result.
func (r *Result) CyclesReport() CyclesReport {
	return CyclesReport{
		Language: r.Language,
		RepoPath: r.RepoPath,
		Stats: ReportStats{
			Files:  r.Deps.FileCount,
			Nodes:  r.Deps.NodeCount,
			Edges:  r.Deps.EdgeCount,
			Cycles: r.Graph.CircularCount(),
		},
		Cycles:     r.Graph.Cycles(),
		CycleEdges: r.Graph.CycleEdges(),
	}
}

// NeighborsReport builds a neighbor view for nodeID.
func (r *Result) NeighborsReport(nodeID string) (NeighborsReport, error) {
	neighbors, err := r.Graph.Neighbors(nodeID)
	if err != nil {
		return NeighborsReport{}, err
	}

	node := r.Deps.Nodes[nodeID]
	return NeighborsReport{
		Language:   r.Language,
		RepoPath:   r.RepoPath,
		Node:       nodeID,
		Label:      node.Label,
		External:   node.External,
		Upstream:   reportNodes(r.Deps, neighbors.Upstream),
		Downstream: reportNodes(r.Deps, neighbors.Downstream),
		Stats: NeighborsStats{
			Upstream:   len(neighbors.Upstream),
			Downstream: len(neighbors.Downstream),
		},
	}, nil
}

func reportNodes(deps *parser.DependencyGraph, ids []string) []ReportNode {
	out := make([]ReportNode, 0, len(ids))
	for _, id := range ids {
		n, ok := deps.Nodes[id]
		if !ok {
			continue
		}
		out = append(out, ReportNode{
			ID:       n.ID,
			Label:    n.Label,
			External: n.External,
			Lang:     n.Lang,
		})
	}
	return out
}
