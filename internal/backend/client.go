// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package backend

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/backend/api/signal"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-sdk/signal/client"
	"golang.org/x/net/http/httpproxy"
	"golang.org/x/xerrors"
)

type Client struct {
	client       *http.Client
	backendURL   *url.URL
	logger       plog.DebugLevelLogger
	session      string
	signalClient *client.Client
	infra        *signal.AgentInfra
	health       *HealthStatus
}

func NewClient(baseURL string, proxy string, logger plog.DebugLevelLogger) (*Client, error) {
	var transport *http.Transport
	if proxy == "" {
		// No user settings. The default transport uses standard global proxy
		// settings *_PROXY environment variables.
		dummyReq, _ := http.NewRequest("GET", baseURL, nil)
		if proxyURL, _ := http.ProxyFromEnvironment(dummyReq); proxyURL != nil {
			logger.Infof("client: using system http proxy `%s` as indicated by the system environment variables http_proxy, https_proxy and no_proxy (or their uppercase alternatives)", proxyURL)
		}
		transport = (http.DefaultTransport).(*http.Transport)
	} else {
		// Use the settings.
		logger.Infof("client: using configured https and http proxy `%s`", proxy)
		proxyCfg := httpproxy.Config{
			HTTPSProxy: proxy,
			HTTPProxy:  proxy,
		}
		proxyURL := proxyCfg.ProxyFunc()
		proxy := func(req *http.Request) (*url.URL, error) {
			return proxyURL(req.URL)
		}
		// Shallow copy the default transport and overwrite its proxy settings.
		transportCopy := *(http.DefaultTransport).(*http.Transport)
		transport = &transportCopy
		transport.Proxy = proxy
	}

	backendURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, sqerrors.Wrapf(err, "could not parse the URL `%s`", backendURL)
	}

	client := &Client{
		client: &http.Client{
			Timeout:   config.BackendHTTPAPIRequestTimeout,
			Transport: transport,
		},
		backendURL: backendURL,
		logger:     logger,
	}

	return client, nil
}

type HealthStatus struct {
	DomainStatus api.SqreenDomainStatusMap
}

func (c *Client) Health() HealthStatus {
	if c.health != nil {
		return *c.health
	}

	health := HealthStatus{
		DomainStatus: api.SqreenDomainStatusMap{},
	}

	var (
		domain = client.DefaultBaseURL
		res    api.PingResponse
		err    error
	)
	req, err := http.NewRequest(config.BackendHTTPAPIEndpoint.Ping.Method, domain+config.BackendHTTPAPIEndpoint.Ping.URL, nil)
	if err == nil {
		err = c.Do(req, nil, &res)
	}
	status := api.SqreenDomainStatus{
		Status: res.Status,
	}
	if err != nil {
		status.Error = err.Error()
	}
	health.DomainStatus[domain] = status

	c.health = &health
	return health
}

func (c *Client) AppLogin(req *api.AppLoginRequest, token string, appName string, disableSignalBackend bool, defaultIngestionUrl *url.URL) (*api.AppLoginResponse, error) {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.AppLogin)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderToken, token)
	if appName != "" {
		httpReq.Header.Set(config.BackendHTTPAPIHeaderAppName, appName)
	}
	res := new(api.AppLoginResponse)
	if err := c.Do(httpReq, req, res); err != nil {
		// Keep the result when it's a HTTP status error as it may contain error
		// reasons sent by the backend
		if !xerrors.As(err, &HTTPStatusError{}) {
			return nil, err
		}
		return res, err
	}

	c.session = res.SessionId

	if !disableSignalBackend && res.Features.UseSignals {
		c.signalClient = client.NewClient(c.client, c.session)
		if defaultIngestionUrl != nil {
			c.signalClient.BaseURL = defaultIngestionUrl
		}

		// If the default signal URL is not healthy, fallback to the general
		// backend URL.
		if !c.Health().DomainStatus[client.DefaultBaseURL].Status {
			c.signalClient.BaseURL = c.backendURL
		}

		c.signalClient.Logger = c.logger
		c.infra = signal.NewAgentInfra(req.AgentVersion, req.OsType, req.Hostname, req.RuntimeVersion)
	}

	return res, nil
}

func (c *Client) AppBeat(ctx context.Context, req *api.AppBeatRequest) (*api.AppBeatResponse, error) {
	if legacyMetrics := req.Metrics; c.signalClient != nil && len(legacyMetrics) > 0 {
		metrics := signal.FromLegacyMetrics(legacyMetrics, c.infra.AgentVersion, c.logger)
		if err := c.signalClient.SignalService().SendBatch(ctx, metrics); err != nil {
			c.logger.Error(sqerrors.Wrap(err, "could not send the batch of metric signals"))
			// The request failed but since we still have the legacy AppBeat request
			// following, we can try again through it by not removing the metrics from
			// the legacy AppBeat request.
		} else {
			// The batch was successfully sent, don't do it again through the
			// following AppBeat request.
			req.Metrics = nil
		}
	}

	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.AppBeat)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)
	res := new(api.AppBeatResponse)
	if err := c.Do(httpReq, req, res); err != nil {
		return nil, err
	}
	return res, nil

}

func (c *Client) AppLogout() error {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.AppLogout)
	if err != nil {
		return err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)
	if err := c.Do(httpReq); err != nil {
		return err
	}
	return nil
}

func (c *Client) Batch(ctx context.Context, req *api.BatchRequest) error {
	if c.signalClient == nil {
		httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.Batch)
		if err != nil {
			return err
		}
		httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)
		return c.Do(httpReq, req)
	}

	batch := signal.FromLegacyBatch(req.Batch, c.infra, c.logger)
	return c.signalClient.SignalService().SendBatch(ctx, batch)
}

func (c *Client) ActionsPack() (*api.ActionsPackResponse, error) {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.ActionsPack)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)
	res := new(api.ActionsPackResponse)
	if err := c.Do(httpReq, nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) RulesPack() (*api.RulesPackResponse, error) {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.RulesPack)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)
	res := new(api.RulesPackResponse)
	if err := c.Do(httpReq, nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) SendAppBundle(req *api.AppBundle) error {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.Bundle)
	if err != nil {
		return err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)
	return c.Do(httpReq, req)
}

// Do performs the request whose body is pbs[0] pointer, while the expected
// response is pbs[1] pointer. They are optional, and must be used according to
// the cases request case.
func (c *Client) Do(req *http.Request, pbs ...interface{}) error {
	var buf bytes.Buffer

	if len(pbs) >= 1 && pbs[0] != nil {
		pbMarshaler := json.NewEncoder(&buf)
		err := pbMarshaler.Encode(pbs[0])
		if err != nil {
			return sqerrors.Wrap(err, "json marshal")
		}
	}
	req.Body = ioutil.NopCloser(&buf)
	req.ContentLength = int64(buf.Len())

	c.logger.Debugf("sending request\n%s\n", (*HTTPRequestStringer)(req))
	res, err := c.client.Do(req)
	if err != nil {
		// Try to unwrap the error to get the stable message part, excluding
		// involved ip addresses
		if urlErr, ok := err.(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok {
				err = sqerrors.WithInfo(err, netErr)
			} else {
				err = sqerrors.WithInfo(err, urlErr)
			}
		}
		return err
	}
	c.logger.Debugf("received response\n%s\n", (*HTTPResponseStringer)(res))

	// As documented (https://golang.org/pkg/net/http/#Response), connections are
	// reused iif the response body was fully drained. The following chunk thus
	// makes sure there's no bytes left before closing the body reader, and thus
	// makes sure EOF is returned. The `net/http` package requires this, while
	// json parsers may stop before.
	// See also: https://github.com/google/go-github/pull/317
	defer func() {
		io.CopyN(ioutil.Discard, res.Body, 1)
		res.Body.Close()
	}()

	if len(pbs) >= 2 && pbs[1] != nil {
		pbUnmarshaler := json.NewDecoder(res.Body)
		err = pbUnmarshaler.Decode(pbs[1])
		if err != nil && err != io.EOF {
			return sqerrors.Wrap(err, "json unmarshal")
		}
	}

	if res.StatusCode != http.StatusOK {
		return NewStatusError(res.StatusCode)
	}
	return nil
}

type HTTPRequestStringer http.Request

func (r *HTTPRequestStringer) String() string {
	dump, _ := httputil.DumpRequestOut((*http.Request)(r), true)
	return string(dump)
}

type HTTPResponseStringer http.Response

func (r *HTTPResponseStringer) String() string {
	dump, _ := httputil.DumpResponse((*http.Response)(r), true)
	return string(dump)
}

// Helper method to build an API endpoint request structure.
func (c *Client) newRequest(descriptor *config.HTTPAPIEndpoint) (*http.Request, error) {
	url, err := c.backendURL.Parse(descriptor.URL)
	if err != nil {
		return nil, sqerrors.Wrap(err, "could not parse the request url")
	}
	req, err := http.NewRequest(
		descriptor.Method,
		url.String(),
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) SendAgentMessage(ctx context.Context, t time.Time, message string, infos map[string]interface{}) error {
	hash := sha1.Sum([]byte(message))
	id := hex.EncodeToString(hash[:])

	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.AgentMessage)
	if err != nil {
		return err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)

	payload := api.AgentMessage{
		Id:      id,
		Kind:    "error",
		Message: message,
	}
	return c.Do(httpReq, payload)
}

// SendAgentMessage is a special client function allowing to send app-level
// messages when the instance is not logged in yet and will not.
func SendAgentMessage(logger plog.DebugLogger, cfg *config.Config, message string) {
	b := new(bytes.Buffer)
	id := sha1.Sum([]byte(message))
	payload := api.AgentMessage{
		Id:      hex.EncodeToString(id[:]),
		Kind:    "error",
		Message: message,
	}
	err := json.NewEncoder(b).Encode(payload)
	if err != nil {
		return
	}
	endpoint := config.BackendHTTPAPIEndpoint.AppAgentMessage
	req, err := http.NewRequest(endpoint.Method, cfg.BackendHTTPAPIBaseURL()+endpoint.URL, b)
	if err != nil {
		return
	}
	req.Header.Add(config.BackendHTTPAPIHeaderToken, cfg.BackendHTTPAPIToken())
	req.Header.Add(config.BackendHTTPAPIHeaderAppName, cfg.AppName())
	req.Header.Add("Content-Type", "application/json")

	logger.Debugf("sending app message:\n%s\n", (*HTTPRequestStringer)(req))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	logger.Debugf("received app exception response:\n%s\n", (*HTTPResponseStringer)(res))
}
