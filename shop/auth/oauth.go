package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/syaiful6/sersan"
	"golang.org/x/oauth2"
)

var (
	OAuth2ContextErrKey = "oauth2.ctx.err"
	ErrTokenMismatch    = errors.New("oauth2 state token mismatch")
	ErrNoToken          = errors.New("oauth2 state token not found in request")
	InvalidToken        = errors.New("oauth2 invalid state token")
)

type OAuth2LoginHandler struct {
	*Authenticator
	Name         string
	RedirectPath string
	Config       *oauth2.Config
	ErrorHandler http.Handler
	Fetcher      UserFetcherFunc
}

// a function that take `oauth2.Token` and return User. This usually get local User
// based on Provider, or create one if not exists yet.
type UserFetcherFunc func(*oauth2.Token) (*User, error)

type OAuth2ErrorResponse struct {
	err, desc string
}

func (err *OAuth2ErrorResponse) Error() string {
	return fmt.Sprintf("oauth error response: %s. detail: %s", err.err, err.desc)
}

func IsOAuth2ErrorResponse(err error) bool {
	if _, ok := err.(*OAuth2ErrorResponse); ok {
		return true
	}
	return false
}

// session key used to store state
func (oa *OAuth2LoginHandler) GetSessionStateKey() string {
	return fmt.Sprintf("oauth2.state.%s", oa.Name)
}

// start login oauth2 flow.
// - Set a random CSRF token in our session
// - Redirect to the Provider's authorization URL
func (oa *OAuth2LoginHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Create nonce
	state := generateSessionState(r)

	sess, err := sersan.GetSession(r)
	if err != nil {
		panic("oauth can't get session store: " + err.Error())
	}

	sess[oa.GetSessionStateKey()] = state

	http.Redirect(w, r, oa.Config.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

// After user complete authorization, user redirect back to our page.
// - Verify the URL's CSRF token matches our session
// - Use the code parameter to fetch an AccessToken
// - Fetch User using provider `UserFetcherFunc`
// - Login and redirect to `RedirectPath`
func (oa *OAuth2LoginHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	err := oa.verifySessionState(w, r)
	if err != nil {
		oa.ErrorHandler.ServeHTTP(w, contextSave(r, OAuth2ContextErrKey, err))
		return
	}

	// check if this error redirect
	hasErr := r.FormValue("error")
	if len(hasErr) > 0 {
		oa.ErrorHandler.ServeHTTP(w, contextSave(r, OAuth2ContextErrKey, &OAuth2ErrorResponse{
			err:  hasErr,
			desc: r.FormValue("error_description"),
		}))
		return
	}

	// exchange
	token, err := oa.Config.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
	if err != nil {
		oa.ErrorHandler.ServeHTTP(w, contextSave(r, OAuth2ContextErrKey, err))
		return
	}

	user, err := oa.Fetcher(token)
	if err != nil {
		oa.ErrorHandler.ServeHTTP(w, contextSave(r, OAuth2ContextErrKey, err))
		return
	}

	nr, err := oa.Login(user, r)
	if err != nil {
		oa.ErrorHandler.ServeHTTP(w, contextSave(r, OAuth2ContextErrKey, err))
		return
	}

	// we have user now, login then redirect to target
	http.Redirect(w, nr, oa.RedirectPath, http.StatusTemporaryRedirect)
}

//
func generateSessionState(r *http.Request) string {
	state := r.URL.Query().Get("state")
	if len(state) > 0 {
		return state
	}

	nonce := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic("oauth: source of randomness unavailable: " + err.Error())
	}

	return base64.URLEncoding.EncodeToString(nonce)
}

// verify session state
func (oa *OAuth2LoginHandler) verifySessionState(w http.ResponseWriter, r *http.Request) error {
	var sessState string

	sess, err := sersan.GetSession(r)
	if err != nil {
		panic("oauth can't get session store: " + err.Error())
	}

	if b, ok := sess[oa.GetSessionStateKey()]; ok {
		sessState, ok = b.(string)
		if !ok {
			return InvalidToken
		}
	} else {
		return ErrNoToken
	}

	delete(sess, oa.GetSessionStateKey())

	if sessState != "" && (sessState != r.FormValue("state")) {
		return ErrTokenMismatch
	}

	return nil
}

func contextSave(r *http.Request, key string, val interface{}) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, key, val)
	return r.WithContext(ctx)
}
