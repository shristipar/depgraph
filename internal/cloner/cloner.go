// Package cloner handles cloning repositories from GitHub and GitLab.
package cloner

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Cloner clones remote Git repositories to a local temp directory.
type Cloner struct {
	// Depth limits how many commits to fetch. 0 = full clone.
	Depth int
}

// New creates a Cloner with shallow clone enabled (depth=1) for speed.
func New() *Cloner {
	return &Cloner{Depth: 1}
}

// Clone clones the given GitHub/GitLab URL into a temporary directory.
// Returns the local path, a cleanup function, and any error.
func (c *Cloner) Clone(rawURL string) (string, func(), error) {
	normalized, err := normalizeURL(rawURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid repository URL %q: %w", rawURL, err)
	}

	tmpDir, err := os.MkdirTemp("", "depgraph-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	if err := c.runGitClone(normalized, tmpDir); err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}

// normalizeURL validates and normalizes GitHub/GitLab URLs.
func normalizeURL(rawURL string) (string, error) {
	// Handle SSH-style git@github.com:user/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		return rawURL, nil
	}

	// Ensure scheme is present
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	host := strings.ToLower(u.Hostname())
	if !isSupported(host) {
		return "", fmt.Errorf("unsupported host %q — only github.com and gitlab.com are supported", host)
	}

	// Strip .git suffix duplicates then add canonical one
	u.Path = strings.TrimSuffix(u.Path, ".git") + ".git"
	// Clear query/fragment for clean clone URL
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), nil
}

func isSupported(host string) bool {
	return host == "github.com" || host == "gitlab.com" ||
		strings.HasSuffix(host, ".github.com") ||
		strings.HasSuffix(host, ".gitlab.com")
}

func (c *Cloner) runGitClone(repoURL, destDir string) error {
	args := []string{"clone", "--quiet"}
	if c.Depth > 0 {
		args = append(args, fmt.Sprintf("--depth=%d", c.Depth))
	}
	args = append(args, repoURL, destDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed for %s: %w", repoURL, err)
	}

	// If dest is a bare clone, look for a nested directory
	entries, _ := os.ReadDir(destDir)
	if len(entries) == 1 && entries[0].IsDir() {
		nested := filepath.Join(destDir, entries[0].Name())
		// Re-point if that nested dir looks like a repo root
		if _, err := os.Stat(filepath.Join(nested, "go.mod")); err == nil {
			return nil // Already in the right place
		}
	}

	return nil
}
