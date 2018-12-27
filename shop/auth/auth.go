package auth

import (
	"context"
	"net/http"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/sessions"
	"github.com/syaiful6/thatique/shop/db"
)

const (
	sessionName = "auth.session"

	userSessionKey = "auth.session.userKey"

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

type UserProvider interface {
	FindUserById(bson.ObjectId) (*User, error)
}

type MgoUserProvider struct {
	conn *db.MongoConn
}

func NewMgoUserProvider(conn *db.MongoConn) *MgoUserProvider {
	return &MgoUserProvider{conn: conn}
}

func (p *MgoUserProvider) FindUserById(id bson.ObjectId) (*User, error) {
	var user *User
	if err := p.conn.Find(user, bson.M{"_id": id}).One(&user); err != nil {
		return nil, err
	}
	return user, nil
}

// authenticator
type Authenticator struct {
	store    sessions.Store
	provider UserProvider
}

func NewAuthenticator(store sessions.Store, provider UserProvider) *Authenticator {
	return &Authenticator{
		store:    store,
		provider: provider,
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
func (a *Authenticator) User(r *http.Request) *User {
	u, ok := r.Context().Value(UserKey).(*User)
	if !ok {
		return &User{}
	}
	return u
}

// Set user only to current request without persisting to session store
func (a *Authenticator) LoginOnce(u *User, r *http.Request) *http.Request {
	return r.WithContext(WithUser(r.Context(), u))
}

// login user to application, return http.Request that can be passed to next http.Handler
// so that user visible.
func (a *Authenticator) Login(u *User, w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	sess, err := a.store.Get(r, sessionName)
	if err != nil {
		return nil, err
	}

	// save the user id to session
	sess.Values[userSessionKey] = u.Id.Hex()
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

func (a *Authenticator) getUserFromSession(r *http.Request) *User {
	sess, err := a.store.Get(r, sessionName)
	var (
		ok   bool
		uid  string
		suid interface{}
	)

	if err != nil {
		return &User{}
	}
	if suid, ok = sess.Values[userSessionKey]; !ok {
		return &User{}
	}

	if uid, ok = suid.(string); !ok {
		return &User{}
	}

	if !bson.IsObjectIdHex(uid) {
		return &User{}
	}

	user, err := a.provider.FindUserById(bson.ObjectIdHex(uid))
	if err != nil {
		return &User{}
	}

	return user
}
