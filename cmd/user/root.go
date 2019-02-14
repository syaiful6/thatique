package user

import (
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/syaiful6/thatique/pkg/emailparser"
)

var (
	Superuser, Staff bool
)

func init() {
	// add
	userAddCommand.Flags().BoolVarP(&Superuser, "superuser", "s", false, "create new superuser")
	userAddCommand.Flags().BoolVarP(&Staff, "staff", "S", false, "create new staff user")
	UserCommand.AddCommand(userAddCommand)

	// changepassword
	UserCommand.AddCommand(changePasswordCommand)
}

//
func ArgEmailValidator(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("Require exactly one argument")
	}

	_, err := emailparser.NewEmail(args[0])
	if err != nil {
		return errors.New("Argument passed must be an email")
	}
	return nil
}

func promptPassword(label string, staff bool) (string, error) {
	passwordValidator := func(input string) error {
		// if this for superuser, then the password's length must be
		// at least 15
		if staff && len(input) < 15 {
			return fmt.Errorf("staff's password must be at least 15 length. You specify %d length", len(input))
		}

		if len(input) < 8 {
			return fmt.Errorf("password must be at least 8 length, you specify %d length", len(input))
		}

		return nil
	}

	passwordPrompt := promptui.Prompt{
		Label:    label,
		Validate: passwordValidator,
		Mask:     '*',
	}

	return passwordPrompt.Run()
}

func promptConfirmPassword(label, password string) (string, error) {
	passwordValidator := func(input string) error {
		if input != password {
			return errors.New("password and confirmation password is not equal")
		}

		return nil
	}

	passwordPrompt := promptui.Prompt{
		Label:    label,
		Validate: passwordValidator,
		Mask:     '*',
	}

	return passwordPrompt.Run()
}

var UserCommand = &cobra.Command{
	Use:   "user",
	Short: "user related command line utility",
	Long:  `cli for managing user in thatique. Adding or modify user attribute`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}
