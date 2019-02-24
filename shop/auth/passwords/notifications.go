package passwords

import (
	"io"
	"net/mail"

	"github.com/emersion/go-message"
	"github.com/syaiful6/thatique/pkg/mailer"
	"github.com/syaiful6/thatique/pkg/queue"
	"github.com/syaiful6/thatique/shop/auth"
)

type Notifier interface {
	// Notify, notify message to user
	Notify(user *auth.User, message io.Reader) error
}

type MailNotifier struct {
	Sender    string
	transport mailer.Transport
	channel   chan<- queue.Job
}

func NewMailNotifier(sender string, m mailer.Transport, channel chan<- queue.Job) *MailNotifier {
	return &MailNotifier{
		Sender:    sender,
		transport: m,
		channel:   channel,
	}
}

func (n *MailNotifier) Notify(user *auth.User, body io.Reader) error {
	to := &mail.Address{
		Name:    user.Profile.Name,
		Address: user.Email,
	}
	from, err := mail.ParseAddressList(n.Sender)
	if err != nil {
		return err
	}

	sender := mailer.FormatAddressList(from)
	h := make(message.Header)
	h.Set("Sender", sender)
	h.Set("From", sender)
	h.Set("To", mailer.FormatAddressList([]*mail.Address{to}))
	h.Set("Subject", "[Thatiq.com] Reset Password")
	h.Set("Content-Type", "text/plain")

	msg, err := message.New(h, body)
	if err != nil {
		return err
	}

	job := mailer.NewJobMail(n.transport, []*message.Entity{msg})

	n.channel <- job

	return nil
}
