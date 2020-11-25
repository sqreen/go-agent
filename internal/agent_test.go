// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_overheadRate(t *testing.T) {
	t.Run("no panics nor special float numbers", func(t *testing.T) {
		limits := []float64{
			math.MaxFloat64,
			-math.MaxFloat64,
			math.NaN(),
			math.Inf(1),
			math.Inf(-1),
			0,
			1,
			-1,
			-1e100,
			1e100,
		}
		for _, req := range limits {
			for _, sq := range limits {
				require.NotPanics(t, func() {
					rate, _ := overheadRate(req, sq)
					require.False(t, math.IsNaN(rate) || math.IsInf(rate, 0))
				})
			}
		}
	})
}
