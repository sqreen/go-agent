package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			filename := newCfgFile(t, ".", envKey+`: `+someValue)
			defer os.Remove(filename)
			manager.ReadInConfig()
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

		New()
		token := BackendHTTPAPIToken()
		require.Equal(binDirToken, token)

		cwdToken := "cwd-token"
		cwdFile := newCfgFile(t, ".", `token: `+cwdToken)
		defer os.Remove(cwdFile)

		New()
		token = BackendHTTPAPIToken()
		require.Equal(cwdToken, token)

		tmpToken := "tmp-token"
		tmpDir := "./" + testlib.RandString(4)
		tmpFile := newCfgFile(t, tmpDir, `token: `+tmpToken)
		defer os.Remove(tmpFile)
		os.Setenv("SQREEN_CONFIG_FILE", tmpFile)
		New()
		token = BackendHTTPAPIToken()
		require.Equal(tmpToken, token)

		os.Unsetenv("SQREEN_CONFIG_FILE")
		New()
		token = BackendHTTPAPIToken()
		require.Equal(cwdToken, token)
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
			filename := newCfgFile(t, ".", envKey+`: `+someValue)
			defer os.Remove(filename)
			manager.ReadInConfig()
			require.Equal(t, getCfgValue(), someValue)
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
