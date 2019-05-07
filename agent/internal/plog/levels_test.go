// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package plog_test

import (
	"testing"
	"time"

	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/stretchr/testify/require"
)

func TestLogLevel(t *testing.T) {
	levels := []plog.LogLevel{
		plog.Disabled,
		plog.Panic,
		plog.Error,
		plog.Info,
		plog.Debug,
	}
	for _, level := range levels {
		require.Equal(t, level, plog.ParseLogLevel(level.String()))
	}
}

func TestFormatTime(t *testing.T) {
	tim, err := time.Parse(time.RFC3339Nano, "2006-01-02T15:04:05.999999999+00:00")
	require.NoError(t, err)
	buf := plog.FormatTime(tim)
	require.Equal(t, "2006-1-2T15:4:5.999999", string(buf))
}
