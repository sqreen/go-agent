// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/internal/event"
	http_protection_mockups "github.com/sqreen/go-agent/internal/protection/http/_testlib/mockups"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	middleware_mockups "github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProtectionAPI(t *testing.T) {
	newMockups := func(t *testing.T, ip net.IP, allowPath, allowIP bool) (*middleware_mockups.RootHTTPProtectionContextMockup, *middleware_mockups.HTTPProtectionConfigMockup, *http_protection_mockups.RequestReaderMockup, *http_protection_mockups.ResponseWriterMockup) {
		r := &middleware_mockups.RootHTTPProtectionContextMockup{}

		cfg := middleware_mockups.NewHTTPProtectionConfigMockup()

		responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}

		requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}

		// Path passlist
		u, err := url.Parse("https://test.com/foo/bar/")
		require.NoError(t, err)
		requestReaderMockup.ExpectURL().Return(u)
		r.ExpectIsPathAllowed(u.Path).Return(allowPath)

		if !allowPath {
			r.ExpectConfig().Return(cfg)

			// The following calls only happen when the path is not passlisted
			// Make ClientIP return a given IP address
			requestReaderMockup.ExpectRemoteAddr().Return(ip.String())
			requestReaderMockup.ExpectHeaders().Return(nil)

			// IP passlist
			r.ExpectIsIPAllowed(ip).Return(allowIP)
		}

		return r, cfg, requestReaderMockup, responseWriterMockup
	}

	t.Run("passlists", func(t *testing.T) {
		t.Run("ip allowed", func(t *testing.T) {
			r, cfg, req, w := newMockups(t, net.ParseIP("1.2.3.4"), false, true)
			defer r.AssertExpectations(t)
			defer cfg.AssertExpectations(t)
			defer req.AssertExpectations(t)
			defer w.AssertExpectations(t)
			p := NewProtectionContext(r, w, req)
			require.Nil(t, p)
		})

		t.Run("path allowed", func(t *testing.T) {
			r, cfg, req, w := newMockups(t, net.ParseIP("1.2.3.4"), true, false)
			defer r.AssertExpectations(t)
			defer cfg.AssertExpectations(t)
			defer req.AssertExpectations(t)
			defer w.AssertExpectations(t)
			p := NewProtectionContext(r, w, req)
			require.Nil(t, p)
		})

		t.Run("none allowed", func(t *testing.T) {
			ip := net.ParseIP("1.2.3.4")
			r, cfg, req, w := newMockups(t, ip, false, false)
			defer r.AssertExpectations(t)
			defer cfg.AssertExpectations(t)
			defer req.AssertExpectations(t)
			defer w.AssertExpectations(t)

			p := NewProtectionContext(r, w, req)
			require.NotNil(t, p)
		})
	})

	t.Run("nil root context", func(t *testing.T) {
		w := &http_protection_mockups.ResponseWriterMockup{}
		req := &http_protection_mockups.RequestReaderMockup{}
		p := NewProtectionContext(nil, w, req)
		require.Nil(t, p)
	})

	t.Run("protection/callback api", func(t *testing.T) {
		ip := net.ParseIP("1.2.3.4")
		r, cfg, req, w := newMockups(t, ip, false, false)
		defer cfg.AssertExpectations(t)
		defer req.AssertExpectations(t)
		defer w.AssertExpectations(t)

		p := NewProtectionContext(r, w, req)
		require.NotNil(t, p)

		require.Equal(t, ip, p.ClientIP())

		// Handle a non-blocking attack
		blocked := p.HandleAttack(false, &event.AttackEvent{})
		require.False(t, blocked)
		r.AssertExpectations(t)

		// Handle a blocking attack
		r.ExpectCancelContext().Once() // the context should be closed
		blocked = p.HandleAttack(true, &event.AttackEvent{})
		require.True(t, blocked)
		r.AssertExpectations(t)

		// Handle a blocking attack without logging an attack
		r.ExpectCancelContext().Once() // the context should be closed
		blocked = p.HandleAttack(true, nil)
		require.True(t, blocked)
		r.AssertExpectations(t)

		// Fake response
		response := &http_protection_mockups.ResponseMockup{}
		response.ExpectStatus().Return(433)
		response.ExpectContentType().Return("sqreen/test")
		response.ExpectContentLength().Return(int64(4321))

		// Fake request
		req.ExpectMethod().Return("GET")
		u, _ := url.Parse("http://test.com/")
		req.ExpectURL().Return(u)
		req.ExpectRequestURI().Return(u.RequestURI())
		req.ExpectHost().Return(u.Host)
		req.ExpectIsTLS().Return(false)
		req.ExpectUserAgent().Return("ua")
		req.ExpectReferer().Return("referer")
		req.ExpectQueryForm().Return(nil)
		req.ExpectPostForm().Return(nil)
		req.ExpectParams().Return(nil)

		// Close the protection context
		r.ExpectClose(mock.MatchedBy(func(closed types.ClosedProtectionContextFace) bool {
			events := closed.Events()
			require.Len(t, events.AttackEvents, 2)
			return true
		}))
		p.Close(response)
		r.AssertExpectations(t)
	})
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

		globalIPv6 := RandGlobalIPv6()
		require.True(t, isGlobal(globalIPv6))
		require.False(t, isPrivate(globalIPv6))

		privateIPv6 := RandPrivateIPv6()
		require.False(t, isGlobal(privateIPv6))
		require.True(t, isPrivate(privateIPv6))

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

			{
				expected:   "2409:aaaa:210c:3333:5fa9:6ecb:7366:4444",
				remoteAddr: "127.0.0.1",
				extraHeaders: map[string]string{
					"X-Forwarded-For": "2409:aaaa:210c:3333:5fa9:6ecb:7366:4444",
				},
			},

			{
				expected:   "2409:aaaa:210c:3333:5fa9:6ecb:7366:4444",
				remoteAddr: "2409:aaaa:210c:3333:5fa9:6ecb:7366:4444",
				extraHeaders: map[string]string{
					"X-Forwarded-For": "127.0.0.1",
				},
			},

			{
				expected:   "152.23.231.25",
				remoteAddr: "127.0.0.1",
				extraHeaders: map[string]string{
					"X-Forwarded-For": "127.0.0.1, 152.23.231.25:98746, 10.1.2.3, 2409:aaaa:210c:3333:5fa9:6ecb:7366:4444, 8.8.8.8",
				},
			},

			{
				expected:   "2409:aaaa:210c:3333:5fa9:6ecb:7366:4444",
				remoteAddr: "127.0.0.1",
				extraHeaders: map[string]string{
					"X-Forwarded-For": "127.0.0.1, 2409:aaaa:210c:3333:5fa9:6ecb:7366:4444, 10.1.2.3, 152.23.231.29:2793, 8.8.8.8",
				},
			},

			{
				expected:   "2409:aaaa:210c:3333:5fa9:6ecb:7366:4444",
				remoteAddr: "127.0.0.1",
				extraHeaders: map[string]string{
					"X-Forwarded-For": "127.0.0.1, [2409:aaaa:210c:3333:5fa9:6ecb:7366:4444]:3333, 10.1.2.3, 152.23.231.29:2793, 8.8.8.8",
				},
			},

			{
				expected:   "2409:aaaa:210c:3333:5fa9:6ecb:7366:4444",
				remoteAddr: "127.0.0.1",
				extraHeaders: map[string]string{
					"X-Forwarded-For": "127.0.0.1, 2409:aaaa:210c:3333:5fa9:6ecb:7366:4444:3333, 10.1.2.3, 152.23.231.29:2793, 8.8.8.8",
				},
			},

			{
				expected:   globalIPv6.String(),
				remoteAddr: RandPrivateIPv6().String(),
				extraHeaders: map[string]string{
					"X-Forwarded-For": RandPrivateIPv6().String() + "," + RandPrivateIPv6().String() + "," + globalIPv6.String() + "," + RandPrivateIPv6().String(),
				},
			},

			{
				expected:   globalIPv6.String(),
				remoteAddr: RandPrivateIPv6().String(),
				extraHeaders: map[string]string{
					"X-Forwarded-For": globalIPv6.String() + "," + globalIP.String() + "," + privateIPv6.String() + "," + privateIP.String(),
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

func RandIPv6() net.IP {
	return net.IP{
		uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()),
		uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()),
		uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()),
		uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()),
	}
}

func RandGlobalIPv4() net.IP {
	for {
		ip := RandIPv4()
		if isGlobal(ip) && !isPrivate(ip) {
			return ip
		}
	}
}

func RandGlobalIPv6() net.IP {
	for {
		ip := RandIPv6()
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

func RandPrivateIPv6() net.IP {
	for {
		ip := RandIPv6()
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
