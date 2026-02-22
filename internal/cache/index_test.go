package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadIndexMissingFile(t *testing.T) {
	idx, err := LoadIndex("/nonexistent/cache.json")
	if err != nil {
		t.Fatal(err)
	}
	if idx.Version != 1 {
		t.Errorf("expected Version 1, got %d", idx.Version)
	}
	if len(idx.Repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(idx.Repos))
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := IndexPath(dir)

	now := time.Now().Truncate(time.Second)
	original := &Index{
		Version: 1,
		Repos: []CachedRepo{
			{
				ID:        "github.com/owner/repo@main",
				URL:       "https://github.com/owner/repo",
				Path:      "repos/github.com/owner/repo/main",
				Branch:    "main",
				Commit:    "abc123",
				ClonedAt:  now,
				UpdatedAt: now,
			},
		},
	}

	if err := Write(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadIndex(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(loaded.Repos))
	}
	if loaded.Repos[0].ID != "github.com/owner/repo@main" {
		t.Errorf("ID mismatch: got %q", loaded.Repos[0].ID)
	}
	if loaded.Repos[0].Commit != "abc123" {
		t.Errorf("Commit mismatch: got %q", loaded.Repos[0].Commit)
	}
}

func TestWriteCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	path := filepath.Join(nested, "cache.json")

	idx := &Index{Version: 1}
	if err := Write(path, idx); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestWriteTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := IndexPath(dir)

	idx := &Index{Version: 1, Repos: []CachedRepo{}}
	Write(path, idx)

	data, _ := os.ReadFile(path)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestIndexPath(t *testing.T) {
	got := IndexPath("/home/user/.pickpocket")
	want := "/home/user/.pickpocket/cache.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetRepoInsert(t *testing.T) {
	idx := &Index{Version: 1}
	idx.SetRepo(CachedRepo{ID: "a@main", Commit: "111"})

	if len(idx.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(idx.Repos))
	}
}

func TestSetRepoUpsert(t *testing.T) {
	idx := &Index{Version: 1}
	idx.SetRepo(CachedRepo{ID: "a@main", Commit: "111"})
	idx.SetRepo(CachedRepo{ID: "a@main", Commit: "222"})

	if len(idx.Repos) != 1 {
		t.Fatalf("expected 1 repo after upsert, got %d", len(idx.Repos))
	}
	if idx.Repos[0].Commit != "222" {
		t.Errorf("expected updated commit '222', got %q", idx.Repos[0].Commit)
	}
}

func TestRemoveRepo(t *testing.T) {
	idx := &Index{Version: 1}
	idx.SetRepo(CachedRepo{ID: "a@main"})

	if !idx.RemoveRepo("a@main") {
		t.Error("expected RemoveRepo to return true")
	}
	if len(idx.Repos) != 0 {
		t.Error("expected 0 repos after removal")
	}
	if idx.RemoveRepo("a@main") {
		t.Error("expected RemoveRepo to return false for missing repo")
	}
}

func TestFindRepo(t *testing.T) {
	idx := &Index{Version: 1}
	idx.SetRepo(CachedRepo{ID: "a@main", Commit: "111"})

	found := idx.FindRepo("a@main")
	if found == nil {
		t.Fatal("expected to find repo")
	}
	if found.Commit != "111" {
		t.Errorf("expected commit '111', got %q", found.Commit)
	}
	if idx.FindRepo("b@dev") != nil {
		t.Error("expected nil for missing repo")
	}
}

func TestDefaultDir(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir == "" {
		t.Error("expected non-empty dir")
	}
	if filepath.Base(dir) != ".pickpocket" {
		t.Errorf("expected .pickpocket dir, got %q", filepath.Base(dir))
	}
}
