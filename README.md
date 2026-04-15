# depgraph — Dependency Graph Generator

A CLI tool that clones a GitHub/GitLab repository and generates a visual dependency graph using **tree-sitter** for accurate AST-based import extraction.

## Features

- 🔗 Clones any public GitHub or GitLab repository (shallow clone for speed)
- 🌳 Uses **tree-sitter** for accurate, language-aware AST parsing
- 🗺️ Generates dependency graphs in **DOT**, **JSON**, or **SVG** formats
- 🔴 Detects **circular dependencies** and highlights them in red
- 🎨 Color-coded: internal packages (blue) vs. external dependencies (orange)
- 🚀 Supports **Go**, **Python**, **Java**, **JavaScript**, and **TypeScript**
- 🔍 Auto-detects the primary language of the repository

## Supported Languages

| Language   | Import styles detected                                      |
|------------|-------------------------------------------------------------|
| Go         | `import "pkg"`, grouped `import ( ... )`                  |
| Python     | `import foo`, `from foo import bar`, relative imports       |
| JavaScript | ES `import`, `export from`, CommonJS `require()`           |
| TypeScript | All JS patterns + TypeScript-specific syntax               |
| Java       | `import` (including `import static` and `.*`)              |

## Installation

```bash
git clone https://github.com/yourname/depgraph
cd depgraph
go mod tidy
go build -o depgraph ./cmd/
```

**Prerequisites:** `git` must be in your PATH. For SVG rendering via DOT, install [Graphviz](https://graphviz.org/).

## Usage

```bash
# Basic usage — auto-detects language, outputs DOT
./depgraph --url https://github.com/user/repo

# Specify language and output format
./depgraph --url https://github.com/gin-gonic/gin --lang go --format svg --output gin-graph

# Generate JSON for tooling integration
./depgraph --url https://gitlab.com/user/repo --format json

# Limit analysis depth (useful for large monorepos)
./depgraph --url https://github.com/user/repo --depth 3
```

## Flags

| Flag        | Default              | Description                                        |
|-------------|----------------------|----------------------------------------------------|
| `--url`     | *(required)*         | GitHub or GitLab repository URL                    |
| `--lang`    | `auto`               | Language: `auto`, `go`, `python`, `javascript`, `typescript`, `java` |
| `--format`  | `dot`                | Output format: `dot`, `json`, `svg`               |
| `--output`  | `dependency_graph`   | Output filename (extension added automatically)    |
| `--depth`   | `0` (unlimited)      | Max directory depth to analyze                     |
| `--verbose` | `false`              | Enable verbose/debug logging                       |

## Output Formats

### DOT (Graphviz)
```bash
./depgraph --url https://github.com/user/repo --format dot
dot -Tsvg dependency_graph.dot -o graph.svg   # render to SVG
dot -Tpng dependency_graph.dot -o graph.png   # render to PNG
```

### SVG (inline, no Graphviz needed)
```bash
./depgraph --url https://github.com/user/repo --format svg
open dependency_graph.svg
```

### JSON (for integration with other tools)
```json
{
  "nodes": [{ "id": "github.com/user/repo", "label": "repo", "external": false, "lang": "go" }],
  "edges": [{ "from": "github.com/user/repo", "to": "fmt" }],
  "stats": { "files": 42, "nodes": 18, "edges": 65, "cycles": 0 }
}
```

## Architecture

```
depgraph/
├── cmd/
│   └── main.go              # CLI entry point
├── internal/
│   ├── cloner/
│   │   └── cloner.go        # Git clone logic (GitHub/GitLab)
│   ├── parser/
│   │   ├── parser.go        # Language detection + Parser interface
│   │   ├── go_parser.go     # Go import extraction (tree-sitter)
│   │   ├── python_parser.go # Python import extraction (tree-sitter)
│   │   ├── js_parser.go     # JS/TS import extraction (tree-sitter)
│   │   └── java_parser.go   # Java import extraction (tree-sitter)
│   └── graph/
│       └── graph.go         # Graph building + DOT/JSON/SVG output
└── go.mod
```

## How It Works

1. **Clone** — shallow-clones the repository to a temp directory (auto-cleaned up)
2. **Detect** — scans file extensions to determine the primary language
3. **Parse** — walks source files, uses tree-sitter queries to extract import statements from the AST (not regex)
4. **Build** — constructs a directed graph of nodes (packages/files) and edges (imports)
5. **Detect Cycles** — DFS-based cycle detection marks circular edges in red
6. **Output** — writes DOT / JSON / SVG to the current directory

## Examples

```bash
# Analyze the popular Go web framework Gin
./depgraph --url https://github.com/gin-gonic/gin --lang go --format svg

# Analyze a Python project
./depgraph --url https://github.com/pallets/flask --lang python --format dot

# Analyze a TypeScript project with depth limit
./depgraph --url https://github.com/microsoft/vscode --lang typescript --depth 2

# Analyze a Java project
./depgraph --url https://github.com/google/guava --lang java --format dot
```
