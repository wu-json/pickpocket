package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const systemPrompt = `# Pickpocket — Vendored Clone Manager for LLM Context

This project uses **pickpocket** to manage vendored git clones as LLM context. Repositories are declared in a ` + "`pickpocket.json`" + ` file and cloned into a local ` + "`.picks/`" + ` directory, giving you fast, local access to external codebases.

## Discover available picks

Run ` + "`pick list --json`" + ` to see all declared picks with their URL, branch, commit, and tags.

## Get filesystem paths

- ` + "`pick path`" + ` — print absolute paths to all vendored repos, one per line
- ` + "`pick path <id>`" + ` — print the absolute path to a specific repo
- ` + "`pick path --tag <tag>`" + ` — print paths filtered by tag

Use these paths to read files and grep through vendored code directly on the filesystem.

## Explore vendored code

The paths returned by ` + "`pick path`" + ` point to full working-tree clones. You can read any file, search with grep, or browse directory listings at those paths — just like any local code.

## Add new repos

` + "`" + `pick <url>` + "`" + ` adds a repo to the Pickfile and clones it. Optional flags:
- ` + "`--tag <tag>`" + ` (repeatable) — attach tags for filtering
- ` + "`--branch <branch>`" + ` — track a specific branch

## Other useful commands

- ` + "`pick info <id>`" + ` — detailed info about a specific pick
- ` + "`pick open <id>`" + ` — create an ephemeral writable worktree for builds or experiments
- ` + "`pick update`" + ` — fetch latest commits for all picks
`

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Output a system prompt that teaches a coding agent how to use pickpocket",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprint(os.Stdout, systemPrompt)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
}
