package internal

import (
	"crypto/rand"
	"fmt"
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

		for _, tc := range []struct {
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
			// Globa IP in the middle of XFF and private IPs everywhere else.
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
}

func RandIPv4() net.IP {
	ip := make([]byte, net.IPv4len)
	rand.Read(ip)
	return net.IP(ip)
}

func RandGlobalIPv4() net.IP {
	for {
		ip := RandIPv4()
		if isGlobal(ip) {
			return ip
		}
	}
}

func RandPrivateIPv4() net.IP {
	for {
		ip := RandIPv4()
		if isPrivate(ip) {
			return ip
		}
	}
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
