// Package verify checks dependency graphs against architectural rules.
// It returns structured pass/fail results for CLI gates and MCP agent loops.
package verify

import (
	"github.com/depgraph/internal/analyze"
	"github.com/depgraph/internal/graph"
)

// Options selects which rules to enforce.
type Options struct {
	NoCycles bool
}

// DefaultOptions enables the standard rule set for agent verify loops.
func DefaultOptions() Options {
	return Options{NoCycles: true}
}

// Violation describes a single failed rule.
type Violation struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Report is the structured output of a verify run.
type Report struct {
	OK         bool        `json:"ok"`
	Language   string      `json:"language"`
	RepoPath   string      `json:"repo_path"`
	Stats      analyze.ReportStats `json:"stats"`
	Violations []Violation `json:"violations"`
}

// Run checks analysis result against the given rules.
func Run(result *analyze.Result, opts Options) Report {
	report := Report{
		OK:       true,
		Language: result.Language,
		RepoPath: result.RepoPath,
		Stats: analyze.ReportStats{
			Files:  result.Deps.FileCount,
			Nodes:  result.Deps.NodeCount,
			Edges:  result.Deps.EdgeCount,
			Cycles: result.Graph.CircularCount(),
		},
	}

	if opts.NoCycles {
		if v := checkNoCycles(result.Graph); v != nil {
			report.OK = false
			report.Violations = append(report.Violations, *v)
		}
	}

	return report
}

func checkNoCycles(g *graph.Graph) *Violation {
	cycles := g.Cycles()
	if len(cycles) == 0 {
		return nil
	}
	return &Violation{
		Rule:    "no_cycles",
		Message: "circular dependencies detected",
		Details: map[string]any{
			"cycle_groups": cycles,
			"cycle_edges":  g.CycleEdges(),
		},
	}
}
