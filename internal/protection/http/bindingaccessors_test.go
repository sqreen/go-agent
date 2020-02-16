// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sqreen/go-agent/internal/binding-accessor"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/assert.v1"
)

func TestRequestBindingAccessors(t *testing.T) {
	expectedClientIP := net.IPv4(64, 81, 32, 89)

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
				`#.Headers['Content-Type']`:      []string{multipartContentTypeHeader},
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
			Body: strings.NewReader(`z=post&both=y&prio=2&=nokey&orphan;empty=&`),
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "POST",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.Headers['Content-Type']`:      []string{`application/x-www-form-urlencoded`},
				`#.URL.RequestURI`:               "/admin/news",
				`#.FilteredParams | flat_values`: FlattenedResult{"post", "y", "2", "nokey", "", ""},
				`#.FilteredParams | flat_keys`:   FlattenedResult{"Form", "z", "both", "prio", "", "orphan", "empty"},
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
				`#.Headers['Content-Type']`:      []string{`application/x-www-form-urlencoded`},
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
			req.ParseForm()
			ctx := http_protection.NewRequestBindingAccessorContext(requestReaderImpl{r: req, clientIP: expectedClientIP})
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

type FlattenedResult []interface{}

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
		require.Failf(t, "missing expected value", "expected `%v` having type `%T`", f, f)
	}
}

type requestReaderImpl struct {
	r        *http.Request
	clientIP net.IP
}

func (r requestReaderImpl) UserAgent() string {
	return r.r.UserAgent()
}

func (r requestReaderImpl) Referer() string {
	return r.r.Referer()
}

func (r requestReaderImpl) Header(header string) (value string) {
	return r.r.Header.Get(header)
}

func (r requestReaderImpl) Headers() http.Header {
	return r.r.Header
}

func (r requestReaderImpl) Method() string {
	return r.r.Method
}

func (r requestReaderImpl) URL() *url.URL {
	return r.r.URL
}

func (r requestReaderImpl) RequestURI() string {
	return r.r.RequestURI
}

func (r requestReaderImpl) Host() string {
	return r.r.Host
}

func (r requestReaderImpl) RemoteAddr() string {
	return r.r.RemoteAddr
}

func (r requestReaderImpl) IsTLS() bool {
	return r.r.TLS != nil
}

func (r requestReaderImpl) Form() url.Values {
	return r.r.Form
}

func (r requestReaderImpl) PostForm() url.Values {
	return r.r.PostForm
}

func (r requestReaderImpl) ClientIP() net.IP {
	return r.clientIP
}
