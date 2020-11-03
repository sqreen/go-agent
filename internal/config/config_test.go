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

	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqsanitize"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestUserConfig(t *testing.T) {
	logger := plog.NewLogger(plog.Debug, os.Stderr, nil)
	cfg, unset := newTestConfig(t, logger)
	defer unset()

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
		{
			Name:          "Disable the agent",
			GetCfgValue:   cfg.Disabled,
			ConfigKey:     configKeyDisable,
			DefaultValue:  false,
			CfgValue:      testlib.RandUTF8String(1, 30),
			ExpectedValue: true,
		},
	}
	for _, tc := range boolValueTests {
		testBoolValue(t, cfg, tc.Name, tc.GetCfgValue, tc.ConfigKey, tc.DefaultValue, tc.CfgValue, tc.ExpectedValue)
	}
}

func TestConfigValidation(t *testing.T) {
	logger := plog.NewLogger(plog.Debug, os.Stderr, nil)

	t.Run("no token", func(t *testing.T) {
		cfg, err := New(logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("non-org token only is a valid config", func(t *testing.T) {
		cwdFile := newCfgFile(t, ".", `token: mytoken`)
		defer os.Remove(cwdFile)
		cfg, err := New(logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("org token without app name", func(t *testing.T) {
		cwdFile := newCfgFile(t, ".", `token: env_org_mytoken`)
		defer os.Remove(cwdFile)
		cfg, err := New(logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("bad sanitization key regexp", func(t *testing.T) {
		cwdFile := newCfgFile(t, ".", `token: mytoken
`+configKeyStripSensitiveKeyRegexp+`: oo(ps`)
		defer os.Remove(cwdFile)
		cfg, err := New(logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("bad sanitization value regexp", func(t *testing.T) {
		cwdFile := newCfgFile(t, ".", `token: mytoken
`+configKeyStripSensitiveValueRegexp+`: oo(ps`)
		defer os.Remove(cwdFile)
		cfg, err := New(logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}

func TestFileLocation(t *testing.T) {
	execFile, err := os.Executable()
	require.NoError(t, err)
	binDir := filepath.Dir(execFile)
	binDirToken := "exec-token"
	binDirFile := newCfgFile(t, binDir, `token: `+binDirToken)
	defer os.Remove(binDirFile)

	logger := plog.NewLogger(plog.Debug, os.Stderr, nil)
	cfg, err := New(logger)
	require.NoError(t, err)

	token := cfg.BackendHTTPAPIToken()
	require.Equal(t, binDirToken, token)

	cwdToken := "cwd-token"
	cwdFile := newCfgFile(t, ".", `token: `+cwdToken)
	defer os.Remove(cwdFile)

	cfg, err = New(logger)
	require.NoError(t, err)

	token = cfg.BackendHTTPAPIToken()
	require.Equal(t, cwdToken, token)

	tmpToken := "tmp-token"
	tmpDir := "./" + testlib.RandPrintableUSASCIIString(4)
	tmpFile := newCfgFile(t, tmpDir, `token: `+tmpToken)
	defer os.RemoveAll(tmpDir)
	os.Setenv("SQREEN_CONFIG_FILE", tmpFile)

	cfg, err = New(logger)
	require.NoError(t, err)

	token = cfg.BackendHTTPAPIToken()
	require.Equal(t, tmpToken, token)

	os.Unsetenv("SQREEN_CONFIG_FILE")

	cfg, err = New(logger)
	require.NoError(t, err)

	token = cfg.BackendHTTPAPIToken()
	require.Equal(t, cwdToken, token)
}

// Helper function return a valid default configuration
func newTestConfig(t *testing.T, logger *plog.Logger) (*Config, func()) {
	envVar := strings.ToUpper(configEnvPrefix) + "_" + strings.ToUpper(configKeyBackendHTTPAPIToken)
	os.Setenv(envVar, testlib.RandPrintableUSASCIIString(2, 30))
	cfg, err := New(logger)
	require.NoError(t, err)
	return cfg, func() {
		os.Unsetenv(envVar)
	}
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

func TestStripRegexp(t *testing.T) {
	t.Run("pii scrubbing default config", func(t *testing.T) {
		cwdFile := newCfgFile(t, ".", `token: mytoken`)
		defer os.Remove(cwdFile)

		logger := plog.NewLogger(plog.Debug, os.Stderr, nil)
		cfg, err := New(logger)

		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.NotNil(t, cfg.StripSensitiveKeyRegexp())
		require.NotNil(t, cfg.StripSensitiveValueRegexp())

		scrubber := sqsanitize.NewScrubber(cfg.StripSensitiveKeyRegexp(), cfg.StripSensitiveValueRegexp(), ScrubberRedactedString)

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

	t.Run("pii scrubbing disabled", func(t *testing.T) {
		cwdFile := newCfgFile(t, ".", `token: mytoken
`+configKeyStripSensitiveKeyRegexp+`: ""
`+configKeyStripSensitiveValueRegexp+`: ""`)
		defer os.Remove(cwdFile)

		logger := plog.NewLogger(plog.Debug, os.Stderr, nil)
		cfg, err := New(logger)

		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Nil(t, cfg.StripSensitiveKeyRegexp())
		require.Nil(t, cfg.StripSensitiveValueRegexp())
	})
}

func TestDefaultConfiguration(t *testing.T) {
	cwdFile := newCfgFile(t, ".", `token: mytoken`)
	defer os.Remove(cwdFile)

	logger := plog.NewLogger(plog.Debug, os.Stderr, nil)
	cfg, err := New(logger)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.StripSensitiveKeyRegexp())
	require.NotNil(t, cfg.StripSensitiveValueRegexp())
}

func TestValidateAppCredentialsConfiguration(t *testing.T) {
	for _, tc := range []struct {
		Name, Token, AppName string
		ShouldFail           bool
	}{
		{
			Name:    "valid org token strings",
			Token:   "org_ok",
			AppName: "ok",
		},
		{
			Name:  "valid non-org",
			Token: "ok",
		},
		{
			Name:       "invalid credentials with empty strings",
			Token:      "",
			AppName:    "",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with empty token and non-empty app-name",
			Token:      "",
			AppName:    "ok",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with valid org token but empty app-name",
			Token:      "org_ok",
			AppName:    "",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "org_ok",
			AppName:    "ko\nko",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "org_ok",
			AppName:    "koko\x00\x01\x02ok",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "org_ok",
			AppName:    "koko\tok",
			ShouldFail: true,
		},
		{
			Name:    "valid credentials with a space in app-name",
			Token:   "org_ok",
			AppName: "ok ok ok",
		},
		{
			Name:       "invalid credentials with invalid token character",
			Token:      "org_ok\nko",
			AppName:    "ok",
			ShouldFail: true,
		},


		{
			Name:       "invalid credentials with valid org token but empty app-name",
			Token:      "env_org_ok",
			AppName:    "",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "env_org_ok",
			AppName:    "ko\nko",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "env_org_ok",
			AppName:    "koko\x00\x01\x02ok",
			ShouldFail: true,
		},
		{
			Name:       "invalid credentials with token ok but invalid app-name",
			Token:      "env_org_ok",
			AppName:    "koko\tok",
			ShouldFail: true,
		},
		{
			Name:    "valid credentials with a space in app-name",
			Token:   "env_org_ok",
			AppName: "ok ok ok",
		},
		{
			Name:       "invalid credentials with invalid token character",
			Token:      "env_org_ok\nko",
			AppName:    "ok",
			ShouldFail: true,
		},
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			err := validateAppCredentials(tc.Token, tc.AppName)
			if tc.ShouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
