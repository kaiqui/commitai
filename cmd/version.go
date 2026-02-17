package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"
var Commit = "unknown"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print commitai version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("commitai %s (commit: %s)\n", Version, Commit)
	},
}
