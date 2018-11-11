package session

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/syaiful6/thatique/shop/auth"
)

var (
	sessionName = "auth.session.user"
	userSessionKey = "auth.session.userKey"
)

type sessionStrategy struct {
	store session.Store
}

func makeSessionStrategy(opts map[string]interface{}) (mux.MiddlewareFunc, error) {
	store, ok = opts["store"].(sessions.Store)
	if !ok {
		return nil, fmt.Error("session auth requires a valid option string store")
	}

	strategy := &sessionStrategy{
		store: store
	}

	return strategy.authenticate, nil
}

func PersistUser(store sessions.Store, user auth.UserInfo, w http.ResponseWriter, req *http.Request) (err error) {
	sess, err := store.Get(req, sessionName)
	if err != nil {
		return err
	}
	sess.Values[userSessionKey] = user
	sess.Save(r, w)
	return
}

func DeleteUser(store sessions.Store, w http.ResponseWriter, req *http.Request) (err error) {
	sess, err := store.Get(req, sessionName)
	if err != nil {
		return err
	}
	delete(sess.Values, userSessionKey)
	sess.Save(r, w)
	return
}

func (ss *sessionStrategy) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
		sess, err := ss.store.Get(req, sessionName)
		if err != nil {
			return next.ServeHTTP(w, req)
		}

		if v, ok := sess.Values[userSessionKey]; ok {
			userInfo := v.(auth.UserInfo)
			return next.ServeHTTP(w, req.WithContext(auth.WithUser(req.Context(), userInfo)))
		}

		next.ServeHTTP(w, req)
	})
}

func init() {
	auth.Register("session", makeSessionStrategy)
}
