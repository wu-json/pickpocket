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

		if _, err := os.Stat(path); err == nil {
			warn := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
			fmt.Fprintln(os.Stderr, warn.Render("Already initialized: "+pickfile.Filename))
			return nil
		}

		pf := &pickfile.Pickfile{Picks: []pickfile.Pick{}}
		if err := pickfile.Write(path, pf); err != nil {
			return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
		}

		success := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		fmt.Fprintln(os.Stderr, success.Render("Created "+pickfile.Filename))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
