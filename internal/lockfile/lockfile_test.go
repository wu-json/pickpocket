package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)

	original := &Lockfile{
		Locked: []LockedPick{
			{URL: "https://github.com/owner/repo", Branch: "main", Commit: "abc123"},
			{URL: "https://github.com/other/lib", Branch: "dev", Commit: "def456"},
		},
	}

	if err := Write(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Locked) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Locked))
	}
	if loaded.Locked[0].Commit != "abc123" {
		t.Errorf("commit mismatch: got %q", loaded.Locked[0].Commit)
	}
}

func TestWriteTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)

	lf := &Lockfile{Locked: []LockedPick{}}
	Write(path, lf)

	data, _ := os.ReadFile(path)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestSetEntryInsert(t *testing.T) {
	lf := &Lockfile{}
	lf.SetEntry(LockedPick{URL: "https://github.com/a/b", Branch: "main", Commit: "aaa"})

	if len(lf.Locked) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(lf.Locked))
	}
}

func TestSetEntryUpsert(t *testing.T) {
	lf := &Lockfile{}
	lf.SetEntry(LockedPick{URL: "https://github.com/a/b", Branch: "main", Commit: "aaa"})
	lf.SetEntry(LockedPick{URL: "https://github.com/a/b", Branch: "main", Commit: "bbb"})

	if len(lf.Locked) != 1 {
		t.Fatalf("expected 1 entry after upsert, got %d", len(lf.Locked))
	}
	if lf.Locked[0].Commit != "bbb" {
		t.Errorf("expected updated commit 'bbb', got %q", lf.Locked[0].Commit)
	}
}

func TestRemoveEntry(t *testing.T) {
	lf := &Lockfile{}
	lf.SetEntry(LockedPick{URL: "https://github.com/a/b", Branch: "main", Commit: "aaa"})

	if !lf.RemoveEntry("https://github.com/a/b", "main") {
		t.Error("expected RemoveEntry to return true")
	}
	if len(lf.Locked) != 0 {
		t.Error("expected 0 entries after removal")
	}
	if lf.RemoveEntry("https://github.com/a/b", "main") {
		t.Error("expected RemoveEntry to return false for missing entry")
	}
}

func TestFindEntry(t *testing.T) {
	lf := &Lockfile{}
	lf.SetEntry(LockedPick{URL: "https://github.com/a/b", Branch: "main", Commit: "aaa"})

	if lf.FindEntry("https://github.com/a/b", "main") == nil {
		t.Error("expected to find entry")
	}
	if lf.FindEntry("https://github.com/a/b", "dev") != nil {
		t.Error("expected nil for wrong branch")
	}
	if lf.FindEntry("https://github.com/other/repo", "main") != nil {
		t.Error("expected nil for wrong URL")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
