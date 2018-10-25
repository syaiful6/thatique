package configuration

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type Configuration struct {
	Version Version `yaml:"version"`

	// Log supports setting various parameters related to the logging
	// subsystem.
	Log struct {
		// AccessLog configures access logging.
		AccessLog struct {
			// Disabled disables access logging.
			Disabled bool `yaml:"disabled,omitempty"`
		} `yaml:"accesslog,omitempty"`

		// Level is the granularity at which registry operations are logged.
		Level Loglevel `yaml:"level,omitempty"`

		// Formatter overrides the default formatter with another. Options
		// include "text", "json" and "logstash".
		Formatter string `yaml:"formatter,omitempty"`

		// Fields allows users to specify static string fields to include in
		// the logger context.
		Fields map[string]interface{} `yaml:"fields,omitempty"`

		// Hooks allows users to configure the log hooks, to enabling the
		// sequent handling behavior, when defined levels of log message emit.
		Hooks []LogHook `yaml:"hooks,omitempty"`
	}

	Mail Mail `yaml:"mail,omitempty"`

	HTTP struct {
		// Addr specifies the bind address
		Addr string `yaml:"addr,omitempty"`

		// Net specifies the net portion of the bind address. A default empty value means tcp.
		Net string `yaml:"net,omitempty"`

		// Host specifies an externally-reachable address for the app, as a fully
		// qualified URL.
		Host string `yaml:"host,omitempty"`

		Prefix string `yaml:"prefix,omitempty"`

		// Secret specifies the secret key which HMAC tokens are created with.
		Secret string `yaml:"secret,omitempty"`

		// Amount of time to wait for connection to drain before shutting down when registry
		// receives a stop signal
		DrainTimeout time.Duration `yaml:"draintimeout,omitempty"`

		// Headers is a set of headers to include in HTTP responses. A common
		// use case for this would be security headers such as
		// Strict-Transport-Security. The map keys are the header names, and
		// the values are the associated header payloads.
		Headers http.Header `yaml:"headers,omitempty"`
	} `yaml:"http,omitempty"`

	Redis struct {
		// Addr specifies the the redis instance available to the application
		Addr string `yaml:"addr,omitempty"`

		Password string `yaml:"password,omitempty"`

		DB int `yaml:"db,omitempty"`

		DialTimeout  time.Duration `yaml:"dialtimeout,omitempty"`  // timeout for connect
		ReadTimeout  time.Duration `yaml:"readtimeout,omitempty"`  // timeout for reads of data
		WriteTimeout time.Duration `yaml:"writetimeout,omitempty"` // timeout for writes of data

		// we always use redis Poop
		// MaxIdle sets the maximum number of idle connections.
		MaxIdle int `yaml:"maxidle,omitempty"`
		// MaxActive sets the maximum number of connections that should be
		// opened before blocking a connection request.
		MaxActive int `yaml:"maxactive,omitempty"`

		// IdleTimeout sets the amount time to wait before closing
		// inactive connections.
		IdleTimeout time.Duration `yaml:"idletimeout,omitempty"`
	} `yaml:"redis,omitempty"`

	MongoDB struct {
		URI  string `yaml:"uri,omitempty"`
		Name string `yaml:"name,omitempty"`
	} `yaml:"mongodb"`
}

// LogHook is composed of hook Level and Type.
// After hooks configuration, it can execute the next handling automatically,
// when defined levels of log message emitted.
// Example: hook can sending an email notification when error log happens in app.
type LogHook struct {
	// Disable lets user select to enable hook or not.
	Disabled bool `yaml:"disabled,omitempty"`

	// Type allows user to select which type of hook handler they want.
	Type string `yaml:"type,omitempty"`

	// Levels set which levels of log message will let hook executed.
	Levels []string `yaml:"levels,omitempty"`

	// Mail allows user to configure email parameters.
	Mail Mail `yaml:"options,omitempty"`
}

type Mail struct {
	SMTP struct {
		// Addr defines smtp host address
		Addr string `yaml:addr,omitempty`

		// Username defines user name to smtp host
		Username string `yaml:"username,omitempty"`

		// Password defines password of login user
		Password string `yaml:"password,omitempty"`

		// Insecure defines if smtp login skips the secure certification.
		Insecure bool `yaml:"insecure,omitempty"`
	}

	From string `yaml:from,omitempty`

	// To defines mail receiving address
	To []string `yaml:"to,omitempty"`
}

// v0_1Configuration is a Version 0.1 Configuration struct
// This is currently aliased to Configuration, as it is the current version
type v0_1Configuration Configuration

// UnmarshalYAML implements the yaml.Unmarshaler interface
// Unmarshals a string of the form X.Y into a Version, validating that X and Y can represent unsigned integers
func (version *Version) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var versionString string
	err := unmarshal(&versionString)
	if err != nil {
		return err
	}

	newVersion := Version(versionString)
	if _, err := newVersion.major(); err != nil {
		return err
	}

	if _, err := newVersion.minor(); err != nil {
		return err
	}

	*version = newVersion
	return nil
}

// CurrentVersion is the most recent Version that can be parsed
var CurrentVersion = MajorMinorVersion(0, 1)

type Loglevel string

// UnmarshalYAML implements the yaml.Umarshaler interface
// Unmarshals a string into a Loglevel, lowercasing the string and validating that it represents a
// valid loglevel
func (loglevel *Loglevel) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var loglevelString string
	err := unmarshal(&loglevelString)
	if err != nil {
		return err
	}

	loglevelString = strings.ToLower(loglevelString)
	switch loglevelString {
	case "error", "warn", "info", "debug":
	default:
		return fmt.Errorf("Invalid loglevel %s Must be one of [error, warn, info, debug]", loglevelString)
	}

	*loglevel = Loglevel(loglevelString)
	return nil
}

// Parameters defines a key-value parameters mapping
type Parameters map[string]interface{}

// Parse parses an input configuration yaml document into a Configuration struct
// This should generally be capable of handling old configuration format versions
//
// Environment variables may be used to override configuration parameters other than version,
// following the scheme below:
// Configuration.Abc may be replaced by the value of THATIQ_ABC,
// Configuration.Abc.Xyz may be replaced by the value of THATIQ_ABC_XYZ, and so forth
func Parse(rd io.Reader) (*Configuration, error) {
	in, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	p := NewParser("thatiq", []VersionedParseInfo{
		{
			Version: MajorMinorVersion(0, 1),
			ParseAs: reflect.TypeOf(v0_1Configuration{}),
			ConversionFunc: func(c interface{}) (interface{}, error) {
				if v0_1, ok := c.(*v0_1Configuration); ok {
					if v0_1.Log.Level == Loglevel("") {
						v0_1.Log.Level = Loglevel("info")
					}

					return (*Configuration)(v0_1), nil
				}
				return nil, fmt.Errorf("Expected *v0_1Configuration, received %#v", c)
			},
		},
	})

	config := new(Configuration)
	err = p.Parse(in, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
