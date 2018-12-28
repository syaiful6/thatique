package middlewares

import (
	"github.com/gorilla/mux"
	"net/http"
)

type IfRequestMiddleware struct {
	Predicate	func(*http.Request) bool
	Middlewares []mux.MiddlewareFunc
}

func (mw *IfRequestMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if mw.Predicate(req) {
			for i := len(mw.Middlewares) - 1; i >= 0; i-- {
				next = mw.Middlewares[i](next)
			}
		}
		next.ServeHTTP(w, req)
	})
}
