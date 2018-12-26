package handlers

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"gopkg.in/boj/redistore.v1"

	"github.com/syaiful6/thatique/configuration"
	scontext "github.com/syaiful6/thatique/context"
	"github.com/syaiful6/thatique/shop/auth"
	"github.com/syaiful6/thatique/shop/data"
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

	Config        *configuration.Configuration
	asset         func(string) ([]byte, error)
	authenticator *auth.Authenticator
	router        *mux.Router
	redis         *redis.Pool
	mongo         *data.MongoConn
	sessionStore  sessions.Store
}

func NewApp(ctx context.Context, asset func(string) ([]byte, error), config *configuration.Configuration) (*App, error) {
	redisPool, err := tredis.NewRedisPool(config.Redis)
	if err != nil {
		return nil, err
	}

	// connect to mongodb
	mongodb, err := data.Dial(config.MongoDB.URI, config.MongoDB.Name)
	if err != nil {
		return nil, err
	}

	redisStore, err := redistore.NewRediStoreWithPool(redisPool,
		createSecretKeys(config.HTTP.SessionKeys...)...)
	if err != nil {
		return nil, err
	}

	authenticator := auth.NewAuthenticator(redisStore)
	app := &App{
		Config:        config,
		Context:       ctx,
		asset:         asset,
		router:        RouterWithPrefix(config.HTTP.Prefix),
		redis:         redisPool,
		mongo:         mongodb,
		sessionStore:  redisStore,
		authenticator: authenticator,
	}

	authWeb := &middlewares.IfRequestMiddleware{
		Inner: authenticator.Middleware,
		Predicate: func(r *http.Request) bool {
			return !strings.HasPrefix(r.URL.Path, "/api")
		},
	}
	app.router.Use(authWeb.Middleware)
	// Register the handler dispatchers.
	app.handle("/", homepageDispatcher).Name("home")

	app.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(&StaticFs{asset: asset, prefix: "assets/static"})))

	app.configureSecret(config)

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

func (app *App) handle(path string, dispatch DispatchFunc) *mux.Route {
	return app.router.Handle(path, app.dispatcher(dispatch))
}

// dispatchFunc takes a context and request and returns a constructed handler
// for the route. The dispatcher will use this to dynamically create request
// specific handlers for each endpoint without creating a new router for each
// request.
type DispatchFunc func(ctx *Context, r *http.Request) http.Handler

// dispatcher returns a handler that constructs a request specific context and
// handler, using the dispatch factory function.
func (app *App) dispatcher(dispatch DispatchFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for headerName, headerValues := range app.Config.HTTP.Headers {
			for _, value := range headerValues {
				w.Header().Add(headerName, value)
			}
		}

		context := app.context(w, r)

		// sync up context on the request.
		r = r.WithContext(context)
		dispatch(context, r).ServeHTTP(w, r)
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

func (app *App) parseTemplate(tpl *template.Template, name string) (*template.Template, error) {
	assetPath := path.Join("assets/templates", filepath.FromSlash(path.Clean("/"+name)))
	if len(assetPath) > 0 && assetPath[0] == '/' {
		assetPath = assetPath[1:]
	}

	var b []byte
	var err error
	if b, err = app.asset(assetPath); err != nil {
		return nil, err
	}

	return tpl.Parse(string(b))
}

func (app *App) template(name string, base string, tpls ...string) (tpl *template.Template, err error) {
	tpl = template.New(name)

	if tpl, err = app.parseTemplate(tpl, base); err != nil {
		return nil, err
	}

	for _, tn := range tpls {
		if tpl, err = app.parseTemplate(tpl, tn); err != nil {
			return nil, err
		}
	}

	return
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
