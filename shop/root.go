package shop

import (
	"github.com/spf13/cobra"

	"github.com/syaiful6/thatique/version"
)


var showVersion bool

func init() {
	RootCmd.AddCommand(ServeCmd)
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show the version and exit")
}

// RootCmd is the main command for the 'registry' binary.
var RootCmd = &cobra.Command{
	Use:   "shop",
	Short: "Thatiq's shop",
	Long:  "Thatiq's shop",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			version.PrintVersion()
			return
		}
		cmd.Usage()
	},
}
