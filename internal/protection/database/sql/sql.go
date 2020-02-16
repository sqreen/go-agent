// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sql

import (
	"context"
	"database/sql/driver"
)

func NewInstrumentedDriver(d driver.Driver, name, dialect string) driver.Driver {
	return &InstrumentedDriver{
		d:       d,
		name:    name,
		dialect: dialect,
	}
}

func (i *InstrumentedDriver) Open(name string) (driver.Conn, error) {
	conn, err := i.d.Open(name)
	if err != nil {
		return nil, err
	}
	return NewInstrumentedConn(i, conn), nil
}

//func (i *InstrumentedDriverWithOpenConnector) OpenConnector(name string) (driver.Connector, error) {
//	if d, ok := i.d.(driver.DriverContext); ok {
//		conn, err := d.OpenConnector(name)
//		if err != nil {
//			return nil, err
//		}
//		return &InstrumentedConnector{
//			connector: conn,
//			driver:    i,
//		}, nil
//	}
//}

func NewInstrumentedConn(d *InstrumentedDriver, conn driver.Conn) driver.Conn {
	var (
		queryer  driver.QueryerContext
		execer   driver.ExecerContext
		preparer driver.ConnPrepareContext
	)

	return &InstrumentedConn{
		d:        d,
		queryer:  queryer,
		execer:   execer,
		preparer: preparer,
	}
}

type InstrumentedDriver struct {
	d       driver.Driver
	name    string
	dialect string
}

func (d *InstrumentedDriver) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if d.execerContext == nil {
		return nil, driver.ErrSkip
	}
	return d.execContext(ctx, query, args)
}

//go:noinline
func (d *InstrumentedDriver) execContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	// dynamically instrumented
	return d.execerContext(ctx, query, args)
}

type skipperDriver struct{}

func (s skipperDriver) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	panic("implement me")
}
