// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package bindingaccessor_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/binding-accessor"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/assert.v1"
)

func TestRequestBindingAccessors(t *testing.T) {
	expectedClientIP := "64.81.32.89"

	var multipartFormBody bytes.Buffer
	mp := multipart.NewWriter(&multipartFormBody)
	f1, err := mp.CreateFormField("field 1")
	require.NoError(t, err)
	f1.Write([]byte("value 1"))
	mp.Close()
	multipartContentTypeHeader := mp.FormDataContentType()

	for _, tc := range []struct {
		Title            string
		Method           string
		URL              string
		Headers          http.Header
		BindingAccessors map[string]interface{}
		Body             io.Reader
	}{
		{
			Title:  "GET with URL parameters",
			Method: "GET",
			URL:    "http://sqreen.com/admin?user=root&password=root",
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "GET",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.URL.RequestURI`:               "/admin?user=root&password=root",
				`#.FilteredParams | flat_values`: FlattenedResult{"root", "root"},
				`#.FilteredParams | flat_keys`:   FlattenedResult{"Form", "user", "password"},
			},
		},
		{
			Title:  "GET without URL parameters",
			Method: "GET",
			URL:    "http://sqreen.com/admin",
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "GET",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.URL.RequestURI`:               "/admin",
				`#.FilteredParams | flat_values`: []interface{}(nil),
				`#.FilteredParams | flat_keys`:   FlattenedResult{"Form"},
			},
		},
		{
			Title:  "POST with multipart form data and URL parameters",
			Method: "POST",
			URL:    "http://sqreen.com/admin/news?user=root&password=root",
			Headers: http.Header{
				"Content-Type": []string{mp.FormDataContentType()},
			},
			Body: &multipartFormBody,
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "POST",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.Header['Content-Type']`:       []string{multipartContentTypeHeader},
				`#.URL.RequestURI`:               "/admin/news?user=root&password=root",
				`#.FilteredParams | flat_values`: FlattenedResult{"root", "root"},             // The multipart form data is not included for now
				`#.FilteredParams | flat_keys`:   FlattenedResult{"Form", "user", "password"}, // The multipart form data is not included for now
			},
		},
		{
			Title:  "POST with urlencoded parameters",
			Method: "POST",
			URL:    "http://sqreen.com/admin/news",
			Headers: http.Header{
				"Content-Type": []string{`application/x-www-form-urlencoded`},
			},
			Body: strings.NewReader("z=post&both=y&prio=2&=nokey&orphan;empty=&"),
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "POST",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.Header['Content-Type']`:       []string{`application/x-www-form-urlencoded`},
				`#.URL.RequestURI`:               "/admin/news",
				`#.FilteredParams | flat_values`: FlattenedResult{"post", "y", "2", "nokey", "", ""},
				`#.FilteredParams | flat_keys`:   FlattenedResult{"Form", "empty", "z", "both", "prio", "", "orphan"},
			},
		},
		{
			Title:  "POST with urlencoded parameters and URL parameters",
			Method: "POST",
			URL:    "http://sqreen.com/admin/news?sqreen=okay",
			Headers: http.Header{
				"Content-Type": []string{`application/x-www-form-urlencoded`},
			},
			Body: strings.NewReader("z=post&both=y&prio=2&=nokey&orphan;empty=&"),
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "POST",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.Header['Content-Type']`:       []string{`application/x-www-form-urlencoded`},
				`#.URL.RequestURI`:               "/admin/news?sqreen=okay",
				`#.FilteredParams | flat_values`: FlattenedResult{"post", "y", "2", "nokey", "", "", "okay"},
				`#.FilteredParams | flat_keys`:   FlattenedResult{"Form", "empty", "z", "both", "prio", "", "orphan", "sqreen"},
			},
		},
	} {
		tc := tc
		t.Run(tc.Title, func(t *testing.T) {
			req := httptest.NewRequest(tc.Method, tc.URL, tc.Body)
			for k, v := range tc.Headers {
				req.Header[k] = v
			}
			ctx := bindingaccessor.NewHTTPRequestBindingAccessorContext(req, expectedClientIP)
			for expr, expected := range tc.BindingAccessors {
				expr := expr
				expected := expected
				t.Run(expr, func(t *testing.T) {
					ba, err := bindingaccessor.Compile(expr)
					require.NoError(t, err)
					value, err := ba(ctx)
					require.NoError(t, err)

					// Quick hack for transformations from maps that return an array that
					// cannot be compared to the expected value because the order of the map
					// accesses is not stable
					if flattened, ok := expected.(FlattenedResult); ok {
						requireEqualFlatResult(t, flattened, value)
					} else {
						require.Equal(t, expected, value)
					}
				})
			}
		})
	}
}

func requireEqualFlatResult(t *testing.T, expected FlattenedResult, value interface{}) {
	got := value.([]interface{})
	require.Equal(t, len(expected), len(got), got)
loop:
	for _, f := range expected {
		for _, g := range got {
			if assert.IsEqual(g, f) {
				continue loop
			}
		}
		require.Failf(t, "missing expected value `%v`", "", f)
	}
}
