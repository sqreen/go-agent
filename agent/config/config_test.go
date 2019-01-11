package config

import (
	"math/rand"
	"os"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var lock sync.Mutex

var _ = Describe("Config", func() {
	DescribeTable("User-configurable values",
		func(getCfgValue func() string, envKey, defaultValue, someValue string) {
			// Specs are run in parallel and this test is not concurrency-safe because of
			// shared env vars and the shared configuration file. This global mutex allows
			// to enforce one test at a time to execute.
			lock.Lock()
			defer lock.Unlock()

			By("default")
			Expect(getCfgValue()).To(Equal(defaultValue))

			By("environment variable")
			envVar := strings.ToUpper(configEnvPrefix) + "_" + strings.ToUpper(envKey)
			os.Setenv(envVar, someValue)
			defer os.Unsetenv(envVar)
			Expect(getCfgValue()).To(Equal(someValue))

			By("configuration file")
			filename := newCfgFile(`url: ` + someValue)
			defer os.Remove(filename)
			viper.ReadInConfig()
			Expect(getCfgValue()).To(Equal(someValue))
		},
		Entry("Backend HTTP API Base URL", BackendHTTPAPIBaseURL, configKeyBackendHTTPAPIBaseURL, configDefaultBackendHTTPAPIBaseURL, "https://"+randString(2+rand.Intn(50))+":80806/is/cool"),
		Entry("Backend HTTP API Token", BackendHTTPAPIToken, configKeyBackendHTTPAPIToken, "", randString(2+rand.Intn(30))),
		Entry("Log Level", LogLevel, configKeyLogLevel, configDefaultLogLevel, randString(2+rand.Intn(30))),
		Entry("App Name", AppName, configKeyAppName, "", randString(2+rand.Intn(30))),
		Entry("IP Header", HTTPClientIPHeader, configKeyHTTPClientIPHeader, "", randString(2+rand.Intn(30))),
	)
})

func newCfgFile(content string) string {
	cfg, err := os.Create("sqreen.yml")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	_, err = cfg.WriteString(content)
	Expect(err).NotTo(HaveOccurred())
	return cfg.Name()
}

func randString(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
