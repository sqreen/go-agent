// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package backend_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sqreen/go-agent/internal"
	"github.com/sqreen/go-agent/internal/backend"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

var (
	logger = plog.NewLogger(plog.Debug, os.Stderr, 0)
	cfg    = config.New(logger)
	fuzzer = fuzz.New().Funcs(FuzzStruct, FuzzCommandRequest, FuzzRuleDataValue, FuzzRule)
)

func TestClient(t *testing.T) {
	RegisterTestingT(t)

	t.Run("AppLogin", func(t *testing.T) {
		g := NewGomegaWithT(t)

		token := testlib.RandHTTPHeaderValue(2, 50)
		appName := testlib.RandHTTPHeaderValue(2, 50)

		statusCode := http.StatusOK

		endpointCfg := &config.BackendHTTPAPIEndpoint.AppLogin

		response := NewRandomAppLoginResponse()
		request := NewRandomAppLoginRequest()

		headers := http.Header{
			config.BackendHTTPAPIHeaderToken:   []string{token},
			config.BackendHTTPAPIHeaderAppName: []string{appName},
		}

		server := initFakeServer(endpointCfg, request, response, statusCode, headers)
		defer server.Close()

		client := backend.NewClient(server.URL(), cfg, logger)

		res, err := client.AppLogin(request, token, appName, false)
		g.Expect(err).NotTo(HaveOccurred())
		// A request has been received
		g.Expect(len(server.ReceivedRequests())).ToNot(Equal(0))
		g.Expect(res).Should(Equal(response))
	})

	t.Run("AppBeat", func(t *testing.T) {
		g := NewGomegaWithT(t)

		statusCode := http.StatusOK

		endpointCfg := &config.BackendHTTPAPIEndpoint.AppBeat

		response := NewRandomAppBeatResponse()
		request := NewRandomAppBeatRequest()

		client, server := initFakeServerSession(endpointCfg, request, response, statusCode, nil)
		defer server.Close()

		res, err := client.AppBeat(context.Background(), request)
		g.Expect(err).NotTo(HaveOccurred())
		// A request has been received
		g.Expect(len(server.ReceivedRequests())).ToNot(Equal(0))
		g.Expect(res).Should(Equal(response))
	})

	t.Run("Batch", func(t *testing.T) {
		g := NewGomegaWithT(t)

		statusCode := http.StatusOK

		endpointCfg := &config.BackendHTTPAPIEndpoint.Batch

		request := NewRandomBatchRequest()
		t.Logf("%#v", request)

		client, server := initFakeServerSession(endpointCfg, request, nil, statusCode, nil)
		defer server.Close()

		err := client.Batch(context.Background(), request)
		g.Expect(err).NotTo(HaveOccurred())
		// A request has been received
		g.Expect(len(server.ReceivedRequests())).ToNot(Equal(0))
	})

	t.Run("ActionsPack", func(t *testing.T) {
		g := NewGomegaWithT(t)

		statusCode := http.StatusOK

		endpointCfg := &config.BackendHTTPAPIEndpoint.ActionsPack

		response := NewRandomActionsPackResponse()

		client, server := initFakeServerSession(endpointCfg, nil, response, statusCode, nil)
		defer server.Close()

		res, err := client.ActionsPack()
		g.Expect(err).NotTo(HaveOccurred())
		// A request has been received
		g.Expect(len(server.ReceivedRequests())).ToNot(Equal(0))
		g.Expect(res).Should(Equal(response))
	})

	t.Run("RulesPack", func(t *testing.T) {
		g := NewGomegaWithT(t)

		statusCode := http.StatusOK

		endpointCfg := &config.BackendHTTPAPIEndpoint.RulesPack

		response := NewRandomRulesPackResponse()

		client, server := initFakeServerSession(endpointCfg, nil, response, statusCode, nil)
		defer server.Close()

		res, err := client.RulesPack()
		g.Expect(err).NotTo(HaveOccurred())
		// A request has been received
		g.Expect(len(server.ReceivedRequests())).ToNot(Equal(0))
		g.Expect(res).Should(Equal(response))
	})

	t.Run("AppLogout", func(t *testing.T) {
		g := NewGomegaWithT(t)

		statusCode := http.StatusOK

		endpointCfg := &config.BackendHTTPAPIEndpoint.AppLogout

		client, server := initFakeServerSession(endpointCfg, nil, nil, statusCode, nil)
		defer server.Close()

		err := client.AppLogout()
		g.Expect(err).NotTo(HaveOccurred())
		// A request has been received
		g.Expect(len(server.ReceivedRequests())).ToNot(Equal(0))
	})
}

func initFakeServer(endpointCfg *config.HTTPAPIEndpoint, request, response interface{}, statusCode int, headers http.Header) *ghttp.Server {
	handlers := []http.HandlerFunc{
		ghttp.VerifyRequest(endpointCfg.Method, endpointCfg.URL),
		ghttp.VerifyHeader(headers),
	}

	if request != nil {
		handlers = append(handlers, ghttp.VerifyJSONRepresenting(request))
	}

	if response != nil {
		handlers = append(handlers, ghttp.RespondWithJSONEncoded(statusCode, response))
	} else {
		handlers = append(handlers, ghttp.RespondWith(statusCode, nil))
	}

	server := ghttp.NewServer()
	server.AppendHandlers(ghttp.CombineHandlers(handlers...))
	return server
}

func initFakeServerSession(endpointCfg *config.HTTPAPIEndpoint, request, response interface{}, statusCode int, headers http.Header) (client *backend.Client, server *ghttp.Server) {
	server = ghttp.NewServer()

	loginReq := NewRandomAppLoginRequest()
	loginRes := NewRandomAppLoginResponse()
	loginRes.SessionId = testlib.RandHTTPHeaderValue(2, 50)
	loginRes.Status = true
	server.AppendHandlers(ghttp.RespondWithJSONEncoded(http.StatusOK, loginRes))

	client = backend.NewClient(server.URL(), cfg, logger)

	token := testlib.RandHTTPHeaderValue(2, 50)
	appName := testlib.RandHTTPHeaderValue(2, 50)
	_, err := client.AppLogin(loginReq, token, appName, false)
	if err != nil {
		panic(err)
	}

	if headers != nil {
		headers.Add(config.BackendHTTPAPIHeaderSession, loginRes.SessionId)
	} else {
		headers = http.Header{
			config.BackendHTTPAPIHeaderSession: []string{loginRes.SessionId},
		}
	}

	handlers := []http.HandlerFunc{
		ghttp.VerifyRequest(endpointCfg.Method, endpointCfg.URL),
		ghttp.VerifyHeader(headers),
	}

	if request != nil {
		handlers = append(handlers, ghttp.VerifyJSONRepresenting(request))
	}

	if response != nil {
		handlers = append(handlers, ghttp.RespondWithJSONEncoded(statusCode, response))
	} else {
		handlers = append(handlers, ghttp.RespondWith(statusCode, nil))
	}

	server.AppendHandlers(ghttp.CombineHandlers(handlers...))

	return client, server
}

func TestProxy(t *testing.T) {
	// ghttp uses gomega global functions so globally register `t` to gomega.
	RegisterTestingT(t)
	t.Run("HTTPS_PROXY", func(t *testing.T) { testProxy(t, "HTTPS_PROXY") })
	t.Run("SQREEN_PROXY", func(t *testing.T) { testProxy(t, "SQREEN_PROXY") })
}

func testProxy(t *testing.T, envVar string) {
	t.Skip()
	// FIXME: (i) use an actual proxy, (ii) check requests go through it, (iii)
	// use a fake backend and check the requests exactly like previous tests
	// (ideally reuse them and add the proxy).
	http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	// Create a fake proxy checking it receives a CONNECT request.
	proxy := ghttp.NewServer()
	defer proxy.Close()
	proxy.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodConnect, ""),
		ghttp.RespondWith(http.StatusOK, nil),
	))

	//back := ghttp.NewUnstartedServer()
	//back.HTTPTestServer.Listener.Close()
	//listener, _ := net.Listen("tcp", testlib.GetNonLoopbackIP().String()+":0")
	//back.HTTPTestServer.Listener = listener
	//back.start()
	//defer back.Close()
	//back.AppendHandlers(ghttp.CombineHandlers(
	//	ghttp.VerifyRequest(http.MethodPost, "/sqreen/v1/app-login"),
	//	ghttp.RespondWith(http.StatusOK, nil),
	//))

	// Setup the configuration
	os.Setenv(envVar, proxy.URL())
	defer os.Unsetenv(envVar)
	require.Equal(t, os.Getenv(envVar), proxy.URL())

	// The new client should take the proxy into account.
	client := backend.NewClient(cfg.BackendHTTPAPIBaseURL(), cfg, logger)
	// Perform a request that should go through the proxy.
	request := NewRandomAppLoginRequest()
	_, err := client.AppLogin(request, "my-token", "my-app", false)
	require.NoError(t, err)
	// A request has been received:
	//require.NotEqual(t, len(back.ReceivedRequests()), 0, "0 request received")
	require.NotEqual(t, len(proxy.ReceivedRequests()), 0, "0 request received")
}

func NewRandomAppLoginResponse() *api.AppLoginResponse {
	pb := new(api.AppLoginResponse)
	fuzzer.Fuzz(pb)
	return pb
}

func NewRandomAppLoginRequest() *api.AppLoginRequest {
	pb := new(api.AppLoginRequest)
	fuzzer.Fuzz(pb)
	return pb
}

func NewRandomAppBeatResponse() *api.AppBeatResponse {
	pb := new(api.AppBeatResponse)
	fuzzer.Fuzz(pb)
	return pb
}

func NewRandomAppBeatRequest() *api.AppBeatRequest {
	pb := new(api.AppBeatRequest)
	fuzzer.Fuzz(pb)
	return pb
}

func NewRandomBatchRequest() *api.BatchRequest {
	pb := new(api.BatchRequest)
	fuzzer.Fuzz(pb)
	return pb
}

func NewRandomActionsPackResponse() *api.ActionsPackResponse {
	pb := new(api.ActionsPackResponse)
	fuzzer.Fuzz(pb)
	return pb
}

func NewRandomRulesPackResponse() *api.RulesPackResponse {
	pb := new(api.RulesPackResponse)
	fuzzer.Fuzz(pb)
	return pb
}

func FuzzStruct(e *api.Struct, c fuzz.Continue) {
	v := struct {
		A string
		B int
		C float64
		D bool
		F []byte
		G struct {
			A string
			B int
			C float64
			D bool
			F []byte
		}
	}{}
	c.Fuzz(&v)
	e.Value = v
}

func FuzzCommandRequest(e *api.CommandRequest, c fuzz.Continue) {
	c.Fuzz(&e.Name)
	c.Fuzz(&e.Uuid)
}

func FuzzRuleDataValue(e *api.RuleDataEntry, c fuzz.Continue) {
	v := &api.CustomErrorPageRuleDataEntry{}
	c.Fuzz(&v.StatusCode)
	e.Value = v
}

func FuzzRule(e *api.Rule, c fuzz.Continue) {
	c.Fuzz(e)
	e.Signature = api.RuleSignature{ECDSASignature: api.ECDSASignature{Message: []byte(`{}`)}}
}

func TestValidateCredentialsConfiguration(t *testing.T) {
	for _, tc := range []struct {
		Name, Token, AppName string
		ShouldFail           bool
	}{
		{
			Name:    "valid org token strings",
			Token:   "org_ok",
			AppName: "ok",
		},
		{
			Name:  "valid non-org",
			Token: "ok",
		},
		{
			Name:       "invalid credentials with empty strings",
			Token:      "",
			AppName:    "",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with empty token and non-empty app-name",
			Token:      "",
			AppName:    "ok",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with valid org token but empty app-name",
			Token:      "org_ok",
			AppName:    "",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "org_ok",
			AppName:    "ko\nko",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "org_ok",
			AppName:    "koko\x00\x01\x02ok",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "org_ok",
			AppName:    "koko\tok",
			ShouldFail: true,
		},
		{
			Name:    "valid credentials with a space in app-name",
			Token:   "org_ok",
			AppName: "ok ok ok",
		},
		{
			Name:       "invalid credentials with invalid token character",
			Token:      "org_ok\nko",
			AppName:    "ok",
			ShouldFail: true,
		},
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			err := internal.ValidateCredentialsConfiguration(tc.Token, tc.AppName)
			if tc.ShouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
