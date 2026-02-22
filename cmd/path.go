package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/giturl"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var (
	pathTags []string
	pathJSON bool
)

var pathCmd = &cobra.Command{
	Use:   "path [id]",
	Short: "Print absolute paths to cached picks",
	Long: `Print absolute paths to cached picks.

If a positional argument is given (e.g. github.com/owner/repo@branch),
only that pick's path is printed. Use --tag to filter by tags.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPath,
}

func init() {
	pathCmd.Flags().StringSliceVarP(&pathTags, "tag", "t", nil, "filter by tag (repeatable)")
	pathCmd.Flags().BoolVar(&pathJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(pathCmd)
}

func runPath(cmd *cobra.Command, args []string) error {
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

	// Resolve which picks to show
	var picks []pickfile.Pick

	if len(args) > 0 {
		// Find single pick by cache ID
		targetID := args[0]
		for _, p := range pf.Picks {
			parsed, err := giturl.Parse(p.URL)
			if err != nil {
				continue
			}
			if parsed.CacheID(p.Branch) == targetID {
				picks = append(picks, p)
				break
			}
		}
		if len(picks) == 0 {
			return fmt.Errorf("no pick found matching %q", targetID)
		}
	} else if len(pathTags) > 0 {
		picks = pf.FindByTag(pathTags)
	} else {
		picks = pf.Picks
	}

	// Build paths, skipping uncached picks
	var paths []string
	for _, p := range picks {
		parsed, err := giturl.Parse(p.URL)
		if err != nil {
			continue
		}
		cacheID := parsed.CacheID(p.Branch)
		if idx.FindRepo(cacheID) == nil {
			continue
		}
		cachePath := parsed.CachePath(p.Branch)
		paths = append(paths, filepath.Join(cacheDir, cachePath))
	}

	if pathJSON {
		data, err := json.MarshalIndent(paths, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}

	for _, p := range paths {
		fmt.Fprintln(os.Stdout, p)
	}
	return nil
}
