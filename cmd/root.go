package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of pickpocket",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("pickpocket version " + Version)
	},
}

var rootCmd = &cobra.Command{
	Use:   "pick [url]",
	Short: "Manage vendored git clones as LLM context",
	Long: `pickpocket is a CLI tool for managing vendored git clones as LLM context.

It lets you declare git repositories in a pickpocket.json file, clone them into
a local .picks/ directory, and keep them in sync — giving your LLM tools
fast, local access to external codebases as context.

Usage:
  pick <url>           Add a git repository to the Pickfile
  pick [command]       Run a subcommand`,
	Args:          cobra.MaximumNArgs(1),
	RunE:          runAdd,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Flags().StringVarP(&addBranch, "branch", "b", "", "branch to track")
	rootCmd.Flags().StringSliceVarP(&addTags, "tag", "t", nil, "tags for this pick (repeatable)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
