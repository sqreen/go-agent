// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package agent

import (
	"github.com/sqreen/go-agent/agent/internal"
)

var agent *internal.Agent

func init() {
	agent = internal.New()
	if agent != nil {
		agent.Start()
	}
}
