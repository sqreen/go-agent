// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package squnsafe_test

import (
	"testing"

	"github.com/sqreen/go-agent/internal/sqlib/squnsafe"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestStringToBytes(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		b := squnsafe.StringToBytes("")
		require.Nil(t, b)
	})
	t.Run("non empty string", func(t *testing.T) {
		s := testlib.RandUTF8String()
		cp := []byte(s)
		b := squnsafe.StringToBytes(s)
		require.Equal(t, cp, b)
	})
}
