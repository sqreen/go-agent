// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"github.com/sqreen/go-agent/agent/internal/metrics"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
)

func (a *Agent) addUserEvent(event userEventFace) {
	if a.config.Disable() || a.metrics == nil {
		// Disabled or not yet initialized agent
		return
	}
	var (
		store  *metrics.Store
		logFmt string
	)
	switch actual := event.(type) {
	case *authUserEvent:
		if actual.loginSuccess {
			store = a.staticMetrics.sdkUserLoginSuccess
			logFmt = "user event: user login success `%v`"
		} else {
			store = a.staticMetrics.sdkUserLoginFailure
			logFmt = "user event: user login failure `%v`"
		}
	case *signupUserEvent:
		store = a.staticMetrics.sdkUserSignup
		logFmt = "user event: user signup `%v`"
	default:
		a.logger.Error(sqerrors.Errorf("user event: unexpected user event type `%T`", actual))
		return
	}
	a.logger.Debug(logFmt, event)
	if err := store.Add(event, 1); err != nil {
		sqErr := sqerrors.Wrap(err, "user event: could not update the user metrics store")
		switch actualErr := err.(type) {
		case metrics.MaxMetricsStoreLengthError:
			a.logger.Debug(sqErr)
			if err := a.staticMetrics.errors.Add(actualErr, 1); err != nil {
				a.logger.Debugf("could not update the metrics store: %v", err)
			}
		default:
			a.logger.Error(sqErr)
		}
	}
}

func (a *Agent) addWhitelistEvent(matchedWhitelistEntry string) {
	if a.config.Disable() || a.metrics == nil {
		// Agent is disabled or not yet initialized
		return
	}
	a.logger.Debug("request whitelisted for `%s`", matchedWhitelistEntry)
	err := a.staticMetrics.whitelistedIP.Add(matchedWhitelistEntry, 1)
	if err != nil {
		sqErr := sqerrors.Wrap(err, "whitelist event: could not update the whitelist metrics store")
		switch actualErr := err.(type) {
		case metrics.MaxMetricsStoreLengthError:
			a.logger.Debug(sqErr)
			if err := a.staticMetrics.errors.Add(actualErr, 1); err != nil {
				a.logger.Debugf("could not update the metrics store: %v", err)
			}
		default:
			a.logger.Error(sqErr)
		}
	}
}
