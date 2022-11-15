package main

import (
	"github.com/informalsystems/ghere/pkg/ghere"
	"github.com/spf13/cobra"
)

func newAddCmd(root *rootCmd) *cobra.Command {
	return &cobra.Command{
		Use:   "add path [path ...]",
		Short: "Add one or more repositories to a local collection",
		Example: `  # Add the repository https://github.com/myorg/repo1 to a local collection
  ghere add myorg/repo1`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			log := root.logger
			log.Info("Loading local collection", "path", root.configFile)
			coll, err := ghere.LoadOrCreateLocalCollection(root.configFile)
			if err != nil {
				log.Error("Failed to load collection", "err", err)
				return err
			}
			for _, arg := range args {
				_, err := coll.NewFromPath(arg)
				if err != nil {
					log.Error("Failed to create repository", "err", err)
					return err
				}
			}
			if err = coll.Save(); err != nil {
				log.Error("Failed to save local collection", "err", err)
				return err
			}
			log.Info("Success")
			return nil
		},
	}
}
