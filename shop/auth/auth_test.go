package auth

import (
	"bytes"
	"errors"
	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/sessions"
	"net/http"
	"net/http/httptest"
	"testing"
)

// DefaultRemoteAddr is the default remote address to return in RemoteAddr if
// an explicit DefaultRemoteAddr isn't set on ResponseRecorder.
const DefaultRemoteAddr = "1.2.3.4"

// The expected user info
var oid = bson.NewObjectId()
var ui = &User{
	Id: oid,
}

type fakeUserProvider struct {
}

func (p *fakeUserProvider) FindUserById(id bson.ObjectId) (*User, error) {
	if id != oid {
		return nil, errors.New("invalid user")
	}

	return &User{
		Id: oid,
	}, nil
}

// NewRecorder returns an initialized ResponseRecorder.
func NewRecorder() *httptest.ResponseRecorder {
	return &httptest.ResponseRecorder{
		HeaderMap: make(http.Header),
		Body:      new(bytes.Buffer),
	}
}

func TestAuthenticatorLogin(t *testing.T) {
	authenticator := NewAuthenticator(sessions.NewCookieStore([]byte("secret-key")), &fakeUserProvider{})

	r := httptest.NewRequest("GET", "http://localhost:8080/", nil)
	w := NewRecorder()
	r, err := authenticator.Login(ui, w, r)
	if err != nil {
		t.Fatalf("error login user to application: %v", err)
	}
	ui2 := authenticator.User(r)
	if ui2.IsAnonymous() || ui2.Id != ui.Id {
		t.Errorf("Go user id %s, expected %s", ui2.Id, ui.Id)
	}

	hdr := w.Header()
	cookies, ok := hdr["Set-Cookie"]
	if !ok || len(cookies) != 1 {
		t.Fatal("No cookies. Header:", hdr)
	}

	// after user login, the system should able to load user from session
	r = httptest.NewRequest("GET", "http://localhost:8080/", nil)
	r.Header.Add("Cookie", cookies[0])
	w = NewRecorder()

	ui2 = authenticator.getUserFromSession(r)
	if ui2.IsAnonymous() || ui2.Id != ui.Id {
		t.Errorf("Go user id %s, expected %s", ui2.Id, ui.Id)
	}
	authenticator.Logout(w, r)
}
