// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"golang.org/x/net/http/httpproxy"
	"golang.org/x/xerrors"
)

type Client struct {
	client     *http.Client
	backendURL string
	logger     *plog.Logger
	session    string
}

func NewClient(backendURL string, cfg *config.Config, logger *plog.Logger) *Client {
	var transport *http.Transport
	if proxySettings := cfg.BackendHTTPAPIProxy(); proxySettings == "" {
		// No user settings. The default transport uses standard global proxy
		// settings *_PROXY environment variables.
		dummyReq, _ := http.NewRequest("GET", backendURL, nil)
		if proxyURL, _ := http.ProxyFromEnvironment(dummyReq); proxyURL != nil {
			logger.Infof("client: using system http proxy `%s` as indicated by the system environment variables http_proxy, https_proxy and no_proxy (or their uppercase alternatives)", proxyURL)
		}
		transport = (http.DefaultTransport).(*http.Transport)
	} else {
		// Use the settings.
		logger.Infof("client: using configured https proxy `%s`", proxySettings)
		proxyCfg := httpproxy.Config{
			HTTPSProxy: proxySettings,
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

	client := &Client{
		client: &http.Client{
			Timeout:   config.BackendHTTPAPIRequestTimeout,
			Transport: transport,
		},
		backendURL: backendURL,
		logger:     logger,
	}

	return client
}

func (c *Client) AppLogin(req *api.AppLoginRequest, token string, appName string) (*api.AppLoginResponse, error) {
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

	return res, nil
}

func (c *Client) AppBeat(req *api.AppBeatRequest) (*api.AppBeatResponse, error) {
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

func (c *Client) Batch(req *api.BatchRequest) error {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.Batch)
	if err != nil {
		return err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, c.session)
	if err := c.Do(httpReq, req); err != nil {
		return err
	}
	return nil
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
				// TODO: update the api to pass these dropped extra details (involved
				//  ip addresses) as error metadata
				err = sqerrors.Wrap(netErr.Err, fmt.Sprintf("%s %s", urlErr.Op, urlErr.URL))
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
	req, err := http.NewRequest(
		descriptor.Method,
		c.backendURL+descriptor.URL,
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}
