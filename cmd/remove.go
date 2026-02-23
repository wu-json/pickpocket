package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a pick from the Pickfile",
	Long: `Remove a pick from the Pickfile.

This only removes the entry from pickpocket.json — the cached clone in
~/.pickpocket/ is left intact. Use 'pick cache remove' to delete a
cached clone from disk.`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	bold := lipgloss.NewStyle().Bold(true)

	// 1. Load pickfile
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

	// 2. Find pick by cache ID
	targetID := args[0]
	pick, parsed, err := findPickByID(pf, targetID)
	if err != nil {
		return err
	}

	repo := parsed.Owner + "/" + parsed.Repo

	// 3. Confirmation prompt (unless --force)
	if !removeForce {
		fmt.Fprintf(os.Stderr, "Remove %s@%s? (y/n) ", repo, pick.Branch)
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	// 4. Remove from pickfile
	pf.RemovePick(pick.URL, pick.Branch)

	// 5. Write pickfile
	if err := pickfile.Write(pfPath, pf); err != nil {
		return fmt.Errorf("writing %s: %w", pickfile.Filename, err)
	}

	// 6. Print confirmation
	fmt.Fprintln(os.Stderr, "Removed "+bold.Render(repo))

	return nil
}
