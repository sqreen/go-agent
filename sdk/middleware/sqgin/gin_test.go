// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqgin"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("without middleware", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		body := testlib.RandUTF8String(4096)
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
			c.String(http.StatusOK, body)
		})
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("without middleware", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		body := testlib.RandUTF8String(4096)
		// Create a Gin router
		router := gin.New()
		router.Use(sqgin.Middleware())
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
			c.String(http.StatusOK, body)
		})
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})
}
