package main

import (
	"github.com/informalsystems/ghere/pkg/ghere"
	"github.com/spf13/cobra"
)

type rootCmd struct {
	*cobra.Command

	configFile string
	verbose    bool
	add        *addCmd
	fetch      *fetchCmd

	logger ghere.Logger
}

func newRootCmd() *rootCmd {
	r := &rootCmd{
		Command: &cobra.Command{
			Use:   "ghere",
			Short: "GitHub repos over _here_ on my local machine.",
			Long: `GitHub repos, with issues, pull requests and comments, over _here_, on my local
machine. Pronounced "gear".`,
			Example: `  # Initialize an empty local collection of GitHub repositories in the current
  # working directory.
  ghere init

  # Add https://github.com/org/repo to the local collection's index. Does not
  # download the remote repository's data yet.
  ghere add org/repo

  # Fetch this collection's local copies of repositories from the remote ones on
  # GitHub.
  ghere fetch`,
			SilenceErrors: true,
		},
	}
	r.PersistentFlags().StringVarP(&r.configFile, "config-file", "c", ghere.CONFIG_FILE_NAME, "path to local GitHub repository collection configuration file")
	r.PersistentFlags().BoolVarP(&r.verbose, "verbose", "v", false, "increase output logging verbosity to debug level")

	r.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		level := ghere.Info
		if r.verbose {
			level = ghere.Debug
		}
		r.logger = ghere.NewZerologLogger(level)
	}

	r.AddCommand(newInitCmd(r))

	r.add = newAddCmd(r)
	r.AddCommand(r.add.Command)

	r.fetch = newFetchCmd(r)
	r.AddCommand(r.fetch.Command)

	r.AddCommand(newVersionCmd())
	return r
}
