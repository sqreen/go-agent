package plog_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
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
			for _, errChanLen := range []int{0, 1, 1024} {
				errChanLen := errChanLen // new scope
				t.Run(fmt.Sprintf("with chan buffer length %d", errChanLen), func(t *testing.T) {
					g := gomega.NewGomegaWithT(t)
					output := gbytes.NewBuffer()
					var errChan chan error
					if errChanLen > 0 {
						errChan = make(chan error, errChanLen)
					}
					logger := plog.NewLogger(level, output, errChan)

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

					// The error should have been sent into the channel
					if errChanLen > 0 {
						g.Eventually(errChan).Should(gomega.Receive(gomega.Equal(err)))
					}
				})
			}
		})
	}
}

func TestWithBackoff(t *testing.T) {
	t.Run("common usage", func(t *testing.T) {
		for _, level := range []plog.LogLevel{
			plog.Disabled,
			plog.Debug,
			plog.Info,
			plog.Error,
		} {
			level := level // new scope
			t.Run(level.String(), func(t *testing.T) {
				for _, errChanLen := range []int{0, 1, 2, 3, 1024} {
					errChanLen := errChanLen // new scope
					t.Run(fmt.Sprintf("with chan buffer length %d", errChanLen), func(t *testing.T) {
						g := gomega.NewGomegaWithT(t)
						output := gbytes.NewBuffer()

						var errChan chan error
						if errChanLen > 0 {
							errChan = make(chan error, errChanLen)
						}
						logger := plog.WithStrictBackoff(plog.NewLogger(level, output, errChan))

						// Perform log calls
						logger.Debug("debug 1", " debug 2", " debug 3")
						logger.Info("info 1 ", "info 2 ", "info 3")
						err := errors.New("error message 0")
						logger.Error(err)
						logger.Error(err)
						logger.Error(err)
						logger.Error(err)
						logger.Error(err)
						err1 := sqerrors.WithKey(errors.New("error message 1"), 1)
						logger.Error(err1)
						logger.Error(err1)
						err2 := sqerrors.WithKey(errors.New("error message 2"), 2)
						logger.Error(err2)
						logger.Error(err2)
						logger.Error(err2)
						logger.Error(err2)
						logger.Error(err2)

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
							errMsg0 := errorRe + " 0"
							g.Expect(output).Should(gbytes.Say(errMsg0))
							g.Expect(output).Should(gbytes.Say(errMsg0))
							g.Expect(output).Should(gbytes.Say(errMsg0))

							errMsg1 := errorRe + " 1"
							g.Expect(output).Should(gbytes.Say(errMsg1))

							errMsg2 := errorRe + " 2"
							g.Expect(output).Should(gbytes.Say(errMsg2))
							g.Expect(output).Should(gbytes.Say(errMsg2))
							g.Expect(output).Should(gbytes.Say(errMsg2))
						}

						// The error should have been sent into the channel
						// The number of sent events has been backoff'd but also limited by
						// the channel buffer size
						expectedErrors := []error{
							err, err, err, err1, err2, err2, err2,
						}
						for i := 0; i < len(expectedErrors) && i < errChanLen; i++ {
							g.Eventually(errChan).Should(gomega.Receive(gomega.Equal(expectedErrors[i])))
						}
					})
				}
			})
		}
	})

	t.Run("with bad key type leading to a panic while looking up the map", func(t *testing.T) {
		output := gbytes.NewBuffer()
		errChan := make(chan error, 1)
		var logger plog.DebugLevelLogger = plog.NewLogger(plog.Debug, output, errChan)
		logger = plog.WithStrictBackoff(logger)

		anError := errors.New("my error")

		// A slice is not hashable and will lead to a panic
		type badErrKey []int
		logger.Error(sqerrors.WithKey(anError, badErrKey{}))

		// The panic was caught and logged along the original error as an error
		// collection
		logged := <-errChan
		var collection sqerrors.ErrorCollection
		require.True(t, xerrors.As(logged, &collection))
		require.Len(t, collection, 2)

		// One of them is the panic error
		var panicErr *sqsafe.PanicError
		require.True(t, xerrors.As(collection[0], &panicErr) || xerrors.As(collection[1], &panicErr))

		// One of them is the original error
		require.True(t, xerrors.Is(collection[0], anError) || xerrors.Is(collection[1], anError))
	})
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
