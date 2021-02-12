// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/backend/api/signal"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/stretchr/testify/require"
)

func Test_newMetricsAPIAdapter(t *testing.T) {
	s, err := metrics.NewPerfHistogram(time.Second, 0.1, 2, 10)
	z := -0.0
	require.NoError(t, err)
	require.NoError(t, s.Add(math.MaxFloat64))
	require.NoError(t, s.Add(-math.MaxFloat64))
	require.NoError(t, s.Add(33/z))
	time.Sleep(time.Second)
	require.True(t, s.Ready())
	perfHist := s.Flush()
	require.Len(t, perfHist, 1, "the period should be long enough to have the two values")

	type args struct {
		logger       plog.ErrorLogger
		readyMetrics map[string]metrics.ReadyStore
	}
	tests := []struct {
		name string
		args args
		want []api.MetricsTimeBucket
	}{
		{
			name: "",
			args: args{
				logger: plog.NewLogger(plog.Debug, os.Stderr, nil),
				readyMetrics: map[string]metrics.ReadyStore{
					"pct": perfHist[0],
				},
			},
			want: []api.MetricsTimeBucket{
				{
					Name: "pct",
					Observation: api.PerfMetricsData{
						Unit: 0.1,
						Base: 2,
						Values: map[string]interface{}{
							"1":    int64(1),
							"1029": int64(1),
							"max":  math.MaxFloat64,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newMetricsAPIAdapter(tt.args.logger, tt.args.readyMetrics)
			//require.Len(t, got, 1)
			//require.Equal(t, tt.want[0].Observation, got[0].Observation)

			sig := signal.FromLegacyMetrics(got, "1.2.3", plog.NewLogger(plog.Debug, os.Stderr, nil))
			//payload := sig[0].(*signal_api.Metric).Payload.(signal_api.BinningMetricsSignalPayload)

			//require.Equal(t, math.MaxFloat64, payload.Max)
			//require.Equal(t, 0.1, payload.Unit)
			//require.Equal(t, 2.0, payload.Base)
			//require.Equal(t, map[string]int64{
			//	"1":    int64(1),
			//	"1029": int64(1),
			//}, payload.Bins)

			enc := json.NewEncoder(os.Stderr)
			enc.SetEscapeHTML(false)
			err := enc.Encode(sig)
			require.NoError(t, err)
		})
	}
}
