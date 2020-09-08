// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ResponseWriterMockup struct {
	mock.Mock
}

func (r *ResponseWriterMockup) Header() http.Header {
	h, _ := r.Called().Get(0).(http.Header)
	return h
}

func (r *ResponseWriterMockup) Write(bytes []byte) (int, error) {
	ret := r.Called(bytes)
	return ret.Int(0), ret.Error(1)
}

func (r *ResponseWriterMockup) WriteHeader(statusCode int) {
	r.Called(statusCode)
}

func (r *ResponseWriterMockup) WriteString(s string) (n int, err error) {
	ret := r.Called(s)
	return ret.Int(0), ret.Error(1)
}

type RequestReaderMockup struct {
	mock.Mock
}

func (r *RequestReaderMockup) Body() []byte {
	value, _ := r.Called().Get(0).([]byte)
	return value
}

func (r *RequestReaderMockup) Header(header string) (value *string) {
	value, _ = r.Called(header).Get(0).(*string)
	return value
}

func (r *RequestReaderMockup) Headers() http.Header {
	h, _ := r.Called().Get(0).(http.Header)
	return h
}

func (r *RequestReaderMockup) Method() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) URL() *url.URL {
	u, _ := r.Called().Get(0).(*url.URL)
	return u
}

func (r *RequestReaderMockup) ExpectURL() *mock.Call {
	return r.On("URL")
}

func (r *RequestReaderMockup) RequestURI() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) Host() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) RemoteAddr() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) IsTLS() bool {
	return r.Called().Bool(0)
}

func (r *RequestReaderMockup) UserAgent() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) Referer() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) QueryForm() url.Values {
	v, _ := r.Called().Get(0).(url.Values)
	return v
}

func (r *RequestReaderMockup) PostForm() url.Values {
	v, _ := r.Called().Get(0).(url.Values)
	return v
}

func (r *RequestReaderMockup) ClientIP() net.IP {
	ip, _ := r.Called().Get(0).(net.IP)
	return ip
}

func (r *RequestReaderMockup) ExpectClientIP() *mock.Call {
	return r.On("ClientIP")
}

func (r *RequestReaderMockup) Params() types.RequestParamMap {
	v, _ := r.Called().Get(0).(types.RequestParamMap)
	return v
}

func TestProtectionAPI(t *testing.T) {
	t.Run("passlists", func(t *testing.T) {
		t.Run("ip allowed", func(t *testing.T) {
			agentMockup := &mockups.AgentMockup{}
			defer agentMockup.AssertExpectations(t)

			responseWriterMockup := &ResponseWriterMockup{}
			defer responseWriterMockup.AssertExpectations(t)

			requestReaderMockup := &RequestReaderMockup{}
			defer requestReaderMockup.AssertExpectations(t)
			ip := net.ParseIP("1.2.3.4")
			requestReaderMockup.ExpectClientIP().Return(ip)
			agentMockup.ExpectIsIPAllowed(ip).Return(true)

			// The request path is not passlisted
			u, err := url.Parse("https://test.com/foo/bar/")
			require.NoError(t, err)
			requestReaderMockup.ExpectURL().Return(u)
			agentMockup.ExpectIsPathAllowed(u.Path).Return(false)

			ctx, _, _ := NewRequestContext(context.Background(), agentMockup, responseWriterMockup, requestReaderMockup)
			require.Nil(t, ctx)
		})

		t.Run("path allowed", func(t *testing.T) {
			agentMockup := &mockups.AgentMockup{}
			defer agentMockup.AssertExpectations(t)

			responseWriterMockup := &ResponseWriterMockup{}
			defer responseWriterMockup.AssertExpectations(t)

			requestReaderMockup := &RequestReaderMockup{}
			defer requestReaderMockup.AssertExpectations(t)

			// The request path is allowed
			u, err := url.Parse("https://test.com/foo/bar/")
			require.NoError(t, err)
			requestReaderMockup.ExpectURL().Return(u)
			agentMockup.ExpectIsPathAllowed(u.Path).Return(true)

			ctx, _, _ := NewRequestContext(context.Background(), agentMockup, responseWriterMockup, requestReaderMockup)
			require.Nil(t, ctx)
		})

		t.Run("none allowed", func(t *testing.T) {
			agentMockup := &mockups.AgentMockup{}
			defer agentMockup.AssertExpectations(t)

			responseWriterMockup := &ResponseWriterMockup{}
			defer responseWriterMockup.AssertExpectations(t)

			requestReaderMockup := &RequestReaderMockup{}
			defer requestReaderMockup.AssertExpectations(t)

			// The IP is not allowed
			ip := net.ParseIP("1.2.3.4")
			requestReaderMockup.ExpectClientIP().Return(ip)
			agentMockup.ExpectIsIPAllowed(ip).Return(false)

			// The request path is not allowed
			u, err := url.Parse("https://test.com/foo/bar/")
			require.NoError(t, err)
			requestReaderMockup.ExpectURL().Return(u)
			agentMockup.ExpectIsPathAllowed(u.Path).Return(false)

			ctx, reqCtx, cancel := NewRequestContext(context.Background(), agentMockup, responseWriterMockup, requestReaderMockup)

			require.NotNil(t, ctx)
			require.NotNil(t, reqCtx)
			require.NotNil(t, cancel)
		})
	})

	// TODO: more test cases
}

func TestParseClientIPHeaderHeaderValue(t *testing.T) {
	// Tests with malformed values
	// A buffer of random bytes.
	var randBuf []byte
	fuzz.New().NilChance(0).NumElements(1, 10000).Fuzz(&randBuf)
	require.NotEmpty(t, randBuf)
	// A random IPv4 value without the expected `:` separator
	randIPv4 := fmt.Sprintf("%X", []byte(RandIPv4()))

	for _, tc := range []string{
		"",
		string(randBuf),
		string(randBuf) + ":",
		randIPv4,
	} {
		tc := tc // new scope
		t.Run("malformed", func(t *testing.T) {
			_, err := parseClientIPHeaderHeaderValue("", tc)
			require.Error(t, err)
		})
	}
}

func TestGetClientIP(t *testing.T) {
	newRequest := func(remoteAddr string) *http.Request {
		header := make(http.Header)
		return &http.Request{
			RemoteAddr: remoteAddr,
			Header:     header,
		}
	}

	t.Run("Without prioritized header", func(t *testing.T) {
		globalIP := RandGlobalIPv4()
		require.True(t, isGlobal(globalIP))
		require.False(t, isPrivate(globalIP))

		privateIP := RandPrivateIPv4()
		require.False(t, isGlobal(privateIP))
		require.True(t, isPrivate(privateIP))

		for i, tc := range []struct {
			expected, remoteAddr string
			extraHeaders         map[string]string
		}{
			// Only a private IP in remote address
			{expected: privateIP.String(), remoteAddr: privateIP.String()},
			// Only a global IP in remote address
			{expected: globalIP.String(), remoteAddr: globalIP.String()},
			// Global IP in XFF
			{
				expected:   globalIP.String(),
				remoteAddr: RandPrivateIPv4().String(),
				extraHeaders: map[string]string{
					"X-Forwarded-For": globalIP.String() + "," + RandIPv4().String() + "," + RandIPv4().String(),
				},
			},
			// Private IPs everywhere.
			{
				expected:   privateIP.String(),
				remoteAddr: RandPrivateIPv4().String(),
				extraHeaders: map[string]string{
					"X-Forwarded-For": privateIP.String() + "," + RandPrivateIPv4().String() + "," + RandPrivateIPv4().String(),
				},
			},
			// Private IPs everywhere but in the remote addr.
			{
				expected:   globalIP.String(),
				remoteAddr: globalIP.String(),
				extraHeaders: map[string]string{
					"X-Forwarded-For": RandPrivateIPv4().String() + "," + RandPrivateIPv4().String() + "," + RandPrivateIPv4().String(),
				},
			},
			// Global IP in the middle of XFF and private IPs everywhere else.
			{
				expected:   globalIP.String(),
				remoteAddr: RandPrivateIPv4().String(),
				extraHeaders: map[string]string{
					"X-Forwarded-For": RandPrivateIPv4().String() + "," + RandPrivateIPv4().String() + "," + globalIP.String() + "," + RandPrivateIPv4().String(),
				},
			},

			{
				expected:   "152.23.231.25",
				remoteAddr: "127.0.0.1",
				extraHeaders: map[string]string{
					"X-Forwarded-For": "127.0.0.1, 152.23.231.25:98746, 10.1.2.3, 152.23.231.29, 8.8.8.8",
				},
			},
		} {
			tc := tc
			t.Run(tc.expected, func(t *testing.T) {
				t.Logf("%d %+v", i, tc)

				req := newRequest(tc.remoteAddr)
				for k, v := range tc.extraHeaders {
					req.Header.Set(k, v)
				}

				ip := ClientIP(req.RemoteAddr, req.Header, "", "")
				require.Equal(t, tc.expected, ip.String())
			})
		}
	})

	t.Run("Custom IP header format", func(t *testing.T) {
		t.Run("HA Proxy X-Unique-Id", func(t *testing.T) {
			uniqueIDGlobalIP, uniqueID := RandHAProxyUniqueID()
			require.True(t, isGlobal(uniqueIDGlobalIP))
			require.False(t, isPrivate(uniqueIDGlobalIP))

			globalIP := RandGlobalIPv4()
			require.True(t, isGlobal(globalIP))
			require.False(t, isPrivate(globalIP))

			privateIP := RandPrivateIPv4()
			require.False(t, isGlobal(privateIP))
			require.True(t, isPrivate(privateIP))

			for i, tc := range []struct {
				expected, remoteAddr, uniqueID string
				extraHeaders                   map[string]string
			}{
				// Empty X-Unique-Id value
				{expected: "127.0.0.1", remoteAddr: "127.0.0.1", uniqueID: ""},
				// Global IP in X-Unique-Id
				{expected: uniqueIDGlobalIP.String(), remoteAddr: "127.0.0.1", uniqueID: uniqueID},
				// Global IP in X-Unique-Id and XFF, but X-Unique-Id is prioritized by the config
				{
					expected:   uniqueIDGlobalIP.String(),
					remoteAddr: "127.0.0.1",
					uniqueID:   uniqueID,
					extraHeaders: map[string]string{
						"X-Forwarded-For": globalIP.String() + "," + RandIPv4().String() + "," + RandIPv4().String(),
					},
				},
				// Private IP in X-Unique-Id which is is prioritized by the config but XFF has a global IP
				{
					expected:   globalIP.String(),
					remoteAddr: "127.0.0.1",
					uniqueID:   HAProxyUniqueID(privateIP),
					extraHeaders: map[string]string{
						"X-Forwarded-For": globalIP.String() + "," + RandIPv4().String() + "," + RandIPv4().String(),
					},
				},
				// Private IPs everywhere.
				{
					expected:   privateIP.String(),
					remoteAddr: RandPrivateIPv4().String(),
					uniqueID:   HAProxyUniqueID(privateIP),
					extraHeaders: map[string]string{
						"X-Forwarded-For": RandPrivateIPv4().String() + "," + RandPrivateIPv4().String() + "," + RandPrivateIPv4().String(),
					},
				},
				// Private IPs everywhere but in the remote addr.
				{
					expected:   globalIP.String(),
					remoteAddr: globalIP.String(),
					uniqueID:   HAProxyUniqueID(privateIP),
					extraHeaders: map[string]string{
						"X-Forwarded-For": RandPrivateIPv4().String() + "," + RandPrivateIPv4().String() + "," + RandPrivateIPv4().String(),
					},
				},
			} {
				tc := tc
				t.Run(tc.expected, func(t *testing.T) {
					t.Logf("%d %+v", i, tc)

					req := newRequest(tc.remoteAddr)
					req.Header.Set("X-Unique-Id", tc.uniqueID)
					for k, v := range tc.extraHeaders {
						req.Header.Set(k, v)
					}

					ip := ClientIP(req.RemoteAddr, req.Header, "x-uNiQue-iD", "it just needs to be set for now")
					require.Equal(t, tc.expected, ip.String())
				})
			}
		})
	})
}

func RandIPv4() net.IP {
	return net.IPv4(uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()))
}

func RandGlobalIPv4() net.IP {
	for {
		ip := RandIPv4()
		if isGlobal(ip) && !isPrivate(ip) {
			return ip
		}
	}
}

func RandPrivateIPv4() net.IP {
	for {
		ip := RandIPv4()
		if !isGlobal(ip) && isPrivate(ip) {
			return ip
		}
	}
}

func RandHAProxyUniqueID() (net.IP, string) {
	ip := RandGlobalIPv4()
	return ip, HAProxyUniqueID(ip)
}

func HAProxyUniqueID(ip net.IP) string {
	var randStr string
	fuzz.New().NilChance(0).Fuzz(&randStr)
	value := fmt.Sprintf("%X:%s", []byte(ip.To4()), randStr)
	return value
}
