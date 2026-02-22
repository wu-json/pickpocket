package pickfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const Filename = ".pickpocket"

// Pickfile represents the contents of a .pickpocket file.
type Pickfile struct {
	Picks []Pick `json:"picks"`
}

// Pick represents a single repository entry.
type Pick struct {
	URL    string   `json:"url"`
	Branch string   `json:"branch,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// Discover walks up from startDir to find the nearest .pickpocket file.
// Returns the full path to the file, or an error if not found.
func Discover(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, Filename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found in %s or any parent directory", Filename, startDir)
		}
		dir = parent
	}
}

// Load reads and parses a Pickfile from the given path.
func Load(path string) (*Pickfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pf Pickfile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &pf, nil
}

// Write atomically writes the Pickfile to the given path.
func Write(path string, pf *Pickfile) error {
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".pickpocket-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}

// AddPick adds a pick to the Pickfile. Returns an error if a pick with
// the same URL and branch already exists.
func (pf *Pickfile) AddPick(p Pick) error {
	if pf.FindPick(p.URL, p.Branch) != nil {
		return fmt.Errorf("pick already exists: %s (branch %q)", p.URL, p.Branch)
	}
	pf.Picks = append(pf.Picks, p)
	return nil
}

// RemovePick removes a pick by URL and branch. Returns true if found and removed.
func (pf *Pickfile) RemovePick(url, branch string) bool {
	for i, p := range pf.Picks {
		if p.URL == url && p.Branch == branch {
			pf.Picks = append(pf.Picks[:i], pf.Picks[i+1:]...)
			return true
		}
	}
	return false
}

// FindPick returns a pointer to the pick with the given URL and branch, or nil.
func (pf *Pickfile) FindPick(url, branch string) *Pick {
	for i := range pf.Picks {
		if pf.Picks[i].URL == url && pf.Picks[i].Branch == branch {
			return &pf.Picks[i]
		}
	}
	return nil
}

// FindByTag returns all picks that have ALL of the given tags (AND logic).
func (pf *Pickfile) FindByTag(tags []string) []Pick {
	var result []Pick
	for _, p := range pf.Picks {
		if hasAllTags(p.Tags, tags) {
			result = append(result, p)
		}
	}
	return result
}

func hasAllTags(pickTags, required []string) bool {
	set := make(map[string]struct{}, len(pickTags))
	for _, t := range pickTags {
		set[t] = struct{}{}
	}
	for _, t := range required {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}
