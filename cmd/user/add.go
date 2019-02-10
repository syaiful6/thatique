package user

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/syaiful6/thatique/configuration"
	"github.com/syaiful6/thatique/pkg/emailparser"
	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/db"
)

func promptPassword(label string) (string, error) {
	passwordValidator := func(input string) error {
		// if this for superuser, then the password's length must be
		// at least 15
		if Superuser && len(input) < 15 {
			return fmt.Errorf("superuser's password must be at least 15 length")
		}

		if len(input) < 8 {
			return fmt.Errorf("password must be at least 8 length")
		}

		return nil
	}

	passwordPrompt := promptui.Prompt{
		Label:    label,
		Validate: passwordValidator,
	}

	return passwordPrompt.Run()
}

var userAddCommand = &cobra.Command{
	Use:   "add",
	Short: "add a user to thatique application",
	Long: `add a user to thatique application, if email your provided already
	in use, then this command will report the error.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("user add require exactly one argument")
		}

		_, err := emailparser.NewEmail(args[0])
		if err != nil {
			return fmt.Errorf("the argument passed to `user add` must be an email")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		email := args[0]

		var args1  []string
		config, err := configuration.Resolve(args1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration err: %v \n", err)
			return
		}

		mongodb, err := db.Dial(config.MongoDB.URI, config.MongoDB.Name)
		if err != nil {
			fmt.Fprint(os.Stderr, "can't connect to mongodb server provided in configuration file")
			return
		}

		user := &auth.User{
			Email:     email,
			Superuser: Superuser,
			Staff:     Staff,
		}
		if mongodb.Exists(user) {
			fmt.Fprintf(os.Stderr, "User with email %s already exists", email)
			return
		}

		pswd1, err := promptPassword("Password")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid password %v \n", err)
			return
		}
		pswd2, err := promptPassword("Confirm Password")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid password %v", err)
			return
		}

		if pswd1 != pswd2 {
			fmt.Fprintf(os.Stderr, "password and confirmation password is not equal")
			return
		}

		err = user.SetPassword([]byte(pswd2))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed setting password with error: %v \n", err)
			return
		}

		_, err = mongodb.Upsert(user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed inserting user document to mongodb with error: %v \n", err)
			return
		}

		fmt.Printf("user created successfully")
	},
}
