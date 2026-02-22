package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Pickfile in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		path := filepath.Join(cwd, pickfile.Filename)

		dim := lipgloss.NewStyle().Faint(true)
		bold := lipgloss.NewStyle().Bold(true)

		if _, err := os.Stat(path); err == nil {
			fmt.Fprintln(os.Stderr, dim.Render("· "+pickfile.Filename+" already exists"))
			return nil
		}

		pf := &pickfile.Pickfile{Picks: []pickfile.Pick{}}
		if err := pickfile.Write(path, pf); err != nil {
			return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
		}

		fmt.Fprintln(os.Stderr, "✓ Created "+bold.Render(pickfile.Filename))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
