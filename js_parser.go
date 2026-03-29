package parser

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
)

// JSParser parses JavaScript and TypeScript files using tree-sitter.
type JSParser struct {
	parser     *sitter.Parser
	typescript bool
}

// NewJSParser creates a parser for JS (typescript=false) or TS (typescript=true).
func NewJSParser(typescript bool) *JSParser {
	p := sitter.NewParser()
	if typescript {
		p.SetLanguage(tsx.GetLanguage())
	} else {
		p.SetLanguage(javascript.GetLanguage())
	}
	return &JSParser{parser: p, typescript: typescript}
}

// Queries for ES module import/export statements:
//   import foo from 'bar'
//   import { a, b } from 'baz'
//   export { x } from 'qux'
//   const x = require('mod')
const jsImportQuery = `
(import_statement
  source: (string) @import)

(export_statement
  source: (string) @import)

(call_expression
  function: (identifier) @fn (#eq? @fn "require")
  arguments: (arguments (string) @import))
`

// ParseDirectory walks a JS/TS project and builds a DependencyGraph.
func (jp *JSParser) ParseDirectory(root string, maxDepth int) (*DependencyGraph, error) {
	g := newGraph()
	lang := "javascript"
	if jp.typescript {
		lang = "typescript"
	}

	exts := map[string]bool{".js": true, ".jsx": true, ".mjs": true, ".cjs": true}
	if jp.typescript {
		exts[".ts"] = true
		exts[".tsx"] = true
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return skipVendor(d, err)
		}
		if !exts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}

		if maxDepth > 0 {
			rel, _ := filepath.Rel(root, path)
			if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
				return nil
			}
		}

		imports, err := jp.parseFile(path)
		if err != nil {
			return nil
		}

		g.FileCount++

		rel, _ := filepath.Rel(root, path)
		fileID := filepath.ToSlash(rel)
		label := filepath.Base(path)
		g.addNode(fileID, label, path, lang, false)

		for _, imp := range imports {
			imp = strings.Trim(imp, `"'`)
			external := !strings.HasPrefix(imp, ".") && !strings.HasPrefix(imp, "/")
			label := imp
			if parts := strings.Split(imp, "/"); len(parts) > 0 {
				// Scoped packages: @scope/pkg
				if strings.HasPrefix(imp, "@") && len(parts) >= 2 {
					label = parts[0] + "/" + parts[1]
				} else {
					label = parts[len(parts)-1]
				}
			}
			g.addNode(imp, label, "", lang, external)
			g.addEdge(fileID, imp)
		}

		return nil
	})

	return g, err
}

// parseFile extracts import source strings from a JS/TS file.
func (jp *JSParser) parseFile(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tree, err := jp.parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var lang sitter.Language
	if jp.typescript {
		lang = *tsx.GetLanguage()
	} else {
		lang = *javascript.GetLanguage()
	}

	q, err := sitter.NewQuery([]byte(jsImportQuery), &lang)
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
			if cap.Index == 0 {
				continue // skip the @fn capture from require()
			}
			imp := cap.Node.Content(src)
			imp = strings.Trim(imp, `"'` + "`")
			if !seen[imp] {
				seen[imp] = true
				imports = append(imports, imp)
			}
		}
	}
	return imports, nil
}
