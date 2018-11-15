package shop

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	logstash "github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/bugsnag/bugsnag-go"
	gorhandlers "github.com/gorilla/handlers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/syaiful6/thatique/assets"
	"github.com/syaiful6/thatique/configuration"
	scontext "github.com/syaiful6/thatique/context"
	"github.com/syaiful6/thatique/shop/handlers"
	"github.com/syaiful6/thatique/shop/listener"
	"github.com/syaiful6/thatique/uuid"
	"github.com/syaiful6/thatique/version"
)

// this channel gets notified when process receives signal. It is global to ease unit testing
var quit = make(chan os.Signal, 1)

// ServeCmd is a cobra command for running the registry.
var ServeCmd = &cobra.Command{
	Use:   "serve <config>",
	Short: "`serve` the application",
	Long:  "`serve` run the shop http server and start sell",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := scontext.WithVersion(scontext.Background(), version.Version)

		config, err := resolveConfiguration(args)

		if err != nil {
			fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
			cmd.Usage()
			os.Exit(1)
		}

		shop, err := NewShop(ctx, config)
		if err != nil {
			log.Fatalln(err)
		}

		if err = shop.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	},
}

type Shop struct {
	config *configuration.Configuration
	server *http.Server
	app    *handlers.App
}

func NewShop(ctx context.Context, config *configuration.Configuration) (*Shop, error) {
	ctx, err := configureLogging(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error configuring logger: %v", err)
	}

	// inject a logger into the uuid library. warns us if there is a problem
	// with uuid generation under low entropy.
	uuid.Loggerf = scontext.GetLogger(ctx).Warnf

	app, err := handlers.NewApp(ctx, assets.Asset, config)
	if err != nil {
		return nil, fmt.Errorf("error creting handlers app: %v", err)
	}

	handler := panicHandler(configureReporting(app))

	if !config.Log.AccessLog.Disabled {
		handler = gorhandlers.CombinedLoggingHandler(os.Stdout, handler)
	}

	server := &http.Server{
		Handler: handler,
	}

	return &Shop{
		app:    app,
		config: config,
		server: server,
	}, nil
}

// ListenAndServe runs the shope's HTTP server.
func (shop *Shop) ListenAndServe() error {
	config := shop.config

	ln, err := listener.NewListener(config.HTTP.Net, config.HTTP.Addr)
	if err != nil {
		return err
	}

	// setup channel to get notified on SIGTERM signal
	signal.Notify(quit, syscall.SIGTERM)
	serveErr := make(chan error)

	// Start serving in goroutine and listen for stop signal in main thread
	go func() {
		serveErr <- shop.server.Serve(ln)
	}()

	select {
	case err := <-serveErr:
		return err

	case <-quit:
		// shutdown the server with a grace period of configured timeout
		scontext.GetLogger(shop.app).Info("stopping server gracefully. Draining connections for ", config.HTTP.DrainTimeout)
		c, cancel := context.WithTimeout(context.Background(), config.HTTP.DrainTimeout)
		defer cancel()
		return shop.server.Shutdown(c)
	}
}

// panicHandler add an HTTP handler to web app. The handler recover the happening
// panic. logrus.Panic transmits panic message to pre-config log hooks, which is
// defined in config.yml.
func panicHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Panic(fmt.Sprintf("%v", err))
			}
		}()
		handler.ServeHTTP(w, r)
	})
}

func configureReporting(app *handlers.App) http.Handler {
	var handler http.Handler = app

	if app.Config.Reporting.Bugsnag.APIKey != "" {
		bugsnagConfig := bugsnag.Configuration{
			APIKey: app.Config.Reporting.Bugsnag.APIKey,
		}
		ver := scontext.GetVersion(app.Context)
		if ver != "" {
			bugsnagConfig.AppVersion = ver
		}
		if app.Config.Reporting.Bugsnag.ReleaseStage != "" {
			bugsnagConfig.ReleaseStage = app.Config.Reporting.Bugsnag.ReleaseStage
		}
		if app.Config.Reporting.Bugsnag.Endpoint != "" {
			bugsnagConfig.Endpoint = app.Config.Reporting.Bugsnag.Endpoint
		}
		bugsnag.Configure(bugsnagConfig)

		handler = bugsnag.Handler(handler)
	}

	return handler
}

// configureLogging prepares the context with a logger using the
// configuration.
func configureLogging(ctx context.Context, config *configuration.Configuration) (context.Context, error) {
	log.SetLevel(logLevel(config.Log.Level))

	formatter := config.Log.Formatter
	if formatter == "" {
		formatter = "text" // default formatter
	}

	switch formatter {
	case "json":
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "text":
		log.SetFormatter(&log.TextFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "logstash":
		log.SetFormatter(&logstash.LogstashFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		// just let the library use default on empty string.
		if config.Log.Formatter != "" {
			return ctx, fmt.Errorf("unsupported logging formatter: %q", config.Log.Formatter)
		}
	}

	if config.Log.Formatter != "" {
		log.Debugf("using %q logging formatter", config.Log.Formatter)
	}

	if len(config.Log.Fields) > 0 {
		// build up the static fields, if present.
		var fields []interface{}
		for k := range config.Log.Fields {
			fields = append(fields, k)
		}

		ctx = scontext.WithValues(ctx, config.Log.Fields)
		ctx = scontext.WithLogger(ctx, scontext.GetLogger(ctx, fields...))
	}

	return ctx, nil
}

func logLevel(level configuration.Loglevel) log.Level {
	l, err := log.ParseLevel(string(level))
	if err != nil {
		l = log.InfoLevel
		log.Warnf("error parsing level %q: %v, using %q	", level, err, l)
	}

	return l
}

func resolveConfiguration(args []string) (*configuration.Configuration, error) {
	var configurationPath string

	if len(args) > 0 {
		configurationPath = args[0]
	} else if os.Getenv("THATIQ_CONFIGURATION_PATH") != "" {
		configurationPath = os.Getenv("THATIQ_CONFIGURATION_PATH")
	}

	if configurationPath == "" {
		return nil, fmt.Errorf("configuration path unspecified")
	}

	fp, err := os.Open(configurationPath)
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	config, err := configuration.Parse(fp)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", configurationPath, err)
	}

	return config, nil
}
