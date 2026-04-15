package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavaParser_ParseDirectory(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "src", "main", "java", "com", "demo")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	code := "package com.demo;\n\nimport java.util.List;\nimport org.slf4j.Logger;\n\npublic class Main {}\n"
	if err := os.WriteFile(filepath.Join(sub, "Main.java"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	jp := NewJavaParser()
	g, err := jp.ParseDirectory(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if g.FileCount != 1 {
		t.Fatalf("FileCount = %d, want 1", g.FileCount)
	}
	if g.EdgeCount != 2 {
		t.Fatalf("EdgeCount = %d, want 2", g.EdgeCount)
	}
}

func TestPackageLCP(t *testing.T) {
	if got := packageLCP([]string{"com.foo.app", "com.foo.bar"}); got != "com.foo" {
		t.Fatalf("packageLCP = %q", got)
	}
	if got := packageLCP([]string{}); got != "" {
		t.Fatalf("empty packageLCP = %q", got)
	}
}
