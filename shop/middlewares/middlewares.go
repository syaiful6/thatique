package middlewares

import (
	"github.com/gorilla/mux"
	"net/http"
)

type IfRequestMiddleware struct {
	Predicate func(*http.Request) bool
	Inner     mux.MiddlewareFunc
}

func (mw *IfRequestMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if mw.Predicate(req) {
			mw.Inner(next).ServeHTTP(w, req)
			return
		}
		next.ServeHTTP(w, req)
	})
}
