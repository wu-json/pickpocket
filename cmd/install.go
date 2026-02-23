package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/git"
	"github.com/wu-json/pickpocket/internal/giturl"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

type installResult struct {
	index  int
	status string // "cloned", "updated", "cached", "failed"
	commit string
	branch string
	err    error
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Clone or update all picks declared in the Pickfile",
	Long: `Clone or update all picks declared in the Pickfile.

Picks are cloned in parallel (up to 4 at a time) into the global cache
at ~/.pickpocket/. If a pick is already cached at the pinned commit, it
is skipped. If the pick has a commit pin that differs from the cache,
a fetch + checkout is performed.`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)

	// Phase A — Load
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	pfPath, err := pickfile.Discover(cwd)
	if err != nil {
		return fmt.Errorf("no %s found: %w", pickfile.Filename, err)
	}

	pf, err := pickfile.Load(pfPath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", pickfile.Filename, err)
	}

	if len(pf.Picks) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("No picks to install."))
		return nil
	}

	cacheDir, err := cache.DefaultDir()
	if err != nil {
		return err
	}
	idxPath := cache.IndexPath(cacheDir)
	idx, err := cache.LoadIndex(idxPath)
	if err != nil {
		return fmt.Errorf("loading cache index: %w", err)
	}

	// Phase B — Parallel clone/fetch
	stop := startSpinner(fmt.Sprintf("Installing %d picks", len(pf.Picks)))

	results := make([]installResult, len(pf.Picks))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup

	for i, pick := range pf.Picks {
		wg.Add(1)
		go func(i int, pick pickfile.Pick) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = processOnePick(pick, i, idx, cacheDir)
		}(i, pick)
	}

	wg.Wait()
	stop()

	// Phase C — Sequential mutations
	for _, r := range results {
		if r.err != nil {
			continue
		}
		pick := &pf.Picks[r.index]

		if r.branch != "" {
			pick.Branch = r.branch
		}
		if r.commit != "" {
			pick.Commit = r.commit
		}

		parsed, err := giturl.Parse(pick.URL)
		if err != nil {
			continue
		}
		cacheID := parsed.CacheID(pick.Branch)
		cachePath := parsed.CachePath(pick.Branch)

		now := time.Now()
		existing := idx.FindRepo(cacheID)
		if existing != nil {
			existing.Commit = pick.Commit
			existing.UpdatedAt = now
		} else {
			idx.SetRepo(cache.CachedRepo{
				ID:        cacheID,
				URL:       pick.URL,
				Path:      cachePath,
				Branch:    pick.Branch,
				Commit:    pick.Commit,
				ClonedAt:  now,
				UpdatedAt: now,
			})
		}
	}

	if err := pickfile.Write(pfPath, pf); err != nil {
		return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
	}
	if err := cache.Write(idxPath, idx); err != nil {
		return fmt.Errorf("writing cache index: %w", err)
	}

	// Phase D — Summary
	var cloned, updated, cached, failed int
	for _, r := range results {
		pick := pf.Picks[r.index]
		parsed, _ := giturl.Parse(pick.URL)
		repo := parsed.Owner + "/" + parsed.Repo
		short := r.commit
		if len(short) > 7 {
			short = short[:7]
		}

		switch r.status {
		case "cloned":
			cloned++
			fmt.Fprintln(os.Stderr, "✓ "+bold.Render(repo)+" "+dim.Render("cloned at "+short))
		case "updated":
			updated++
			fmt.Fprintln(os.Stderr, "✓ "+bold.Render(repo)+" "+dim.Render("updated to "+short))
		case "cached":
			cached++
			fmt.Fprintln(os.Stderr, "· "+bold.Render(repo)+" "+dim.Render("cached at "+short))
		case "failed":
			failed++
			fmt.Fprintln(os.Stderr, "✗ "+bold.Render(repo)+" "+dim.Render(r.err.Error()))
		}
	}

	parts := []string{}
	if cloned > 0 {
		parts = append(parts, fmt.Sprintf("%d cloned", cloned))
	}
	if updated > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", updated))
	}
	if cached > 0 {
		parts = append(parts, fmt.Sprintf("%d cached", cached))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	summary := fmt.Sprintf("\n%d picks", len(pf.Picks))
	for i, p := range parts {
		if i == 0 {
			summary += ": " + p
		} else {
			summary += ", " + p
		}
	}
	fmt.Fprintln(os.Stderr, dim.Render(summary))

	if failed > 0 {
		return fmt.Errorf("%d pick(s) failed to install", failed)
	}
	return nil
}

func processOnePick(pick pickfile.Pick, index int, idx *cache.Index, cacheDir string) installResult {
	parsed, err := giturl.Parse(pick.URL)
	if err != nil {
		return installResult{index: index, status: "failed", err: fmt.Errorf("invalid URL: %w", err)}
	}

	branch := pick.Branch
	if branch == "" {
		branch, err = git.DefaultBranch(pick.URL)
		if err != nil {
			return installResult{index: index, status: "failed", err: fmt.Errorf("detecting default branch: %w", err)}
		}
	}

	cacheID := parsed.CacheID(branch)
	cachePath := parsed.CachePath(branch)
	destDir := filepath.Join(cacheDir, cachePath)

	existing := idx.FindRepo(cacheID)

	if existing != nil {
		// Repo is cached
		if pick.Commit != "" && pick.Commit == existing.Commit {
			// Pinned commit matches cache — skip
			return installResult{index: index, status: "cached", commit: pick.Commit, branch: branch}
		}

		if pick.Commit != "" {
			// Pinned commit differs from cache — fetch + checkout
			if err := git.Fetch(destDir); err != nil {
				return installResult{index: index, status: "failed", err: fmt.Errorf("fetching: %w", err)}
			}
			if err := git.Checkout(destDir, pick.Commit); err != nil {
				return installResult{index: index, status: "failed", err: fmt.Errorf("checkout: %w", err)}
			}
			return installResult{index: index, status: "updated", commit: pick.Commit, branch: branch}
		}

		// No pinned commit — use cache HEAD
		return installResult{index: index, status: "cached", commit: existing.Commit, branch: branch}
	}

	// Not cached — clone
	commitSHA, err := git.Clone(pick.URL, branch, destDir)
	if err != nil {
		return installResult{index: index, status: "failed", err: fmt.Errorf("cloning: %w", err)}
	}

	if pick.Commit != "" && pick.Commit != commitSHA {
		// Pinned to a specific commit that isn't branch tip — checkout
		if err := git.Checkout(destDir, pick.Commit); err != nil {
			return installResult{index: index, status: "failed", err: fmt.Errorf("checkout after clone: %w", err)}
		}
		commitSHA = pick.Commit
	}

	return installResult{index: index, status: "cloned", commit: commitSHA, branch: branch}
}
