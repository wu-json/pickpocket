package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var infoCmd = &cobra.Command{
	Use:   "info <id>",
	Short: "Show detailed information about a pick",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	pfPath, err := pickfile.Discover(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, dim.Render("No pickpocket.json found. Run `pick init` to get started."))
		return nil
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

	// Find the pick by cache ID
	targetID := args[0]
	pick, parsed, err := findPickByID(pf, targetID)
	if err != nil {
		return err
	}

	cacheID := parsed.CacheID(pick.Branch)
	cachePath := parsed.CachePath(pick.Branch)
	absPath := filepath.Join(cacheDir, cachePath)
	entry := idx.FindRepo(cacheID)

	// Header
	fmt.Fprintln(os.Stderr, bold.Render(parsed.Owner+"/"+parsed.Repo))
	fmt.Fprintln(os.Stderr)

	// Fields
	printField := func(label, value string) {
		fmt.Fprintf(os.Stderr, "  %s  %s\n", dim.Render(fmt.Sprintf("%10s", label)), value)
	}

	printField("url", pick.URL)
	printField("branch", pick.Branch)
	printField("commit", pick.Commit)
	if len(pick.Tags) > 0 {
		printField("tags", strings.Join(pick.Tags, ", "))
	}

	fmt.Fprintln(os.Stderr)

	if entry != nil {
		// Replace home dir with ~ for display
		displayPath := absPath
		if home, err := os.UserHomeDir(); err == nil {
			displayPath = strings.Replace(absPath, home, "~", 1)
		}
		printField("path", displayPath)

		size, err := dirSize(absPath)
		if err == nil {
			printField("size", size)
		}

		printField("cloned", entry.ClonedAt.Format("2006-01-02 15:04:05"))
		printField("updated", entry.UpdatedAt.Format("2006-01-02 15:04:05"))
	} else {
		printField("path", dim.Render("not cached"))
	}

	return nil
}

func dirSize(path string) (string, error) {
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
	if err != nil {
		return "", err
	}
	return formatBytes(total), nil
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
