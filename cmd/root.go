package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pick",
	Short: "Manage vendored git clones as LLM context",
	Long: `pickpocket is a CLI tool for managing vendored git clones as LLM context.

It lets you declare git repositories in a .pickpocket file, clone them into
a local .picks/ directory, and keep them in sync — giving your LLM tools
fast, local access to external codebases as context.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
