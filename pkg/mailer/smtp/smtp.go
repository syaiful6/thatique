package smtptransport

import (
	"crypto/tls"
	"errors"
	"github.com/emersion/go-message"
	"net"
	"net/mail"
	"net/smtp"
	"sync"
)

type Options struct {
	// The addr must include a port, as in "mail.example.com:smtp".
	Addr string
	// Authentication
	Auth smtp.Auth
}

type SMTPTransport struct {
	locker sync.Mutex
	conn   *smtp.Client
	option *Options

	serverName string
}

func NewSMTPTransport(option *Options) *SMTPTransport {
	host, _, _ := net.SplitHostPort(option.Addr)

	return &SMTPTransport{
		option:     option,
		serverName: host,
	}
}

func (t *SMTPTransport) Open() error {
	t.locker.Lock()
	defer t.locker.Unlock()

	if t.conn != nil {
		return nil
	}

	c, err := smtp.Dial(t.option.Addr)
	if err != nil {
		return err
	}

	if err = c.Hello("localhost"); err != nil {
		return err
	}

	// Start TLS if possible
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: t.serverName}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}

	// auth is non nil
	if t.option.Auth != nil {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("mailer.smptp: server doesn't support AUTH")
		}

		if err = c.Auth(t.option.Auth); err != nil {
			return err
		}
	}

	// connection establish, store it and return
	t.conn = c

	return nil
}

func (t *SMTPTransport) Close() error {
	t.locker.Lock()
	defer t.locker.Unlock()

	if t.conn == nil {
		return nil
	}

	err := t.conn.Quit()
	t.conn = nil
	return err
}

func (t *SMTPTransport) SendMessages(messages []*message.Entity) (int, error) {
	if len(messages) == 0 {
		return 0, nil
	}

	t.locker.Lock()
	defer t.locker.Unlock()

	var (
		numsent int
		err     error
		sent    bool
	)

	// fail silently?
	if t.conn == nil {
		return numsent, errors.New("mailer.smtp: connection not opened.")
	}

	for _, msg := range messages {
		sent, err = t.sendMessage(msg)
		if err != nil {
			return numsent, err
		}
		if sent {
			numsent += 1
		}
	}

	return numsent, nil
}

func (t *SMTPTransport) sendMessage(msg *message.Entity) (bool, error) {
	var (
		headerPrefix string
		fromAddrStr  string
		err          error
		addressList  []*mail.Address
	)
	resent := msg.Header.Get("Resent-Date")
	if resent != "" {
		headerPrefix = "Resent-"
	}

	if sender := msg.Header.Get(headerPrefix + "Sender"); sender != "" {
		fromAddrStr = sender
	} else if sender = msg.Header.Get(headerPrefix + "From"); sender != "" {
		fromAddrStr = sender
	}

	fromAddrs, err := mail.ParseAddressList(fromAddrStr)
	if err != nil || len(fromAddrs) == 0 {
		return false, nil
	}

	var toAddrs []*mail.Address
	for _, key := range []string{"To", "Bcc", "Cc"} {
		addrList := msg.Header.Get(headerPrefix + key)
		if addrList == "" {
			continue
		}
		addressList, err = mail.ParseAddressList(addrList)
		if err != nil {
			continue
		}
		toAddrs = append(toAddrs, addressList...)
	}

	if len(toAddrs) == 0 {
		return false, nil
	}

	if err = t.conn.Mail(fromAddrs[0].Address); err != nil {
		return false, err
	}

	for _, addr := range toAddrs {
		if err = t.conn.Rcpt(addr.Address); err != nil {
			return false, err
		}
	}

	w, err := t.conn.Data()
	if err != nil {
		return false, err
	}
	msg.WriteTo(w)
	err = w.Close()
	if err != nil {
		return false, err
	}

	return true, nil
}
