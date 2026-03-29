// Package graph builds and renders dependency graphs.
package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/depgraph/internal/parser"
)

// Graph wraps a DependencyGraph and provides rendering capabilities.
type Graph struct {
	data     *parser.DependencyGraph
	circular int
}

// New creates a Graph from parsed dependency data and detects cycles.
func New(data *parser.DependencyGraph) *Graph {
	g := &Graph{data: data}
	g.circular = g.detectCycles()
	return g
}

// CircularCount returns the number of detected circular dependency edges.
func (g *Graph) CircularCount() int {
	return g.circular
}

// ------------------------------------------------------------------
// DOT output (Graphviz)
// ------------------------------------------------------------------

// WriteDOT writes the dependency graph in Graphviz DOT format.
func (g *Graph) WriteDOT(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	cycles := g.cycleEdges()

	fmt.Fprintln(f, `digraph dependencies {`)
	fmt.Fprintln(f, `  graph [rankdir=LR, fontname="Helvetica", splines=ortho, nodesep=0.5];`)
	fmt.Fprintln(f, `  node  [shape=box, style=filled, fontname="Helvetica", fontsize=10];`)
	fmt.Fprintln(f, `  edge  [fontsize=9];`)
	fmt.Fprintln(f)

	// Sort node IDs for deterministic output
	nodeIDs := make([]string, 0, len(g.data.Nodes))
	for id := range g.data.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	// Nodes
	for _, id := range nodeIDs {
		n := g.data.Nodes[id]
		color := `"#ddeeff"` // internal
		if n.External {
			color = `"#ffe0cc"` // external / third-party
		}
		label := escape(n.Label)
		safeID := dotID(id)
		fmt.Fprintf(f, "  %s [label=%q, fillcolor=%s];\n", safeID, label, color)
	}

	fmt.Fprintln(f)

	// Edges
	for _, e := range g.data.Edges {
		fromID := dotID(e.From)
		toID := dotID(e.To)
		cycleKey := e.From + "->" + e.To
		if cycles[cycleKey] {
			fmt.Fprintf(f, "  %s -> %s [color=red, penwidth=2, label=\"cycle\"];\n", fromID, toID)
		} else {
			fmt.Fprintf(f, "  %s -> %s;\n", fromID, toID)
		}
	}

	fmt.Fprintln(f, `}`)
	return nil
}

// ------------------------------------------------------------------
// JSON output
// ------------------------------------------------------------------

type jsonNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	External bool   `json:"external"`
	Lang     string `json:"lang"`
}

type jsonEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type jsonGraph struct {
	Nodes []*jsonNode `json:"nodes"`
	Edges []*jsonEdge `json:"edges"`
	Stats struct {
		Files   int `json:"files"`
		Nodes   int `json:"nodes"`
		Edges   int `json:"edges"`
		Cycles  int `json:"cycles"`
	} `json:"stats"`
}

// WriteJSON writes the dependency graph as JSON.
func (g *Graph) WriteJSON(path string) error {
	jg := &jsonGraph{}

	nodeIDs := make([]string, 0, len(g.data.Nodes))
	for id := range g.data.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for _, id := range nodeIDs {
		n := g.data.Nodes[id]
		jg.Nodes = append(jg.Nodes, &jsonNode{
			ID:       id,
			Label:    n.Label,
			External: n.External,
			Lang:     n.Lang,
		})
	}

	for _, e := range g.data.Edges {
		jg.Edges = append(jg.Edges, &jsonEdge{From: e.From, To: e.To})
	}

	jg.Stats.Files = g.data.FileCount
	jg.Stats.Nodes = g.data.NodeCount
	jg.Stats.Edges = g.data.EdgeCount
	jg.Stats.Cycles = g.circular

	data, err := json.MarshalIndent(jg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ------------------------------------------------------------------
// SVG output (inline, no Graphviz needed)
// ------------------------------------------------------------------

// WriteSVG produces a simple force-directed SVG using a layered layout.
func (g *Graph) WriteSVG(path string) error {
	const (
		W         = 1400
		H         = 900
		nodeW     = 140
		nodeH     = 32
		padding   = 20
	)

	// Assign layers: internal nodes left, external right
	type pos struct{ x, y float64 }
	positions := map[string]pos{}

	var internal, external []string
	for id, n := range g.data.Nodes {
		if n.External {
			external = append(external, id)
		} else {
			internal = append(internal, id)
		}
	}
	sort.Strings(internal)
	sort.Strings(external)

	spread := func(nodes []string, xCenter float64) {
		n := len(nodes)
		if n == 0 {
			return
		}
		totalH := float64(n)*(nodeH+padding) - padding
		startY := (H-totalH)/2 + float64(nodeH)/2
		for i, id := range nodes {
			positions[id] = pos{xCenter, startY + float64(i)*(nodeH+padding)}
		}
	}

	spread(internal, float64(W)*0.25)
	spread(external, float64(W)*0.75)

	cycles := g.cycleEdges()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, W, H, W, H))
	sb.WriteString("\n<style>")
	sb.WriteString(`
    text { font: 11px Helvetica, Arial, sans-serif; }
    .internal rect { fill:#ddeeff; stroke:#5599cc; stroke-width:1.5; rx:4; }
    .external rect { fill:#ffe0cc; stroke:#cc7744; stroke-width:1.5; }
    .edge { stroke:#888; stroke-width:1; fill:none; marker-end:url(#arr); }
    .edge.cycle { stroke:red; stroke-width:2; }
    .legend text { font-size:12px; }
  `)
	sb.WriteString("</style>\n")

	// Arrow marker
	sb.WriteString(`<defs>
  <marker id="arr" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
    <path d="M0,0 L0,6 L8,3 z" fill="#888"/>
  </marker>
  <marker id="arr-red" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
    <path d="M0,0 L0,6 L8,3 z" fill="red"/>
  </marker>
</defs>
`)

	// Background
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#f8f8f8"/>`, W, H))

	// Section labels
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="30" text-anchor="middle" font-weight="bold" font-size="14" fill="#336">Internal Packages</text>`, int(W*0.25)))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="30" text-anchor="middle" font-weight="bold" font-size="14" fill="#633">External Dependencies</text>`, int(W*0.75)))

	// Edges
	for _, e := range g.data.Edges {
		fp, ok1 := positions[e.From]
		tp, ok2 := positions[e.To]
		if !ok1 || !ok2 {
			continue
		}
		cycleKey := e.From + "->" + e.To
		cls := "edge"
		marker := "url(#arr)"
		if cycles[cycleKey] {
			cls = "edge cycle"
			marker = "url(#arr-red)"
		}
		x1 := fp.x + nodeW/2
		y1 := fp.y
		x2 := tp.x - nodeW/2
		y2 := tp.y
		mx := (x1 + x2) / 2
		sb.WriteString(fmt.Sprintf(
			`<path class="%s" d="M%.0f,%.0f C%.0f,%.0f %.0f,%.0f %.0f,%.0f" marker-end="%s"/>`,
			cls, x1, y1, mx, y1, mx, y2, x2, y2, marker,
		))
		sb.WriteString("\n")
	}

	// Nodes
	for id, p := range positions {
		n := g.data.Nodes[id]
		cls := "internal"
		if n.External {
			cls = "external"
		}
		x := p.x - nodeW/2
		y := p.y - nodeH/2
		label := truncate(n.Label, 18)
		sb.WriteString(fmt.Sprintf(
			`<g class="%s" transform="translate(%.0f,%.0f)"><rect width="%d" height="%d" rx="4"/><text x="%d" y="%d" text-anchor="middle">%s</text></g>`,
			cls, x, y, nodeW, nodeH, nodeW/2, nodeH/2+4, label,
		))
		sb.WriteString("\n")
	}

	// Legend
	sb.WriteString(fmt.Sprintf(`<g class="legend" transform="translate(%d,%d)">`, W-220, H-90))
	sb.WriteString(`<rect width="200" height="80" fill="white" stroke="#ccc" rx="4"/>`)
	sb.WriteString(`<rect x="10" y="15" width="16" height="12" fill="#ddeeff" stroke="#5599cc"/>`)
	sb.WriteString(`<text x="32" y="25">Internal package</text>`)
	sb.WriteString(`<rect x="10" y="35" width="16" height="12" fill="#ffe0cc" stroke="#cc7744"/>`)
	sb.WriteString(`<text x="32" y="45">External dependency</text>`)
	sb.WriteString(`<line x1="10" y1="60" x2="26" y2="60" stroke="red" stroke-width="2"/>`)
	sb.WriteString(`<text x="32" y="64">Circular dependency</text>`)
	sb.WriteString("</g>\n")

	sb.WriteString("</svg>")

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// ------------------------------------------------------------------
// Cycle detection (DFS)
// ------------------------------------------------------------------

func (g *Graph) detectCycles() int {
	adj := map[string][]string{}
	for _, e := range g.data.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	visited := map[string]bool{}
	inStack := map[string]bool{}
	cycles := 0

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		inStack[node] = true
		for _, nb := range adj[node] {
			if !visited[nb] {
				if dfs(nb) {
					cycles++
				}
			} else if inStack[nb] {
				cycles++
			}
		}
		inStack[node] = false
		return false
	}

	for id := range g.data.Nodes {
		if !visited[id] {
			dfs(id)
		}
	}
	return cycles
}

func (g *Graph) cycleEdges() map[string]bool {
	adj := map[string][]string{}
	for _, e := range g.data.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	visited := map[string]int{} // 0=unvisited,1=in-stack,2=done
	cyclic := map[string]bool{}

	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = 1
		for _, nb := range adj[node] {
			if visited[nb] == 0 {
				dfs(nb)
			} else if visited[nb] == 1 {
				cyclic[node+"->"+nb] = true
			}
		}
		visited[node] = 2
	}

	for id := range g.data.Nodes {
		if visited[id] == 0 {
			dfs(id)
		}
	}
	return cyclic
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

// dotID converts an arbitrary string to a safe Graphviz node identifier.
func dotID(s string) string {
	r := strings.NewReplacer("/", "_", ".", "_", "-", "_", "@", "_", ":", "_", " ", "_")
	return `n_` + r.Replace(s)
}

func escape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
