// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("using the sdk without middleware", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/hello", nil)
		body := testlib.RandUTF8String(4096)
		// Create a router
		router := http.NewServeMux()
		// Add an endpoint accessing the SDK handle
		subrouter := http.NewServeMux()
		subrouter.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
			sq := sdk.FromContext(req.Context())
			require.NotNil(t, sq)
			sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
			sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			w.Write([]byte(body))
			w.WriteHeader(http.StatusOK)
		})
		router.Handle("/", subrouter)
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("using the sdk with the middleware", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/hello", nil)
		body := testlib.RandUTF8String(4096)
		// Create a router
		router := http.NewServeMux()
		// Add an endpoint accessing the SDK handle
		subrouter := http.NewServeMux()
		subrouter.Handle("/hello", Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sq := sdk.FromContext(req.Context())
			require.NotNil(t, sq)
			sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
			sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			w.Write([]byte(body))
			w.WriteHeader(http.StatusOK)
		})))
		router.Handle("/", subrouter)
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	// Test how the control flows between middleware and handler functions
	t.Run("control flow", func(t *testing.T) {
		middlewareResponseBody := testlib.RandUTF8String(4096)
		middlewareResponseStatus := 433
		handlerResponseBody := testlib.RandUTF8String(4096)
		handlerResponseStatus := 533
		agent := &mockups.AgentMockup{}
		agent.ExpectConfig().Return(&mockups.AgentConfigMockup{})
		agent.ExpectIsIPAllowed(mock.Anything).Return(false)
		agent.ExpectIsPathAllowed(mock.Anything).Return(false)
		agent.ExpectSendClosedRequestContext(mock.Anything).Return(nil)
		defer agent.AssertExpectations(t) // inaccurate but worth it

		for _, tc := range []struct {
			name  string
			agent protectioncontext.AgentFace
		}{
			{
				name:  "agent disabled",
				agent: nil,
			},
			{
				name:  "agent enabled",
				agent: agent,
			},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				for _, tc := range []struct {
					name     string
					handlers http.Handler
					test     func(t *testing.T, rec *httptest.ResponseRecorder)
				}{
					{
						name: "sqreen first/handler writes the response",
						handlers: middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(handlerResponseStatus)
							io.WriteString(w, handlerResponseBody)
						}), tc.agent),
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, handlerResponseStatus, rec.Code)
							require.Equal(t, handlerResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after another middleware/the middleware writes the response first",
						handlers: func(next http.Handler) http.Handler {
							return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								w.WriteHeader(middlewareResponseStatus)
								io.WriteString(w, middlewareResponseBody)
								next.ServeHTTP(w, r)
								io.WriteString(w, middlewareResponseBody)
							})
						}(middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							io.WriteString(w, handlerResponseBody)
						}), tc.agent)),
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody+handlerResponseBody+middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after another middleware/the middleware writes the response after the handler",
						handlers: func(next http.Handler) http.Handler {
							return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								next.ServeHTTP(w, r)
								w.WriteHeader(middlewareResponseStatus) // too late
								io.WriteString(w, middlewareResponseBody)
							})
						}(middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							io.WriteString(w, handlerResponseBody) // involves a 200 status code
						}), tc.agent)),
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, http.StatusOK, rec.Code)
							require.Equal(t, handlerResponseBody+middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after another middleware/issue AGO-29: only the middleware writes the response",
						handlers: func(next http.Handler) http.Handler {
							return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								next.ServeHTTP(w, r)
								w.WriteHeader(middlewareResponseStatus)
								io.WriteString(w, middlewareResponseBody)
							})
						}(middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), tc.agent)),
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},
				} {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						// Perform the request and record the output
						rec := httptest.NewRecorder()
						req, _ := http.NewRequest("GET", "/", nil)
						tc.handlers.ServeHTTP(rec, req)

						// Check the request was performed as expected
						tc.test(t, rec)
					})
				}
			})
		}
	})

	t.Run("response observation", func(t *testing.T) {
		expectedStatusCode := 433

		agent := &mockups.AgentMockup{}
		agent.ExpectConfig().Return(&mockups.AgentConfigMockup{}).Once()
		agent.ExpectIsIPAllowed(mock.Anything).Return(false).Once()
		agent.ExpectIsPathAllowed(mock.Anything).Return(false).Once()
		var responseStatusCode int
		agent.ExpectSendClosedRequestContext(mock.MatchedBy(func(recorded types.ClosedRequestContextFace) bool {
			resp := recorded.Response()
			responseStatusCode = resp.Status()
			return true
		})).Return(nil)
		defer agent.AssertExpectations(t)

		req, _ := http.NewRequest("GET", "/", nil)
		// Create a router
		router := http.NewServeMux()
		router.Handle("/", middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(expectedStatusCode)
		}), agent))
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// Check the request was performed as expected
		require.Equal(t, expectedStatusCode, responseStatusCode)
		require.Equal(t, expectedStatusCode, rec.Code)
	})
}

func middleware(next http.HandlerFunc, agent protectioncontext.AgentFace) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middlewareHandler(agent, next, w, r)
	})
}

func TestRequestReaderImpl(t *testing.T) {
	t.Run("framework params", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)
			reqReader := requestReaderImpl{req}

			frameworkParams := reqReader.FrameworkParams()
			require.Empty(t, frameworkParams)
		})

		t.Run("url segments", func(t *testing.T) {
			req, err := http.NewRequest("GET", `/a/bb/cc/%2Ffoo//bar///zyz/"\"\\"\\\"/`, nil)
			require.NoError(t, err)
			reqReader := requestReaderImpl{req}

			frameworkParams := reqReader.FrameworkParams()
			require.NotEmpty(t, frameworkParams)

			require.Equal(t, []string{`a`, `bb`, `cc`, `foo`, `bar`, `zyz`, `"\"\\"\\\"`}, frameworkParams[urlSegmentsFrameworkParamsKey])
		})
	})
}
