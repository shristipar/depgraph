package parser

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// PythonParser parses Python source files using tree-sitter.
type PythonParser struct {
	parser *sitter.Parser
}

// NewPythonParser creates a parser backed by the tree-sitter Python grammar.
func NewPythonParser() *PythonParser {
	p := sitter.NewParser()
	p.SetLanguage(python.GetLanguage())
	return &PythonParser{parser: p}
}

// Queries for Python import statements:
//   import foo
//   import foo.bar
//   from foo import bar
//   from foo.bar import baz
const pythonImportQuery = `
(import_statement
  name: (dotted_name) @import)

(import_from_statement
  module_name: (dotted_name) @import)

(import_from_statement
  module_name: (relative_import) @import)
`

// ParseDirectory walks a Python project and builds a DependencyGraph.
func (pp *PythonParser) ParseDirectory(root string, maxDepth int) (*DependencyGraph, error) {
	g := newGraph()

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return skipVendor(d, err)
		}
		if filepath.Ext(path) != ".py" {
			return nil
		}

		if maxDepth > 0 {
			rel, _ := filepath.Rel(root, path)
			if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
				return nil
			}
		}

		imports, err := pp.parseFile(path)
		if err != nil {
			return nil
		}

		g.FileCount++

		rel, _ := filepath.Rel(root, path)
		fileID := filepath.ToSlash(strings.TrimSuffix(rel, ".py"))
		label := strings.TrimSuffix(filepath.Base(path), ".py")
		g.addNode(fileID, label, path, "python", false)

		for _, imp := range imports {
			// Relative imports start with dots — mark them as internal
			external := !strings.HasPrefix(imp, ".")
			parts := strings.Split(imp, ".")
			topLevel := parts[0]
			g.addNode(imp, topLevel, "", "python", external)
			g.addEdge(fileID, imp)
		}

		return nil
	})

	return g, err
}

// parseFile extracts import module names from a Python file using tree-sitter.
func (pp *PythonParser) parseFile(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tree, err := pp.parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	q, err := sitter.NewQuery([]byte(pythonImportQuery), python.GetLanguage())
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, tree.RootNode())

	seen := map[string]bool{}
	var imports []string
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, cap := range m.Captures {
			imp := cap.Node.Content(src)
			if !seen[imp] {
				seen[imp] = true
				imports = append(imports, imp)
			}
		}
	}
	return imports, nil
}
