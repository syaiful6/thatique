package auth

import (
	"testing"
)

func TestPasswordHasher(t *testing.T) {
	userData := []struct {
		email    string
		password string
	}{
		{
			email:    "nami@pub.example.com",
			password: "secret",
		},
		{
			email:    "luci@machine.example",
			password: "secret12333longpasswordssssssssssssssaaaaaa",
		},
	}
	for i, data := range userData {
		user, err := Create(data.email, data.password)
		if err != nil {
			t.Errorf("Failed to create user for %d", i)
			return
		}
		if user.Password == data.password {
			t.Error("You should not store plain password")
		}
		if !user.VerifyPassword(data.password) {
			t.Errorf("passwords for %s should correct", user.Email)
			return
		}
	}
}
