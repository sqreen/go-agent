package internal

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
		} {
			tc := tc
			t.Run(tc.expected, func(t *testing.T) {
				t.Logf("%d %+v", i, tc)
				cfg := &GetClientIPConfigMockup{}
				defer cfg.AssertExpectations(t)
				cfg.On("HTTPClientIPHeader").Return("")

				req := newRequest(tc.remoteAddr)
				for k, v := range tc.extraHeaders {
					req.Header.Set(k, v)
				}

				ip := getClientIP(req, cfg)
				require.Equal(t, tc.expected, ip)
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
					cfg := &GetClientIPConfigMockup{}
					defer cfg.AssertExpectations(t)
					cfg.On("HTTPClientIPHeader").Return("x-uNiQue-iD")                                   // check it works even with a random case
					cfg.On("HTTPClientIPHeaderFormat").Return("it just needs to be set for now").Maybe() // depends on the testcase

					req := newRequest(tc.remoteAddr)
					req.Header.Set("X-Unique-Id", tc.uniqueID)
					for k, v := range tc.extraHeaders {
						req.Header.Set(k, v)
					}

					ip := getClientIP(req, cfg)
					require.Equal(t, tc.expected, ip)
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

type GetClientIPConfigMockup struct {
	mock.Mock
}

func (m *GetClientIPConfigMockup) HTTPClientIPHeader() string {
	ret := m.Called()
	return ret.String(0)
}

func (m *GetClientIPConfigMockup) HTTPClientIPHeaderFormat() string {
	ret := m.Called()
	return ret.String(0)
}
