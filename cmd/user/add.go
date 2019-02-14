package user

import (
	"fmt"
	"os"
	osuser "os/user"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/syaiful6/thatique/configuration"
	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/db"
)

func promptProfileName() (string, error) {
	var defaultName string
	user := osuser.Current()
	defaultName = user.Name
	if len(defaultName) == 0 {
		defaultName = user.Username
	}

	var validator = func(input string) error {
		if input == "" {
			return fmt.Errorf("name can't be empty")
		}
		return nil
	}

	profileNamePrompt := promptui.Prompt{
		Label:    "Name",
		Default:  defaultName,
		Validate: validator,
	}

	return profileNamePrompt.Run()
}

var userAddCommand = &cobra.Command{
	Use:   "add",
	Short: "add a user to thatique application",
	Long: `add a user to thatique application, if email your provided already
in use, then this command will report the error.

Flags:
-s flag to create a superuser
-S flag to create a staff`,
	Args: ArgEmailValidator,
	Run: func(cmd *cobra.Command, args []string) {
		email := args[0]

		var args1 []string
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

		// Superuser is always staff
		if Superuser {
			user.Staff = true
		}
		if mongodb.Exists(user) {
			fmt.Fprintf(os.Stderr, "User with email %s already exists", email)
			return
		}

		name, err := promptProfileName()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid user's profile name: %v \n", err)
			return
		}

		user.Profile = auth.Profile{Name: name}

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

		_, err = mongodb.Upsert(user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed inserting user document to mongodb with error: %v \n", err)
			return
		}

		fmt.Printf("user created successfully")
	},
}
