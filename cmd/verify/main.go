// Command verify is the "gate" in an agent loop: analyze deps, check rules, exit non-zero on failure.
//
// Usage:
//
//	depgraph-verify --path .
//	depgraph-verify --path . --json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/depgraph/internal/analyze"
	"github.com/depgraph/internal/verify"
)

func main() {
	var (
		repoURL  = flag.String("url", "", "GitHub/GitLab repository URL")
		repoPath = flag.String("path", "", "Local repository path")
		language = flag.String("lang", "auto", "Language: auto, go, python, javascript, typescript, java")
		maxDepth = flag.Int("depth", 0, "Max directory depth (0 = unlimited)")
		jsonOut  = flag.Bool("json", false, "Print verify report as JSON")
	)
	flag.Parse()

	if *repoURL == "" && *repoPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --url or --path is required")
		flag.PrintDefaults()
		os.Exit(2)
	}

	log.SetOutput(os.Stderr)

	result, cleanup, err := analyze.Run(context.Background(), analyze.Options{
		URL:      *repoURL,
		Path:     *repoPath,
		Lang:     *language,
		MaxDepth: *maxDepth,
	})
	if err != nil {
		log.Fatalf("analysis failed: %v", err)
	}
	defer cleanup()

	report := verify.Run(result, verify.DefaultOptions())

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			log.Fatalf("encode report: %v", err)
		}
	} else {
		printHuman(report)
	}

	if !report.OK {
		os.Exit(1)
	}
}

func printHuman(report verify.Report) {
	if report.OK {
		fmt.Println("✅ verify passed")
	} else {
		fmt.Println("❌ verify failed")
	}
	fmt.Printf("   repo     : %s\n", report.RepoPath)
	fmt.Printf("   language : %s\n", report.Language)
	fmt.Printf("   files    : %d\n", report.Stats.Files)
	fmt.Printf("   cycles   : %d\n", report.Stats.Cycles)
	for _, v := range report.Violations {
		fmt.Printf("   [%s] %s\n", v.Rule, v.Message)
	}
}
