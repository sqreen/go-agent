// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhttp

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("sdk methods", func(t *testing.T) {
		// Define a handler performing 4 track events (aka custom event internally)
		h := func(w http.ResponseWriter, r *http.Request) {
			// using the request context
			sq := sdk.FromContext(r.Context())
			require.NotNil(t, sq)
			sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
			sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())

			body, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)
			n, err := io.WriteString(w, string(body))
			require.NoError(t, err)
			require.Len(t, body, n)
		}

		for _, tc := range []struct {
			name string
			test func(t *testing.T, h http.Handler, w http.ResponseWriter, r *http.Request)
		}{
			{
				name: "without middleware",
				test: func(t *testing.T, h http.Handler, w http.ResponseWriter, r *http.Request) {
					require.NotPanics(t, func() {
						h.ServeHTTP(w, r)
					})
				},
			},

			{
				name: "with middleware",
				test: func(t *testing.T, h http.Handler, w http.ResponseWriter, r *http.Request) {
					ctx := mockups.NewRootHTTPProtectionContextMockup(context.Background(), mock.Anything, mock.Anything)
					ctx.ExpectClose(mock.MatchedBy(func(closed types.ClosedProtectionContextFace) bool {
						require.Equal(t, 2, len(closed.Events().CustomEvents))
						return true
					}))
					defer ctx.AssertExpectations(t)

					// Wrap and call the handler
					h = middleware(ctx, h)

					// Wrap and call the handler
					require.NotPanics(t, func() {
						h.ServeHTTP(w, r)
					})
				},
			},

			{
				name: "without agent",
				test: func(t *testing.T, h http.Handler, w http.ResponseWriter, r *http.Request) {
					// Wrap and call the handler
					h = middleware(nil, h)
					// Wrap and call the handler
					require.NotPanics(t, func() {
						h.ServeHTTP(w, r)
					})
				},
			},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				// Perform the request and record the output
				rec := httptest.NewRecorder()
				body := testlib.RandUTF8String(4096)
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

				// Perfom the test
				tc.test(t, http.HandlerFunc(h), rec, req)
				// Check the request was performed as expected
				require.Equal(t, http.StatusOK, rec.Code)
				require.Equal(t, body, rec.Body.String())
			})
		}
	})

	// Test how the control flows between middleware and handler functions
	t.Run("data and control flow", func(t *testing.T) {
		middlewareResponseBody := testlib.RandUTF8String(4096)
		middlewareResponseStatus := 433
		handlerResponseBody := testlib.RandUTF8String(4096)
		handlerResponseStatus := 533

		root := mockups.NewRootHTTPProtectionContextMockup(context.Background(), mock.Anything, mock.Anything)
		root.ExpectClose(mock.Anything)
		defer root.AssertExpectations(t) // inaccurate but worth it

		for _, tc := range []struct {
			name string
			ctx  types.RootProtectionContext
		}{
			{
				name: "agent disabled",
				ctx:  nil,
			},
			{
				name: "agent enabled",
				ctx:  root,
			},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				for _, tc := range []struct {
					name     string
					handlers http.Handler
					test     func(t *testing.T, rec *httptest.ResponseRecorder)
				}{
					//
					// Control flow tests
					// When an handlers, including middlewares, block.
					//

					{
						name: "sqreen first/handler writes the response",
						handlers: middleware(tc.ctx, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(handlerResponseStatus)
							io.WriteString(w, handlerResponseBody)
						})),
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
						}(middleware(tc.ctx, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							io.WriteString(w, handlerResponseBody)
						}))),
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
						}(middleware(tc.ctx, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							io.WriteString(w, handlerResponseBody) // involves a 200 status code
						}))),
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
						}(middleware(tc.ctx, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))),
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					//
					// Context data flow tests
					//
					{
						name: "middleware, sqreen, handler",
						handlers: func(next http.Handler) http.Handler {
							return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								r = r.WithContext(context.WithValue(r.Context(), "m", "v"))
								next.ServeHTTP(w, r)
							})
						}(middleware(tc.ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							ctx := r.Context()
							if v, ok := ctx.Value("m").(string); !ok || v != "v" {
								panic("couldn't get the context value m")
							}

							w.WriteHeader(http.StatusOK)
						}))),
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, http.StatusOK, rec.Code)
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
		var (
			responseStatusCode    int
			responseContentType   string
			responseContentLength int64
		)
		root := mockups.NewRootHTTPProtectionContextMockup(context.Background(), mock.Anything, mock.Anything)
		root.ExpectClose(mock.MatchedBy(func(closed types.ClosedProtectionContextFace) bool {
			resp := closed.Response()
			responseStatusCode = resp.Status()
			responseContentLength = resp.ContentLength()
			responseContentType = resp.ContentType()
			return true
		}))
		defer root.AssertExpectations(t)

		expectedStatusCode := 433
		expectedContentLength := int64(len(`"hello"`))
		expectedContentType := "application/json"

		req, _ := http.NewRequest("GET", "/", nil)
		// Create a router
		router := http.NewServeMux()
		router.Handle("/", middleware(root, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(expectedStatusCode)
			w.Write([]byte(`"hello"`))
		})))
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// Check the request was performed as expected
		require.Equal(t, expectedStatusCode, rec.Code)
		require.Equal(t, expectedStatusCode, responseStatusCode)
		require.Equal(t, expectedContentLength, responseContentLength)
		require.Equal(t, expectedContentType, responseContentType)
	})
}

func middleware(ctx types.RootProtectionContext, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middlewareHandlerFromRootProtectionContext(ctx, next, w, r)
	})
}

func TestRequestReaderImpl(t *testing.T) {
	t.Run("framework params", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)
			reqReader := requestReaderImpl{Request: req}

			frameworkParams := reqReader.Params()
			require.Empty(t, frameworkParams)
		})

		t.Run("url segments", func(t *testing.T) {
			req, err := http.NewRequest("GET", `/a/bb/cc/%2Ffoo//bar///zyz/"\"\\"\\\"/`, nil)
			require.NoError(t, err)
			reqReader := requestReaderImpl{Request: req}

			frameworkParams := reqReader.Params()
			require.NotEmpty(t, frameworkParams)

			require.Equal(t, []interface{}{[]string{`a`, `bb`, `cc`, `foo`, `bar`, `zyz`, `"\"\\"\\\"`}}, frameworkParams[urlSegmentsFrameworkParamsKey])
		})
	})
}
