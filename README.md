# depgraph — Dependency Graph Generator

A CLI and MCP server that analyzes source-code dependencies using **tree-sitter** for accurate AST-based import extraction. Clone a remote repository or analyze a local checkout, then output a dependency graph as DOT, JSON, or SVG—or expose structured graph queries to AI agents via MCP.

## Features

- Clones public GitHub or GitLab repositories (shallow clone for speed), or analyzes a **local path**
- Uses **tree-sitter** for language-aware AST parsing (not regex)
- Generates dependency graphs in **DOT**, **JSON**, or **SVG**
- Detects **circular dependencies** and highlights them in visual output
- Color-coded graphs: internal packages (blue) vs. external dependencies (orange)
- Supports **Go**, **Python**, **Java**, **JavaScript**, and **TypeScript**
- Auto-detects the primary language of a repository
- **MCP server** for agent workflows: full graph analysis, cycle listing, and neighbor queries

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
make build        # builds ./depgraph
make build-mcp    # builds ./depgraph-mcp
```

Or build manually:

```bash
go build -o depgraph ./cmd/
go build -o depgraph-mcp ./cmd/depgraph-mcp/
```

**Prerequisites:** `git` must be in your PATH when using `--url`. For SVG rendering via DOT, install [Graphviz](https://graphviz.org/).

## CLI Usage

```bash
# Remote repo — auto-detect language, output DOT
./depgraph --url https://github.com/user/repo

# Local repo (no clone)
./depgraph --path /path/to/repo --lang go --format json

# Specify language and output format
./depgraph --url https://github.com/gin-gonic/gin --lang go --format svg --output gin-graph

# Limit analysis depth (useful for large monorepos)
./depgraph --url https://github.com/user/repo --depth 3
```

### CLI Flags

| Flag        | Default              | Description                                        |
|-------------|----------------------|----------------------------------------------------|
| `--url`     | *(see below)*        | GitHub or GitLab repository URL                    |
| `--path`    | *(see below)*        | Local repository path (alternative to `--url`)     |
| `--lang`    | `auto`               | Language: `auto`, `go`, `python`, `javascript`, `typescript`, `java` |
| `--format`  | `dot`                | Output format: `dot`, `json`, `svg`               |
| `--output`  | `dependency_graph`   | Output filename (extension added automatically)    |
| `--depth`   | `0` (unlimited)      | Max directory depth to analyze                     |
| `--verbose` | `false`              | Enable verbose/debug logging                       |

Either `--url` or `--path` is required (not both).

## Output Formats

### DOT (Graphviz)

```bash
./depgraph --url https://github.com/user/repo --format dot
dot -Tsvg dependency_graph.dot -o graph.svg
dot -Tpng dependency_graph.dot -o graph.png
```

### SVG (inline, no Graphviz needed)

```bash
./depgraph --url https://github.com/user/repo --format svg
open dependency_graph.svg
```

### JSON (for tooling integration)

```json
{
  "nodes": [{ "id": "github.com/user/repo", "label": "repo", "external": false, "lang": "go" }],
  "edges": [{ "from": "github.com/user/repo", "to": "fmt" }],
  "stats": { "files": 42, "nodes": 18, "edges": 65, "cycles": 0 }
}
```

## MCP Server

depgraph includes an MCP server so agents (Cursor, Claude Desktop, etc.) can query dependency structure programmatically.

### Build and configure

```bash
make build-mcp
```

Project-local Cursor config (`.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "depgraph": {
      "command": "go",
      "args": ["run", "./cmd/depgraph-mcp/"]
    }
  }
}
```

Or point to a built binary:

```json
{
  "mcpServers": {
    "depgraph": {
      "command": "/absolute/path/to/depgraph-mcp"
    }
  }
}
```

Reload MCP in Cursor after changing the config.

### MCP Tools

| Tool             | Description |
|------------------|-------------|
| `analyze_repo`   | Full dependency graph: nodes, edges, stats, and cycle edges |
| `list_cycles`    | Circular dependency groups (strongly connected components) |
| `get_neighbors`  | Upstream and downstream dependencies for a specific node |

Common arguments for all tools:

| Argument | Description |
|----------|-------------|
| `url`    | GitHub/GitLab URL (omit when using `path`) |
| `path`   | Local repository path (preferred when the agent has the repo open) |
| `lang`   | `auto`, `go`, `python`, `javascript`, `typescript`, or `java` (default: `auto`) |
| `depth`  | Max directory depth (`0` = unlimited) |

`get_neighbors` also requires `node_id` (a package/module ID from `analyze_repo` output).

Example agent flow:

1. Call `analyze_repo` with `{ "path": "/path/to/repo", "lang": "go" }`
2. Call `get_neighbors` with `{ "path": "/path/to/repo", "node_id": "github.com/user/repo/internal/auth" }`
3. Call `list_cycles` with `{ "path": "/path/to/repo" }`

### Test with MCP Inspector

```bash
npx @modelcontextprotocol/inspector go run ./cmd/depgraph-mcp/
```

### Example MCP response (`analyze_repo`)

```json
{
  "language": "go",
  "repo_path": "/path/to/repo",
  "stats": { "files": 42, "nodes": 18, "edges": 65, "cycles": 2 },
  "nodes": [{ "id": "github.com/user/repo/internal/auth", "label": "auth", "external": false, "lang": "go" }],
  "edges": [{ "from": "github.com/user/repo/cmd", "to": "github.com/user/repo/internal/auth" }],
  "cycle_edges": [{ "from": "a", "to": "b" }, { "from": "b", "to": "a" }]
}
```

## Architecture

```
depgraph/
├── cmd/
│   ├── main.go              # CLI entry point
│   └── depgraph-mcp/
│       └── main.go          # MCP server (stdio)
├── internal/
│   ├── analyze/
│   │   ├── analyze.go       # Shared clone/parse/graph pipeline
│   │   ├── report.go        # Structured JSON reports
│   │   └── query.go         # Cycles and neighbors reports
│   ├── cloner/
│   │   └── cloner.go        # Git clone logic (GitHub/GitLab)
│   ├── parser/
│   │   ├── parser.go        # Language detection + Parser interface
│   │   ├── go_parser.go     # Go import extraction (tree-sitter)
│   │   ├── python_parser.go # Python import extraction (tree-sitter)
│   │   ├── js_parser.go     # JS/TS import extraction (tree-sitter)
│   │   └── java_parser.go   # Java import extraction (tree-sitter)
│   └── graph/
│       ├── graph.go         # Graph rendering + cycle detection
│       └── query.go         # Cycles(), Neighbors()
├── .cursor/
│   └── mcp.json             # Cursor MCP configuration
└── go.mod
```

## How It Works

1. **Resolve** — shallow-clone a remote repo to a temp directory, or use a local `--path`
2. **Detect** — scan file extensions to determine the primary language
3. **Parse** — walk source files and use tree-sitter queries to extract imports from the AST
4. **Build** — construct a directed graph of nodes (packages/modules) and edges (imports)
5. **Detect cycles** — DFS and strongly connected components; cyclic edges highlighted in red
6. **Output** — write DOT / JSON / SVG (CLI), or return structured JSON (MCP)

## Examples

```bash
# Analyze the Go web framework Gin
./depgraph --url https://github.com/gin-gonic/gin --lang go --format svg

# Analyze a local checkout
./depgraph --path ~/projects/myapp --lang go --format json

# Analyze a Python project
./depgraph --url https://github.com/pallets/flask --lang python --format dot

# Analyze a TypeScript project with depth limit
./depgraph --url https://github.com/microsoft/vscode --lang typescript --depth 2

# Run tests
make test
```
