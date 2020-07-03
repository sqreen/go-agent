// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types_test

import (
	"testing"

	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/stretchr/testify/require"
)

func TestRequestParamMap_Add(t *testing.T) {
	t.Run("add to default nil value", func(t *testing.T) {
		var m types.RequestParamMap = nil
		m.Add("k1", "v1")
		m.Add("k1", 2)
		m.Add("k1", true)
		m.Add("k2", "v2")
		m.Add("k3", 3)
		m.Add("k3", false)

		expected := types.RequestParamMap{
			"k1": types.RequestParamValueSlice{"v1", 2, true},
			"k2": types.RequestParamValueSlice{"v2"},
			"k3": types.RequestParamValueSlice{3, false},
		}
		require.Equal(t, expected, m)
	})
}
