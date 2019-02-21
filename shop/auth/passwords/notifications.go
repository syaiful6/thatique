package passwords

import (
	"github.com/syaiful6/thatique/shop/auth"
	"io"
)

type Notifier interface {
	// Notify, notify message to user
	Notify(user *auth.User, message io.Reader) error
}
