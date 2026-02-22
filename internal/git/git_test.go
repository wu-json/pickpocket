package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// addCommitToBare clones the bare repo, adds a commit, pushes, and returns the new SHA.
func addCommitToBare(t *testing.T, bareDir string) string {
	t.Helper()

	tmpDir := filepath.Join(t.TempDir(), "pusher")
	cmd := exec.Command("git", "clone", bareDir, tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone for push: %s: %v", out, err)
	}

	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = tmpDir
		c.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	run("commit", "--allow-empty", "-m", "extra commit")
	run("push", "origin", "main")

	sha, err := HeadCommit(tmpDir)
	if err != nil {
		t.Fatalf("HeadCommit after push: %v", err)
	}
	return sha
}

func TestFetch(t *testing.T) {
	bareDir := setupBareRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	if _, err := Clone(bareDir, "main", cloneDir); err != nil {
		t.Fatalf("Clone() error: %v", err)
	}

	// Add a new commit to the bare repo
	newSHA := addCommitToBare(t, bareDir)

	// Fetch should bring in the new commit
	if err := Fetch(cloneDir); err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	// origin/main should now point to the new commit
	cmd := exec.Command("git", "-C", cloneDir, "rev-parse", "origin/main")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse origin/main: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != newSHA {
		t.Errorf("origin/main = %q, want %q", got, newSHA)
	}
}

func TestFetchInvalidDir(t *testing.T) {
	if err := Fetch("/nonexistent/dir"); err == nil {
		t.Error("expected error for invalid dir, got nil")
	}
}

func TestCheckout(t *testing.T) {
	bareDir := setupBareRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	initialSHA, err := Clone(bareDir, "main", cloneDir)
	if err != nil {
		t.Fatalf("Clone() error: %v", err)
	}

	// Add a second commit and fetch it
	addCommitToBare(t, bareDir)
	if err := Fetch(cloneDir); err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	// Checkout the initial commit
	if err := Checkout(cloneDir, initialSHA); err != nil {
		t.Fatalf("Checkout() error: %v", err)
	}

	got, err := HeadCommit(cloneDir)
	if err != nil {
		t.Fatalf("HeadCommit() error: %v", err)
	}
	if got != initialSHA {
		t.Errorf("HeadCommit() = %q, want %q", got, initialSHA)
	}
}

func TestCheckoutInvalidRef(t *testing.T) {
	bareDir := setupBareRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	if _, err := Clone(bareDir, "main", cloneDir); err != nil {
		t.Fatalf("Clone() error: %v", err)
	}

	if err := Checkout(cloneDir, "nonexistent-ref-abc123"); err == nil {
		t.Error("expected error for invalid ref, got nil")
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
