// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Agent configuration package.

// This package includes both compile-time and run-time configuration of the
// agent. Variables are made configurable at run-time when necessary for users.

package config

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"

	"github.com/spf13/viper"
)

type Config struct {
	*viper.Viper
}

// Error messages.
const (
	ErrorMessage_UnsupportedCommand = "command is not supported"
)

const PublicKey string = `-----BEGIN PUBLIC KEY-----
MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQA39oWMHR8sxb9LRaM5evZ7mw03iwJ
WNHuDeGqgPo1HmvuMfLnAyVLwaMXpGPuvbqhC1U65PG90bTJLpvNokQf0VMA5Tpi
m+NXwl7bjqa03vO/HErLbq3zBRysrZnC4OhJOF1jazkAg0psQOea2r5HcMcPHgMK
fnWXiKWnZX+uOWPuerE=
-----END PUBLIC KEY-----`

type HTTPAPIEndpoint struct {
	Method, URL string
}

// Error metrics store period.
const ErrorMetricsPeriod = time.Minute

// Default value of network timeouts.
const DefaultNetworkTimeout = 5 * time.Second

// Backend client configuration.
var (
	// Timeout value of a HTTP request. See http.Client.Timeout.
	BackendHTTPAPIRequestTimeout = DefaultNetworkTimeout

	// List of endpoint addresses, relative to the base URL.
	BackendHTTPAPIEndpoint = struct {
		AppLogin, AppLogout, AppBeat, AppException, Batch, ActionsPack, RulesPack HTTPAPIEndpoint
	}{
		AppLogin:     HTTPAPIEndpoint{http.MethodPost, "/sqreen/v1/app-login"},
		AppLogout:    HTTPAPIEndpoint{http.MethodGet, "/sqreen/v0/app-logout"},
		AppBeat:      HTTPAPIEndpoint{http.MethodPost, "/sqreen/v1/app-beat"},
		AppException: HTTPAPIEndpoint{http.MethodPost, "/sqreen/v0/app_sqreen_exception"},
		Batch:        HTTPAPIEndpoint{http.MethodPost, "/sqreen/v0/batch"},
		ActionsPack:  HTTPAPIEndpoint{http.MethodGet, "/sqreen/v0/actionspack"},
		RulesPack:    HTTPAPIEndpoint{http.MethodGet, "/sqreen/v0/rulespack"},
	}

	// Header name of the API token.
	BackendHTTPAPIHeaderToken = "X-Api-Key"

	// Header name of the API session.
	BackendHTTPAPIHeaderSession = "X-Session-Key"

	// Header name of the App name.
	BackendHTTPAPIHeaderAppName = "X-App-Name"

	BackendHTTPAPIOrganizationTokenPrefix = "org_"

	// BackendHTTPAPIRequestRetryPeriod is the time period to retry failed backend
	// HTTP requests.
	BackendHTTPAPIRequestRetryPeriod = time.Minute

	// BackendHTTPAPIBackoffRate is the backoff rate to compute the next sleep
	// duration before retrying the failed request.
	BackendHTTPAPIBackoffRate = 2.0

	// BackendHTTPAPIBackoffMaxDuration is the maximum backoff's sleep duration.
	BackendHTTPAPIBackoffMaxDuration = 30 * time.Minute

	// BackendHTTPAPIBackoffMaxDuration is the minimum backoff's sleep duration.
	BackendHTTPAPIBackoffMinDuration = time.Millisecond

	// BackendHTTPAPIDefaultHeartbeatDelay is the default heartbeat delay when not
	// correctly provided by the backend.
	BackendHTTPAPIDefaultHeartbeatDelay = time.Minute

	// EventBatchMaxStaleness is the time when the data in the event manager's
	// batch is considered too long, and is therefore immediatly sent to the
	// backend, without waiting for the batch to become full.
	EventBatchMaxStaleness = 20 * time.Second
)

const (
	MaxEventsPerHeatbeat = 1000
)

var (
	TrackedHTTPHeaders = []string{
		"X-Forwarded-For",
		"X-Forwarded-Host",
		"X-Forwarded-Proto",
		"X-Client-Ip",
		"X-Real-Ip",
		"X-Forwarded",
		"X-Cluster-Client-Ip",
		"Forwarded-For",
		"Forwarded",
		"Via",
		"Content-Type",
		"Content-Length",
		"Host",
		"X-Requested-With",
		"X-Request-Id",
		"HTTP_X_FORWARDED_FOR",
		"HTTP_X_REAL_IP",
		"HTTP_CLIENT_IP",
		"HTTP_X_FORWARDED",
		"HTTP_X_CLUSTER_CLIENT_IP",
		"HTTP_FORWARDED_FOR",
		"HTTP_FORWARDED",
		"HTTP_VIA",
	}

	IPRelatedHTTPHeaders = []string{
		"X-Forwarded-For",
		"X-Client-Ip",
		"X-Real-Ip",
		"X-Forwarded",
		"X-Cluster-Client-Ip",
		"Forwarded-For",
		"Forwarded",
		"Via",
		"HTTP_X_FORWARDED_FOR",
		"HTTP_X_REAL_IP",
		"HTTP_CLIENT_IP",
		"HTTP_X_FORWARDED",
		"HTTP_X_CLUSTER_CLIENT_IP",
		"HTTP_FORWARDED_FOR",
		"HTTP_FORWARDED",
		"HTTP_VIA",
	}
)

// Helper function to return the IP network out of a string.
func ipnet(s string) *net.IPNet {
	_, n, _ := net.ParseCIDR(s)
	return n
}

// IP networks allowing to compute whether to
var (
	IPv4PrivateNetworks = []*net.IPNet{
		ipnet("0.0.0.0/8"),
		ipnet("10.0.0.0/8"),
		ipnet("127.0.0.0/8"),
		ipnet("169.254.0.0/16"),
		ipnet("172.16.0.0/12"),
		ipnet("192.0.0.0/29"),
		ipnet("192.0.0.170/31"),
		ipnet("192.0.2.0/24"),
		ipnet("192.168.0.0/16"),
		ipnet("198.18.0.0/15"),
		ipnet("198.51.100.0/24"),
		ipnet("203.0.113.0/24"),
		ipnet("240.0.0.0/4"),
		ipnet("255.255.255.255/32"),
	}

	IPv4PublicNetwork = ipnet("100.64.0.0/10")

	IPv6PrivateNetworks = []*net.IPNet{
		ipnet("::1/128"),
		ipnet("::/128"),
		ipnet("::ffff:0:0/96"),
		ipnet("100::/64"),
		ipnet("2001::/23"),
		ipnet("2001:2::/48"),
		ipnet("2001:db8::/32"),
		ipnet("2001:10::/28"),
		ipnet("fc00::/7"),
		ipnet("fe80::/10"),
	}
)

const (
	configEnvPrefix    = `sqreen`
	configFileBasename = `sqreen`
)

const (
	configEnvKeyConfigFile = `config_file`

	configKeyBackendHTTPAPIBaseURL    = `url`
	configKeyBackendHTTPAPIToken      = `token`
	configKeyLogLevel                 = `log_level`
	configKeyAppName                  = `app_name`
	configKeyHTTPClientIPHeader       = `ip_header`
	configKeyHTTPClientIPHeaderFormat = `ip_header_format`
	configKeyBackendHTTPAPIProxy      = `proxy`
	configKeyDisable                  = `disable`
	configKeyStripHTTPReferer         = `strip_http_referer`
	configKeyRules                    = `rules`
	configKeySDKMetricsPeriod         = `sdk_metrics_period`
	configKeyMaxMetricsStoreLength    = `max_metrics_store_length`
)

// User configuration's default values.
const (
	configDefaultBackendHTTPAPIBaseURL = `https://back.sqreen.com`
	configDefaultLogLevel              = `info`
	configDefaultSDKMetricsPeriod      = 60
	configDefaultMaxMetricsStoreLength = 100 * 1024 * 1024
)

func New(logger *plog.Logger) *Config {
	manager := viper.New()
	manager.SetEnvPrefix(configEnvPrefix)
	manager.AutomaticEnv()
	manager.SetConfigName(configFileBasename)

	// Default values of configurable parameters
	parameters := []struct {
		key            string
		defaultValue   interface{}
		secretFromChar int
		hidden         bool
	}{
		{key: configKeyBackendHTTPAPIBaseURL, defaultValue: configDefaultBackendHTTPAPIBaseURL},
		{key: configKeyLogLevel, defaultValue: configDefaultLogLevel},
		{key: configKeyBackendHTTPAPIToken, defaultValue: "", secretFromChar: 6},
		{key: configKeyAppName, defaultValue: ""},
		{key: configKeyHTTPClientIPHeader, defaultValue: ""},
		{key: configKeyHTTPClientIPHeaderFormat, defaultValue: ""},
		{key: configKeyBackendHTTPAPIProxy, defaultValue: ""},
		{key: configKeyDisable, defaultValue: ""},
		{key: configKeyStripHTTPReferer, defaultValue: ""},
		{key: configKeyRules, defaultValue: "", hidden: true},
		{key: configKeySDKMetricsPeriod, defaultValue: configDefaultSDKMetricsPeriod, hidden: true},
		{key: configKeyMaxMetricsStoreLength, defaultValue: configDefaultMaxMetricsStoreLength, hidden: true},
	}
	for _, p := range parameters {
		manager.SetDefault(p.key, p.defaultValue)
	}

	// Configuration file settings
	configFileEnvVar := strings.ToUpper(configEnvPrefix + "_" + configEnvKeyConfigFile)
	configFile := os.Getenv(configFileEnvVar)
	if configFile != "" {
		// File location enforced by the user
		manager.SetConfigFile(configFile)
		logger.Infof("config: configuration file enforced by the environment variable `%s` to `%s`", configFileEnvVar, configFile)
	} else {
		// Not enforced: add possible paths in precedence order
		// 1. Current working directory path:
		manager.AddConfigPath(`.`)
		// 2. Executable path
		exec, err := os.Executable()
		if err != nil {
			logger.Error(sqerrors.Wrap(err, "config: could not read the executable file path"))
		} else {
			manager.AddConfigPath(filepath.Dir(exec))
		}
	}
	// Try to read a configuration file according to the previous settings
	if readErr, fileUsed := manager.ReadInConfig(), manager.ConfigFileUsed(); readErr != nil && fileUsed != "" {
		// Could not read despite the fact of having found a file
		logger.Error(sqerrors.Wrap(readErr, fmt.Sprintf("config: could not read the configuration file `%s`: falling back to environment variables", fileUsed)))
	} else if fileUsed != "" {
		// A file was found and no error reading it
		logger.Infof("config: reading configuration settings from file `%s`", fileUsed)
	} else {
		logger.Infof("config: reading configuration settings from environment variables")
	}

	cfg := &Config{manager}
	if cfg.LogLevel() == plog.Debug {
		logger.Infof("config: setting: %s = %q", configFileEnvVar, configFile)
		for _, p := range parameters {
			if !p.hidden {
				v := cfg.GetString(p.key)
				if p.secretFromChar > 0 && len(v) > 0 {
					secret := make([]byte, 0, len(v))
					secret = append(secret, v[:p.secretFromChar]...)
					for range v[p.secretFromChar:] {
						secret = append(secret, '*')
					}
					v = string(secret)
				}
				logger.Infof("config: settings: %s = %q", p.key, v)
			}
		}
	}
	return cfg
}

// BackendHTTPAPIBaseURL returns the base URL of the backend HTTP API.
func (c *Config) BackendHTTPAPIBaseURL() string {
	return sanitizeString(c.GetString(configKeyBackendHTTPAPIBaseURL))
}

// BackendHTTPAPIToken returns the access token to the backend API.
func (c *Config) BackendHTTPAPIToken() string {
	return sanitizeString(c.GetString(configKeyBackendHTTPAPIToken))
}

// LogLevel returns the log level.
func (c *Config) LogLevel() plog.LogLevel {
	return plog.ParseLogLevel(sanitizeString(c.GetString(configKeyLogLevel)))
}

// AppName returns the app name.
func (c *Config) AppName() string {
	return sanitizeString(c.GetString(configKeyAppName))
}

// HTTPClientIPHeader returns the header to first lookup to find the client ip of a HTTP request.
func (c *Config) HTTPClientIPHeader() string {
	return sanitizeString(c.GetString(configKeyHTTPClientIPHeader))
}

// HTTPClientIPHeaderFormat returns the header format of the `ip_header` value.
func (c *Config) HTTPClientIPHeaderFormat() string {
	return sanitizeString(c.GetString(configKeyHTTPClientIPHeaderFormat))
}

// Proxy returns the proxy configuration to use for backend HTTP calls.
func (c *Config) BackendHTTPAPIProxy() string {
	return sanitizeString(c.GetString(configKeyBackendHTTPAPIProxy))
}

// Disable returns true when the agent should be disabled, false otherwise.
func (c *Config) Disable() bool {
	disable := sanitizeString(c.GetString(configKeyDisable))
	return disable != "" || c.BackendHTTPAPIToken() == ""
}

// Disable returns true when the agent should be strip the `Referer` HTTP
// header, false otherwise.
func (c *Config) StripHTTPReferer() bool {
	strip := sanitizeString(c.GetString(configKeyStripHTTPReferer))
	return strip != ""
}

// LocalRulesFile returns a JSON file containing custom rules in an array. They
// are added to the rules received from server.
func (c *Config) LocalRulesFile() string {
	return sanitizeString(c.GetString(configKeyRules))
}

// SDKMetricsPeriod returns the period to use for the SDK metric stores.
// This is temporary until the SDK rules are implemented and required for
// integration tests which require a shorter time.
func (c *Config) SDKMetricsPeriod() int {
	p, err := strconv.Atoi(sanitizeString(c.GetString(configKeySDKMetricsPeriod)))
	if err != nil {
		return configDefaultSDKMetricsPeriod
	}
	return p
}

// MaxMetricsStoreLength returns the maximum length a metrics store should not
// exceed. After this limit, new metrics values will be dropped.
func (c *Config) MaxMetricsStoreLength() uint {
	n := c.GetInt(configKeyMaxMetricsStoreLength)
	if n < 0 {
		n = 0
	}
	return uint(n)
}

func sanitizeString(s string) string {
	return strings.TrimSpace(s)
}
