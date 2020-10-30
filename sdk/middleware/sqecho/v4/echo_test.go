// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
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
		h := func(c echo.Context) error {
			{
				// using echo's context
				sq := FromContext(c)
				require.NotNil(t, sq)
				sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
				sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			}
			{
				// using the request context
				sq := sdk.FromContext(c.Request().Context())
				require.NotNil(t, sq)
				sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
				sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			}
			body, err := ioutil.ReadAll(c.Request().Body)
			if err != nil {
				return err
			}
			return c.String(http.StatusOK, string(body))
		}

		for _, tc := range []struct {
			name string
			test func(t *testing.T, h echo.HandlerFunc, c echo.Context)
		}{
			{
				name: "without middleware",
				test: func(t *testing.T, h echo.HandlerFunc, c echo.Context) {
					require.NoError(t, h(c))
				},
			},

			{
				name: "with middleware",
				test: func(t *testing.T, h echo.HandlerFunc, c echo.Context) {
					ctx := mockups.NewRootHTTPProtectionContextMockup(context.Background(), mock.Anything, mock.Anything)
					ctx.ExpectClose(mock.MatchedBy(func(closed types.ClosedProtectionContextFace) bool {
						require.Equal(t, 4, len(closed.Events().CustomEvents))
						return true
					}))
					defer ctx.AssertExpectations(t)

					// Create the middleware function
					m := middleware(ctx)

					// Wrap and call the handler
					err := m(h)(c)
					require.NoError(t, err)
				},
			},

			{
				name: "without agent",
				test: func(t *testing.T, h echo.HandlerFunc, c echo.Context) {
					// Create the middleware function with a nil root context
					m := middleware(nil)
					// Wrap and call the handler
					err := m(h)(c)
					require.NoError(t, err)
				},
			},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				// Perform the request and record the output
				rec := httptest.NewRecorder()
				body := testlib.RandUTF8String(4096)
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
				// Create a context
				c := echo.New().NewContext(req, rec)
				// Perfom the test
				tc.test(t, h, c)
				// Check the request was performed as expected
				require.Equal(t, http.StatusOK, rec.Code)
				require.Equal(t, body, rec.Body.String())
			})
		}
	})

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
					middlewares []echo.MiddlewareFunc
					handler     func(echo.Context) error
					test        func(t *testing.T, rec *httptest.ResponseRecorder, err error)
				}{
					//
					// Control flow tests
					// When an handlers, including middlewares, block.
					//

					{
						name: "sqreen first/the middleware aborts before the handler",
						middlewares: []echo.MiddlewareFunc{
							middleware(tc.ctx),
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									c.String(middlewareResponseStatus, middlewareResponseBody)
									return errors.New("middleware abort")
								}
							},
						},
						handler: func(echo.Context) error {
							panic("unexpected control flow")
							return nil
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.Error(t, err)
							require.Equal(t, "middleware abort", err.Error())
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen first/the handler aborts",
						middlewares: []echo.MiddlewareFunc{
							middleware(tc.ctx),
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									err := next(c)
									require.Error(t, err)
									return err
								}
							},
						},
						handler: func(c echo.Context) error {
							c.String(handlerResponseStatus, handlerResponseBody)
							return errors.New("handler abort")
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.Error(t, err)
							require.Equal(t, "handler abort", err.Error())
							require.Equal(t, handlerResponseStatus, rec.Code)
							require.Equal(t, handlerResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen first/no one aborts",
						middlewares: []echo.MiddlewareFunc{
							middleware(tc.ctx),
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									err := c.String(middlewareResponseStatus, middlewareResponseBody)
									require.NoError(t, err)
									err = next(c)
									require.NoError(t, err)
									err = c.String(middlewareResponseStatus, middlewareResponseBody)
									require.NoError(t, err)
									return err
								}
							},
						},
						handler: func(c echo.Context) error {
							return c.String(handlerResponseStatus, handlerResponseBody)
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.NoError(t, err)
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody+handlerResponseBody+middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen first/the middleware aborts after the handler",
						middlewares: []echo.MiddlewareFunc{
							middleware(tc.ctx),
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									err := next(c)
									require.NoError(t, err)
									err = c.String(middlewareResponseStatus, middlewareResponseBody)
									require.NoError(t, err)
									return errors.New("middleware abort")
								}
							},
						},
						handler: func(c echo.Context) error {
							return nil
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.Error(t, err)
							require.Equal(t, "middleware abort", err.Error())
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after another middleware/issue AGO-29: the middleware aborts after the next handler",
						middlewares: []echo.MiddlewareFunc{
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									err := next(c)
									require.NoError(t, err)
									err = c.String(middlewareResponseStatus, middlewareResponseBody)
									require.NoError(t, err)
									return errors.New("middleware abort")
								}
							},
							middleware(tc.ctx),
						},
						handler: func(c echo.Context) error {
							// Do nothing so that the calling middleware can handle the response.
							return nil
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.Error(t, err)
							require.Equal(t, "middleware abort", err.Error())
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after another middleware/the middleware aborts before the next handler",
						middlewares: []echo.MiddlewareFunc{
							func(echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									err := c.String(middlewareResponseStatus, middlewareResponseBody)
									require.NoError(t, err)
									return errors.New("middleware abort")
								}
							},
							func(echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									// Make sure echo doesn't call the next middleware when the
									// previous one returns an error.
									panic("unexpected control flow")
								}
							},
							middleware(tc.ctx),
						},
						handler: func(echo.Context) error {
							// Make sure echo doesn't call the handler when one of the
							// previous middlewares return an error.
							panic("unexpected control flow")
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.Error(t, err)
							require.Equal(t, "middleware abort", err.Error())
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after another middleware/the handler aborts",
						middlewares: []echo.MiddlewareFunc{
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									err := next(c)
									require.Error(t, err)
									return err
								}
							},
							middleware(tc.ctx),
						},
						handler: func(c echo.Context) error {
							err := c.String(handlerResponseStatus, handlerResponseBody)
							require.NoError(t, err)
							return errors.New("handler abort")
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.Error(t, err)
							require.Equal(t, "handler abort", err.Error())
							require.Equal(t, handlerResponseStatus, rec.Code)
							require.Equal(t, handlerResponseBody, rec.Body.String())
						},
					},

					{
						name: "sqreen after another middleware/no one aborts",
						middlewares: []echo.MiddlewareFunc{
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									err := c.String(middlewareResponseStatus, middlewareResponseBody)
									require.NoError(t, err)
									err = next(c)
									require.NoError(t, err)
									err = c.String(middlewareResponseStatus, middlewareResponseBody)
									require.NoError(t, err)
									return nil
								}
							},
							middleware(tc.ctx),
						},
						handler: func(c echo.Context) error {
							err := c.String(handlerResponseStatus, handlerResponseBody)
							require.NoError(t, err)
							return nil
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.NoError(t, err)
							require.Equal(t, middlewareResponseStatus, rec.Code)
							require.Equal(t, middlewareResponseBody+handlerResponseBody+middlewareResponseBody, rec.Body.String())
						},
					},

					//
					// Context data flow tests
					//
					{
						name: "middleware1, sqreen, middleware2, handler",
						middlewares: []echo.MiddlewareFunc{
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									c.Set("m10", "v10")
									c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), "m11", "v11")))
									return next(c)
								}
							},
							middleware(tc.ctx),
							func(next echo.HandlerFunc) echo.HandlerFunc {
								return func(c echo.Context) error {
									c.Set("m20", "v20")
									c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), "m21", "v21")))
									return next(c)
								}
							},
						},
						handler: func(c echo.Context) error {
							// From Gin's context
							if v, ok := c.Get("m10").(string); !ok || v != "v10" {
								panic("couldn't get the context value m10")
							}
							if v, ok := c.Get("m20").(string); !ok || v != "v20" {
								panic("couldn't get the context value m20")
							}

							// From the request context
							reqCtx := c.Request().Context()
							if v, ok := reqCtx.Value("m11").(string); !ok || v != "v11" {
								panic("couldn't get the context value m11")
							}
							if v, ok := reqCtx.Value("m21").(string); !ok || v != "v21" {
								panic("couldn't get the context value m21")
							}

							return c.NoContent(http.StatusOK)
						},
						test: func(t *testing.T, rec *httptest.ResponseRecorder, err error) {
							require.NoError(t, err)
							require.Equal(t, http.StatusOK, rec.Code)
						},
					},
				} {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						// Create a echo router
						router := echo.New()
						// Setup the middleware
						router.Use(tc.middlewares...)
						// Add the endpoint
						router.GET("/", tc.handler)

						// Perform the request and record the output
						rec := httptest.NewRecorder()
						req, _ := http.NewRequest("GET", "/", nil)
						var err error
						router.HTTPErrorHandler = func(e error, _ echo.Context) {
							err = e
						}
						router.ServeHTTP(rec, req)

						// Check the request was performed as expected
						tc.test(t, rec, err)
					})
				}
			})
		}
	})

	t.Run("response observation", func(t *testing.T) {
		t.Run("direct http header write", func(t *testing.T) {
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
			expectedContentLength := int64(len("\"hello\"\n"))
			expectedContentType := echo.MIMEApplicationJSONCharsetUTF8

			h := func(c echo.Context) error {
				return c.JSON(expectedStatusCode, "hello")
			}

			// Perform the request and record the output
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/", nil)

			m := middleware(root)
			c := echo.New().NewContext(req, rec)

			// Wrap and call the handler
			err := m(h)(c)

			// Check the result
			require.NoError(t, err)
			require.Equal(t, expectedStatusCode, rec.Code)
			require.Equal(t, expectedStatusCode, responseStatusCode)
			require.Equal(t, expectedContentLength, responseContentLength)
			require.Equal(t, expectedContentType, responseContentType)
		})

		t.Run("echo handler error", func(t *testing.T) {
			var (
				responseStatusCode int
			)
			root := mockups.NewRootHTTPProtectionContextMockup(context.Background(), mock.Anything, mock.Anything)
			root.ExpectClose(mock.MatchedBy(func(closed types.ClosedProtectionContextFace) bool {
				resp := closed.Response()
				responseStatusCode = resp.Status()
				return true
			}))
			defer root.AssertExpectations(t)

			expectedError := echo.ErrNotFound

			h := func(c echo.Context) error {
				return expectedError
			}

			// Perform the request and record the output
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/", nil)

			m := middleware(root)
			c := echo.New().NewContext(req, rec)

			// Wrap and call the handler
			err := m(h)(c)

			// Check the result
			require.Error(t, err)
			require.Equal(t, expectedError, err)
			require.Equal(t, expectedError.Code, responseStatusCode)
		})
	})
}

func middleware(p types.RootProtectionContext) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return middlewareHandlerFromRootProtectionContext(p, next, c)
		}
	}
}
