// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsql_test

import (
	"database/sql/driver"
	"testing"

	"github.com/sqreen/go-agent/internal/sqlib/sqsql"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/module/apmsql"
)

// myFakeDriver implements driver.Driver
type myFakeDriver struct{}

func (m myFakeDriver) Open(string) (driver.Conn, error) {
	return nil, nil
}

func (m myFakeDriver) SomethingElse() {}

type myDriverWrapper struct {
	driver.Driver
}

func (w myDriverWrapper) Unwrap() driver.Driver {
	return w.Driver
}

func TestUnwrap(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		unwrapped := sqsql.Unwrap(nil)
		require.Equal(t, nil, unwrapped)
	})

	t.Run("not wrapped", func(t *testing.T) {
		actual := myFakeDriver{}
		unwrapped := sqsql.Unwrap(actual)
		require.Equal(t, actual, unwrapped)
	})

	t.Run("nested", func(t *testing.T) {
		actual := myFakeDriver{}
		var d driver.Driver = actual
		d = myDriverWrapper{d}
		d = myDriverWrapper{d}
		d = myDriverWrapper{d}
		d = myDriverWrapper{d}
		d = myDriverWrapper{d}

		unwrapped := sqsql.Unwrap(d)
		require.Equal(t, actual, unwrapped)
	})

	t.Run("Elastic APM SQL Tracing Wrapper", func(t *testing.T) {
		t.Skip("unskip when merged: https://github.com/elastic/apm-agent-go/issues/848")
		actual := myFakeDriver{}
		drv := apmsql.Wrap(actual)
		unwrapped := sqsql.Unwrap(drv)
		require.Equal(t, actual, unwrapped)
	})
}
