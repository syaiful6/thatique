package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

type OAuth2LoginHandler struct {
	*Authenticator
	Name string
	RedirectError string
	Config *oauth2.Config
}

func (oa *OAuth2LoginHandler) GetSessionStateKey() {
	return fmt.Sprintf("oauth2.state.%s", oa.Name)
}

// start login
func (oa *OAuth2LoginHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Create nonce
	nonce := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return
	}

	state := base64.URLEncoding.EncodeToString(nonce)
	sess, err := a.store.Get(r, sessionName)
	if err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return
	}

	sess.Values[oa.GetSessionStateKey()] = state
	if err = sess.Save(r, w); err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return
	}

	url := oa.Config.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (oa *OAuth2LoginHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	err := oa.verifySessionState(w, r)
	if err != nil {
		return
	}

	// check if this error redirect
	hasErr := r.FormValue("error")
	if len(hasErr) > 0 {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return
	}

	// exchange
	code := r.FormValue("code")
	token, err := oa.Config.Exchange(oauth2.NoContext, code)
	if err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return
	}
}

func (oa *OAuth2LoginHandler) verifySessionState(w http.ResponseWriter, r *http.Request) error {
	sess, err := a.store.Get(r, sessionName)
	if err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return err
	}

	if sessState, ok = sess.Values[oa.GetSessionStateKey()]; !ok {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return err
	}

	delete(sess.Values, oa.GetSessionStateKey())
	if err = sess.Save(r, w); err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return err
	}

	state, err := base64.URLEncoding.DecodeString(r.FormValue("state"))
	if err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return err
	}

	wantState, err := base64.URLEncoding.DecodeString(sessState)
	if err != nil {
		http.Redirect(w, r, oa.RedirectError, http.StatusTemporaryRedirect)
		return err
	}

	if subtle.ConstantTimeCompare(state, wantState) != 1 {
		http.Error(w, "Invalid OAuth2 state token", http.StatusForbidden)
		return errors.New("Invalid OAuth2 state token")
	}

	return nil
}
