package pickfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	// Create a temp dir structure: root/.pickpocket, root/a/b/
	root := t.TempDir()
	pfPath := filepath.Join(root, Filename)
	os.WriteFile(pfPath, []byte(`{"picks":[]}`), 0644)

	nested := filepath.Join(root, "a", "b")
	os.MkdirAll(nested, 0755)

	// Discover from nested dir should find root's .pickpocket
	got, err := Discover(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != pfPath {
		t.Errorf("got %q, want %q", got, pfPath)
	}
}

func TestDiscoverFromSameDir(t *testing.T) {
	root := t.TempDir()
	pfPath := filepath.Join(root, Filename)
	os.WriteFile(pfPath, []byte(`{"picks":[]}`), 0644)

	got, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != pfPath {
		t.Errorf("got %q, want %q", got, pfPath)
	}
}

func TestDiscoverNotFound(t *testing.T) {
	root := t.TempDir()
	_, err := Discover(root)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)

	original := &Pickfile{
		Picks: []Pick{
			{URL: "https://github.com/owner/repo", Branch: "main", Tags: []string{"llm", "tools"}},
			{URL: "https://github.com/other/lib"},
		},
	}

	if err := Write(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Picks) != 2 {
		t.Fatalf("expected 2 picks, got %d", len(loaded.Picks))
	}
	if loaded.Picks[0].URL != original.Picks[0].URL {
		t.Errorf("URL mismatch: got %q", loaded.Picks[0].URL)
	}
	if loaded.Picks[0].Branch != "main" {
		t.Errorf("Branch mismatch: got %q", loaded.Picks[0].Branch)
	}
	if len(loaded.Picks[0].Tags) != 2 {
		t.Errorf("Tags mismatch: got %v", loaded.Picks[0].Tags)
	}
}

func TestOmitempty(t *testing.T) {
	// A pick with no branch and nil tags should not have those keys in JSON
	pf := &Pickfile{
		Picks: []Pick{
			{URL: "https://github.com/owner/repo"},
		},
	}

	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Unmarshal into a map to check keys
	var raw map[string][]map[string]interface{}
	json.Unmarshal(data, &raw)

	pick := raw["picks"][0]
	if _, ok := pick["branch"]; ok {
		t.Error("expected branch to be omitted")
	}
	if _, ok := pick["commit"]; ok {
		t.Error("expected commit to be omitted")
	}
	if _, ok := pick["tags"]; ok {
		t.Error("expected tags to be omitted")
	}
}

func TestWriteTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)

	pf := &Pickfile{Picks: []Pick{}}
	Write(path, pf)

	data, _ := os.ReadFile(path)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestAddPickDuplicate(t *testing.T) {
	pf := &Pickfile{}
	p := Pick{URL: "https://github.com/owner/repo", Branch: "main"}
	if err := pf.AddPick(p); err != nil {
		t.Fatal(err)
	}
	if err := pf.AddPick(p); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestAddPickDifferentBranch(t *testing.T) {
	pf := &Pickfile{}
	pf.AddPick(Pick{URL: "https://github.com/owner/repo", Branch: "main"})
	// Same URL, different branch — should be allowed
	if err := pf.AddPick(Pick{URL: "https://github.com/owner/repo", Branch: "dev"}); err != nil {
		t.Fatalf("same URL different branch should be allowed: %v", err)
	}
	if len(pf.Picks) != 2 {
		t.Errorf("expected 2 picks, got %d", len(pf.Picks))
	}
}

func TestRemovePick(t *testing.T) {
	pf := &Pickfile{}
	pf.AddPick(Pick{URL: "https://github.com/owner/repo", Branch: "main"})

	if !pf.RemovePick("https://github.com/owner/repo", "main") {
		t.Error("expected RemovePick to return true")
	}
	if len(pf.Picks) != 0 {
		t.Error("expected 0 picks after removal")
	}
	if pf.RemovePick("https://github.com/owner/repo", "main") {
		t.Error("expected RemovePick to return false for missing pick")
	}
}

func TestFindPick(t *testing.T) {
	pf := &Pickfile{}
	pf.AddPick(Pick{URL: "https://github.com/owner/repo", Branch: "main"})

	if pf.FindPick("https://github.com/owner/repo", "main") == nil {
		t.Error("expected to find pick")
	}
	if pf.FindPick("https://github.com/owner/repo", "dev") != nil {
		t.Error("expected nil for wrong branch")
	}
}

func TestFindByTag(t *testing.T) {
	pf := &Pickfile{
		Picks: []Pick{
			{URL: "a", Tags: []string{"llm", "tools"}},
			{URL: "b", Tags: []string{"llm"}},
			{URL: "c", Tags: []string{"tools"}},
			{URL: "d"},
		},
	}

	// Single tag
	got := pf.FindByTag([]string{"llm"})
	if len(got) != 2 {
		t.Errorf("expected 2 matches for 'llm', got %d", len(got))
	}

	// AND logic: both tags required
	got = pf.FindByTag([]string{"llm", "tools"})
	if len(got) != 1 {
		t.Errorf("expected 1 match for 'llm'+'tools', got %d", len(got))
	}
	if got[0].URL != "a" {
		t.Errorf("expected URL 'a', got %q", got[0].URL)
	}

	// No matches
	got = pf.FindByTag([]string{"nonexistent"})
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}
