package handlers

import (
	"net/http"
	"net/url"

	"github.com/gorilla/csrf"
	"github.com/gorilla/handlers"

	"github.com/syaiful6/sersan"
	"github.com/syaiful6/thatique/context"
	"github.com/syaiful6/thatique/pkg/httputil"
	"github.com/syaiful6/thatique/shop/auth"
)

type signinLimitDispatcher struct {
	finder  auth.UserFinder
	limiter *RateLimiter
}

type signinHandler struct {
	*Context
	finder  auth.UserFinder
	limiter *RateLimiter
}

func NewSigninLimitDispatcher(finder auth.UserFinder, n, b int) *signinLimitDispatcher {
	return &signinLimitDispatcher{
		finder:  finder,
		limiter: NewRateLimiter(n, b, httputil.GetSourceIP),
	}
}

func (sd *signinLimitDispatcher) DispatchHTTP(ctx *Context, r *http.Request) http.Handler {
	// if user already logged in, redirect to homepage
	if ctx.Auth.User(r) != nil {
		return http.RedirectHandler("/", http.StatusFound)
	}
	sgHandler := &signinHandler{
		Context: ctx,
		finder:  sd.finder,
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	limiter := sg.limiter.Get(r)
	if limiter.Allow() == false {
		w.WriteHeader(http.StatusTooManyRequests)
		if err = sg.renderForm(w, map[string]interface{}{
			"Title":          "Signin",
			"Description":    "Signin to Thatiq",
			"Errors":         []string{http.StatusText(429)},
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			sg.App.handleErrorHTML(w, err)
		}
		return
	}

	form := &auth.SigninForm{
		Finder:   sg.finder,
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	user, errs, ok := form.Validate()
	if !ok {
		if err = sg.renderForm(w, map[string]interface{}{
			"Title":          "Signin",
			"Description":    "Signin to Thatiq",
			"Email":          form.Email,
			"Errors":         sg.toSlice(errs),
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			sg.App.handleErrorHTML(w, err)
		}
		return
	}

	// check success
	r, err = sg.Context.Auth.Login(user, r)
	if err != nil {
		panic(err)
	}

	w.Header().Del("Content-Type")
	redirectURL := r.FormValue("next")
	if redirectURL == "" {
		redirectURL = "/"
	} else {
		redirectURL, _ = url.QueryUnescape(redirectURL)
	}

	if !httputil.IsSameSiteURLPath(redirectURL) {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (sg *signinHandler) toSlice(m map[string]string) []string {
	var xs []string
	for _, v := range m {
		xs = append(xs, v)
	}

	return xs
}
