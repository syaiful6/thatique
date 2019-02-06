package handlers

import (
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/csrf"
	"github.com/gorilla/handlers"

	"github.com/syaiful6/sersan"
	"github.com/syaiful6/thatique/context"
	"github.com/syaiful6/thatique/shop/auth"
)

type signinLimitDispatcher struct {
	limiter *RateLimiter
}

type signinHandler struct {
	*Context
	limiter *RateLimiter
}

func NewSigninLimitDispatcher(n, b int) *signinLimitDispatcher {
	return &signinLimitDispatcher{NewIpVisitor(n, b)}
}

func (sd *signinLimitDispatcher) DispatchHTTP(ctx *Context, r *http.Request) http.Handler {
	sgHandler := &signinHandler{
		Context: ctx,
		limiter: sd.limiter,
	}

	return handlers.MethodHandler{
		"GET":  http.HandlerFunc(sgHandler.showSignupForm),
		"POST": http.HandlerFunc(sgHandler.postSignupForm),
	}
}

func (sg *signinHandler) showSignupForm(w http.ResponseWriter, r *http.Request) {
	var emailValue string

	sess, err := sersan.GetSession(r)
	if err != nil {
		sg.App.handleErrorHTML(w, err)
		return
	}

	if v, ok := sess["input.signup.email.flash"]; ok {
		emailValue = v.(string)
		delete(sess, "input.signup.email.flash")
	}

	if err = sg.renderForm(w, map[string]interface{}{
		"Title":          "Signin",
		"Description":    "Signin to Thatiq",
		"Email":          emailValue,
		csrf.TemplateTag: csrf.TemplateField(r),
	}); err != nil {
		sg.App.handleErrorHTML(w, err)
	}
}

func (sg *signinHandler) renderForm(w http.ResponseWriter, data map[string]interface{}) error {
	tpl, err := sg.App.Template("auth/sign", "base.html", "auth/signin.html")
	if err != nil {
		return err
	}
	if err = tpl.Execute(w, data); err != nil {
		context.GetLogger(sg).Debugf("unexpected error when executing auth/signin.html: %v", err)
		return err
	}

	return nil
}

func (sg *signinHandler) postSignupForm(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		context.GetLogger(sg).Debugf("error encountered when parsing form: %v", err)
		// this is likely parsing error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	limiter := sg.limiter.Get(r)
	if limiter.Allow() == false {
		http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	if len(email) == 0 || len(password) == 0 {
		if err = sg.renderForm(w, map[string]interface{}{
			"Title":          "Signin",
			"Description":    "Signin to Thatiq",
			"Email":          email,
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			sg.App.handleErrorHTML(w, err)
		}
		return
	}

	var user *auth.User
	conn := sg.App.mongo.Copy()
	defer conn.Close()
	if err := conn.Find(user, bson.M{"email": email}).One(&user); err != nil {
		if err == mgo.ErrNotFound {
			auth.NewUser(email, password)
			if err = sg.renderForm(w, map[string]interface{}{
				"Title":          "Signin",
				"Description":    "Signin to Thatiq",
				"Email":          email,
				csrf.TemplateTag: csrf.TemplateField(r),
			}); err != nil {
				sg.App.handleErrorHTML(w, err)
			}
			return
		}
		sg.App.handleErrorHTML(w, err)
		return
	}

	if !user.VerifyPassword(password) {
		if err = sg.renderForm(w, map[string]interface{}{
			"Title":          "Signin",
			"Description":    "Signin to Thatiq",
			"Email":          email,
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			sg.App.handleErrorHTML(w, err)
		}
		return
	}

	redirectURL := r.FormValue("next")
	if redirectURL != "" {
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
	return
}
