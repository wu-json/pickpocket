package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const indexFilename = "cache.json"

// Worktree represents an ephemeral writable worktree created from a cached clone.
type Worktree struct {
	RepoID    string    `json:"repo_id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

// Index is the top-level cache index structure.
type Index struct {
	Version   int          `json:"version"`
	Repos     []CachedRepo `json:"repos"`
	Worktrees []Worktree   `json:"worktrees,omitempty"`
}

// CachedRepo represents a cached git clone.
type CachedRepo struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	Commit    string    `json:"commit"`
	ClonedAt  time.Time `json:"cloned_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DefaultDir returns the default cache directory (~/.pickpocket/).
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".pickpocket"), nil
}

// IndexPath returns the path to the cache index file.
func IndexPath(cacheDir string) string {
	return filepath.Join(cacheDir, indexFilename)
}

// LoadIndex reads the cache index from the given path.
// If the file doesn't exist, returns an empty Index with Version=1.
func LoadIndex(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Index{Version: 1}, nil
		}
		return nil, err
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &idx, nil
}

// Write atomically writes the cache index to the given path, creating parent dirs.
func Write(path string, idx *Index) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".cache-*.tmp")
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

// SetRepo upserts a cached repo by ID.
func (idx *Index) SetRepo(repo CachedRepo) {
	for i, r := range idx.Repos {
		if r.ID == repo.ID {
			idx.Repos[i] = repo
			return
		}
	}
	idx.Repos = append(idx.Repos, repo)
}

// RemoveRepo removes a cached repo by ID. Returns true if found.
func (idx *Index) RemoveRepo(id string) bool {
	for i, r := range idx.Repos {
		if r.ID == id {
			idx.Repos = append(idx.Repos[:i], idx.Repos[i+1:]...)
			return true
		}
	}
	return false
}

// FindRepo returns a pointer to the repo with the given ID, or nil.
func (idx *Index) FindRepo(id string) *CachedRepo {
	for i := range idx.Repos {
		if idx.Repos[i].ID == id {
			return &idx.Repos[i]
		}
	}
	return nil
}

// AddWorktree appends a worktree entry to the index.
func (idx *Index) AddWorktree(wt Worktree) {
	idx.Worktrees = append(idx.Worktrees, wt)
}

// RemoveWorktree removes a worktree entry by path. Returns true if found.
func (idx *Index) RemoveWorktree(path string) bool {
	for i, wt := range idx.Worktrees {
		if wt.Path == path {
			idx.Worktrees = append(idx.Worktrees[:i], idx.Worktrees[i+1:]...)
			return true
		}
	}
	return false
}

// PruneWorktrees removes entries older than 24h or whose Path no longer exists on disk.
// For each removed entry, it calls git worktree remove on the cached clone via the
// repoDir function which maps a RepoID to the clone's absolute path.
func (idx *Index) PruneWorktrees(repoDir func(repoID string) string) []Worktree {
	cutoff := time.Now().Add(-24 * time.Hour)
	var keep []Worktree
	var removed []Worktree

	for _, wt := range idx.Worktrees {
		stale := wt.CreatedAt.Before(cutoff)
		_, statErr := os.Stat(wt.Path)
		missing := statErr != nil

		if stale || missing {
			if !missing {
				dir := repoDir(wt.RepoID)
				if dir != "" {
					exec.Command("git", "-C", dir, "worktree", "remove", "--force", wt.Path).Run()
				}
			}
			removed = append(removed, wt)
		} else {
			keep = append(keep, wt)
		}
	}

	idx.Worktrees = keep
	return removed
}
