package user

import (
	"fmt"
	"os"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/spf13/cobra"
	"github.com/syaiful6/thatique/configuration"
	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/db"
)

var changePasswordCommand = &cobra.Command{
	Use:   "changepassword [useremail]",
	Short: "Change user's password",
	Long: `Change user's password used for authentication. Without normal change
password workflow.`,
	Args: ArgEmailValidator,
	Run: func(cmd *cobra.Command, args []string) {
		email := args[0]

		var args1 []string
		config, err := configuration.Resolve(args1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration err: %v \n", err)
			return
		}

		conn, err := db.Dial(config.MongoDB.URI, config.MongoDB.Name)
		if err != nil {
			fmt.Fprint(os.Stderr, "can't connect to mongodb server provided in configuration file")
			return
		}

		var user *auth.User
		err = conn.Find(user, bson.M{"email": email}).One(&user)
		if err != nil {
			if err == mgo.ErrNotFound {
				fmt.Fprintf(os.Stderr, "there are no user with %s email", email)
				return
			}

			fmt.Fprintf(os.Stderr, "error querying mongodb server, it return error: %v", err)
			return
		}

		// we have user right now
		pswd1, err := promptPassword("Password", user.Staff)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid password %v \n", err)
			return
		}
		pswd2, err := promptConfirmPassword("Confirm Password", pswd1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid password %v", err)
			return
		}

		err = user.SetPassword([]byte(pswd2))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed setting password with error: %v \n", err)
			return
		}

		_, err = conn.Upsert(user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed updating user document to mongodb with error: %v \n", err)
			return
		}

		fmt.Printf("user's password changed successfully")
	},
}
