// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqecho"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("without middleware", func(t *testing.T) {
		body := testlib.RandString(1, 100)
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

		agent := &testlib.AgentMockup{}
		defer agent.AssertExpectations(t)
		sdk.SetAgent(agent)

		// Create an Echo context
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// Define a Echo handler
		h := func(c echo.Context) error {
			require.Nil(t, sqecho.FromContext(c))
			require.Nil(t, sdk.FromContext(c.Request().Context()))
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

	t.Run("without agent", func(t *testing.T) {
		body := testlib.RandString(1, 100)
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

		sdk.SetAgent(nil)

		// Create an Echo context
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// Define a Echo handler
		h := func(c echo.Context) error {
			require.Nil(t, sqecho.FromContext(c))
			require.Nil(t, sdk.FromContext(c.Request().Context()))
			body, err := ioutil.ReadAll(c.Request().Body)
			if err != nil {
				return err
			}
			return c.String(http.StatusOK, string(body))
		}
		// Perform the request and record the output
		mw := sqecho.Middleware()
		err := mw(h)(c)
		// Check the request was performed as expected
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("without security response", func(t *testing.T) {
		t.Run("without handler error", func(t *testing.T) {
			body := testlib.RandString(1, 100)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

			agent, record := testlib.NewAgentForMiddlewareTestsWithoutSecurityResponse()
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)

			// Create an Echo context
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			// Define a Echo handler
			h := func(c echo.Context) error {
				require.NotNil(t, sqecho.FromContext(c), "The middleware should attach its handle object to Gin's context")
				require.NotNil(t, sdk.FromContext(c.Request().Context()), "The middleware should attach its handle object to the request's context")
				body, err := ioutil.ReadAll(c.Request().Body)
				if err != nil {
					return err
				}
				return c.String(http.StatusOK, string(body))
			}
			// Perform the request and record the output
			mw := sqecho.Middleware()
			err := mw(h)(c)
			// Check the request was performed as expected
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, body, rec.Body.String())
		})

		t.Run("with a handler error", func(t *testing.T) {
			body := testlib.RandString(1, 100)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

			agent, record := testlib.NewAgentForMiddlewareTestsWithoutSecurityResponse()
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)

			// Create an Echo context
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			// Define a Echo handler
			theError := errors.New("oh my error")
			h := func(c echo.Context) error {
				return theError
			}
			// Perform the request and record the output
			mw := sqecho.Middleware()
			err := mw(h)(c)
			// Check the request was performed as expected
			require.Error(t, err)
			require.Equal(t, theError, err)
		})
	})

	t.Run("with a security response", func(t *testing.T) {
		t.Run("with ip response", func(t *testing.T) {
			body := testlib.RandString(1, 100)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

			status := http.StatusBadRequest
			agent, record := testlib.NewAgentForMiddlewareTestsWithSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)

			// Create an Echo context
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			// Define a Echo handler
			h := func(c echo.Context) error {
				panic("must not be called")
			}
			// Perform the request and record the output
			mw := sqecho.Middleware()
			err := mw(h)(c)
			// Check the request was performed as expected
			require.NoError(t, err)
			require.Equal(t, rec.Code, status)
			require.Equal(t, rec.Body.String(), "")
		})

		t.Run("with user response", func(t *testing.T) {
			body := testlib.RandString(1, 100)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

			status := http.StatusBadRequest
			agent, record := testlib.NewAgentForMiddlewareTestsWithUserSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			uid := sdk.EventUserIdentifiersMap{}
			record.ExpectIdentify(uid)
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)

			// Create an Echo context
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			// Define a Echo handler
			h := func(c echo.Context) error {
				sqreen := sdk.FromContext(c.Request().Context())
				sqUser := sqreen.ForUser(uid)
				sqUser.Identify()
				match, err := sqUser.MatchSecurityResponse()
				require.True(t, match)
				require.Error(t, err)
				return err
			}
			// Perform the request and record the output
			mw := sqecho.Middleware()
			err := mw(h)(c)
			// Check the request was performed as expected
			require.NoError(t, err)
			require.Equal(t, rec.Code, status)
			require.Equal(t, rec.Body.String(), "")
		})
	})
}
