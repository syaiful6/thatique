package handlers

import (
	"html/template"
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/syaiful6/thatique/context"
)

type homepageHandler struct {
	*Context
	template *template.Template
}

func homepageDispatcher(ctx *Context, r *http.Request) http.Handler {
	tpl, err := ctx.App.template("homepage", "base.html", "homepage.html")
	if err != nil {
		panic(err)
	}
	hpHandlers := &homepageHandler{
		Context:  ctx,
		template: tpl,
	}

	mhandler := handlers.MethodHandler{
		"GET":     http.HandlerFunc(hpHandlers.GetHomepage),
		"OPTIONS": http.HandlerFunc(hpHandlers.GetHomepage),
	}

	return mhandler
}

func (h *homepageHandler) GetHomepage(w http.ResponseWriter, r *http.Request) {
	if err := h.template.Execute(w, map[string]interface{}{
		"Title":       "Thatiq",
		"Description": "Executive",
	}); err != nil {
		context.GetLogger(h).Debugf("unexpected error when executing homepage.html: %v", err)
		w.WriteHeader(500)
	}
}
