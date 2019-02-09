package handlers

import (
	"html/template"
	"net/http"
)

func Handler404(tpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if err := tpl.Execute(w, map[string]interface{}{
			"Title":       "404: Page Not Found",
			"Description": "This is not the web page you are looking for",
		}); err != nil {
			panic(err)
		}
	})
}
