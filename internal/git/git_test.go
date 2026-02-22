package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupBareRepo creates a local bare repo with one commit and returns its path.
func setupBareRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Create a normal repo with a commit
	workDir := filepath.Join(dir, "work")
	bareDir := filepath.Join(dir, "bare.git")

	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %s: %v", name, args, out, err)
		}
	}

	os.MkdirAll(workDir, 0755)
	run("git", "init", "-b", "main")
	run("git", "commit", "--allow-empty", "-m", "initial")

	// Clone to bare repo
	cmd := exec.Command("git", "clone", "--bare", workDir, bareDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone --bare: %s: %v", out, err)
	}

	return bareDir
}

func TestCloneLocalRepo(t *testing.T) {
	bareDir := setupBareRepo(t)
	destDir := filepath.Join(t.TempDir(), "clone")

	sha, err := Clone(bareDir, "main", destDir)
	if err != nil {
		t.Fatalf("Clone() error: %v", err)
	}

	if len(sha) != 40 {
		t.Errorf("expected 40-char SHA, got %q (len %d)", sha, len(sha))
	}

	// Verify the clone directory exists
	if _, err := os.Stat(filepath.Join(destDir, ".git")); err != nil {
		t.Errorf("expected .git dir in clone: %v", err)
	}
}

func TestHeadCommit(t *testing.T) {
	bareDir := setupBareRepo(t)
	destDir := filepath.Join(t.TempDir(), "clone")

	expected, err := Clone(bareDir, "main", destDir)
	if err != nil {
		t.Fatalf("Clone() error: %v", err)
	}

	got, err := HeadCommit(destDir)
	if err != nil {
		t.Fatalf("HeadCommit() error: %v", err)
	}

	if got != expected {
		t.Errorf("HeadCommit() = %q, want %q", got, expected)
	}
}

func TestHeadCommitInvalidDir(t *testing.T) {
	_, err := HeadCommit("/nonexistent/dir")
	if err == nil {
		t.Error("expected error for invalid dir, got nil")
	}
}

func TestDefaultBranch(t *testing.T) {
	// Network test — skip if we can't reach github.com
	branch, err := DefaultBranch("https://github.com/charmbracelet/lipgloss")
	if err != nil {
		t.Skipf("skipping network test: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("DefaultBranch() = %q, expected main or master", branch)
	}
}
