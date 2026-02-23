package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// validateRepo checks that repoDir contains a .git directory.
// Returns an actionable error if the directory is missing or corrupt.
func validateRepo(repoDir string) error {
	gitDir := filepath.Join(repoDir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return fmt.Errorf("not a git repository: %s (missing .git — the cache entry may be corrupt, try 'pick cache remove' and reinstall)", repoDir)
	}
	// .git can be a directory (normal clone) or a file (worktree)
	if !info.IsDir() && info.Mode().IsRegular() {
		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("not a git repository: %s (corrupt .git entry)", repoDir)
	}
	return nil
}

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
	if err := validateRepo(repoDir); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", repoDir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Fetch fetches the latest refs from origin in the given repo directory.
func Fetch(repoDir string) error {
	if err := validateRepo(repoDir); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", repoDir, "fetch", "origin")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch in %s: %s: %w", repoDir, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Checkout checks out the given ref (branch, tag, or commit SHA) in the repo directory.
func Checkout(repoDir, ref string) error {
	if err := validateRepo(repoDir); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", repoDir, "checkout", ref)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s in %s: %s: %w", ref, repoDir, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// ResetHard resets the repo to the given ref, discarding local changes.
func ResetHard(repoDir, ref string) error {
	if err := validateRepo(repoDir); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", repoDir, "reset", "--hard", ref)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git reset --hard %s in %s: %s: %w", ref, repoDir, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// WorktreeAdd creates a new worktree at worktreePath checked out to the given commit.
func WorktreeAdd(repoDir, worktreePath, commit string) error {
	if err := validateRepo(repoDir); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", "--detach", worktreePath, commit)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add in %s: %s: %w", repoDir, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// WorktreeRemove removes the worktree at worktreePath.
// Ignores errors if the worktree path doesn't exist.
func WorktreeRemove(repoDir, worktreePath string) error {
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return nil
	}
	if err := validateRepo(repoDir); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", repoDir, "worktree", "remove", "--force", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove in %s: %s: %w", repoDir, strings.TrimSpace(string(out)), err)
	}
	return nil
}
