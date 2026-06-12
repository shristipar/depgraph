// Package analyze orchestrates cloning, parsing, and graph construction.
package analyze

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/depgraph/internal/cloner"
	"github.com/depgraph/internal/graph"
	"github.com/depgraph/internal/parser"
)

// Options configures a dependency analysis run.
type Options struct {
	URL      string // GitHub/GitLab URL; ignored when Path is set
	Path     string // local repository root; skips clone
	Lang     string // language or "auto"
	MaxDepth int    // max directory depth (0 = unlimited)
}

// Result holds the output of a completed analysis.
type Result struct {
	Deps     *parser.DependencyGraph
	Graph    *graph.Graph
	Language string
	RepoPath string
}

// Run clones or opens a repository, parses imports, and builds the graph.
// The returned cleanup function removes a temporary clone directory; it is a
// no-op when analyzing a local Path. Call cleanup when finished.
func Run(ctx context.Context, opts Options) (*Result, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	if opts.URL == "" && opts.Path == "" {
		return nil, nil, fmt.Errorf("either URL or Path is required")
	}
	if opts.URL != "" && opts.Path != "" {
		return nil, nil, fmt.Errorf("provide URL or Path, not both")
	}

	lang := opts.Lang
	if lang == "" {
		lang = "auto"
	}

	repoPath, cleanup, err := resolveRepoPath(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	if lang == "auto" {
		lang, err = parser.DetectLanguage(repoPath)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("detect language: %w", err)
		}
	}

	if err := ctx.Err(); err != nil {
		cleanup()
		return nil, nil, err
	}

	p, err := parser.New(lang)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	deps, err := p.ParseDirectory(repoPath, opts.MaxDepth)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("parse dependencies: %w", err)
	}

	return &Result{
		Deps:     deps,
		Graph:    graph.New(deps),
		Language: lang,
		RepoPath: repoPath,
	}, cleanup, nil
}

func resolveRepoPath(ctx context.Context, opts Options) (string, func(), error) {
	if opts.Path != "" {
		info, err := os.Stat(opts.Path)
		if err != nil {
			return "", nil, fmt.Errorf("local path: %w", err)
		}
		if !info.IsDir() {
			return "", nil, fmt.Errorf("local path %q is not a directory", opts.Path)
		}
		abs, err := filepath.Abs(opts.Path)
		if err != nil {
			return "", nil, fmt.Errorf("local path: %w", err)
		}
		return abs, func() {}, nil
	}

	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	repoPath, cleanup, err := cloner.New().Clone(opts.URL)
	if err != nil {
		return "", nil, fmt.Errorf("clone repository: %w", err)
	}
	return repoPath, cleanup, nil
}
