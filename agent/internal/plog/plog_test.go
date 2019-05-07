// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package plog_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	const re = "SQREEN/%s - [0-9]{4}(-[0-9]{1,2}){2}T([0-9]{1,2}:){2}[0-9]{1,2}.[0-9]{1,6} - %s"

	for _, tc := range []plog.LogLevel{
		plog.Disabled,
		plog.Panic,
		plog.Error,
		plog.Info,
		plog.Debug,
	} {
		t.Run(tc.String(), func(t *testing.T) {
			output := gbytes.NewBuffer()
			logger := plog.NewLogger(tc, output)

			logger.Error("error 1", "error 2", "error 3")
			logger.Info("info 1", "info 2", "info 3")
			logger.Debug("debug 1", "debug 2", "debug 3")
			require.Panics(t, func() { logger.Panic(errors.New("panic error"), "panic 1", "panic 2", "panic 3") })

			gomega := gomega.NewGomegaWithT(t)
			switch tc {
			case plog.Disabled:
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "ERROR", "error 1error 2error 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "INFO", "info 1info 2info 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "DEBUG", "debug 1debug 2debug 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "PANIC", "panic 1panic 2panic 3")))
			case plog.Error:
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "ERROR", "error 1error 2error 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "INFO", "info 1info 2info 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "DEBUG", "debug 1debug 2debug 3")))
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "PANIC", "panic 1panic 2panic 3")))
			case plog.Info:
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "ERROR", "error 1error 2error 3")))
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "INFO", "info 1info 2info 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "DEBUG", "debug 1debug 2debug 3")))
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "PANIC", "panic 1panic 2panic 3")))
			case plog.Debug:
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "ERROR", "error 1error 2error 3")))
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "INFO", "info 1info 2info 3")))
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "DEBUG", "debug 1debug 2debug 3")))
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "PANIC", "panic 1panic 2panic 3")))
			case plog.Panic:
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "ERROR", "error 1error 2error 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "INFO", "info 1info 2info 3")))
				gomega.Expect(output).ShouldNot(gbytes.Say(fmt.Sprintf(re, "DEBUG", "debug 1debug 2debug 3")))
				gomega.Expect(output).Should(gbytes.Say(fmt.Sprintf(re, "PANIC", "panic 1panic 2panic 3")))
			}
		})
	}
}
