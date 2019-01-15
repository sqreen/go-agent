package backend

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/sqreen/go-agent/agent/config"
	"github.com/sqreen/go-agent/agent/plog"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/sqreen/go-agent/agent/backend/api"
	"golang.org/x/net/http/httpproxy"
)

var logger = plog.NewLogger("agent/backend")

type Client struct {
	client      *http.Client
	backendURL  string
	pbMarshaler jsonpb.Marshaler
}

func NewClient(backendURL string) (*Client, error) {
	proxyCfg := httpproxy.Config{
		HTTPSProxy: config.BackendHTTPAPIProxy(),
	}
	proxyURL := proxyCfg.ProxyFunc()
	proxy := func(req *http.Request) (*url.URL, error) {
		return proxyURL(req.URL)
	}

	transport := *(http.DefaultTransport).(*http.Transport)
	transport.Proxy = proxy

	client := &Client{
		client: &http.Client{
			Timeout:   config.BackendHTTPAPIRequestTimeout,
			Transport: &transport,
		},
		backendURL:  backendURL,
		pbMarshaler: api.DefaultJSONPBMarshaler,
	}

	return client, nil
}

func (c *Client) AppLogin(req *api.AppLoginRequest, token string) (*api.AppLoginResponse, error) {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.AppLogin)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderToken, token)
	res := new(api.AppLoginResponse)
	if err := c.Do(httpReq, req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) AppBeat(req *api.AppBeatRequest, session string) (*api.AppBeatResponse, error) {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.AppBeat)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, session)
	res := new(api.AppBeatResponse)
	if err := c.Do(httpReq, req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) AppLogout(session string) error {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.AppLogout)
	if err != nil {
		return err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, session)
	if err := c.Do(httpReq); err != nil {
		return err
	}
	return nil
}

func (c *Client) Batch(req *api.BatchRequest, session string) error {
	httpReq, err := c.newRequest(&config.BackendHTTPAPIEndpoint.Batch)
	if err != nil {
		return err
	}
	httpReq.Header.Set(config.BackendHTTPAPIHeaderSession, session)
	if err := c.Do(httpReq, req); err != nil {
		return err
	}
	return nil
}

func (c *Client) Do(req *http.Request, pbs ...proto.Message) error {
	var buf bytes.Buffer

	if len(pbs) >= 1 {
		err := c.pbMarshaler.Marshal(&buf, pbs[0])
		if err != nil {
			return err
		}
	}

	req.Body = ioutil.NopCloser(&buf)
	req.ContentLength = int64(buf.Len())

	dumpReq, _ := httputil.DumpRequestOut(req, true)
	logger.Debugf("sending request\n%s", dumpReq)

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	dumpRes, _ := httputil.DumpResponse(res, true)
	logger.Debugf("received response\n%s", dumpRes)

	// As documented (https://golang.org/pkg/net/http/#Response), connections are
	// reused iif the response body was fully drained. The following chunk thus
	// makes sure there's no bytes left before closing the body reader, and thus
	// makes sure EOF is returned. The `net/http` package requires this, while
	// json parsers may stop before.
	// See also: https://github.com/google/go-github/pull/317
	defer func() {
		// fixme: log when n > 00
		io.CopyN(ioutil.Discard, res.Body, 1)
		res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return NewStatusError(res.StatusCode)
	}

	if len(pbs) >= 2 {
		err = jsonpb.Unmarshal(res.Body, pbs[1])
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
