package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/globalsign/mgo/bson"
	"github.com/syaiful6/sersan"
)

const (
	UserSessionKey = "auth.session.userKey"

	// UserKey is used to get the user object from
	// a user context
	UserKey = "auth.user"

	UserIdKey = "auth.user.id"
)

func WithUser(ctx context.Context, user *User) context.Context {
	return userInfoContext{
		Context: ctx,
		user:    user,
	}
}

type userInfoContext struct {
	context.Context
	user *User
}

func (uic userInfoContext) Value(key interface{}) interface{} {
	switch key {
	case UserKey:
		return uic.user
	case UserIdKey:
		return uic.user.Id
	}

	return uic.Context.Value(key)
}

// authenticator for the web
type Authenticator struct {
	provider FinderById
}

func NewAuthenticator(provider FinderById) *Authenticator {
	return &Authenticator{
		provider: provider,
	}
}

// Middleware that load user from session and set it current user if success
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := a.getUserFromSession(r)
		if user == nil {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, a.LoginOnce(user, r))
	})
}

// get current user
func (a *Authenticator) User(r *http.Request) *User {
	u, ok := r.Context().Value(UserKey).(*User)
	if !ok {
		return nil
	}
	return u
}

// Set user only to current request without persisting to session store
func (a *Authenticator) LoginOnce(u *User, r *http.Request) *http.Request {
	return r.WithContext(WithUser(r.Context(), u))
}

// login user to application, return http.Request that can be passed to next http.Handler
// so that user visible.
func (a *Authenticator) Login(u *User, r *http.Request) (*http.Request, error) {
	if u == nil {
		return r, errors.New("user passed to login can't be nil user")
	}
	err := a.updateSession(u, r)
	if err != nil {
		return r, err
	}
	return a.LoginOnce(u, r), nil
}

// logout user from application, remove userInfoContext if it can
func (a *Authenticator) Logout(r *http.Request) (*http.Request, error) {
	sess, err := sersan.GetSession(r)
	if err != nil {
		return nil, err
	}

	delete(sess, UserSessionKey)

	// remove user from context
	ctx, ok := r.Context().(userInfoContext)
	if !ok {
		return r, nil
	}

	return r.WithContext(ctx.Context), nil
}

// update session
func (a *Authenticator) updateSession(user *User, r *http.Request) error {
	sess, err := sersan.GetSession(r)
	if err != nil {
		return err
	}

	sess[UserSessionKey] = user.Id.Hex()
	return nil
}

func (a *Authenticator) getUserFromSession(r *http.Request) *User {
	sess, err := sersan.GetSession(r)
	var (
		ok   bool
		uid  string
		suid interface{}
	)

	if err != nil {
		return nil
	}
	if suid, ok = sess[UserSessionKey]; !ok {
		return nil
	}

	if uid, ok = suid.(string); !ok {
		return nil
	}

	if !bson.IsObjectIdHex(uid) {
		return nil
	}

	user, err := a.provider.FindById(bson.ObjectIdHex(uid))
	if err != nil {
		return nil
	}

	return user
}
