package plog_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	for _, level := range []plog.LogLevel{
		plog.Disabled,
		plog.Debug,
		plog.Info,
		plog.Error,
	} {
		level := level // new scope
		t.Run(level.String(), func(t *testing.T) {
			for _, errChanLen := range []int{-1, 0, 1, 1024} {
				errChanLen := errChanLen // new scope
				t.Run(fmt.Sprintf("with chan buffer length %d", errChanLen), func(t *testing.T) {
					g := gomega.NewGomegaWithT(t)
					output := gbytes.NewBuffer()
					logger := plog.NewLogger(level, output, errChanLen)

					// Perform log calls
					logger.Debug("debug 1", " debug 2", " debug 3")
					logger.Info("info 1 ", "info 2 ", "info 3")
					err := errors.New("error message")
					logger.Error(err)

					var (
						re      = "sqreen/%s - [0-9]{4}(-[0-9]{2}){2}T([0-9]{2}:){2}[0-9]{2}.?[0-9]{0,6} - %s"
						debugRe = fmt.Sprintf(re, plog.Debug, "debug 1 debug 2 debug 3")
						errorRe = fmt.Sprintf(re, plog.Error, "error message")
						infoRe  = fmt.Sprintf(re, plog.Info, "info 1 info 2 info 3")
					)
					switch level {
					case plog.Disabled:
						g.Expect(output).ShouldNot(gbytes.Say(debugRe))
						g.Expect(output).ShouldNot(gbytes.Say(infoRe))
						g.Expect(output).ShouldNot(gbytes.Say(errorRe))
					case plog.Debug:
						g.Expect(output).Should(gbytes.Say(debugRe))
						fallthrough
					case plog.Info:
						g.Expect(output).Should(gbytes.Say(infoRe))
						fallthrough
					case plog.Error:
						g.Expect(output).Should(gbytes.Say(errorRe))
					}

					if errChanLen > 0 {
						// The error should have been sent into the channel
						g.Eventually(logger.ErrChan()).Should(gomega.Receive(gomega.Equal(err)))
					}
				})
			}
		})
	}
}

func TestTimeFormat(t *testing.T) {
	for _, tc := range []struct {
		timestamp string
		expected  string
	}{
		{
			timestamp: "2006-01-02T15:04:05.000000",
			expected:  "2006-01-02T15:04:05",
		},
		{
			timestamp: "2006-01-02T15:04:05.1",
			expected:  "2006-01-02T15:04:05.1",
		},
		{
			timestamp: "2006-01-02T15:04:05.99999999",
			expected:  "2006-01-02T15:04:05.999999",
		},
		{
			timestamp: "2006-01-02T15:04:05.999000",
			expected:  "2006-01-02T15:04:05.999",
		},
		{
			timestamp: "2006-01-02T15:04:05.999999",
			expected:  "2006-01-02T15:04:05.999999",
		},
	} {
		t.Run(tc.timestamp, func(t *testing.T) {
			tim, err := time.Parse(plog.TimestampLayout, tc.timestamp)
			require.NoError(t, err)
			got := tim.Format(plog.TimestampLayout)
			require.Equal(t, tc.expected, got)
		})
	}
}
