package cmd

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/git"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var openClean bool

var openCmd = &cobra.Command{
	Use:   "open <id>",
	Short: "Create an ephemeral writable worktree from a cached pick",
	Long: `Create a git worktree in /tmp/pickpocket/ from a cached clone.

The worktree path is printed to stdout, suitable for use with $().
Stale worktrees (>24h or missing) are pruned automatically.

Use --clean to remove all tracked worktrees.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOpen,
}

func init() {
	openCmd.Flags().BoolVar(&openClean, "clean", false, "remove all tracked worktrees")
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	cacheDir, err := cache.DefaultDir()
	if err != nil {
		return err
	}
	idxPath := cache.IndexPath(cacheDir)
	idx, err := cache.LoadIndex(idxPath)
	if err != nil {
		return fmt.Errorf("loading cache index: %w", err)
	}

	repoDirFn := func(repoID string) string {
		repo := idx.FindRepo(repoID)
		if repo == nil {
			return ""
		}
		return filepath.Join(cacheDir, repo.Path)
	}

	if openClean {
		return runOpenClean(idx, idxPath, repoDirFn)
	}

	if len(args) == 0 {
		return fmt.Errorf("requires a pick ID argument (or use --clean)")
	}

	// Prune stale worktrees as side effect
	idx.PruneWorktrees(repoDirFn)

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

	// Find pick by cache ID
	targetID := args[0]
	pick, parsed, err := findPickByID(pf, targetID)
	if err != nil {
		return err
	}

	cacheID := parsed.CacheID(pick.Branch)
	entry := idx.FindRepo(cacheID)
	if entry == nil {
		return fmt.Errorf("pick %q is not cached — run 'pick install' first", targetID)
	}

	// Build worktree path
	shortCommit := entry.Commit
	if len(shortCommit) > 7 {
		shortCommit = shortCommit[:7]
	}
	wtDir := fmt.Sprintf("%s-%s-%s-%s", parsed.Repo, pick.Branch, shortCommit, randomSuffix(4))
	wtPath := filepath.Join("/tmp", "pickpocket", wtDir)

	// Create worktree
	repoDir := filepath.Join(cacheDir, entry.Path)
	if err := git.WorktreeAdd(repoDir, wtPath, entry.Commit); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	// Track in index
	idx.AddWorktree(cache.Worktree{
		RepoID:    cacheID,
		Path:      wtPath,
		CreatedAt: time.Now(),
	})
	if err := cache.Write(idxPath, idx); err != nil {
		return fmt.Errorf("saving cache index: %w", err)
	}

	// Print path to stdout
	fmt.Fprintln(os.Stdout, wtPath)
	return nil
}

func randomSuffix(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func runOpenClean(idx *cache.Index, idxPath string, repoDirFn func(string) string) error {
	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)

	// Remove all worktrees, regardless of age
	var removed []cache.Worktree
	for _, wt := range idx.Worktrees {
		dir := repoDirFn(wt.RepoID)
		if dir != "" {
			git.WorktreeRemove(dir, wt.Path)
		}
		// Also remove the directory if it still exists (belt and suspenders)
		os.RemoveAll(wt.Path)
		removed = append(removed, wt)
	}
	idx.Worktrees = nil

	if err := cache.Write(idxPath, idx); err != nil {
		return fmt.Errorf("saving cache index: %w", err)
	}

	if len(removed) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("no worktrees to clean"))
	} else {
		fmt.Fprintf(os.Stderr, "%s %d worktree(s)\n", bold.Render("Removed"), len(removed))
		for _, wt := range removed {
			fmt.Fprintf(os.Stderr, "  %s\n", dim.Render(wt.Path))
		}
	}

	return nil
}
