// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/internal/sqlib/sqsanitize"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestUserConfig(t *testing.T) {
	logger := plog.NewLogger(plog.Debug, os.Stderr, 0)
	cfg := New(logger)

	stringValueTests := []struct {
		Name         string
		GetCfgValue  func() string
		ConfigKey    string
		DefaultValue string
		SomeValue    string
	}{
		{
			Name:         "Backend HTTP API Base URL",
			GetCfgValue:  cfg.BackendHTTPAPIBaseURL,
			ConfigKey:    configKeyBackendHTTPAPIBaseURL,
			DefaultValue: configDefaultBackendHTTPAPIBaseURL,
			SomeValue:    testlib.RandUTF8String(2, 50),
		},
		{
			Name:        "Backend HTTP API Token",
			GetCfgValue: cfg.BackendHTTPAPIToken,
			ConfigKey:   configKeyBackendHTTPAPIToken,
			SomeValue:   testlib.RandUTF8String(2, 30),
		},
		{
			Name:        "App Name",
			GetCfgValue: cfg.AppName,
			ConfigKey:   configKeyAppName,
			SomeValue:   testlib.RandUTF8String(2, 30),
		},
		{
			Name:        "IP Header",
			GetCfgValue: cfg.HTTPClientIPHeader,
			ConfigKey:   configKeyHTTPClientIPHeader,
			SomeValue:   testlib.RandUTF8String(2, 30),
		},
		{
			Name:        "Backend HTTP API Proxy",
			GetCfgValue: cfg.BackendHTTPAPIProxy,
			ConfigKey:   configKeyBackendHTTPAPIProxy,
			SomeValue:   testlib.RandUTF8String(2, 30),
		},
	}
	for _, tc := range stringValueTests {
		testStringValue(t, cfg, tc.Name, tc.GetCfgValue, tc.ConfigKey, tc.DefaultValue, tc.SomeValue)
	}

	// The reflect package could be used instead
	boolValueTests := []struct {
		Name          string
		GetCfgValue   func() bool
		ConfigKey     string
		DefaultValue  bool
		CfgValue      string
		ExpectedValue bool
	}{
		{
			Name:          "Strip the Referer HTTP Header",
			GetCfgValue:   cfg.StripHTTPReferer,
			ConfigKey:     configKeyStripHTTPReferer,
			DefaultValue:  false,
			CfgValue:      testlib.RandUTF8String(1, 30),
			ExpectedValue: true,
		},
	}
	for _, tc := range boolValueTests {
		testBoolValue(t, cfg, tc.Name, tc.GetCfgValue, tc.ConfigKey, tc.DefaultValue, tc.CfgValue, tc.ExpectedValue)
	}

	// The disable which is a special config case which also depends on the sqreen
	// token value.
	t.Run("Disable", func(t *testing.T) {
		os.Setenv("SQREEN_TOKEN", testlib.RandUTF8String(2, 30))
		defer os.Unsetenv("SQREEN_TOKEN")

		getCfgValue := cfg.Disable
		defaultValue := false
		envKey := configKeyDisable
		someValue := testlib.RandUTF8String(2, 30)

		t.Run("Default value", func(t *testing.T) {
			require.Equal(t, getCfgValue(), defaultValue)
		})

		t.Run("Set through environment variable", func(t *testing.T) {
			envVar := strings.ToUpper(configEnvPrefix) + "_" + strings.ToUpper(envKey)
			os.Setenv(envVar, someValue)
			defer os.Unsetenv(envVar)
			require.NotEqual(t, getCfgValue(), defaultValue)
		})

		t.Run("Set through configuration file", func(t *testing.T) {
			filename := newCfgFile(t, ".", envKey+`: `+someValue)
			defer os.Remove(filename)
			require.NoError(t, cfg.ReadInConfig())
			require.Equal(t, getCfgValue(), !defaultValue)
		})
	})

	t.Run("File location", func(t *testing.T) {
		require := require.New(t)

		execFile, err := os.Executable()
		require.NoError(err)
		binDir := filepath.Dir(execFile)
		binDirToken := "exec-token"
		binDirFile := newCfgFile(t, binDir, `token: `+binDirToken)
		defer os.Remove(binDirFile)

		cfg := New(logger)
		token := cfg.BackendHTTPAPIToken()
		require.Equal(binDirToken, token)

		cwdToken := "cwd-token"
		cwdFile := newCfgFile(t, ".", `token: `+cwdToken)
		defer os.Remove(cwdFile)

		cfg = New(logger)
		token = cfg.BackendHTTPAPIToken()
		require.Equal(cwdToken, token)

		tmpToken := "tmp-token"
		tmpDir := "./" + testlib.RandPrintableUSASCIIString(4)
		tmpFile := newCfgFile(t, tmpDir, `token: `+tmpToken)
		defer os.Remove(tmpFile)
		os.Setenv("SQREEN_CONFIG_FILE", tmpFile)
		cfg = New(logger)
		token = cfg.BackendHTTPAPIToken()
		require.Equal(tmpToken, token)

		os.Unsetenv("SQREEN_CONFIG_FILE")
		cfg = New(logger)
		token = cfg.BackendHTTPAPIToken()
		require.Equal(cwdToken, token)
	})
}

func testStringValue(t *testing.T, cfg *Config, name string, getCfgValue func() string, envKey, defaultValue, someValue string) {
	t.Run(name, func(t *testing.T) {
		t.Run("Default value", func(t *testing.T) {
			require.Equal(t, defaultValue, getCfgValue())
		})

		t.Run("Set through environment variable", func(t *testing.T) {
			envVar := strings.ToUpper(configEnvPrefix) + "_" + strings.ToUpper(envKey)
			os.Setenv(envVar, someValue)
			defer os.Unsetenv(envVar)
			require.Equal(t, someValue, getCfgValue())
		})

		t.Run("Set through configuration file", func(t *testing.T) {
			filename := newCfgFile(t, ".", envKey+`: `+someValue)
			defer os.Remove(filename)
			require.NoError(t, cfg.ReadInConfig())
			require.Equal(t, someValue, getCfgValue())
		})
	})
}

func testBoolValue(t *testing.T, cfg *Config, name string, getCfgValue func() bool, envKey string, defaultValue bool, cfgValue string, expectedValue bool) {
	t.Run(name, func(t *testing.T) {
		t.Run("Default value", func(t *testing.T) {
			require.Equal(t, getCfgValue(), defaultValue)
		})

		t.Run("Set through environment variable", func(t *testing.T) {
			envVar := strings.ToUpper(configEnvPrefix) + "_" + strings.ToUpper(envKey)
			os.Setenv(envVar, cfgValue)
			defer os.Unsetenv(envVar)
			require.Equal(t, getCfgValue(), expectedValue)
		})

		t.Run("Set through configuration file", func(t *testing.T) {
			filename := newCfgFile(t, ".", envKey+`: `+strconv.Quote(cfgValue))
			defer os.Remove(filename)
			require.NoError(t, cfg.ReadInConfig())
			require.Equal(t, getCfgValue(), expectedValue)
		})
	})
}

func newCfgFile(t *testing.T, path string, content string) string {
	os.MkdirAll(path, 0700)
	cfg, err := os.Create(path + "/sqreen.yml")
	require.NoError(t, err)
	defer cfg.Close()
	_, err = cfg.WriteString(content)
	require.NoError(t, err)
	return cfg.Name()
}

func TestDefaultConfiguration(t *testing.T) {
	t.Run("pii scrubbing default config", func(t *testing.T) {
		scrubber, err := sqsanitize.NewScrubber(ScrubberKeyRegexp, ScrubberValueRegexp, ScrubberRedactedString)
		require.NoError(t, err)

		t.Run("the key regexp should match", func(t *testing.T) {
			for _, key := range []string{
				"passwd", "password", "passphrase", "secret", "authorization", "api_key",
				"apikey", "accesstoken", "access_token", "token",
			} {
				key := key
				t.Run(key, func(t *testing.T) {
					v := map[string]string{
						key: testlib.RandUTF8String(),
					}
					scrubbed, err := scrubber.Scrub(&v, nil)
					require.NoError(t, err)
					require.True(t, scrubbed)
					require.Equal(t, ScrubberRedactedString, v[key])
				})
			}
		})

		t.Run("the value regexp should match", func(t *testing.T) {
			for _, value := range []string{
				"0000-1111-2222-3333", "9999888877776666", "0000 1111 2222 3333",
			} {
				value := value
				t.Run(value, func(t *testing.T) {
					v := value
					scrubbed, err := scrubber.Scrub(&v, nil)
					require.NoError(t, err)
					require.True(t, scrubbed)
					require.Equal(t, ScrubberRedactedString, v)
				})
			}
		})
	})
}
