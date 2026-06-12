# Verify loop: how agents use depgraph

This guide walks through building and using a **verify loop** step by step.

## The big picture

An "agent" here is **not** a separate program you write inside depgraph. The agent is **Cursor** (or Claude, etc.) — an LLM that can call **tools**. depgraph exposes tools via **MCP**.

```text
┌─────────────────────────────────────────────────────────────┐
│  You (human)                                                │
│    "Fix circular deps in this repo"                         │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Host agent (Cursor chat)                                   │
│    - reads your goal                                        │
│    - plans steps                                            │
│    - calls MCP tools                                        │
│    - edits files in the workspace                           │
│    - repeats until verify passes                            │
└───────────────────────────┬─────────────────────────────────┘
                            │ MCP (stdio JSON-RPC)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  depgraph-mcp (this repo)                                   │
│    analyze_repo  → facts (graph JSON)                       │
│    list_cycles   → where cycles are                         │
│    verify_repo   → pass/fail + violations  ◄── THE GATE     │
└─────────────────────────────────────────────────────────────┘
```

**depgraph = facts + gate.** The LLM = planner + coder.

## Step 1 — Facts engine (`internal/analyze`)

Clone or open a repo, parse imports, build a graph.

```go
result, cleanup, err := analyze.Run(ctx, analyze.Options{Path: "."})
defer cleanup()
```

This step answers: *what depends on what?*

## Step 2 — Rules engine (`internal/verify`)

Turn graph facts into **pass/fail** with structured violations.

```go
report := verify.Run(result, verify.DefaultOptions())
// report.OK == true  → agent can stop
// report.OK == false → agent reads report.Violations and fixes code
```

First rule: **`no_cycles`** — fail if any circular dependency exists.

Example violation JSON:

```json
{
  "ok": false,
  "violations": [
    {
      "rule": "no_cycles",
      "message": "circular dependencies detected",
      "details": {
        "cycle_groups": [{ "nodes": ["a", "b"], "edges": [...] }]
      }
    }
  ]
}
```

## Step 3 — CLI gate (`cmd/verify`)

Humans and CI can run the same check without an LLM:

```bash
go build -o depgraph-verify ./cmd/verify/
./depgraph-verify --path . --json
echo $?   # 0 = pass, 1 = fail, 2 = usage error
```

Shell scripts and GitHub Actions use exit codes. Agents use the JSON body.

## Step 4 — MCP tool (`verify_repo`)

Cursor loads `.cursor/mcp.json`, which starts `depgraph-mcp`. The host agent discovers tools and calls them.

`verify_repo` runs analyze + verify and returns structured JSON. The agent reads `ok` and `violations` to decide what to fix next.

## Step 5 — The agent loop (what happens in chat)

```text
1. User: "Remove circular dependencies"

2. Agent calls verify_repo({ "path": "/workspace" })
   → { "ok": false, "violations": [...] }

3. Agent calls list_cycles({ "path": "/workspace" })
   → understands which packages are involved

4. Agent edits import statements in the workspace

5. Agent calls verify_repo again
   → { "ok": true }

6. Agent tells you it's done
```

You do **not** implement step 4–6 in Go — Cursor does that when it has the `verify_repo` tool.

## Try it yourself

### Terminal (no agent)

```bash
make build-verify
./depgraph-verify --path "$(pwd)" --lang go --json
echo $?   # 0 = pass, 1 = fail
```

Use an absolute path (or `$(pwd)`) so analysis resolves the repo root correctly.

### Cursor (with agent)

1. Ensure `.cursor/mcp.json` points at `depgraph-mcp`
2. Reload MCP
3. Ask: *"Run verify_repo on this repo. If it fails, explain the violations."*

### MCP Inspector (tool debugging without agent)

```bash
npx @modelcontextprotocol/inspector go run ./cmd/depgraph-mcp/
```

Call `verify_repo` manually and inspect JSON.

## What to add next

| Step | Feature | Purpose |
|------|---------|---------|
| 6 | Analysis cache | Avoid re-cloning on every tool call |
| 7 | `max_fan_out` rule | Fail when a package imports too many others |
| 8 | YAML policy file | Custom layer rules (`api` must not import `db`) |
| 9 | Graph diff vs `main` | PR agent: only fail on new violations |

## Mental checklist

- [ ] **analyze** produces facts
- [ ] **verify** produces pass/fail
- [ ] **CLI** uses exit codes (CI)
- [ ] **MCP** exposes the same logic (agents)
- [ ] **Host LLM** loops: verify → fix → verify
