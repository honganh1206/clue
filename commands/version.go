package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Adrift",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Adrift version %s (commit: %s, built: %s)\n", Version, GitCommit, BuildTime)
	},
}
