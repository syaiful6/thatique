package cmd

import (
	"github.com/spf13/cobra"

	"github.com/syaiful6/thatique/shop"
	"github.com/syaiful6/thatique/version"
)

func init() {
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show the version and exit")

	RootCmd.AddCommand(shop.ServeCmd)

	// secret
	secretKeyCommand.AddCommand(secretKeyGenerate)
	RootCmd.AddCommand(secretKeyCommand)
}

var showVersion bool

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
