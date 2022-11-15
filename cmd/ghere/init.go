package main

import (
	"github.com/informalsystems/ghere/pkg/ghere"
	"github.com/spf13/cobra"
)

func newInitCmd(root *rootCmd) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a local collection of GitHub repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			log := root.logger
			log.Info("Initializing local collection", "path", root.configFile)
			coll, err := ghere.LoadOrCreateLocalCollection(root.configFile)
			if err != nil {
				log.Error("Failed to load collection", "err", err)
				return err
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
