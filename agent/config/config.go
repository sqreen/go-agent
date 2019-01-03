// Agent configuration package.

// This package includes both compile-time and run-time configuration of the
// agent. Variables are made configurable at run-time when necessary for users.

package config

import (
	"net/http"
	"time"

	"github.com/spf13/viper"
	"github.com/sqreen/AgentGo/agent/plog"
)

type HTTPAPIEndpoint struct {
	Method, URL string
}

const (
	// Default value of network timeouts.
	DefaultNetworkTimeout = 5 * time.Second
)

// Backend client configuration.
var (
	// Timeout value of a HTTP request. See http.Client.Timeout.
	BackendHTTPAPIRequestTimeout = DefaultNetworkTimeout

	// List of endpoint addresses, relative to the base URL.
	BackendHTTPAPIEndpoint = struct {
		AppLogin, AppLogout, AppBeat HTTPAPIEndpoint
	}{
		AppLogin:  HTTPAPIEndpoint{http.MethodPost, "/v1/app-login"},
		AppLogout: HTTPAPIEndpoint{http.MethodGet, "/v0/app-logout"},
		AppBeat:   HTTPAPIEndpoint{http.MethodPost, "/v1/app-beat"},
	}

	// Header name of the API token.
	BackendHTTPAPIHeaderToken = "X-Api-Key"

	// Header name of the API session.
	BackendHTTPAPIHeaderSession = "X-Session-Key"
)

const (
	configEnvPrefix    = `sqreen`
	configFileBasename = `sqreen`
	configFilePath     = `.`
)

const (
	configKeyBackendHTTPAPIBaseURL = `url`
	configKeyBackendHTTPAPIToken   = `token`
	configKeyLogLevel              = `log_level`
)

const (
	configDefaultBackendHTTPAPIBaseURL = `https://back.sqreen.io/sqreen`
	configDefaultLogLevel              = `debug`
)

func init() {
	viper.SetEnvPrefix(configEnvPrefix)
	viper.AutomaticEnv()
	viper.SetConfigName(configFileBasename)
	viper.AddConfigPath(configFilePath)

	viper.SetDefault(configKeyBackendHTTPAPIBaseURL, configDefaultBackendHTTPAPIBaseURL)
	viper.SetDefault(configKeyLogLevel, configDefaultLogLevel)

	logger := plog.NewLogger("sqreen/agent/config")

	err := viper.ReadInConfig()
	if err != nil {
		logger.Error("configuration file read error:", err)
	}
}

// BackendHTTPAPIBaseURL returns the base URL of the backend HTTP API.
func BackendHTTPAPIBaseURL() string {
	return viper.GetString(configKeyBackendHTTPAPIBaseURL)
}

// BackendHTTPAPIToken returns the access token to the backend API.
func BackendHTTPAPIToken() string {
	return viper.GetString(configKeyBackendHTTPAPIToken)
}

// LogLevel returns the default log level.
func LogLevel() string {
	return viper.GetString(configKeyLogLevel)
}
