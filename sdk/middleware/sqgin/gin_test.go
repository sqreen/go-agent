// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgin

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ExpectationsAsserter interface {
	AssertExpectations(mock.TestingT) bool
}

func TestMiddleware(t *testing.T) {
	t.Run("sdk methods", func(t *testing.T) {
		// Define a handler performing 4 track events (aka custom event internally)
		h := func(c *gin.Context) {
			{
				// using gin's context
				sq := sdk.FromContext(c)
				require.NotNil(t, sq)
				sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
				sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			}
			{
				// using the request context
				sq := sdk.FromContext(c.Request.Context())
				require.NotNil(t, sq)
				sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
				sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			}
			body, err := ioutil.ReadAll(c.Request.Body)
			require.NoError(t, err)
			c.String(http.StatusOK, string(body))
		}

		for _, tc := range []struct {
			name  string
			setup func(t *testing.T, e *gin.Engine) ExpectationsAsserter
		}{
			{
				name: "without middleware",
				setup: func(t *testing.T, e *gin.Engine) ExpectationsAsserter {
					// No middleware
					return nil
				},
			},

			{
				name: "with middleware",
				setup: func(t *testing.T, e *gin.Engine) ExpectationsAsserter {
					ctx := mockups.NewRootHTTPProtectionContextMockup(context.Background(), mock.Anything, mock.Anything)
					ctx.ExpectClose(mock.MatchedBy(func(closed types.ClosedProtectionContextFace) bool {
						require.Equal(t, 4, len(closed.Events().CustomEvents))
						return true
					}))

					// Setup the middleware
					m := middleware(ctx)
					e.Use(m)

					return ctx
				},
			},

			{
				name: "without agent",
				setup: func(t *testing.T, e *gin.Engine) ExpectationsAsserter {
					// Create and set the middleware function with a nil root context
					m := middleware(nil)
					e.Use(m)
					return nil
				},
			},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				// Perform the request and record the output
				rec := httptest.NewRecorder()
				body := testlib.RandUTF8String(4096)
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

				// Create a Gin context
				c, r := gin.CreateTestContext(rec)
				c.Request = req

				// Setup the middleware
				if m := tc.setup(t, r); m != nil {
					defer m.AssertExpectations(t)
				}

				// Register the route
				r.POST("/", h)

				// Serve the request
				require.NotPanics(t, func() {
					r.HandleContext(c)
				})

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
					name        string
					middlewares []gin.HandlerFunc
					handler     func(*gin.Context)
					test        func(t *testing.T, rec *httptest.ResponseRecorder)
				}{
					//
					// Control flow tests
					// When an handlers, including middlewares, block.
					//

					{
						name: "sqreen first/next middleware aborts before the handler",
						middlewares: []gin.HandlerFunc{
							middleware(tc.ctx),
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
							middleware(tc.ctx),
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
							middleware(tc.ctx),
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
							middleware(tc.ctx),
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
							middleware(tc.ctx),
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
							middleware(tc.ctx),
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
							middleware(tc.ctx),
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
							middleware(tc.ctx),
						},
						handler: func(*gin.Context) {
							// Do nothing so that the calling middleware can handle the response.
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					//
					// Context data flow tests
					//
					{
						name: "middleware1, sqreen, middleware2, handler",
						middlewares: []gin.HandlerFunc{
							func(c *gin.Context) {
								c.Set("m10", "v10")
								c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "m11", "v11"))
							},
							middleware(tc.ctx),
							func(c *gin.Context) {
								c.Set("m20", "v20")
								c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "m21", "v21"))
							},
						},
						handler: func(c *gin.Context) {
							// From Gin's context
							if v, ok := c.Value("m10").(string); !ok || v != "v10" {
								panic("couldn't get the context value m10")
							}
							if v, ok := c.Value("m20").(string); !ok || v != "v20" {
								panic("couldn't get the context value m20")
							}

							// From the request context
							reqCtx := c.Request.Context()
							if v, ok := reqCtx.Value("m11").(string); !ok || v != "v11" {
								panic("couldn't get the context value m11")
							}
							if v, ok := reqCtx.Value("m21").(string); !ok || v != "v21" {
								panic("couldn't get the context value m21")
							}

							c.Status(http.StatusOK)
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder) {
							require.Equal(t, http.StatusOK, rec.Code)
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
		expectedContentType := "application/json; charset=utf-8"

		// Create a route
		router := gin.New()
		router.Use(middleware(root))
		router.GET("/", func(c *gin.Context) {
			c.JSON(expectedStatusCode, "hello")
		})

		// Perform the request and record the output
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(rec, req)

		// Check the result
		require.Equal(t, expectedStatusCode, rec.Code)
		require.Equal(t, expectedStatusCode, responseStatusCode)
		require.Equal(t, expectedContentLength, responseContentLength)
		require.Equal(t, expectedContentType, responseContentType)
	})
}

func middleware(p types.RootProtectionContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		middlewareHandlerFromRootProtectionContext(p, c)
	}
}
