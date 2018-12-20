package backend

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"sqreen/agent/backend/api"
	"sqreen/agent/config"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

type Client struct {
	client      *http.Client
	backendURL  string
	httpRequest struct {
		appLogin, appLogout, appBeat request
	}
	pbMarshaler jsonpb.Marshaler
}

type request struct {
	*http.Request
	buf bytes.Buffer
}

func NewClient(backendURL string) (*Client, error) {
	client := &Client{
		client: &http.Client{
			Timeout: config.BackendHTTPAPIRequestTimeout,
		},
		backendURL: backendURL,
		pbMarshaler: jsonpb.Marshaler{
			OrigName:     true,
			EmitDefaults: true,
		},
	}

	var err error

	err = client.httpRequest.appLogin.prepare(&config.BackendHTTPAPIEndpoint.AppLogin)
	if err != nil {
		return nil, err
	}

	err = client.httpRequest.appLogout.prepare(&config.BackendHTTPAPIEndpoint.AppLogout)
	if err != nil {
		return nil, err
	}

	err = client.httpRequest.appBeat.prepare(&config.BackendHTTPAPIEndpoint.AppBeat)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) AppLogin(req *api.AppLoginRequest, token string) (*api.AppLoginResponse, error) {
	c.httpRequest.appLogin.Header.Set(config.BackendHTTPAPIHeaderToken, token)
	res := new(api.AppLoginResponse)
	if err := c.Do(&c.httpRequest.appLogin, req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) AppBeat(req *api.AppBeatRequest, session string) (*api.AppBeatResponse, error) {
	c.httpRequest.appLogout.Header.Set(config.BackendHTTPAPIHeaderSession, session)
	res := new(api.AppBeatResponse)
	if err := c.Do(&c.httpRequest.appBeat, req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) AppLogout(session string) error {
	c.httpRequest.appLogout.Header.Set(config.BackendHTTPAPIHeaderSession, session)
	if err := c.Do(&c.httpRequest.appLogout, nil, nil); err != nil {
		return err
	}
	return nil
}

// Helper method to build an API endpoint request structure.
func (r *request) prepare(descriptor *config.HTTPAPIEndpoint) error {
	req, err := http.NewRequest(
		descriptor.Method,
		config.BackendHTTPAPIBaseURL+descriptor.URL,
		&r.buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	r.Request = req
	return nil
}

func (c *Client) Do(r *request, request, response proto.Message) error {
	r.buf.Reset()
	if request != nil {
		err := c.pbMarshaler.Marshal(&r.buf, request)
		if err != nil {
			return err
		}
	}

	res, err := c.client.Do(r.Request)
	if err != nil {
		return err
	}

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

	if response != nil {
		err = jsonpb.Unmarshal(res.Body, response)
		if err != nil && err != io.EOF {
			return err
		}
	}

	return nil
}
