package user

import (
	"github.com/spf13/cobra"
)

var (
	Superuser, Staff bool
)

func init() {
	// add
    userAddCommand.Flags().BoolVarP(&Superuser, "superuser", "s", false, "create new superuser")
	userAddCommand.Flags().BoolVarP(&Staff, "staff", "S", false, "create new staff user")
	UserCommand.AddCommand(userAddCommand)
}

var UserCommand = &cobra.Command{
	Use:   "user",
	Short: "user related command line utility",
	Long:  `cli for managing user in thatique. Adding or modify user attribute`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}
