package graph

import (
	"sort"

	"github.com/depgraph/internal/parser"
)

// CycleGroup is a strongly connected component that contains a cycle.
type CycleGroup struct {
	Nodes []string    `json:"nodes"`
	Edges []CycleEdge `json:"edges"`
}

// Neighbors holds upstream and downstream dependencies for a node.
type Neighbors struct {
	Node       string   `json:"node"`
	Upstream   []string `json:"upstream"`
	Downstream []string `json:"downstream"`
}

// Cycles returns strongly connected components that contain cycles.
func (g *Graph) Cycles() []CycleGroup {
	groups := make([]CycleGroup, 0)
	for _, component := range g.stronglyConnectedComponents() {
		if !componentHasCycle(component, g.data.Edges) {
			continue
		}
		nodeSet := make(map[string]bool, len(component))
		for _, id := range component {
			nodeSet[id] = true
		}
		edges := make([]CycleEdge, 0)
		for _, e := range g.data.Edges {
			if nodeSet[e.From] && nodeSet[e.To] {
				edges = append(edges, CycleEdge{From: e.From, To: e.To})
			}
		}
		sort.Strings(component)
		groups = append(groups, CycleGroup{Nodes: component, Edges: edges})
	}
	return groups
}

// Neighbors returns nodes that depend on the given node (upstream) and nodes
// it depends on (downstream). Returns an error if nodeID is not in the graph.
func (g *Graph) Neighbors(nodeID string) (Neighbors, error) {
	if _, ok := g.data.Nodes[nodeID]; !ok {
		return Neighbors{}, errNodeNotFound(nodeID)
	}

	upstreamSet := map[string]bool{}
	downstreamSet := map[string]bool{}
	for _, e := range g.data.Edges {
		if e.To == nodeID {
			upstreamSet[e.From] = true
		}
		if e.From == nodeID {
			downstreamSet[e.To] = true
		}
	}

	return Neighbors{
		Node:       nodeID,
		Upstream:   sortedKeys(upstreamSet),
		Downstream: sortedKeys(downstreamSet),
	}, nil
}

type nodeNotFoundError string

func (e nodeNotFoundError) Error() string {
	return "node not found: " + string(e)
}

func errNodeNotFound(id string) error {
	return nodeNotFoundError(id)
}

func componentHasCycle(nodes []string, edges []*parser.Edge) bool {
	if len(nodes) > 1 {
		return true
	}
	if len(nodes) == 1 {
		id := nodes[0]
		for _, e := range edges {
			if e.From == id && e.To == id {
				return true
			}
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (g *Graph) stronglyConnectedComponents() [][]string {
	adj := map[string][]string{}
	for id := range g.data.Nodes {
		adj[id] = nil
	}
	for _, e := range g.data.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	index := 0
	stack := []string{}
	onStack := map[string]bool{}
	indices := map[string]int{}
	lowlink := map[string]int{}
	var sccs [][]string

	var strongConnect func(v string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adj[v] {
			if _, seen := indices[w]; !seen {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] && indices[w] < lowlink[v] {
				lowlink[v] = indices[w]
			}
		}

		if lowlink[v] == indices[v] {
			component := []string{}
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				component = append(component, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, component)
		}
	}

	nodes := make([]string, 0, len(g.data.Nodes))
	for id := range g.data.Nodes {
		nodes = append(nodes, id)
	}
	sort.Strings(nodes)
	for _, id := range nodes {
		if _, seen := indices[id]; !seen {
			strongConnect(id)
		}
	}
	return sccs
}
