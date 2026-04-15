// Package parser extracts import/dependency information from source files
// using the tree-sitter parsing library.
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DependencyGraph holds the parsed dependency information.
type DependencyGraph struct {
	// Nodes maps a module/file identifier to its metadata.
	Nodes map[string]*Node
	// Edges is a list of directed dependency edges (from → to).
	Edges []*Edge

	// Stats
	FileCount int
	NodeCount int
	EdgeCount int
}

// Node represents a source file or module in the graph.
type Node struct {
	ID       string // unique identifier (e.g., package path or file path)
	Label    string // short display label
	File     string // absolute file path (empty for external deps)
	External bool   // true for third-party/external dependencies
	Lang     string // source language
}

// Edge represents an import from one node to another.
type Edge struct {
	From    string // Node ID
	To      string // Node ID
	Imports []string // specific symbols imported (if extractable)
}

// Parser extracts dependency information from source files.
type Parser interface {
	ParseDirectory(root string, maxDepth int) (*DependencyGraph, error)
}

// New returns a Parser for the given language.
func New(lang string) (Parser, error) {
	switch strings.ToLower(lang) {
	case "go":
		return NewGoParser(), nil
	case "python", "py":
		return NewPythonParser(), nil
	case "javascript", "js":
		return NewJSParser(false), nil
	case "typescript", "ts":
		return NewJSParser(true), nil
	case "java":
		return NewJavaParser(), nil
	default:
		return nil, fmt.Errorf("unsupported language %q — supported: go, python, javascript, typescript, java", lang)
	}
}

// DetectLanguage inspects the repository root and returns the dominant language.
func DetectLanguage(root string) (string, error) {
	counts := map[string]int{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return skipVendor(d, err)
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".go":
			counts["go"]++
		case ".py":
			counts["python"]++
		case ".ts", ".tsx":
			counts["typescript"]++
		case ".js", ".jsx":
			counts["javascript"]++
		case ".java":
			counts["java"]++
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	best, max := "", 0
	for lang, n := range counts {
		if n > max {
			best, max = lang, n
		}
	}
	if best == "" {
		return "", fmt.Errorf("no supported source files found in %s", root)
	}
	return best, nil
}

// skipVendor skips vendor, node_modules, and hidden directories.
func skipVendor(d os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		name := d.Name()
		if name == "vendor" || name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
	}
	return nil
}

// newGraph creates an empty DependencyGraph.
func newGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*Node),
	}
}

// addNode adds a node if not already present.
func (g *DependencyGraph) addNode(id, label, file, lang string, external bool) {
	if _, ok := g.Nodes[id]; !ok {
		g.Nodes[id] = &Node{
			ID:       id,
			Label:    label,
			File:     file,
			External: external,
			Lang:     lang,
		}
		g.NodeCount++
	}
}

// addEdge appends an edge and increments counter.
func (g *DependencyGraph) addEdge(from, to string) {
	g.Edges = append(g.Edges, &Edge{From: from, To: to})
	g.EdgeCount++
}
