package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "subpub version",
	Long:  `the version number of subscriber and publisher server`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("subpub version: %s\n", Version)
	},
}
