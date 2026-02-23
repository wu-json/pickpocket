package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var slashCmd = &cobra.Command{
	Use:   "slash",
	Short: "Install the pick slash command for a coding agent",
	Long: `Install the pick slash command for a coding agent.

Use 'slash claude' to install for Claude Code or 'slash codex' to install
for Codex CLI.`,
}

var slashClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Install /pick slash command for Claude Code",
	RunE:  runSlashClaude,
}

var slashCodexCmd = &cobra.Command{
	Use:   "codex",
	Short: "Install /pick slash command for Codex CLI",
	RunE:  runSlashCodex,
}

func init() {
	slashCmd.AddCommand(slashClaudeCmd, slashCodexCmd)
	rootCmd.AddCommand(slashCmd)
}

func runSlashClaude(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}
	dir := filepath.Join(home, ".claude", "commands")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	path := filepath.Join(dir, "pick.md")
	if err := os.WriteFile(path, []byte(systemPrompt), 0o644); err != nil {
		return fmt.Errorf("writing slash command: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Installed /pick slash command for Claude Code")
	return nil
}

func runSlashCodex(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}
	dir := filepath.Join(home, ".codex", "prompts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	path := filepath.Join(dir, "pick.md")
	if err := os.WriteFile(path, []byte(systemPrompt), 0o644); err != nil {
		return fmt.Errorf("writing slash command: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Installed /pick slash command for Codex CLI")
	return nil
}
