package lockfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const Filename = ".pickpocket.lock"

// Lockfile records the exact commit for each locked pick.
type Lockfile struct {
	Locked []LockedPick `json:"locked"`
}

// LockedPick represents a pinned repository at a specific commit.
type LockedPick struct {
	URL    string `json:"url"`
	Branch string `json:"branch"`
	Commit string `json:"commit"`
}

// Load reads and parses a Lockfile from the given path.
func Load(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lf Lockfile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &lf, nil
}

// Write atomically writes the Lockfile to the given path.
func Write(path string, lf *Lockfile) error {
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".pickpocket-lock-*.tmp")
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

// SetEntry upserts a locked pick by URL+branch.
func (lf *Lockfile) SetEntry(entry LockedPick) {
	for i, e := range lf.Locked {
		if e.URL == entry.URL && e.Branch == entry.Branch {
			lf.Locked[i] = entry
			return
		}
	}
	lf.Locked = append(lf.Locked, entry)
}

// RemoveEntry removes a locked pick by URL and branch. Returns true if found.
func (lf *Lockfile) RemoveEntry(url, branch string) bool {
	for i, e := range lf.Locked {
		if e.URL == url && e.Branch == branch {
			lf.Locked = append(lf.Locked[:i], lf.Locked[i+1:]...)
			return true
		}
	}
	return false
}

// FindEntry returns a pointer to the entry with the given URL and branch, or nil.
func (lf *Lockfile) FindEntry(url, branch string) *LockedPick {
	for i := range lf.Locked {
		if lf.Locked[i].URL == url && lf.Locked[i].Branch == branch {
			return &lf.Locked[i]
		}
	}
	return nil
}
