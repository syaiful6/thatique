package auth

import (
	"fmt"

	"github.com/syaiful6/thatique/pkg/emailparser"
)

// SigninForm represent signin form inputs that user submit
type SigninForm struct {
	Finder   FinderByEmail
	Email    string
	Password string
}

func (form *SigninForm) Validate() (user *User, err map[string]string, ok bool) {
	err = make(map[string]string)
	ok = form.validateInput(err)
	if !ok {
		return
	}

	user, verr := form.Finder.FindByEmail(form.Email)
	if verr != nil {
		ok = false
		err["email_password"] = "your email or password is incorrect"
		return
	}

	if !user.VerifyPassword(form.Password) {
		ok = false
		err["email_password"] = "your email or password is incorrect"
		ok = false
		return
	}

	if !user.IsActive() {
		var msg string
		if user.Status == UserStatusInactive {
			msg = "your account status is inactive, validate your email"
		} else {
			msg = "your account is locked up"
		}

		err["status"] = msg
		ok = false
		return
	}

	ok = true

	return
}

func (form *SigninForm) validateInput(m map[string]string) bool {
	ok := true
	if form.Email == "" {
		m["Email"] = "email field is required"
		ok = false
	} else if emailparser.IsValidEmail(form.Email) {
		m["Email"] = fmt.Sprintf("%s is not valid email", form.Email)
		ok = false
	}

	if form.Password == "" {
		m["Password"] = "password field is required"
		ok = false
	}

	return ok
}
