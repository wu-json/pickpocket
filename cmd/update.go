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

var updateTags []string

var updateCmd = &cobra.Command{
	Use:   "update [id...]",
	Short: "Fetch the latest commits for picks",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().StringSliceVarP(&updateTags, "tag", "t", nil, "filter picks by tag (repeatable)")
	rootCmd.AddCommand(updateCmd)
}

type updateResult struct {
	index     int
	oldCommit string
	newCommit string
	err       error
}

func runUpdate(cmd *cobra.Command, args []string) error {
	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)

	// 1. Load pickfile and cache index
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

	cacheDir, err := cache.DefaultDir()
	if err != nil {
		return err
	}
	idxPath := cache.IndexPath(cacheDir)
	idx, err := cache.LoadIndex(idxPath)
	if err != nil {
		return fmt.Errorf("loading cache index: %w", err)
	}

	// 2. Resolve which picks to update
	type indexedPick struct {
		index int
		pick  pickfile.Pick
	}
	var targets []indexedPick

	if len(args) > 0 {
		// By positional cache IDs
		for _, targetID := range args {
			found := false
			for i, p := range pf.Picks {
				parsed, err := giturl.Parse(p.URL)
				if err != nil {
					continue
				}
				if parsed.CacheID(p.Branch) == targetID {
					targets = append(targets, indexedPick{index: i, pick: p})
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("no pick found matching %q", targetID)
			}
		}
	} else if len(updateTags) > 0 {
		// By tag filter
		tagged := pf.FindByTag(updateTags)
		for _, tp := range tagged {
			for i, p := range pf.Picks {
				if p.URL == tp.URL && p.Branch == tp.Branch {
					targets = append(targets, indexedPick{index: i, pick: p})
					break
				}
			}
		}
	} else {
		// All picks
		for i, p := range pf.Picks {
			targets = append(targets, indexedPick{index: i, pick: p})
		}
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("No picks to update."))
		return nil
	}

	// 3. Filter to only cached picks
	var cachedTargets []indexedPick
	for _, t := range targets {
		parsed, err := giturl.Parse(t.pick.URL)
		if err != nil {
			continue
		}
		cacheID := parsed.CacheID(t.pick.Branch)
		if idx.FindRepo(cacheID) != nil {
			cachedTargets = append(cachedTargets, t)
		} else {
			repo := parsed.Owner + "/" + parsed.Repo
			fmt.Fprintln(os.Stderr, dim.Render("⚠ "+repo+" not cached, skipping (run pick install first)"))
		}
	}

	if len(cachedTargets) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("No cached picks to update."))
		return nil
	}

	// 4. Parallel fetch + reset
	stop := startSpinner(fmt.Sprintf("Updating %d picks", len(cachedTargets)))

	results := make([]updateResult, len(cachedTargets))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup

	for i, t := range cachedTargets {
		wg.Add(1)
		go func(ri int, t indexedPick) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			parsed, err := giturl.Parse(t.pick.URL)
			if err != nil {
				results[ri] = updateResult{index: t.index, err: fmt.Errorf("invalid URL: %w", err)}
				return
			}

			cachePath := parsed.CachePath(t.pick.Branch)
			destDir := filepath.Join(cacheDir, cachePath)

			oldCommit := t.pick.Commit

			if err := git.Fetch(destDir); err != nil {
				results[ri] = updateResult{index: t.index, err: fmt.Errorf("fetch: %w", err)}
				return
			}

			if err := git.ResetHard(destDir, "origin/"+t.pick.Branch); err != nil {
				results[ri] = updateResult{index: t.index, err: fmt.Errorf("reset: %w", err)}
				return
			}

			newCommit, err := git.HeadCommit(destDir)
			if err != nil {
				results[ri] = updateResult{index: t.index, err: fmt.Errorf("reading HEAD: %w", err)}
				return
			}

			results[ri] = updateResult{index: t.index, oldCommit: oldCommit, newCommit: newCommit}
		}(i, t)
	}

	wg.Wait()
	stop()

	// 5. Sequential mutations
	for _, r := range results {
		if r.err != nil {
			continue
		}

		pick := &pf.Picks[r.index]
		pick.Commit = r.newCommit

		parsed, err := giturl.Parse(pick.URL)
		if err != nil {
			continue
		}
		cacheID := parsed.CacheID(pick.Branch)
		entry := idx.FindRepo(cacheID)
		if entry != nil {
			entry.Commit = r.newCommit
			entry.UpdatedAt = time.Now()
		}
	}

	// 6. Write pickfile + cache index
	if err := pickfile.Write(pfPath, pf); err != nil {
		return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
	}
	if err := cache.Write(idxPath, idx); err != nil {
		return fmt.Errorf("writing cache index: %w", err)
	}

	// 7. Print per-pick summary
	var updated, unchanged, failed int
	for i, r := range results {
		pick := cachedTargets[i].pick
		parsed, _ := giturl.Parse(pick.URL)
		repo := parsed.Owner + "/" + parsed.Repo

		if r.err != nil {
			failed++
			fmt.Fprintln(os.Stderr, "✗ "+bold.Render(repo)+"  "+dim.Render(r.err.Error()))
			continue
		}

		oldShort := r.oldCommit
		if len(oldShort) > 7 {
			oldShort = oldShort[:7]
		}
		newShort := r.newCommit
		if len(newShort) > 7 {
			newShort = newShort[:7]
		}

		if r.oldCommit == r.newCommit {
			unchanged++
			fmt.Fprintln(os.Stderr, "· "+bold.Render(repo)+"  "+dim.Render("unchanged"))
		} else {
			updated++
			fmt.Fprintln(os.Stderr, "✓ "+bold.Render(repo)+"  "+dim.Render(oldShort+" → "+newShort))
		}
	}

	// Totals
	parts := []string{}
	if updated > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", updated))
	}
	if unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", unchanged))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	summary := fmt.Sprintf("\n%d picks", len(cachedTargets))
	for i, p := range parts {
		if i == 0 {
			summary += ": " + p
		} else {
			summary += ", " + p
		}
	}
	fmt.Fprintln(os.Stderr, dim.Render(summary))

	if failed > 0 {
		return fmt.Errorf("%d pick(s) failed to update", failed)
	}
	return nil
}
