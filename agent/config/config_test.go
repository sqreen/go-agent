package config

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestUserConfig(t *testing.T) {
	stringValueTests := []struct {
		Name         string
		GetCfgValue  func() string
		ConfigKey    string
		DefaultValue string
		SomeValue    string
	}{
		{
			Name:         "Backend HTTP API Base URL",
			GetCfgValue:  BackendHTTPAPIBaseURL,
			ConfigKey:    configKeyBackendHTTPAPIBaseURL,
			DefaultValue: configDefaultBackendHTTPAPIBaseURL,
			SomeValue:    "https://" + testlib.RandString(2, 50) + ":80806/is/cool",
		},
		{
			Name:        "Backend HTTP API Token",
			GetCfgValue: BackendHTTPAPIToken,
			ConfigKey:   configKeyBackendHTTPAPIToken,
			SomeValue:   testlib.RandString(2, 30),
		},
		{
			Name:         "Log Level",
			GetCfgValue:  LogLevel,
			ConfigKey:    configKeyLogLevel,
			DefaultValue: configDefaultLogLevel,
			SomeValue:    testlib.RandString(2, 30),
		},
		{
			Name:        "App Name",
			GetCfgValue: AppName,
			ConfigKey:   configKeyAppName,
			SomeValue:   testlib.RandString(2, 30),
		},
		{
			Name:        "IP Header",
			GetCfgValue: HTTPClientIPHeader,
			ConfigKey:   configKeyHTTPClientIPHeader,
			SomeValue:   testlib.RandString(2, 30),
		},
		{
			Name:        "Backend HTTP API Proxy",
			GetCfgValue: BackendHTTPAPIProxy,
			ConfigKey:   configKeyBackendHTTPAPIProxy,
			SomeValue:   testlib.RandString(2, 30),
		},
	}

	for _, tc := range stringValueTests {
		testStringValue(t, tc.Name, tc.GetCfgValue, tc.ConfigKey, tc.DefaultValue, tc.SomeValue)
	}

	t.Run("Disable", func(t *testing.T) {
		os.Setenv("SQREEN_TOKEN", testlib.RandString(2, 30))
		defer os.Unsetenv("SQREEN_TOKEN")

		getCfgValue := Disable
		defaultValue := false
		envKey := configKeyDisable
		someValue := testlib.RandString(2, 30)

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
			filename := newCfgFile(t, envKey+`: `+someValue)
			defer os.Remove(filename)
			viper.ReadInConfig()
			require.Equal(t, getCfgValue(), !defaultValue)
		})
	})
}

func testStringValue(t *testing.T, name string, getCfgValue func() string, envKey, defaultValue, someValue string) {
	t.Run(name, func(t *testing.T) {
		t.Run("Default value", func(t *testing.T) {
			require.Equal(t, getCfgValue(), defaultValue)
		})

		t.Run("Set through environment variable", func(t *testing.T) {
			envVar := strings.ToUpper(configEnvPrefix) + "_" + strings.ToUpper(envKey)
			os.Setenv(envVar, someValue)
			defer os.Unsetenv(envVar)
			require.Equal(t, getCfgValue(), someValue)
		})

		t.Run("Set through configuration file", func(t *testing.T) {
			filename := newCfgFile(t, envKey+`: `+someValue)
			defer os.Remove(filename)
			viper.ReadInConfig()
			require.Equal(t, getCfgValue(), someValue)
		})
	})
}

func newCfgFile(t *testing.T, content string) string {
	cfg, err := os.Create("sqreen.yml")
	require.Equal(t, err, nil)
	defer cfg.Close()
	_, err = cfg.WriteString(content)
	require.Equal(t, err, nil)
	return cfg.Name()
}
