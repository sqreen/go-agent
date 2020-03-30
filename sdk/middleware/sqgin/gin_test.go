// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("sdk calls", func(t *testing.T) {
		t.Run("without middleware", func(t *testing.T) {
			responseBody := testlib.RandUTF8String(4096)
			// Create a Gin router
			router := gin.New()
			// Add an endpoint accessing the SDK handle
			router.GET("/", func(c *gin.Context) {
				{
					// using gin's context interface
					sq := sdk.FromContext(c)
					require.NotNil(t, sq)
					sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
					sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
				}
				{
					// using request context interface
					sq := sdk.FromContext(c.Request.Context())
					require.NotNil(t, sq)
					sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
					sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
				}
				c.String(http.StatusOK, responseBody)
			})

			// Perform the request and record the output
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/", nil)
			router.ServeHTTP(rec, req)

			// Check the request was performed as expected
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, responseBody, rec.Body.String())
		})

		t.Run("with middleware", func(t *testing.T) {
			responseBody := testlib.RandUTF8String(4096)
			// Create a Gin router
			router := gin.New()
			router.Use(Middleware())
			// Add an endpoint accessing the SDK handle
			router.GET("/", func(c *gin.Context) {
				{
					// using gin's context interface
					sq := sdk.FromContext(c)
					require.NotNil(t, sq)
					sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
					sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
				}
				{
					// using gin's context interface
					sq := sdk.FromContext(c)
					require.NotNil(t, sq)
					sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
					sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
				}
				c.String(http.StatusOK, responseBody)
			})

			// Perform the request and record the output
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/", nil)
			router.ServeHTTP(rec, req)

			// Check the request was performed as expected
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, responseBody, rec.Body.String())
		})
	})

	// Test how the control flows between middleware and handler functions
	t.Run("control flow", func(t *testing.T) {
		middlewareResponseBody := testlib.RandUTF8String(4096)
		middlewareResponseStatus := 433
		handlerResponseBody := testlib.RandUTF8String(4096)
		handlerResponseStatus := 533
		agent := &mockups.AgentMockup{}
		agent.ExpectConfig().Return(&mockups.AgentConfigMockup{})
		agent.ExpectIsIPWhitelisted(mock.Anything).Return(false)
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
					name        string
					middlewares []gin.HandlerFunc
					handler     func(*gin.Context)
					test        func(t *testing.T, rec *httptest.ResponseRecorder)
				}{
					{
						name: "sqreen first/next middleware aborts before the handler",
						middlewares: []gin.HandlerFunc{
							middleware(tc.agent),
							func(c *gin.Context) {
								c.String(middlewareResponseStatus, middlewareResponseBody)
								c.Abort()
							},
						},
						handler: func(*gin.Context) {
							panic("unexpected control flow")
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen first/the handler aborts",
						middlewares: []gin.HandlerFunc{
							middleware(tc.agent),
							func(c *gin.Context) {
								c.Next()
								if !c.IsAborted() {
									panic("unexpected flow")
								}
							},
						},
						handler: func(c *gin.Context) {
							c.String(handlerResponseStatus, handlerResponseBody)
							c.Abort()
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, handlerResponseStatus, rec.Code)
							require.Equal(t, handlerResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen first/no one aborts",
						middlewares: []gin.HandlerFunc{
							middleware(tc.agent),
							func(c *gin.Context) {
								c.Writer.WriteString(middlewareResponseBody)
								c.Next()
								c.Writer.WriteString(middlewareResponseBody)
							},
						},
						handler: func(c *gin.Context) {
							c.String(handlerResponseStatus, handlerResponseBody)
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							// Because the middleware writes first, it involves the default 200
							// status code
							require.Equal(t, http.StatusOK, rec.Code)
							require.Equal(t, middlewareResponseBody+handlerResponseBody+middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen first/next middleware aborts after the handler",
						middlewares: []gin.HandlerFunc{
							middleware(tc.agent),
							func(c *gin.Context) {
								c.Next()
								c.String(middlewareResponseStatus, middlewareResponseBody)
								c.Abort()
							},
						},
						handler: func(c *gin.Context) {
							// Do nothing so that the calling middleware can handle the response.
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after a middleware/the middleware aborts before the next handler",
						middlewares: []gin.HandlerFunc{
							func(c *gin.Context) {
								c.String(middlewareResponseStatus, middlewareResponseBody)
								c.Abort()
							},
							func(*gin.Context) {
								// Make sure gin doesn't call the next middleware when the previous
								// one aborts.
								panic("unexpected control flow")
							},
							middleware(tc.agent),
						},
						handler: func(*gin.Context) {
							panic("unexpected control flow")
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after a middleware/no one aborts",
						middlewares: []gin.HandlerFunc{
							func(c *gin.Context) {
								c.Writer.WriteString(middlewareResponseBody)
								c.Next()
								c.Writer.WriteString(middlewareResponseBody)
							},
							middleware(tc.agent),
						},
						handler: func(c *gin.Context) {
							c.String(handlerResponseStatus, handlerResponseBody)
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							// Because the middleware writes first, it involves the default 200
							// status code
							require.Equal(t, http.StatusOK, rec.Code)
							require.Equal(t, middlewareResponseBody+handlerResponseBody+middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after a middleware/handler aborts",
						middlewares: []gin.HandlerFunc{
							func(c *gin.Context) {
								c.Next()
								if !c.IsAborted() {
									panic("unexpected control flow")
								}
							},
							middleware(tc.agent),
						},
						handler: func(c *gin.Context) {
							c.String(handlerResponseStatus, handlerResponseBody)
							c.Abort()
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, handlerResponseStatus, rec.Code)
							require.Equal(t, handlerResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after a middleware/issue AGO-29: the middleware aborts after the handler",
						middlewares: []gin.HandlerFunc{
							func(c *gin.Context) {
								c.Next()
								if !c.IsAborted() {
									panic("unexpected control flow")
								}
							},
							func(c *gin.Context) {
								c.Next()
								c.String(middlewareResponseStatus, middlewareResponseBody)
								c.Abort()
							},
							middleware(tc.agent),
						},
						handler: func(*gin.Context) {
							// Do nothing so that the calling middleware can handle the response.
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},
				} {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						// Create a Gin router
						router := gin.New()
						// Setup the middleware
						router.Use(tc.middlewares...)
						// Add the endpoint
						router.GET("/", tc.handler)

						// Perform the request and record the output
						rec := httptest.NewRecorder()
						req, _ := http.NewRequest("GET", "/", nil)
						router.ServeHTTP(rec, req)

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
		agent.ExpectIsIPWhitelisted(mock.Anything).Return(false).Once()
		var responseStatusCode int
		agent.ExpectSendClosedRequestContext(mock.MatchedBy(func(recorded types.ClosedRequestContextFace) bool {
			resp := recorded.Response()
			responseStatusCode = resp.Status()
			return true
		})).Return(nil)
		defer agent.AssertExpectations(t)

		// Create a route
		router := gin.New()
		router.Use(middleware(agent))
		router.GET("/", func(c *gin.Context) {
			c.Status(expectedStatusCode)
		})

		// Perform the request and record the output
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(rec, req)

		// Check the result
		require.Equal(t, expectedStatusCode, responseStatusCode)
		require.Equal(t, expectedStatusCode, responseStatusCode)
	})
}
