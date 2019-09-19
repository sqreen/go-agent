// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testmock

import "github.com/stretchr/testify/mock"

type LoggerMockup struct {
	mock.Mock
}

func (l LoggerMockup) Debug(v ...interface{}) {
	l.Called(v)
}

func (l LoggerMockup) Debugf(format string, v ...interface{}) {
	l.Called(format, v)
}

func (l LoggerMockup) Error(err error) {
	l.Called(err)
}
