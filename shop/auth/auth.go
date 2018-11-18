package auth

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	sessionName = "auth.session"

	userSessionKey = "auth.session.userKey"

	// UserKey is used to get the user object from
	// a user context
	UserKey = "auth.user"

	UserIdKey = "auth.user.id"
)

type UserInfo struct {
	Id string
}

func (u UserInfo) IsAnonymous() bool {
	return u.Id == ""
}

var anonymous = UserInfo{Id: ""}

func WithUser(ctx context.Context, user UserInfo) context.Context {
	return userInfoContext{
		Context: ctx,
		user:    user,
	}
}

type userInfoContext struct {
	context.Context
	user UserInfo
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

// authenticator
type Authenticator struct {
	store sessions.Store
}

func NewAuthenticator(store sessions.Store) *Authenticator {
	return &Authenticator{
		store: store,
	}
}

// Middleware that load user from session and set it current user if success
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userInfo := a.getUserFromSession(r)
		if userInfo.IsAnonymous() {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, a.LoginOnce(userInfo, r))
	})
}

// get current user
func (a *Authenticator) User(r *http.Request) UserInfo {
	u, ok := r.Context().Value(UserKey).(UserInfo)
	if !ok {
		return anonymous
	}
	return u
}

// Set user only to current request without persisting to session store
func (a *Authenticator) LoginOnce(u UserInfo, r *http.Request) *http.Request {
	return r.WithContext(WithUser(r.Context(), u))
}

// login user to application, return http.Request that can be passed to next http.Handler
// so that user visible.
func (a *Authenticator) Login(u UserInfo, w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	sess, err := a.store.Get(r, sessionName)
	if err != nil {
		return nil, err
	}

	// save the user id to session
	sess.Values[userSessionKey] = u.Id
	if err = sess.Save(r, w); err != nil {
		return nil, err
	}

	return a.LoginOnce(u, r), nil
}

// logout user from application, remove userInfoContext if it can
func (a *Authenticator) Logout(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	sess, err := a.store.Get(r, sessionName)
	if err != nil {
		return nil, err
	}

	delete(sess.Values, userSessionKey)
	if err = sess.Save(r, w); err != nil {
		return nil, err
	}

	// remove user from context
	ctx, ok := r.Context().(userInfoContext)
	if !ok {
		return r, nil
	}

	return r.WithContext(ctx.Context), nil
}

func (a *Authenticator) getUserFromSession(r *http.Request) UserInfo {
	sess, err := a.store.Get(r, sessionName)
	var (
		ok   bool
		uid  string
		suid interface{}
	)

	if err != nil {
		return anonymous
	}
	if suid, ok = sess.Values[userSessionKey]; !ok {
		return anonymous
	}

	if uid, ok = suid.(string); !ok {
		return anonymous
	}

	return UserInfo{Id: uid}
}
