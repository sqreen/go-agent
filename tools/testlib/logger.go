// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testlib

import "github.com/stretchr/testify/mock"

type LoggerMockup struct {
	mock.Mock
}

func (l LoggerMockup) Debug(v ...interface{}) {
	l.Called(v...)
}

func (l LoggerMockup) Debugf(format string, v ...interface{}) {
	args := make([]interface{}, 0, len(v)+1)
	args = append(args, format)
	args = append(args, v...)
	l.Called(args...)
}

func (l LoggerMockup) Info(v ...interface{}) {
	l.Called(v...)
}

func (l LoggerMockup) Infof(format string, v ...interface{}) {
	args := make([]interface{}, 0, len(v)+1)
	args = append(args, format)
	args = append(args, v...)
	l.Called(args...)
}

func (l LoggerMockup) Error(v ...interface{}) {
	l.Called(v...)
}

func (l LoggerMockup) Errorf(format string, v ...interface{}) {
	args := make([]interface{}, 0, len(v)+1)
	args = append(args, format)
	args = append(args, v...)
	l.Called(args...)
}

func (l LoggerMockup) Panic(err error, v ...interface{}) {
	args := make([]interface{}, 0, len(v)+1)
	args = append(args, err)
	args = append(args, v...)
	l.Called(args...)
}

func (l LoggerMockup) Panicf(err error, format string, v ...interface{}) {
	args := make([]interface{}, 0, len(v)+2)
	args = append(args, err)
	args = append(args, format)
	args = append(args, v...)
	l.Called(args...)
}
