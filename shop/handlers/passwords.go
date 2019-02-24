package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	"github.com/gorilla/csrf"
	"github.com/gorilla/handlers"

	"github.com/syaiful6/sersan"
	"github.com/syaiful6/thatique/context"
	"github.com/syaiful6/thatique/pkg/httputil"
	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/auth/passwords"
	"github.com/syaiful6/thatique/shop/auth/passwords/buntdbtoken"
)

const (
	userPasswordPath        = "/users/passwords"
	internalSetToken        = "_set-password"
	internalSetTokenSession = "_password_token_"
)

type passwordsDispatcher struct {
	broker  *passwords.PasswordBroker
	limiter *RateLimiter
	repo    auth.UserRepository
}

type passwordsHTTPHandler struct {
	broker  *passwords.PasswordBroker
	limiter *RateLimiter
}

func NewPasswordHTTPDispatcher(app *App, repo auth.UserRepository) (*passwordsDispatcher, error) {
	if err := os.MkdirAll(path.Join(app.Config.DataPath, userPasswordPath), 0777); err != nil {
		return nil, err
	}

	gen, err := buntdbtoken.NewFileTokenGenerator(path.Join(app.Config.DataPath, userPasswordPath, "token.buntdb"))
	if err != nil {
		return nil, err
	}

	var (
		sender string
		config = app.Config.Mail
	)
	if config.From != "" {
		sender = config.From
	}
	notifier := passwords.NewMailNotifier(sender, app.mailTransport, app.jobQueue)

	broker := passwords.NewPasswordBroker(gen, notifier)
	broker.ResetURL = fmt.Sprintf("%s/auth/passwords", app.Config.HTTP.Host)
	broker.Finder = repo

	return &passwordsDispatcher{
		broker:  broker,
		repo:    repo,
		limiter: NewRateLimiter(10, 2, httputil.GetSourceIP),
	}, nil
}

type resetLinkHandler struct {
	*Context
	broker  *passwords.PasswordBroker
	limiter *RateLimiter
}

func (p *passwordsDispatcher) dispatchResetLink(ctx *Context, r *http.Request) http.Handler {
	hd := &resetLinkHandler{
		Context: ctx,
		broker:  p.broker,
		limiter: p.limiter,
	}

	return handlers.MethodHandler{
		"GET": http.HandlerFunc(hd.renderSendResetLink),
		"POST": http.HandlerFunc(hd.sendResetLink),
	}
}

func (sg *resetLinkHandler) renderForm(w http.ResponseWriter, data map[string]interface{}) error {
	tpl, err := sg.App.Template("auth/password_reset_form", "base.html", "auth/password_reset_form.html")
	if err != nil {
		return err
	}
	if err = tpl.Execute(w, data); err != nil {
		context.GetLogger(sg).Debugf("unexpected error when executing auth/password_reset_form.html: %v", err)
		return err
	}

	return nil
}

func (h *resetLinkHandler) renderSendResetLink(w http.ResponseWriter, r *http.Request) {
	if err := h.renderForm(w, map[string]interface{}{
		"Title":          "Reset Password",
		"Description":    "Reset your Password",
		"Email":          "",
		csrf.TemplateTag: csrf.TemplateField(r),
	}); err != nil {
		h.App.handleErrorHTML(w, err)
	}
}

func (h *resetLinkHandler) sendResetLink(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		context.GetLogger(h).Debugf("error encountered when parsing form: %v", err)
		// this is likely parsing error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	limiter := h.limiter.Get(r)
	if limiter.Allow() == false {
		w.WriteHeader(http.StatusTooManyRequests)
		if err = h.renderForm(w, map[string]interface{}{
			"Title":          "Reset Password",
			"Description":    "Reset your Password",
			"Errors":         []string{http.StatusText(429)},
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			h.App.handleErrorHTML(w, err)
		}
		return
	}

	err = h.broker.SendResetLink(httputil.GetSourceIP(r), r.FormValue("email"))
	if err != nil {
		if err = h.renderForm(w, map[string]interface{}{
			"Title":          "Reset Password",
			"Description":    "Reset your Password",
			"Email":          r.FormValue("email"),
			"Errors":         []string{err.Error()},
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			h.App.handleErrorHTML(w, err)
		}
		return
	}

	if err = h.renderForm(w, map[string]interface{}{
		"Title":          "Reset Password",
		"Description":    "Reset your Password",
		"Email":          "",
		"Errors":         []string{"Password reset sudah terkirim ke inbox anda."},
		csrf.TemplateTag: csrf.TemplateField(r),
	}); err != nil {
		h.App.handleErrorHTML(w, err)
	}
	return
}

type resetPasswordHandler struct {
	*Context
	repo    auth.UserRepository
	req     *passwords.ResetRequest
	broker  *passwords.PasswordBroker
}

func (p *passwordsDispatcher) dispatchResetPassword(ctx *Context, r *http.Request) http.Handler {
	vars := mux.Vars(r)

	var (
		uid   = vars["uid"]
		token = vars["token"]
	)

	sess, err := sersan.GetSession(r)
	if err != nil {
		panic(err)
	}
	if token == internalSetToken {
		if v, ok := sess[internalSetTokenSession]; ok {
			token = v.(string)
		}

		req, ok := p.broker.ValidateReset(vars["uid"], token)
		if !ok {
			// it's not valid request
			return ctx.App.templateHandler(map[string]interface{}{
				"Title":          "403 Forbidden",
				"Description":    "Invalid URL",
			}, "403", "base.html", "403.html")
		}

		hd := &resetPasswordHandler{Context: ctx, repo: p.repo, req: req, broker: p.broker,}
		return handlers.MethodHandler{
			"GET": http.HandlerFunc(hd.renderChangePassword),
			"POST": http.HandlerFunc(hd.changePassword),
		}
	}

	sess[internalSetTokenSession] = token
	url := fmt.Sprintf("/auth/passwords/%s/%s", uid, internalSetToken)
	return http.RedirectHandler(url, http.StatusFound)
}

func (sg *resetPasswordHandler) renderForm(w http.ResponseWriter, data map[string]interface{}) error {
	tpl, err := sg.App.Template("auth/password_change_form", "base.html", "auth/password_change_form.html")
	if err != nil {
		return err
	}
	if err = tpl.Execute(w, data); err != nil {
		context.GetLogger(sg).Debugf("unexpected error when executing auth/password_change_form.html: %v", err)
		return err
	}

	return nil
}

func (p *resetPasswordHandler) renderChangePassword(w http.ResponseWriter, r *http.Request) {
	if err := p.renderForm(w, map[string]interface{}{
		"Title":          "Change Password",
		"Description":    "Change your Password",
		csrf.TemplateTag: csrf.TemplateField(r),
	}); err != nil {
		p.App.handleErrorHTML(w, err)
	}
}

func (p *resetPasswordHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		context.GetLogger(p).Debugf("error encountered when parsing form: %v", err)
		// this is likely parsing error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	p.req.Password1 = r.FormValue("password")
	p.req.Password2 = r.FormValue("confirm-password")

	errCode := p.broker.Resets(p.req, func(user *auth.User, pswd string) error {
		err = user.SetPassword([]byte(pswd))
		if err != nil {
			return err
		}

		return p.repo.Save(user)
	})
	if errCode != passwords.NoError {
		if err = p.renderForm(w, map[string]interface{}{
			"Title":          "Change Password",
			"Description":    "Change your Password",
			"Errors":         []string{errCode.ErrorDescription()},
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			p.App.handleErrorHTML(w, err)
		}
		return
	}

	sess, err := sersan.GetSession(r)
	if err != nil {
		p.App.handleErrorHTML(w, err)
		return
	}

	delete(sess, internalSetTokenSession)
	r, err = p.App.Auth.Login(p.req.GetUser(), r)
	if err != nil {
		p.App.handleErrorHTML(w, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
