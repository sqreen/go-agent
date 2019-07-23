// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import "github.com/sqreen/go-agent/agent/internal/metrics"

func (a *Agent) addUserEvent(event userEventFace) {
	if a.config.Disable() || a.metrics == nil {
		// Disabled or not yet initialized agent
		return
	}
	var store *metrics.Store
	switch actual := event.(type) {
	case *authUserEvent:
		if actual.loginSuccess {
			store = a.staticMetrics.sdkUserLoginSuccess
		} else {
			store = a.staticMetrics.sdkUserLoginFailure
		}
	case *signupUserEvent:
		store = a.staticMetrics.sdkUserSignup
	default:
		// TODO: log error
		return
	}
	store.Add(event, 1)
}

func (a *Agent) addWhitelistEvent(matchedWhitelistEntry string) {
	if a.config.Disable() || a.metrics == nil {
		// Agent is disabled or not yet initialized
		return
	}
	a.staticMetrics.whitelistedIP.Add(matchedWhitelistEntry, 1)
}
