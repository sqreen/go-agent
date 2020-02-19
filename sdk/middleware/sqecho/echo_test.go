// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqecho"
	"github.com/sqreen/go-agent/tools/testlib"
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
				sq := sqecho.FromContext(c)
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
}
