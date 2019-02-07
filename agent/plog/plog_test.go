package plog_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/sqreen/go-agent/agent/plog"
)

var _ = Describe("plog", func() {
	Describe("a logger", func() {
		var (
			logger *plog.Logger
			re     = "ns:%s: [0-9]{4}(/[0-9]{2}){2} ([0-9]{2}:){2}[0-9]{2}.[0-9]{6} %[1]s"
		)

		JustBeforeEach(func() {
			logger = plog.NewLogger("ns")
		})

		Context("setting its output", func() {
			var output *gbytes.Buffer

			JustBeforeEach(func() {
				output = gbytes.NewBuffer()
				logger.SetOutput(output)
			})

			Measure("it should be faster when disabled, slower when enabled", func(b Benchmarker) {
				doLog := func() {
					logger.Debug("debug")
					logger.Info("info")
					logger.Warn("warn")
					logger.Error("error")
				}

				var allDurationAvg, disabledDurationAvg uint64
				for n := uint64(1); n <= 5000; n++ {
					logger.SetLevel(plog.Debug)
					allDuration := b.Time("info log level", doLog)
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "debug")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "info")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "warn")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					allDurationAvg = allDurationAvg*(n-1)/n + uint64(allDuration)/n

					logger.SetLevel(plog.Disabled)
					disabledDuration := b.Time("back to disabled", doLog)
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "warn")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "error")))
					disabledDurationAvg = disabledDurationAvg*(n-1)/n + uint64(disabledDuration)/n
				}

				Expect(allDurationAvg).Should(BeNumerically(">", disabledDurationAvg))
			}, 1)

			It("should be disabled", func() {
				logger.Debug("debug")
				logger.Info("info")
				logger.Warn("warn")
				logger.Error("error")
				Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
				Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
				Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "warn")))
				Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "error")))
			})

			Context("toggling from debug to disabled", func() {
				It("should log and no longer log", func() {
					logger.SetLevel(plog.Debug)
					logger.Debug("debug")
					logger.Info("info")
					logger.Warn("warn")
					logger.Error("error")
					logger.SetLevel(plog.Disabled)
					logger.Debug("debug")
					logger.Info("info")
					logger.Warn("warn")
					logger.Error("error")
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "debug")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "info")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "warn")))
					Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
					Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "warn")))
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
					logger.Warn("warn")
					logger.Error("error")
				})

				Context("to debug level", func() {
					BeforeEach(func() {
						level = plog.Debug
					})

					It("should log", func() {
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "debug")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "info")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "warn")))
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
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "warn")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					})
				})

				Context("to warn level", func() {
					BeforeEach(func() {
						level = plog.Warn
					})

					It("should log", func() {
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "warn")))
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
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "warn")))
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "error")))
					})
				})

				Context("to fatal level", func() {
					BeforeEach(func() {
						level = plog.Panic
					})

					It("should log", func() {
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "debug")))
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "info")))
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "warn")))
						Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "error")))
						Expect(func() { logger.Panic(errors.New("fatal"), "fatal") }).To(Panic())
						Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "fatal")))
					})
				})
			})
		})
	})
})
