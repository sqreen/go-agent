// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("without middleware", func(t *testing.T) {
		body := testlib.RandUTF8String(4096)
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

		// Create an Echo context
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// Define a Echo handler
		h := func(c echo.Context) error {
			{
				// using echo's context interface
				sq := FromContext(c)
				require.NotNil(t, sq)
				sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
				sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			}
			{
				// using echo's context interface
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
		// Perform the request and record the output
		err := h(c)
		// Check the request was performed as expected
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("without middleware", func(t *testing.T) {
		body := testlib.RandUTF8String(4096)
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

		// Create an Echo context
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// Define a Echo handler
		h := func(c echo.Context) error {
			{
				// using echo's context interface
				sq := FromContext(c)
				require.NotNil(t, sq)
				sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
				sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(time.Now())
			}
			{
				// using echo's context interface
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
		// Perform the request and record the output
		err := Middleware()(h)(c)
		// Check the request was performed as expected
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

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
					name        string
					middlewares []echo.MiddlewareFunc
					handler     func(echo.Context) error
					test        func(t *testing.T, rec *httptest.ResponseRecorder, err error)
				}{
					{
						name: "sqreen first/the middleware aborts before the handler",
						middlewares: []echo.MiddlewareFunc{
							middleware(tc.agent),
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
							middleware(tc.agent),
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
							middleware(tc.agent),
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
							middleware(tc.agent),
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
							middleware(tc.agent),
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
							middleware(tc.agent),
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
							middleware(tc.agent),
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
							middleware(tc.agent),
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
		expectedStatusCode := 433
		expectedContentLength := int64(len("\"hello\"\n"))
		expectedContentType := echo.MIMEApplicationJSONCharsetUTF8

		agent := &mockups.AgentMockup{}
		agent.ExpectConfig().Return(&mockups.AgentConfigMockup{}).Once()
		agent.ExpectIsIPAllowed(mock.Anything).Return(false).Once()
		agent.ExpectIsPathAllowed(mock.Anything).Return(false).Once()
		var (
			responseStatusCode int
			responseContentType string
			responseContentLength int64
		)
		agent.ExpectSendClosedRequestContext(mock.MatchedBy(func(recorded types.ClosedRequestContextFace) bool {
			resp := recorded.Response()
			responseStatusCode = resp.Status()
			responseContentLength = resp.ContentLength()
			responseContentType = resp.ContentType()
			return true
		})).Return(nil)
		defer agent.AssertExpectations(t)

		// Create a route
		router := echo.New()
		router.Use(middleware(agent))
		router.GET("/", func(c echo.Context) error {
			return c.JSON(expectedStatusCode, "hello")
		})

		// Perform the request and record the output
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		var err error
		router.HTTPErrorHandler = func(e error, _ echo.Context) {
			err = e
		}
		router.ServeHTTP(rec, req)

		// Check the result
		require.NoError(t, err)
		require.Equal(t, expectedStatusCode, rec.Code)
		require.Equal(t, expectedStatusCode, responseStatusCode)
		require.Equal(t, expectedContentLength, responseContentLength)
		require.Equal(t, expectedContentType, responseContentType)
	})

}

func middleware(agent protectioncontext.AgentFace) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return middlewareHandler(agent, next, c)
		}
	}
}
