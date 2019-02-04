package handlers

import (
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/syaiful6/thatique/context"
)

type homepageHandler struct {
	*Context
}

func homepageDispatcher(ctx *Context, r *http.Request) http.Handler {
	hpHandlers := &homepageHandler{
		Context: ctx,
	}

	mhandler := handlers.MethodHandler{
		"GET":     http.HandlerFunc(hpHandlers.getHomepage),
		"OPTIONS": http.HandlerFunc(hpHandlers.getHomepage),
	}

	return mhandler
}

func (h *homepageHandler) getHomepage(w http.ResponseWriter, r *http.Request) {
	tpl, err := h.App.Template("homepage", "base.html", "homepage.html")
	if err != nil {
		panic(err)
	}
	if err = tpl.Execute(w, map[string]interface{}{
		"Title":       "Thatiq",
		"Description": "Executive",
	}); err != nil {
		context.GetLogger(h).Debugf("unexpected error when executing homepage.html: %v", err)
		w.WriteHeader(500)
	}
}
