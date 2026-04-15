package parser

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// JavaParser parses Java source files using tree-sitter to extract import declarations.
type JavaParser struct {
	parser *sitter.Parser
}

// NewJavaParser creates a JavaParser backed by the tree-sitter Java grammar.
func NewJavaParser() *JavaParser {
	p := sitter.NewParser()
	p.SetLanguage(java.GetLanguage())
	return &JavaParser{parser: p}
}

const javaImportQuery = `
(import_declaration
  (identifier) @import)

(import_declaration
  (scoped_identifier) @import)
`

const javaPackageQuery = `
(package_declaration
  (identifier) @pkg)

(package_declaration
  (scoped_identifier) @pkg)
`

// ParseDirectory walks the directory tree and builds a DependencyGraph.
func (jp *JavaParser) ParseDirectory(root string, maxDepth int) (*DependencyGraph, error) {
	packages, err := jp.collectPackages(root, maxDepth)
	if err != nil {
		return nil, err
	}
	lcp := packageLCP(packages)

	g := newGraph()

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return skipVendor(d, err)
		}
		if filepath.Ext(path) != ".java" {
			return nil
		}

		if maxDepth > 0 {
			rel, _ := filepath.Rel(root, path)
			if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
				return nil
			}
		}

		imports, err := jp.parseImports(path)
		if err != nil {
			return nil
		}

		g.FileCount++

		rel, _ := filepath.Rel(root, path)
		fileID := filepath.ToSlash(rel)
		label := filepath.Base(path)
		g.addNode(fileID, label, path, "java", false)

		for _, imp := range imports {
			external := isJDKImport(imp) || !underJavaProject(imp, lcp)
			parts := strings.Split(imp, ".")
			label := imp
			if len(parts) > 0 {
				label = parts[len(parts)-1]
			}
			g.addNode(imp, label, "", "java", external)
			g.addEdge(fileID, imp)
		}

		return nil
	})

	return g, err
}

func (jp *JavaParser) collectPackages(root string, maxDepth int) ([]string, error) {
	var packages []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return skipVendor(d, err)
		}
		if filepath.Ext(path) != ".java" {
			return nil
		}
		if maxDepth > 0 {
			rel, _ := filepath.Rel(root, path)
			if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
				return nil
			}
		}
		pkg, err := jp.parsePackage(path)
		if err != nil || pkg == "" {
			return nil
		}
		packages = append(packages, pkg)
		return nil
	})
	return packages, err
}

func packageLCP(packages []string) string {
	if len(packages) == 0 {
		return ""
	}
	parts := strings.Split(packages[0], ".")
	for _, pkg := range packages[1:] {
		seg := strings.Split(pkg, ".")
		n := len(parts)
		if len(seg) < n {
			n = len(seg)
		}
		i := 0
		for i < n && parts[i] == seg[i] {
			i++
		}
		parts = parts[:i]
		if len(parts) == 0 {
			return ""
		}
	}
	return strings.Join(parts, ".")
}

func isJDKImport(imp string) bool {
	if imp == "java" || imp == "javax" || imp == "jdk" {
		return true
	}
	prefixes := []string{
		"java.", "javax.", "jdk.", "com.sun.", "sun.", "org.w3c.", "org.xml.",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(imp, p) {
			return true
		}
	}
	return false
}

func underJavaProject(imp, lcp string) bool {
	if lcp == "" {
		return false
	}
	return imp == lcp || strings.HasPrefix(imp, lcp+".")
}

func (jp *JavaParser) parsePackage(path string) (string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return jp.parsePackageFromSource(src)
}

func (jp *JavaParser) parsePackageFromSource(src []byte) (string, error) {
	tree, err := jp.parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return "", err
	}
	defer tree.Close()

	q, err := sitter.NewQuery([]byte(javaPackageQuery), java.GetLanguage())
	if err != nil {
		return "", err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, tree.RootNode())

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, cap := range m.Captures {
			return cap.Node.Content(src), nil
		}
	}
	return "", nil
}

func (jp *JavaParser) parseImports(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tree, err := jp.parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	q, err := sitter.NewQuery([]byte(javaImportQuery), java.GetLanguage())
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
