package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/depgraph/internal/analyze"
)

func main() {
	var (
		repoURL    = flag.String("url", "", "GitHub/GitLab repository URL")
		repoPath   = flag.String("path", "", "Local repository path (alternative to --url)")
		outputFile = flag.String("output", "dependency_graph", "Output file name (without extension)")
		outputFmt  = flag.String("format", "dot", "Output format: dot, json, svg")
		language   = flag.String("lang", "auto", "Language to analyze: auto, go, python, javascript, typescript, java")
		maxDepth   = flag.Int("depth", 0, "Max import depth (0 = unlimited)")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *repoURL == "" && *repoPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --url or --path is required")
		fmt.Fprintln(os.Stderr, "\nUsage:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  depgraph --url https://github.com/user/repo")
		fmt.Fprintln(os.Stderr, "  depgraph --path /path/to/local/repo --format svg")
		fmt.Fprintln(os.Stderr, "  depgraph --url https://github.com/user/repo --lang go --depth 3")
		os.Exit(1)
	}

	log.SetFlags(0)
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	opts := analyze.Options{
		URL:      *repoURL,
		Path:     *repoPath,
		Lang:     *language,
		MaxDepth: *maxDepth,
	}

	if opts.URL != "" {
		fmt.Printf("🔗 Cloning repository: %s\n", opts.URL)
	} else {
		fmt.Printf("📂 Analyzing local repository: %s\n", opts.Path)
	}

	result, cleanup, err := analyze.Run(context.Background(), opts)
	if err != nil {
		log.Fatalf("❌ Analysis failed: %v", err)
	}
	defer cleanup()

	if opts.URL != "" {
		fmt.Printf("✅ Cloned to: %s\n", result.RepoPath)
	}
	if *language == "auto" {
		fmt.Printf("🔍 Detected language: %s\n", result.Language)
	}

	fmt.Printf("🌳 Parsing dependencies using tree-sitter...\n")
	fmt.Printf("📦 Found %d files with %d dependency edges\n", result.Deps.FileCount, result.Deps.EdgeCount)
	fmt.Printf("📊 Building dependency graph...\n")

	outPath := *outputFile
	switch *outputFmt {
	case "dot":
		outPath += ".dot"
		err = result.Graph.WriteDOT(outPath)
	case "json":
		outPath += ".json"
		err = result.Graph.WriteJSON(outPath)
	case "svg":
		outPath += ".svg"
		err = result.Graph.WriteSVG(outPath)
	default:
		log.Fatalf("❌ Unknown format: %s (use dot, json, or svg)", *outputFmt)
	}

	if err != nil {
		log.Fatalf("❌ Failed to write output: %v", err)
	}

	absPath, _ := filepath.Abs(outPath)
	fmt.Printf("✅ Dependency graph written to: %s\n", absPath)

	fmt.Println()
	fmt.Println("📈 Summary:")
	fmt.Printf("   Files analyzed : %d\n", result.Deps.FileCount)
	fmt.Printf("   Unique modules : %d\n", result.Deps.NodeCount)
	fmt.Printf("   Dependency edges: %d\n", result.Deps.EdgeCount)
	fmt.Printf("   Circular deps  : %d\n", result.Graph.CircularCount())
	if *outputFmt == "dot" {
		fmt.Println()
		fmt.Println("💡 Render with Graphviz:")
		fmt.Printf("   dot -Tsvg %s -o graph.svg\n", outPath)
		fmt.Printf("   dot -Tpng %s -o graph.png\n", outPath)
	}
}
