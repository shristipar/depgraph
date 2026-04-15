package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/depgraph/internal/cloner"
	"github.com/depgraph/internal/graph"
	"github.com/depgraph/internal/parser"
)

func main() {
	var (
		repoURL    = flag.String("url", "", "GitHub/GitLab repository URL (required)")
		outputFile = flag.String("output", "dependency_graph", "Output file name (without extension)")
		outputFmt  = flag.String("format", "dot", "Output format: dot, json, svg")
		language   = flag.String("lang", "auto", "Language to analyze: auto, go, python, javascript, typescript, java")
		maxDepth   = flag.Int("depth", 0, "Max import depth (0 = unlimited)")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *repoURL == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		fmt.Fprintln(os.Stderr, "\nUsage:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  depgraph --url https://github.com/user/repo")
		fmt.Fprintln(os.Stderr, "  depgraph --url https://gitlab.com/user/repo --format svg --output mygraph")
		fmt.Fprintln(os.Stderr, "  depgraph --url https://github.com/user/repo --lang go --depth 3")
		os.Exit(1)
	}

	log.SetFlags(0)
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Step 1: Clone repository
	fmt.Printf("🔗 Cloning repository: %s\n", *repoURL)
	cl := cloner.New()
	repoPath, cleanup, err := cl.Clone(*repoURL)
	if err != nil {
		log.Fatalf("❌ Failed to clone repository: %v", err)
	}
	defer cleanup()
	fmt.Printf("✅ Cloned to: %s\n", repoPath)

	// Step 2: Detect language if auto
	lang := *language
	if lang == "auto" {
		lang, err = parser.DetectLanguage(repoPath)
		if err != nil {
			log.Fatalf("❌ Failed to detect language: %v", err)
		}
		fmt.Printf("🔍 Detected language: %s\n", lang)
	}

	// Step 3: Parse dependencies
	fmt.Printf("🌳 Parsing dependencies using tree-sitter...\n")
	p, err := parser.New(lang)
	if err != nil {
		log.Fatalf("❌ Failed to create parser: %v", err)
	}

	deps, err := p.ParseDirectory(repoPath, *maxDepth)
	if err != nil {
		log.Fatalf("❌ Failed to parse dependencies: %v", err)
	}
	fmt.Printf("📦 Found %d files with %d dependency edges\n", deps.FileCount, deps.EdgeCount)

	// Step 4: Build graph
	fmt.Printf("📊 Building dependency graph...\n")
	g := graph.New(deps)

	// Step 5: Output
	outPath := *outputFile
	switch *outputFmt {
	case "dot":
		outPath += ".dot"
		err = g.WriteDOT(outPath)
	case "json":
		outPath += ".json"
		err = g.WriteJSON(outPath)
	case "svg":
		outPath += ".svg"
		err = g.WriteSVG(outPath)
	default:
		log.Fatalf("❌ Unknown format: %s (use dot, json, or svg)", *outputFmt)
	}

	if err != nil {
		log.Fatalf("❌ Failed to write output: %v", err)
	}

	absPath, _ := filepath.Abs(outPath)
	fmt.Printf("✅ Dependency graph written to: %s\n", absPath)

	// Print summary
	fmt.Println()
	fmt.Println("📈 Summary:")
	fmt.Printf("   Files analyzed : %d\n", deps.FileCount)
	fmt.Printf("   Unique modules : %d\n", deps.NodeCount)
	fmt.Printf("   Dependency edges: %d\n", deps.EdgeCount)
	fmt.Printf("   Circular deps  : %d\n", g.CircularCount())
	if *outputFmt == "dot" {
		fmt.Println()
		fmt.Println("💡 Render with Graphviz:")
		fmt.Printf("   dot -Tsvg %s -o graph.svg\n", outPath)
		fmt.Printf("   dot -Tpng %s -o graph.png\n", outPath)
	}
}
