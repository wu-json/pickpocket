package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/git"
	"github.com/wu-json/pickpocket/internal/giturl"
	"github.com/wu-json/pickpocket/internal/lockfile"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var (
	addBranch string
	addTags   []string
)

var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a git repository to the Pickfile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		styles := struct {
			success lipgloss.Style
			warn    lipgloss.Style
			info    lipgloss.Style
		}{
			success: lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			warn:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			info:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		}

		// 1. Parse URL
		parsed, err := giturl.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
		normalizedURL := parsed.NormalizedURL()

		// 2. Resolve branch
		branch := addBranch
		if branch == "" {
			fmt.Fprintln(os.Stderr, styles.info.Render("Detecting default branch..."))
			branch, err = git.DefaultBranch(normalizedURL)
			if err != nil {
				return fmt.Errorf("detecting default branch: %w", err)
			}
		}

		// 3. Discover or create Pickfile
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		pfPath, err := pickfile.Discover(cwd)
		if err != nil {
			// No Pickfile found — create one in cwd
			pfPath = filepath.Join(cwd, pickfile.Filename)
			pf := &pickfile.Pickfile{Picks: []pickfile.Pick{}}
			if err := pickfile.Write(pfPath, pf); err != nil {
				return fmt.Errorf("creating %s: %w", pickfile.Filename, err)
			}
			fmt.Fprintln(os.Stderr, styles.info.Render("Created "+pickfile.Filename))
		}

		// 4. Load Pickfile and add pick
		pf, err := pickfile.Load(pfPath)
		if err != nil {
			return fmt.Errorf("loading %s: %w", pickfile.Filename, err)
		}

		pick := pickfile.Pick{
			URL:    normalizedURL,
			Branch: branch,
			Tags:   addTags,
		}
		if err := pf.AddPick(pick); err != nil {
			return fmt.Errorf("duplicate: %s@%s already in Pickfile", normalizedURL, branch)
		}

		// 5. Check cache
		cacheDir, err := cache.DefaultDir()
		if err != nil {
			return err
		}
		idxPath := cache.IndexPath(cacheDir)
		idx, err := cache.LoadIndex(idxPath)
		if err != nil {
			return fmt.Errorf("loading cache index: %w", err)
		}

		cacheID := parsed.CacheID(branch)
		cachePath := parsed.CachePath(branch)
		destDir := filepath.Join(cacheDir, cachePath)

		var commitSHA string

		if existing := idx.FindRepo(cacheID); existing != nil {
			// Already cached
			commitSHA = existing.Commit
			fmt.Fprintln(os.Stderr, styles.warn.Render("Already cached: "+cacheID))
		} else {
			// 6. Clone with spinner
			stop := startSpinner("Cloning "+normalizedURL+"@"+branch)
			commitSHA, err = git.Clone(normalizedURL, branch, destDir)
			stop()
			if err != nil {
				return fmt.Errorf("cloning: %w", err)
			}

			// Update cache index
			now := time.Now()
			idx.SetRepo(cache.CachedRepo{
				ID:        cacheID,
				URL:       normalizedURL,
				Path:      cachePath,
				Branch:    branch,
				Commit:    commitSHA,
				ClonedAt:  now,
				UpdatedAt: now,
			})
			if err := cache.Write(idxPath, idx); err != nil {
				return fmt.Errorf("writing cache index: %w", err)
			}
		}

		// 7. Write Pickfile (only after successful clone)
		if err := pickfile.Write(pfPath, pf); err != nil {
			return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
		}

		// 8. Load or create lockfile, update entry
		lfDir := filepath.Dir(pfPath)
		lfPath := filepath.Join(lfDir, lockfile.Filename)
		lf, err := lockfile.Load(lfPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				lf = &lockfile.Lockfile{}
			} else {
				return fmt.Errorf("loading lockfile: %w", err)
			}
		}

		lf.SetEntry(lockfile.LockedPick{
			URL:    normalizedURL,
			Branch: branch,
			Commit: commitSHA,
		})
		if err := lockfile.Write(lfPath, lf); err != nil {
			return fmt.Errorf("writing lockfile: %w", err)
		}

		fmt.Fprintln(os.Stderr, styles.success.Render(fmt.Sprintf("Added %s@%s", normalizedURL, branch)))
		return nil
	},
}

// startSpinner runs a braille spinner on stderr until the returned stop func is called.
func startSpinner(msg string) func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	var done atomic.Bool
	go func() {
		i := 0
		for !done.Load() {
			fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], msg)
			i++
			time.Sleep(80 * time.Millisecond)
		}
		fmt.Fprintf(os.Stderr, "\r\033[2K") // clear line
	}()
	return func() { done.Store(true); time.Sleep(100 * time.Millisecond) }
}

func init() {
	addCmd.Flags().StringVarP(&addBranch, "branch", "b", "", "branch to track")
	addCmd.Flags().StringSliceVarP(&addTags, "tag", "t", nil, "tags for this pick (repeatable)")
	rootCmd.AddCommand(addCmd)
}
