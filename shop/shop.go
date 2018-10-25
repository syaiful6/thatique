package shop

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/syaiful6/thatique/configuration"
	scontext "github.com/syaiful6/thatique/context"
	sredis "github.com/syaiful6/thatique/redis"
	"github.com/syaiful6/thatique/shop/listener"
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
	Redis  *redis.Pool
}

func NewShop(ctx context.Context, config *configuration.Configuration) (*Shop, error) {
	redis, err := dialRedis(config)
	if err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	router.HandleFunc("/", HomeHandler)

	server := &http.Server{
		Handler: router,
	}

	return &Shop{
		config: config,
		server: server,
		Redis:  redis,
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
		c, cancel := context.WithTimeout(context.Background(), config.HTTP.DrainTimeout)
		defer cancel()
		return shop.server.Shutdown(c)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Path
	message = strings.TrimPrefix(message, "/")
	message = "Hello " + message
	w.Write([]byte(message))
}

func dialRedis(config *configuration.Configuration) (*redis.Pool, error) {
	var pool *redis.Pool

	size, network, address, db, password := 10, "tcp", ":6379", 0, ""

	if config.Redis.MaxIdle != 0 {
		size = config.Redis.MaxIdle
	}
	if config.Redis.Addr != "" {
		address = config.Redis.Addr
	}
	if config.Redis.DB != 0 {
		db = config.Redis.DB
	}
	if config.Redis.Password != "" {
		password = config.Redis.Password
	}
	pool, err := sredis.NewRedisPool(size, network, address, password, db)

	if err != nil {
		fmt.Fprintf(os.Stderr, "connection to Redis server failed: %v\n", err)
	}

	return pool, err
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
