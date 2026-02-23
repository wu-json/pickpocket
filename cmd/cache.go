package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/giturl"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var (
	cacheRemoveForce bool
	cacheCleanForce  bool
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the global pickpocket cache",
	Long: `Manage the global pickpocket cache.

Cloned repositories are stored in ~/.pickpocket/ and shared across all
projects. Use 'cache list' to see what's cached, 'cache remove' to
delete a single entry, or 'cache clean' to wipe everything.`,
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached repositories with disk usage",
	RunE:  runCacheList,
}

var cacheRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a cached repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runCacheRemove,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all cached repositories",
	RunE:  runCacheClean,
}

func init() {
	cacheRemoveCmd.Flags().BoolVarP(&cacheRemoveForce, "force", "f", false, "skip confirmation")
	cacheCleanCmd.Flags().BoolVarP(&cacheCleanForce, "force", "f", false, "skip confirmation")
	cacheCmd.AddCommand(cacheListCmd, cacheRemoveCmd, cacheCleanCmd)
	rootCmd.AddCommand(cacheCmd)
}

func runCacheList(cmd *cobra.Command, args []string) error {
	dim := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)

	cacheDir, err := cache.DefaultDir()
	if err != nil {
		return err
	}
	idx, err := cache.LoadIndex(cache.IndexPath(cacheDir))
	if err != nil {
		return fmt.Errorf("loading cache index: %w", err)
	}

	if len(idx.Repos) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("Cache is empty."))
		return nil
	}

	type row struct {
		id      string
		commit  string
		size    string
		updated string
	}

	var totalBytes int64
	rows := make([]row, len(idx.Repos))
	for i, repo := range idx.Repos {
		commit := repo.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}

		repoPath := filepath.Join(cacheDir, repo.Path)
		bytes, err := dirSizeBytes(repoPath)
		var sizeStr string
		if err != nil {
			sizeStr = "?"
		} else {
			totalBytes += bytes
			sizeStr = formatBytes(bytes)
		}

		rows[i] = row{
			id:      repo.ID,
			commit:  commit,
			size:    sizeStr,
			updated: relativeTime(repo.UpdatedAt),
		}
	}

	// Compute column widths
	headers := row{id: "ID", commit: "COMMIT", size: "SIZE", updated: "UPDATED"}
	widths := [4]int{len(headers.id), len(headers.commit), len(headers.size), len(headers.updated)}
	for _, r := range rows {
		if len(r.id) > widths[0] {
			widths[0] = len(r.id)
		}
		if len(r.commit) > widths[1] {
			widths[1] = len(r.commit)
		}
		if len(r.size) > widths[2] {
			widths[2] = len(r.size)
		}
		if len(r.updated) > widths[3] {
			widths[3] = len(r.updated)
		}
	}

	pad := func(s string, w int) string {
		if len(s) >= w {
			return s
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	// Header
	fmt.Fprintf(os.Stdout, "%s  %s  %s  %s\n",
		bold.Render(pad(headers.id, widths[0])),
		bold.Render(pad(headers.commit, widths[1])),
		bold.Render(pad(headers.size, widths[2])),
		bold.Render(headers.updated))

	// Data rows
	for _, r := range rows {
		fmt.Fprintf(os.Stdout, "%s  %s  %s  %s\n",
			bold.Render(pad(r.id, widths[0])),
			dim.Render(pad(r.commit, widths[1])),
			dim.Render(pad(r.size, widths[2])),
			dim.Render(r.updated))
	}

	// Summary
	fmt.Fprintf(os.Stdout, "\nTotal: %d repos, %s\n", len(idx.Repos), formatBytes(totalBytes))

	return nil
}

func runCacheRemove(cmd *cobra.Command, args []string) error {
	bold := lipgloss.NewStyle().Bold(true)

	cacheDir, err := cache.DefaultDir()
	if err != nil {
		return err
	}
	idxPath := cache.IndexPath(cacheDir)
	idx, err := cache.LoadIndex(idxPath)
	if err != nil {
		return fmt.Errorf("loading cache index: %w", err)
	}

	targetID := args[0]
	repo := idx.FindRepo(targetID)
	if repo == nil {
		msg := fmt.Sprintf("no cached repo found matching %q", targetID)
		if s := suggestID(targetID, collectCacheIDs(idx)); s != "" {
			msg += fmt.Sprintf("\n\n  did you mean %s?", s)
		}
		return errors.New(msg)
	}

	// Confirmation prompt
	if !cacheRemoveForce {
		fmt.Fprintf(os.Stderr, "Remove %s? (y/n) ", targetID)
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	// Delete the clone directory
	clonePath := filepath.Join(cacheDir, repo.Path)
	if err := os.RemoveAll(clonePath); err != nil {
		return fmt.Errorf("removing %s: %w", clonePath, err)
	}

	// Update index
	idx.RemoveRepo(targetID)
	if err := cache.Write(idxPath, idx); err != nil {
		return fmt.Errorf("writing cache index: %w", err)
	}

	// Best-effort warning if current project references this cache entry
	warnIfReferenced(targetID)

	fmt.Fprintln(os.Stderr, "Removed "+bold.Render(targetID))
	return nil
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	cacheDir, err := cache.DefaultDir()
	if err != nil {
		return err
	}
	idxPath := cache.IndexPath(cacheDir)
	idx, err := cache.LoadIndex(idxPath)
	if err != nil {
		return fmt.Errorf("loading cache index: %w", err)
	}

	repoCount := len(idx.Repos)
	if repoCount == 0 {
		fmt.Fprintln(os.Stderr, "Cache is already empty.")
		return nil
	}

	// Compute total size for prompt
	reposDir := filepath.Join(cacheDir, "repos")
	var totalBytes int64
	if bytes, err := dirSizeBytes(reposDir); err == nil {
		totalBytes = bytes
	}

	// Confirmation prompt
	if !cacheCleanForce {
		fmt.Fprintf(os.Stderr, "Remove all %d repos (%s)? (y/n) ", repoCount, formatBytes(totalBytes))
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	// Wipe repos directory
	if err := os.RemoveAll(reposDir); err != nil {
		return fmt.Errorf("removing repos directory: %w", err)
	}

	// Reset index
	idx.Repos = nil
	idx.Worktrees = nil
	if err := cache.Write(idxPath, idx); err != nil {
		return fmt.Errorf("writing cache index: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Removed %d repos\n", repoCount)
	return nil
}

// dirSizeBytes returns the total size in bytes of all files under path.
func dirSizeBytes(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// warnIfReferenced checks if any pick in the current project's Pickfile
// references the given cache ID, and prints a warning if so.
func warnIfReferenced(cacheID string) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	pfPath, err := pickfile.Discover(cwd)
	if err != nil {
		return
	}
	pf, err := pickfile.Load(pfPath)
	if err != nil {
		return
	}
	for _, p := range pf.Picks {
		parsed, err := giturl.Parse(p.URL)
		if err != nil {
			continue
		}
		if parsed.CacheID(p.Branch) == cacheID {
			dim := lipgloss.NewStyle().Faint(true)
			fmt.Fprintln(os.Stderr, dim.Render(fmt.Sprintf("Warning: %s is referenced in %s", cacheID, pickfile.Filename)))
			return
		}
	}
}
