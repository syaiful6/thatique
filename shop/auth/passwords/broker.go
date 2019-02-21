package passwords

import (
	"bytes"
	"fmt"
	"encoding/base64"
	"text/template"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/syaiful6/thatique/shop/auth"
)

type ErrorCode int

const (
	NoError ErrorCode = iota
	ErrPassNotMatch
	ErrMinimumStaffPassLen
	ErrMinimPassword
	ErrUpstream
)

type PasswordBroker struct {
	ResetURL string
	Finder   auth.UserFinder
	token    TokenGenerator
	notifier Notifier
}

type ResetRequest struct {
	user *auth.User
	token, Password1, Password2 string
}

const resetMessage = `
Ganti Password

Baru baru ini ada permintaan pergantian password di Thatiq.com. Jika itu bukan Anda
maka abaikan pesan ini.

Permintaan pergantian password ini akan habis masa berlakunya dalam 2 jam. Jika kamu
tidak mengganti password dalam 2 jam, maka kamu harus melakukan pemintaan kembali.

Untuk mengganti password Anda, klik link di bawah:
{{ .Link }}

Email {{ .Email }}
IP: {{ .IP }}
Dibuat: {{ .CreatedAt }}
`

func (b *PasswordBroker) SendResetLink(ip string, user *auth.User) error {
	token, err := b.token.Generate(user)
	if err != nil {
		return err
	}
	uid := base64.RawURLEncoding.EncodeToString([]byte(user.Id))
	link := fmt.Sprintf("%s/%s/%s", b.ResetURL, uid, token)

	t := template.Must(template.New("reset").Parse(resetMessage))
	var buf *bytes.Buffer
	if err = t.Execute(buf, map[string]interface{}{
		"Link": link,
		"Email": user.Email,
		"CreatedAt": time.Now().UTC().Format(time.RFC1123),
	}); err != nil {
		return err
	}

	return b.notifier.Notify(user, bytes.NewReader(buf.Bytes()))
}

func (b *PasswordBroker) Resets(req *ResetRequest, fn func(user *auth.User, pswd string) error) ErrorCode {
	if req.Password1 != req.Password2 {
		return ErrPassNotMatch
	}

	if req.user.Staff && len(req.Password1) < 15 {
		return ErrMinimumStaffPassLen
	}

	if len(req.Password1) < 8 {
		return ErrMinimPassword
	}

	err := fn(req.user, req.Password1)
	if err != nil {
		return ErrUpstream
	}

	if err = b.token.Delete(req.token); err != nil {
		return ErrUpstream
	}

	return NoError
}

// Validate reset token link variable.
func (b *PasswordBroker) ValidateReset(uid string, token string) (req *ResetRequest, ok bool) {
	if uid == "" || token == "" {
		ok = false
		return
	}

	objectid, ok := b.validateUid(uid)
	if !ok {
		ok = false
		return
	}

	user, err := b.Finder.FindUserById(objectid)
	if err != nil {
		ok = false
		return
	}

	req.user = user

	ok = b.token.IsValid(user, token)
	if !ok {
		ok = false
		return
	}

	req.token = token
	ok = true
	return
}

func (b *PasswordBroker) validateUid(uid string) (bson.ObjectId, bool) {
	bs, err := base64.RawURLEncoding.DecodeString(uid)
	if err != nil {
		return bson.ObjectId(""), false
	}

	objectid := bson.ObjectId(bs[:])
	return objectid, objectid.Valid()
}
