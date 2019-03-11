package backend

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"golang.org/x/net/http/httpproxy"
)

type Client struct {
	client     *http.Client
	backendURL string
	logger     *plog.Logger
	session    string
}

func NewClient(backendURL string, cfg *config.Config, logger *plog.Logger) (*Client, error) {
	logger = plog.NewLogger("client", logger)

	var transport *http.Transport
	if proxySettings := cfg.BackendHTTPAPIProxy(); proxySettings == "" {
		// No user settings. The default transport uses standard global proxy
		// settings *_PROXY environment variables.
		logger.Info("using proxy settings as indicated by the environment variables HTTP_PROXY, HTTPS_PROXY and NO_PROXY (or the lowercase versions)")
		transport = (http.DefaultTransport).(*http.Transport)
	} else {
		// Use the settings.
		logger.Info("using configured https proxy ", proxySettings)
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

	return client, nil
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
		return nil, err
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

// Do performs the request whose body is pbs[0] pointer, while the expected
// response is pbs[1] pointer. They are optional, and must be used according to
// the cases request case.
func (c *Client) Do(req *http.Request, pbs ...interface{}) error {
	var buf bytes.Buffer
	pbMarshaler := json.NewEncoder(&buf)

	if len(pbs) >= 1 && pbs[0] != nil {
		err := pbMarshaler.Encode(pbs[0])
		if err != nil {
			return err
		}
	}
	req.Body = ioutil.NopCloser(&buf)
	req.ContentLength = int64(buf.Len())

	dumpReq, _ := httputil.DumpRequestOut(req, true)
	c.logger.Debugf("sending request\n%s", dumpReq)

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	dumpRes, _ := httputil.DumpResponse(res, true)
	c.logger.Debugf("received response\n%s", dumpRes)

	// As documented (https://golang.org/pkg/net/http/#Response), connections are
	// reused iif the response body was fully drained. The following chunk thus
	// makes sure there's no bytes left before closing the body reader, and thus
	// makes sure EOF is returned. The `net/http` package requires this, while
	// json parsers may stop before.
	// See also: https://github.com/google/go-github/pull/317
	defer func() {
		// FIXME: log when n > 00
		io.CopyN(ioutil.Discard, res.Body, 1)
		res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return NewStatusError(res.StatusCode)
	}

	if len(pbs) >= 2 && pbs[1] != nil {
		pbUnmarshaler := json.NewDecoder(res.Body)
		err = pbUnmarshaler.Decode(pbs[1])
		if err != nil && err != io.EOF {
			return err
		}
	}

	return nil
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
