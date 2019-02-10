package cmd

import (
	"github.com/spf13/cobra"

	"github.com/syaiful6/thatique/cmd/user"
	"github.com/syaiful6/thatique/shop"
	"github.com/syaiful6/thatique/version"
)

func init() {
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show the version and exit")

	RootCmd.AddCommand(shop.ServeCmd)

	// secret
	secretKeyGenerate.Flags().BoolVarP(&secretKey64Len, "long", "L", false, "Generate the 64 bytes key version")
	secretKeyCommand.AddCommand(secretKeyGenerate)
	RootCmd.AddCommand(secretKeyCommand)

	// user
	RootCmd.AddCommand(user.UserCommand)
}

var showVersion bool

var RootCmd = &cobra.Command{
	Use:   "Thatique",
	Short: "Thatique's CLI application to manage thatique server",
	Long:  `Thatique's CLI application to manage thatique server.`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			version.PrintVersion()
			return
		}
		cmd.Usage()
	},
}
