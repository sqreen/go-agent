package plog_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/sqreen/go-agent/agent/internal/plog"
)

var _ = Describe("plog", func() {
	Describe("a logger", func() {
		var (
			logger *plog.Logger
			re     = "ns:%s: [0-9]{4}(/[0-9]{2}){2} ([0-9]{2}:){2}[0-9]{2}.[0-9]{6} %[1]s"
		)

		JustBeforeEach(func() {
			logger = plog.NewLogger("ns", nil)
		})

		Context("setting its output", func() {
			var output *gbytes.Buffer

			JustBeforeEach(func() {
				output = gbytes.NewBuffer()
				logger.SetOutput(output)
			})

			It("should be disabled", func() {
				logger.Debug("debug")
				logger.Info("info")
				logger.Error("error")
				Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
				Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
				Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "error")))
			})

			Context("toggling from debug to disabled", func() {
				It("should log and no longer log", func() {
					logger.SetLevel(plog.Debug)
					logger.Debug("debug")
					logger.Info("info")
					logger.Error("error")
					logger.SetLevel(plog.Disabled)
					logger.Debug("debug")
					logger.Info("info")
					logger.Error("error")
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "debug")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "info")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "error")))
				})
			})

			Context("enabling it", func() {
				var level plog.LogLevel

				JustBeforeEach(func() {
					logger.SetLevel(level)
				})

				JustBeforeEach(func() {
					logger.Debug("debug")
					logger.Info("info")
					logger.Error("error")
				})

				Context("to debug level", func() {
					BeforeEach(func() {
						level = plog.Debug
					})

					It("should log", func() {
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "debug")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "info")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					})
				})

				Context("to info level", func() {
					BeforeEach(func() {
						level = plog.Info
					})

					It("should log", func() {
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "info")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					})
				})

				Context("to error level", func() {
					BeforeEach(func() {
						level = plog.Error
					})

					It("should log", func() {
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					})
				})

				Context("to panic level", func() {
					BeforeEach(func() {
						level = plog.Panic
					})

					It("should log", func() {
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "error")))
						Expect(func() { logger.Panic(errors.New("panic"), "panic") }).To(Panic())
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "panic")))
					})
				})
			})
		})
	})
})
