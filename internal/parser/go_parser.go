package parser

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// GoParser parses Go source files using tree-sitter to extract import paths.
type GoParser struct {
	parser *sitter.Parser
}

// NewGoParser creates a GoParser backed by the tree-sitter Go grammar.
func NewGoParser() *GoParser {
	p := sitter.NewParser()
	p.SetLanguage(golang.GetLanguage())
	return &GoParser{parser: p}
}

// importQuery is the tree-sitter S-expression query for Go import declarations.
// It captures both single-import and grouped import specs.
const goImportQuery = `
(import_declaration
  (import_spec_list
    (import_spec path: (interpreted_string_literal) @import)))

(import_declaration
  (import_spec path: (interpreted_string_literal) @import))
`

// ParseDirectory walks the directory tree and builds a DependencyGraph.
func (gp *GoParser) ParseDirectory(root string, maxDepth int) (*DependencyGraph, error) {
	g := newGraph()

	// First pass: collect module name from go.mod
	moduleName := goModuleName(root)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return skipVendor(d, err)
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Enforce max depth relative to root
		if maxDepth > 0 {
			rel, _ := filepath.Rel(root, path)
			depth := strings.Count(rel, string(os.PathSeparator))
			if depth > maxDepth {
				return nil
			}
		}

		imports, err := gp.parseFile(path)
		if err != nil {
			return nil // skip unparseable files
		}

		g.FileCount++

		// Derive the package ID from the path relative to root
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		pkgID := packageID(moduleName, rel)

		g.addNode(pkgID, filepath.Base(filepath.Dir(path)), filepath.Dir(path), "go", false)

		for _, imp := range imports {
			imp = strings.Trim(imp, `"`)
			external := !strings.HasPrefix(imp, moduleName) && moduleName != ""
			label := imp
			if parts := strings.Split(imp, "/"); len(parts) > 0 {
				label = parts[len(parts)-1]
			}
			g.addNode(imp, label, "", "go", external)
			g.addEdge(pkgID, imp)
		}

		return nil
	})

	return g, err
}

// parseFile extracts import paths from a single Go file using tree-sitter.
func (gp *GoParser) parseFile(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tree, err := gp.parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	q, err := sitter.NewQuery([]byte(goImportQuery), golang.GetLanguage())
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, tree.RootNode())

	var imports []string
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, cap := range m.Captures {
			imports = append(imports, cap.Node.Content(src))
		}
	}
	return imports, nil
}

// goModuleName reads the module name from go.mod.
func goModuleName(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// packageID constructs a stable package identifier.
func packageID(moduleName, rel string) string {
	if moduleName == "" {
		return rel
	}
	if rel == "." || rel == "" {
		return moduleName
	}
	return moduleName + "/" + filepath.ToSlash(rel)
}
