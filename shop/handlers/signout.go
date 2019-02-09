package handlers

import (
	"net/http"
	"net/url"

	"github.com/gorilla/handlers"
	"github.com/syaiful6/thatique/pkg/httputil"
)

type signoutHandler struct {
	*Context
}

func signoutDispatcher(ctx *Context, r *http.Request) http.Handler {
	authenticator := ctx.App.authenticator
	// user not logged in
	if authenticator.User(r) == nil {
		return http.RedirectHandler("/", http.StatusFound)
	}

	sghandler := &signoutHandler{Context: ctx}

	return handlers.MethodHandler{
		"GET":  sghandler,
		"POST": sghandler,
	}
}

func (sg *signoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r, err := sg.App.authenticator.Logout(r)
	if err != nil {
		sg.App.handleErrorHTML(w, err)
		return
	}

	nextPage := r.URL.Query().Get("next")
	if nextPage == "" {
		nextPage = "/"
	} else {
		nextPage, _ = url.QueryUnescape(nextPage)
	}

	if !httputil.IsSameSiteURLPath(nextPage) {
		nextPage = "/"
	}

	http.Redirect(w, r, nextPage, http.StatusFound)
}
