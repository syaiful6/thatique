package handlers

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"

	"github.com/syaiful6/thatique/configuration"
	scontext "github.com/syaiful6/thatique/context"
	"github.com/syaiful6/thatique/shop/data"
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

	Config *configuration.Configuration

	router *mux.Router
	asset func(string) ([]byte, error)
	redis *redis.Pool
	mongo *data.MongoConn
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

	app := &App{
		Config:  config,
		Context: ctx,
		asset:   asset,
		router:  RouterWithPrefix(config.HTTP.Prefix),
		redis:   redisPool,
		mongo:   mongodb,
	}

	// Register the handler dispatchers.
	app.handle("/", func(ctx *Context, r *http.Request) http.Handler {
		return http.HandlerFunc(homeHandlerFunc)
	}).Name("home")

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
		scontext.GetLogger(app).Warn("No HTTP secret provided - generated random secret. This may cause problems with uploads if multiple registries are behind a load-balancer. To provide a shared secret, fill in http.secret in the configuration file or set the THATIQ_HTTP_SECRET environment variable.")
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

func homeHandlerFunc(w http.ResponseWriter, r *http.Request) {
	const emptyJSON = "{}"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(emptyJSON)))

	fmt.Fprint(w, emptyJSON)
}
