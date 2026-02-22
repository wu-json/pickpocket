package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultBranch detects the default branch of a remote repository
// by parsing the output of `git ls-remote --symref <url> HEAD`.
func DefaultBranch(repoURL string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "--symref", repoURL, "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote %s: %w", repoURL, err)
	}

	// Look for line: ref: refs/heads/<branch>\tHEAD
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "ref: refs/heads/") {
			// Split on tab first to isolate "ref: refs/heads/<branch>"
			tabParts := strings.SplitN(line, "\t", 2)
			branch := strings.TrimPrefix(tabParts[0], "ref: refs/heads/")
			if branch != "" {
				return branch, nil
			}
		}
	}

	return "", fmt.Errorf("could not detect default branch for %s", repoURL)
}

// Clone clones a repository into destDir, checking out only the given branch.
// Returns the HEAD commit SHA of the clone.
func Clone(repoURL, branch, destDir string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return "", fmt.Errorf("creating parent dirs: %w", err)
	}

	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", repoURL, destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return HeadCommit(destDir)
}

// HeadCommit returns the full SHA of HEAD in the given repo directory.
func HeadCommit(repoDir string) (string, error) {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", repoDir, err)
	}
	return strings.TrimSpace(string(out)), nil
}
