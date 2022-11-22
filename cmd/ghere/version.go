package main

import (
	"fmt"

	"github.com/informalsystems/ghere/pkg/ghere"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the application version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(ghere.VERSION)
		},
	}
}
