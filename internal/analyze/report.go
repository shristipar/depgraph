package analyze

import (
	"sort"

	"github.com/depgraph/internal/graph"
)

// Report is a structured dependency analysis result for APIs and MCP tools.
type Report struct {
	Language    string            `json:"language"`
	RepoPath    string            `json:"repo_path"`
	Stats       ReportStats       `json:"stats"`
	Nodes       []ReportNode      `json:"nodes"`
	Edges       []ReportEdge      `json:"edges"`
	CycleEdges  []graph.CycleEdge `json:"cycle_edges"`
}

// ReportStats summarizes graph size and cycle count.
type ReportStats struct {
	Files  int `json:"files"`
	Nodes  int `json:"nodes"`
	Edges  int `json:"edges"`
	Cycles int `json:"cycles"`
}

// ReportNode is a node in the dependency graph.
type ReportNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	External bool   `json:"external"`
	Lang     string `json:"lang"`
}

// ReportEdge is a directed dependency edge.
type ReportEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Report builds a structured report from the analysis result.
func (r *Result) Report() Report {
	nodes := make([]ReportNode, 0, len(r.Deps.Nodes))
	nodeIDs := make([]string, 0, len(r.Deps.Nodes))
	for id := range r.Deps.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)
	for _, id := range nodeIDs {
		n := r.Deps.Nodes[id]
		nodes = append(nodes, ReportNode{
			ID:       n.ID,
			Label:    n.Label,
			External: n.External,
			Lang:     n.Lang,
		})
	}

	edges := make([]ReportEdge, 0, len(r.Deps.Edges))
	for _, e := range r.Deps.Edges {
		edges = append(edges, ReportEdge{From: e.From, To: e.To})
	}

	return Report{
		Language:   r.Language,
		RepoPath:   r.RepoPath,
		Stats: ReportStats{
			Files:  r.Deps.FileCount,
			Nodes:  r.Deps.NodeCount,
			Edges:  r.Deps.EdgeCount,
			Cycles: r.Graph.CircularCount(),
		},
		Nodes:      nodes,
		Edges:      edges,
		CycleEdges: r.Graph.CycleEdges(),
	}
}
