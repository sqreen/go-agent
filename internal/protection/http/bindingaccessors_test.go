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
	"net/textproto"
	"net/url"
	"strings"
	"testing"

	"github.com/sqreen/go-agent/internal/binding-accessor"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestRequestBindingAccessors(t *testing.T) {
	expectedClientIP := net.IPv4(64, 81, 32, 89)
	randString := testlib.RandPrintableUSASCIIString()

	var multipartFormBody bytes.Buffer
	mp := multipart.NewWriter(&multipartFormBody)
	f1, err := mp.CreateFormField("field 1")
	require.NoError(t, err)
	f1.Write([]byte("value 1"))
	mp.Close()
	multipartContentTypeHeader := mp.FormDataContentType()
	multipartExpectedBody := multipartFormBody.Bytes()

	for _, tc := range []struct {
		Title            string
		Method           string
		URL              string
		Headers          http.Header
		Body             io.Reader
		RequestParams    types.RequestParamMap
		BindingAccessors map[string]interface{}
	}{
		{
			Title:  "GET with URL parameters",
			Method: "GET",
			URL:    "http://sqreen.com/admin?user=uid&password=pwd",
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "GET",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.URL.RequestURI`:               "/admin?user=uid&password=pwd",
				`#.FilteredParams | flat_values`: FlattenedResult{"uid", "pwd"},
				`#.FilteredParams | flat_keys`:   FlattenedResult{"QueryForm", "user", "password"},
				`#.Body.String`:                  "",
				`#.Body.Bytes`:                   []byte(nil),
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
				`#.FilteredParams | flat_keys`:   []interface{}(nil),
				`#.Body.String`:                  "",
				`#.Body.Bytes`:                   []byte(nil),
			},
		},
		{
			Title:  "POST with multipart form data and URL parameters",
			Method: "POST",
			URL:    "http://sqreen.com/admin/news?user=uid&password=pwd",
			Headers: http.Header{
				"Content-Type": []string{multipartContentTypeHeader},
			},
			Body: &multipartFormBody,
			BindingAccessors: map[string]interface{}{
				`#.Method`:                       "POST",
				`#.Host`:                         "sqreen.com",
				`#.ClientIP`:                     expectedClientIP,
				`#.Headers['Content-Type']`:      []string{multipartContentTypeHeader},
				`#.URL.RequestURI`:               "/admin/news?user=uid&password=pwd",
				`#.FilteredParams | flat_values`: FlattenedResult{"uid", "pwd", "value 1"},                                // The multipart form data is not included for now
				`#.FilteredParams | flat_keys`:   FlattenedResult{"QueryForm", "user", "password", "PostForm", "field 1"}, // The multipart form data is not included for now
				`#.Body.String`:                  string(multipartExpectedBody),
				`#.Body.Bytes`:                   multipartExpectedBody,
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
				`#.FilteredParams | flat_keys`:   FlattenedResult{"PostForm", "z", "both", "prio", "", "orphan", "empty"},
				`#.Body.String`:                  `z=post&both=y&prio=2&=nokey&orphan;empty=&`,
				`#.Body.Bytes`:                   []byte(`z=post&both=y&prio=2&=nokey&orphan;empty=&`),
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
				`#.FilteredParams | flat_keys`:   FlattenedResult{"PostForm", "empty", "z", "both", "prio", "", "orphan", "QueryForm", "sqreen"},
				`#.Body.String`:                  `z=post&both=y&prio=2&=nokey&orphan;empty=&`,
				`#.Body.Bytes`:                   []byte(`z=post&both=y&prio=2&=nokey&orphan;empty=&`),
			},
		},
		{
			Title:  "Headers accesses",
			Method: "GET",
			URL:    "http://sqreen.com/",
			Headers: http.Header{
				"Content-Type":    []string{`application/x-www-form-urlencoded`},
				"X-Protected-By":  []string{"Sqreen", "ATF", "Julio"},
				"X-Forwarded-For": []string{"1.2.3.4,5.6.7.8"},
				"Rand-String":     []string{randString},
			},
			Body: strings.NewReader("z=post&both=y&prio=2&=nokey&orphan;empty=&"),
			BindingAccessors: map[string]interface{}{
				`#.Method`:                     "GET",
				`#.Host`:                       "sqreen.com",
				`#.ClientIP`:                   expectedClientIP,
				`#.Headers['Content-Type']`:    []string{`application/x-www-form-urlencoded`},
				`#.Headers['X-Protected-By']`:  []string{"Sqreen", "ATF", "Julio"},
				`#.Headers['X-Forwarded-For']`: []string{"1.2.3.4,5.6.7.8"},
				`#.Headers`: http.Header{
					"Content-Type":    []string{`application/x-www-form-urlencoded`},
					"X-Protected-By":  []string{"Sqreen", "ATF", "Julio"},
					"X-Forwarded-For": []string{"1.2.3.4,5.6.7.8"},
					"Rand-String":     []string{randString},
				},
				`#.Header['do not exist']`: nil,
				`#.Header['rand-STRING']`:  &randString,
				// The body should not be read by ParseForm when the request method is
				// GET
				`#.Body.String`: ``,
				`#.Body.Bytes`:  []byte(nil),
			},
		},

		{
			Title:  "Extra request params",
			Method: "GET",
			URL:    "http://sqreen.com/admin?user=uid&password=pwd",
			RequestParams: types.RequestParamMap{
				"json": types.RequestParamValueSlice{
					map[string]interface{}{
						"k1": 1,
						"k2": "2",
						"k3": []bool{true, false},
					},
				},
			},
			BindingAccessors: map[string]interface{}{
				`#.Method`:         "GET",
				`#.Host`:           "sqreen.com",
				`#.ClientIP`:       expectedClientIP,
				`#.URL.RequestURI`: "/admin?user=uid&password=pwd",
				`#.FilteredParams`: http_protection.RequestParamMap{
					"QueryForm": []interface{}{
						map[string][]string{
							"user":     {"uid"},
							"password": {"pwd"},
						},
					},
					"json": types.RequestParamValueSlice{
						[]interface{}{
							map[string]interface{}{
								"k1": 1,
								"k2": "2",
								"k3": []bool{true, false},
							},
						},
					},
				},
				`#.Body.String`: "",
				`#.Body.Bytes`:  []byte(nil),
			},
		},
	} {
		tc := tc
		t.Run(tc.Title, func(t *testing.T) {
			var rawBodyBuffer bytes.Buffer
			body := io.TeeReader(tc.Body, &rawBodyBuffer)
			req := httptest.NewRequest(tc.Method, tc.URL, body)
			for k, v := range tc.Headers {
				req.Header[k] = v
			}
			require.NoError(t, req.ParseForm())
			if req.Header.Get("Content-Type") == multipartContentTypeHeader {
				require.NoError(t, req.ParseMultipartForm(1024))
			}

			var rr types.RequestReader = requestReaderImpl{
				r:             req,
				clientIP:      expectedClientIP,
				rawBodyBuffer: rawBodyBuffer,
				requestParams: tc.RequestParams,
			}

			ctx := http_protection.NewRequestBindingAccessorContext(rr)

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
						require.ElementsMatch(t, flattened, value)
					} else {
						require.Equal(t, expected, value)
					}
				})
			}
		})
	}
}

type FlattenedResult []interface{}

type requestReaderImpl struct {
	r        *http.Request
	clientIP net.IP
	// The body read so far
	rawBodyBuffer bytes.Buffer
	requestParams types.RequestParamMap
}

func (r requestReaderImpl) Params() types.RequestParamMap {
	return r.requestParams
}

func (r requestReaderImpl) Body() []byte {
	return r.rawBodyBuffer.Bytes()
}

func (r requestReaderImpl) UserAgent() string {
	return r.r.UserAgent()
}

func (r requestReaderImpl) Referer() string {
	return r.r.Referer()
}

func (r requestReaderImpl) Header(header string) (value *string) {
	headers := r.r.Header
	if headers == nil {
		return nil
	}
	v := headers[textproto.CanonicalMIMEHeaderKey(header)]
	if len(v) == 0 {
		return nil
	}
	return &v[0]
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

func (r requestReaderImpl) QueryForm() url.Values {
	return r.r.URL.Query()
}

func (r requestReaderImpl) PostForm() url.Values {
	return r.r.PostForm
}

func (r requestReaderImpl) ClientIP() net.IP {
	return r.clientIP
}
