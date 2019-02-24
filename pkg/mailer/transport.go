package mailer

import (
	"net/mail"
	"strings"

	"github.com/emersion/go-message"
)

type Transport interface {
	// Open a network connection. This method can be overwritten by backend
	// implementations to open a network connection.
	//
	// This method can be called by applications to force a single
	// network connection to be used when sending mails. See the
	// SendMessages() method of the SMTP backend for a reference
	// implementation.
	Open() error

	// Close a network connection.
	Close() error

	// Send one or more message.Entity objects and return the number of email
	// messages sent. Before calling this method, you need to make sure to call
	// Open, otherwise most implementation might choose to always return 0.
	SendMessages(messages []*message.Entity) (int, error)
}

func FormatAddressList(xs []*mail.Address) string {
	formatted := make([]string, len(xs))
	for i, a := range xs {
		formatted[i] = a.String()
	}

	return strings.Join(formatted, ", ")
}
