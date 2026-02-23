package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/giturl"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var tagCmd = &cobra.Command{
	Use:   "tag [id] [tags...]",
	Short: "Manage tags on picks",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runTagAdd,
}

var tagAddCmd = &cobra.Command{
	Use:   "add <id> <tags...>",
	Short: "Add tags to a pick",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runTagAdd,
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove <id> <tags...>",
	Short: "Remove tags from a pick",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runTagRemove,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags with pick counts",
	Args:  cobra.NoArgs,
	RunE:  runTagList,
}

func init() {
	tagCmd.AddCommand(tagAddCmd, tagRemoveCmd, tagListCmd)
	rootCmd.AddCommand(tagCmd)
}

func runTagAdd(cmd *cobra.Command, args []string) error {
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

	targetID := args[0]
	tags := args[1:]

	pick, parsed, err := findPickByID(pf, targetID)
	if err != nil {
		return err
	}

	// Append new tags, skipping duplicates
	existing := make(map[string]struct{}, len(pick.Tags))
	for _, t := range pick.Tags {
		existing[t] = struct{}{}
	}
	for _, t := range tags {
		if _, ok := existing[t]; !ok {
			pick.Tags = append(pick.Tags, t)
			existing[t] = struct{}{}
		}
	}

	if err := pickfile.Write(pfPath, pf); err != nil {
		return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
	}

	// Print confirmation
	var plusTags []string
	for _, t := range tags {
		plusTags = append(plusTags, "+"+t)
	}
	fmt.Fprintf(os.Stderr, "✓ %s/%s  %s\n", parsed.Owner, parsed.Repo, strings.Join(plusTags, " "))

	return nil
}

func runTagRemove(cmd *cobra.Command, args []string) error {
	dim := lipgloss.NewStyle().Faint(true)

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

	targetID := args[0]
	tags := args[1:]

	pick, parsed, err := findPickByID(pf, targetID)
	if err != nil {
		return err
	}

	// Build set of tags to remove
	toRemove := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		toRemove[t] = struct{}{}
	}

	// Check for tags not present and warn
	existing := make(map[string]struct{}, len(pick.Tags))
	for _, t := range pick.Tags {
		existing[t] = struct{}{}
	}
	for _, t := range tags {
		if _, ok := existing[t]; !ok {
			fmt.Fprintf(os.Stderr, "%s\n", dim.Render(fmt.Sprintf("tag %q not present on %s/%s", t, parsed.Owner, parsed.Repo)))
		}
	}

	// Filter out removed tags
	filtered := pick.Tags[:0]
	for _, t := range pick.Tags {
		if _, ok := toRemove[t]; !ok {
			filtered = append(filtered, t)
		}
	}
	pick.Tags = filtered

	if err := pickfile.Write(pfPath, pf); err != nil {
		return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
	}

	// Print confirmation
	var minusTags []string
	for _, t := range tags {
		minusTags = append(minusTags, "-"+t)
	}
	fmt.Fprintf(os.Stderr, "✓ %s/%s  %s\n", parsed.Owner, parsed.Repo, strings.Join(minusTags, " "))

	return nil
}

func runTagList(cmd *cobra.Command, args []string) error {
	dim := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)

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

	// Count tag occurrences
	counts := make(map[string]int)
	for _, p := range pf.Picks {
		for _, t := range p.Tags {
			counts[t]++
		}
	}

	if len(counts) == 0 {
		fmt.Fprintln(os.Stderr, dim.Render("No tags in use."))
		return nil
	}

	// Sort tags alphabetically
	tags := make([]string, 0, len(counts))
	for t := range counts {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	// Compute column widths
	tagWidth := len("TAG")
	for _, t := range tags {
		if len(t) > tagWidth {
			tagWidth = len(t)
		}
	}

	pad := func(s string, w int) string {
		if len(s) >= w {
			return s
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	// Header
	fmt.Fprintf(os.Stdout, "%s  %s\n",
		bold.Render(pad("TAG", tagWidth)),
		bold.Render("PICKS"))

	// Data rows
	for _, t := range tags {
		fmt.Fprintf(os.Stdout, "%s  %s\n",
			bold.Render(pad(t, tagWidth)),
			dim.Render(fmt.Sprintf("%d", counts[t])))
	}

	return nil
}

// findPickByID resolves a pick by its cache ID.
func findPickByID(pf *pickfile.Pickfile, id string) (*pickfile.Pick, giturl.ParsedURL, error) {
	for i, p := range pf.Picks {
		pu, err := giturl.Parse(p.URL)
		if err != nil {
			continue
		}
		if pu.CacheID(p.Branch) == id {
			return &pf.Picks[i], pu, nil
		}
	}
	return nil, giturl.ParsedURL{}, fmt.Errorf("no pick found matching %q", id)
}
