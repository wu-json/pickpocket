package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/giturl"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var (
	listTags []string
	listJSON bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all picks in the Pickfile",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringSliceVarP(&listTags, "tag", "t", nil, "filter by tag (repeatable)")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	dim := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)

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

	if len(pf.Picks) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("No picks declared."))
		return nil
	}

	picks := pf.Picks
	if len(listTags) > 0 {
		picks = pf.FindByTag(listTags)
	}

	if len(picks) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("No picks match the given tags."))
		return nil
	}

	if listJSON {
		data, err := json.MarshalIndent(picks, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}

	// Load cache index for updated timestamps
	cacheDir, err := cache.DefaultDir()
	if err != nil {
		return err
	}
	idx, err := cache.LoadIndex(cache.IndexPath(cacheDir))
	if err != nil {
		return fmt.Errorf("loading cache index: %w", err)
	}

	// Build table rows
	type row struct {
		id      string
		commit  string
		tags    string
		updated string
	}

	rows := make([]row, len(picks))
	for i, p := range picks {
		parsed, err := giturl.Parse(p.URL)
		if err != nil {
			rows[i] = row{id: p.URL}
			continue
		}
		commit := p.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		cacheID := parsed.CacheID(p.Branch)
		var updated string
		if entry := idx.FindRepo(cacheID); entry != nil {
			updated = relativeTime(entry.UpdatedAt)
		}
		rows[i] = row{
			id:      cacheID,
			commit:  commit,
			tags:    strings.Join(p.Tags, ", "),
			updated: updated,
		}
	}

	// Compute column widths
	headers := row{id: "ID", commit: "COMMIT", tags: "TAGS", updated: "UPDATED"}
	widths := [4]int{len(headers.id), len(headers.commit), len(headers.tags), len(headers.updated)}
	for _, r := range rows {
		if len(r.id) > widths[0] {
			widths[0] = len(r.id)
		}
		if len(r.commit) > widths[1] {
			widths[1] = len(r.commit)
		}
		if len(r.tags) > widths[2] {
			widths[2] = len(r.tags)
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
		bold.Render(pad(headers.tags, widths[2])),
		bold.Render(headers.updated))

	// Data rows
	for _, r := range rows {
		fmt.Fprintf(os.Stdout, "%s  %s  %s  %s\n",
			bold.Render(pad(r.id, widths[0])),
			dim.Render(pad(r.commit, widths[1])),
			dim.Render(pad(r.tags, widths[2])),
			dim.Render(r.updated))
	}

	return nil
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
