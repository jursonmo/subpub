package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "nel proxy version",
	Long:  `the version number of nel proxy`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("nel proxy version: %s", Version)
	},
}
