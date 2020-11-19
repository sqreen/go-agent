// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testmock

import "github.com/stretchr/testify/mock"

type TimeHistogramMockup struct {
	mock.Mock
}

func (t *TimeHistogramMockup) Add(key interface{}, delta uint64) error {
	return t.Called(key, delta).Error(0)
}

func (t *TimeHistogramMockup) ExpectAdd(key interface{}, delta uint64) *mock.Call {
	return t.On("Add", key, delta)
}

type PerfHistogramMockup struct {
	mock.Mock
}

func (t *PerfHistogramMockup) Add(v float64) error {
	return t.Called(v).Error(0)
}

func (t *PerfHistogramMockup) ExpectAdd(v interface{}) *mock.Call {
	return t.On("Add", v)
}
