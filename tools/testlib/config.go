// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testlib

import "github.com/stretchr/testify/mock"

type ConfigMockup struct {
	mock.Mock
}

func (c ConfigMockup) BackendHTTPAPIProxy() string {
	return c.Called().String(0)
}
