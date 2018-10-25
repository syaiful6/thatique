package configuration

import (
	"bytes"
	"net/http"
	"os"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)


// Hook up gocheck into the "go test" runner
func Test(t *testing.T) { TestingT(t) }

var configStruct = Configuration{
	Version: "0.1",
	Log: struct {
		AccessLog struct {
			Disabled bool `yaml:"disabled,omitempty"`
		} `yaml:"accesslog,omitempty"`
		Level     Loglevel               `yaml:"level,omitempty"`
		Formatter string                 `yaml:"formatter,omitempty"`
		Fields    map[string]interface{} `yaml:"fields,omitempty"`
		Hooks     []LogHook              `yaml:"hooks,omitempty"`
	}{
		Level:  "info",
		Fields: map[string]interface{}{"environment": "test"},
	},

	HTTP: struct {
		Addr string `yaml:"addr,omitempty"`
		Net string `yaml:"net,omitempty"`
		Host string `yaml:"host,omitempty"`
		Prefix string `yaml:"prefix,omitempty"`
		Secret string `yaml:"secret,omitempty"`
		DrainTimeout time.Duration `yaml:"draintimeout,omitempty"`
		Headers http.Header `yaml:"headers,omitempty"`
	}{
		Addr: "localhost",
		Headers: http.Header{
			"X-Content-Type-Options": []string{"nosniff"},
		},
	},

	Redis: struct {
		Addr string `yaml:"addr,omitempty"`
		Password string `yaml:"password,omitempty"`
		DB int `yaml:"db,omitempty"`
		DialTimeout  time.Duration `yaml:"dialtimeout,omitempty"`
		ReadTimeout  time.Duration `yaml:"readtimeout,omitempty"`
		WriteTimeout time.Duration `yaml:"writetimeout,omitempty"`
		MaxIdle int `yaml:"maxidle,omitempty"`
		MaxActive int `yaml:"maxactive,omitempty"`
		IdleTimeout time.Duration `yaml:"idletimeout,omitempty"`
	}{
		Addr: "localhost",
		Password: "secret",
		DB: 1,
	},

	MongoDB: struct {
		URI string `yaml:"uri,omitempty"`
		Name string `yaml:"name,omitempty"`
	}{
		URI: "mongodb://localhost:2701",
	},
}

// configYamlV0_1 is a Version 0.1 yaml document representing configStruct
var configYamlV0_1 = `
version: 0.1
log:
  level: info
  fields:
    environment: test
http:
  addr: localhost
  headers:
    X-Content-Type-Options: [nosniff]
redis:
  addr: localhost
  password: secret
  db: 1
mongodb:
  uri: "mongodb://localhost:2701"
`

type ConfigSuite struct {
	expectedConfig *Configuration
}

var _ = Suite(new(ConfigSuite))

func (suite *ConfigSuite) SetUpTest(c *C) {
	os.Clearenv()
	suite.expectedConfig = copyConfig(configStruct)
}

// TestMarshalRoundtrip validates that configStruct can be marshaled and
// unmarshaled without changing any parameters
func (suite *ConfigSuite) TestMarshalRoundtrip(c *C) {
	configBytes, err := yaml.Marshal(suite.expectedConfig)
	c.Assert(err, IsNil)
	config, err := Parse(bytes.NewReader(configBytes))
	c.Log(string(configBytes))
	c.Assert(err, IsNil)
	c.Assert(config, DeepEquals, suite.expectedConfig)
}

// TestParseSimple validates that configYamlV0_1 can be parsed into a struct
// matching configStruct
func (suite *ConfigSuite) TestParseSimple(c *C) {
	config, err := Parse(bytes.NewReader([]byte(configYamlV0_1)))
	c.Assert(err, IsNil)
	c.Assert(config, DeepEquals, suite.expectedConfig)
}

func copyConfig(config Configuration) *Configuration {
	configCopy := new(Configuration)

	configCopy.Version = MajorMinorVersion(config.Version.Major(), config.Version.Minor())

	configCopy.Log = config.Log
	configCopy.Log.Fields = make(map[string]interface{}, len(config.Log.Fields))
	for k, v := range config.Log.Fields {
		configCopy.Log.Fields[k] = v
	}

	configCopy.HTTP = config.HTTP
	configCopy.HTTP.Headers = make(http.Header)
	for k, v := range config.HTTP.Headers {
		configCopy.HTTP.Headers[k] = v
	}

	configCopy.Redis = config.Redis
	configCopy.MongoDB = config.MongoDB

	return configCopy
}