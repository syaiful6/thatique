package handlers

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/syaiful6/sersan"
	redistore "github.com/syaiful6/sersan/redis"

	"github.com/syaiful6/thatique/configuration"
	scontext "github.com/syaiful6/thatique/context"
	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/db"
	"github.com/syaiful6/thatique/shop/middlewares"
	tredis "github.com/syaiful6/thatique/shop/redis"
)

// randomSecretSize is the number of random bytes to generate if no secret
// was specified.
const randomSecretSize = 32

// defaultCheckInterval is the default time in between health checks
const defaultCheckInterval = 10 * time.Second

// App is a global thatiq application object. Shared resources can be placed
// on this object that will be accessible from all requests. Any writable
// fields should be protected.
type App struct {
	context.Context
	*renderer

	Config        *configuration.Configuration
	asset         func(string) ([]byte, error)
	authenticator *auth.Authenticator
	router        *mux.Router
	redis         *redis.Pool
	mongo         *db.MongoConn
}

func NewApp(ctx context.Context, asset func(string) ([]byte, error), config *configuration.Configuration) (*App, error) {
	redisPool, err := tredis.NewRedisPool(config.Redis)
	if err != nil {
		return nil, err
	}

	// connect to mongodb
	mongodb, err := db.Dial(config.MongoDB.URI, config.MongoDB.Name)
	if err != nil {
		return nil, err
	}

	sersanstore, err := redistore.NewRediStore(redisPool)
	if err != nil {
		return nil, err
	}
	sessionstate := sersan.NewServerSessionState(sersanstore,
		createSecretKeys(config.HTTP.SessionKeys...)...)
	sessionstate.AuthKey = auth.UserSessionKey
	sessionstate.Options.Secure = config.HTTP.Secure

	authenticator := auth.NewAuthenticator(auth.NewMgoUserProvider(mongodb))
	app := &App{
		renderer:      newTemplateRenderer(asset),
		Config:        config,
		Context:       ctx,
		asset:         asset,
		router:        RouterWithPrefix(config.HTTP.Prefix),
		redis:         redisPool,
		mongo:         mongodb,
		authenticator: authenticator,
	}

	app.configureSecret(config)

	tpl404, err := app.Template("404", "base.html", "404.html")
	if err != nil {
		return nil, err
	}
	app.router.NotFoundHandler = Handler404(tpl404)

	webMiddlewares := &middlewares.IfRequestMiddleware{
		Predicate: isNotApiRoute,
		Middlewares: []mux.MiddlewareFunc{
			sersan.SessionMiddleware(sessionstate),
			authenticator.Middleware,
			csrf.Protect([]byte(config.HTTP.Secret), csrf.Secure(config.HTTP.Secure)),
		},
	}
	app.router.Use(webMiddlewares.Middleware)

	// Register the handler dispatchers.
	app.router.Handle("/", app.dispatchFunc(homepageDispatcher)).Name("home")

	// auth
	authRouter := app.router.PathPrefix("/auth").Subrouter()
	authRouter.Handle("", app.dispatch(NewSigninLimitDispatcher(5, 3))).Name("auth.signin")

	app.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(&StaticFs{asset: asset, prefix: "assets/static"})))

	return app, err
}

func RouterWithPrefix(prefix string) *mux.Router {
	rootRouter := mux.NewRouter()
	router := rootRouter
	if prefix != "" {
		router = router.PathPrefix(prefix).Subrouter()
	}

	router.StrictSlash(true)

	return rootRouter
}

// configureSecret creates a random secret if a secret wasn't included in the
// configuration.
func (app *App) configureSecret(configuration *configuration.Configuration) {
	if configuration.HTTP.Secret == "" {
		var secretBytes [randomSecretSize]byte
		if _, err := cryptorand.Read(secretBytes[:]); err != nil {
			panic(fmt.Sprintf("could not generate random bytes for HTTP secret: %v", err))
		}
		configuration.HTTP.Secret = string(secretBytes[:])
		scontext.GetLogger(app).Warn("No HTTP secret provided - generated random secret.")
	}
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close() // ensure that request body is always closed.

	// Prepare the context with our own little decorations.
	ctx := r.Context()
	ctx = scontext.WithRequest(ctx, r)
	ctx, w = scontext.WithResponseWriter(ctx, w)
	ctx = scontext.WithLogger(ctx, scontext.GetRequestLogger(ctx))
	r = r.WithContext(ctx)

	defer func() {
		status, ok := ctx.Value("http.response.status").(int)
		if ok && status >= 200 && status <= 399 {
			scontext.GetResponseLogger(r.Context()).Infof("response completed")
		}
	}()

	app.router.ServeHTTP(w, r)
}

// Disptcher takes a context and request and returns a constructed handler
// for the route. The dispatcher will use this to dynamically create request
// specific handlers for each endpoint without creating a new router for each
// request.
type Dispatcher interface {
	DispatchHTTP(ctx *Context, r *http.Request) http.Handler
}

type DispatcherFunc func(ctx *Context, r *http.Request) http.Handler

func (d DispatcherFunc) DispatchHTTP(ctx *Context, r *http.Request) http.Handler {
	return d(ctx, r)
}

func (app *App) dispatchFunc(dispatch DispatcherFunc) http.Handler {
	return app.dispatch(dispatch)
}

// dispatch returns a handler that constructs a request specific context and
// handler, using the dispatch factory function.
func (app *App) dispatch(dispatch Dispatcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for headerName, headerValues := range app.Config.HTTP.Headers {
			for _, value := range headerValues {
				w.Header().Add(headerName, value)
			}
		}

		context := app.context(w, r)

		// sync up context on the request.
		r = r.WithContext(context)
		dispatch.DispatchHTTP(context, r).ServeHTTP(w, r)
	})
}

// context constructs the context object for the application. This only be
// called once per request.
func (app *App) context(w http.ResponseWriter, r *http.Request) *Context {
	ctx := r.Context()
	ctx = scontext.WithVars(ctx, r)
	ctx = scontext.WithLogger(ctx, scontext.GetLogger(ctx,
		"vars.name",
		"vars.uuid"))

	return &Context{
		App:     app,
		Context: ctx,
	}
}

func (app *App) handleErrorHTML(w http.ResponseWriter, err error) {
	var (
		tpl  *template.Template
		err2 error
	)
	if err == mgo.ErrNotFound {
		tpl, err2 = app.Template("404", "base.html", "404.html")
		if err2 == nil {
			w.WriteHeader(http.StatusNotFound)
			if err2 = tpl.Execute(w, map[string]interface{}{
				"Title":       "404: Page Not Found",
				"Description": "This is not the web page you are looking for",
			}); err2 == nil {
				return
			}
		}
		err = err2
	}

	if err == auth.ErrTokenMismatch || err == auth.ErrNoToken || err == auth.InvalidToken {
		tpl, err2 = app.Template("403", "base.html", "403.html")
		if err2 == nil {
			w.WriteHeader(http.StatusForbidden)
			if err2 = tpl.Execute(w, map[string]interface{}{
				"Title":       "403: Forbidden",
				"Description": err.Error(),
			}); err2 == nil {
				return
			}
		}
		err = err2
	}

	// otherwise just render 500
	var b []byte
	if b, err2 = app.asset("assets/templates/50x.html"); err2 != nil {
		panic(err2)
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write(b)
}

func createSecretKeys(keyPairs ...string) [][]byte {
	xs := make([][]byte, len(keyPairs))
	var (
		err error
		key []byte
	)
	for _, s := range keyPairs {
		if strings.HasPrefix(s, "base64:") {
			key, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(s, "base64:"))
			if err != nil {
				continue
			}
			xs = append(xs, key)
		} else {
			xs = append(xs, []byte(s))
		}
	}
	return xs
}

func isNotApiRoute(r *http.Request) bool {
	return !strings.HasPrefix(r.URL.Path, "/api")
}
