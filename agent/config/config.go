package config

import (
	"net/http"
	"time"
)

type HTTPAPIEndpoint struct {
	Method, URL string
}

const (
	// Default value of network timeouts.
	DefaultNetworkTimeout = 60 * time.Second
)

// Backend client configuration.
var (
	// Timeout value of a HTTP request. See http.Client.Timeout.
	BackendHTTPAPIRequestTimeout = DefaultNetworkTimeout

	// Base URL of the backend HTTP API.
	BackendHTTPAPIBaseURL = "https://back.sqreen.io/sqreen"

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
