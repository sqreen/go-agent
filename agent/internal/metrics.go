// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"encoding/json"

	"github.com/sqreen/go-agent/agent/internal/metrics"
	"github.com/sqreen/go-agent/agent/internal/record"
	"github.com/sqreen/go-agent/agent/internal/sqlib/sqerrors"
)

func (a *Agent) AddUserEvent(event record.UserEventFace) {
	if a.config.Disable() || a.metrics == nil {
		// Disabled or not yet initialized agent
		return
	}
	var (
		store  *metrics.Store
		logFmt string
	)
	var uevent *record.UserEvent
	switch actual := event.(type) {
	case *record.AuthUserEvent:
		uevent = actual.UserEvent
		if actual.LoginSuccess {
			store = a.staticMetrics.sdkUserLoginSuccess
			logFmt = "user event: user login success `%+v`"
		} else {
			store = a.staticMetrics.sdkUserLoginFailure
			logFmt = "user event: user login failure `%+v`"
		}
	case *record.SignupUserEvent:
		uevent = actual.UserEvent
		store = a.staticMetrics.sdkUserSignup
		logFmt = "user event: user signup `%+v`"
	default:
		a.logger.Error(sqerrors.Errorf("user event: unexpected user event type `%T`", actual))
		return
	}
	key, err := UserEventMetricsStoreKey(uevent)
	if err != nil {
		a.logger.Error(sqerrors.Wrap(err, "user event: could not create a metrics store key"))
		return
	}
	a.logger.Debugf(logFmt, key)

	if err := store.Add(key, 1); err != nil {
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

func (a *Agent) AddWhitelistEvent(matchedWhitelistEntry string) {
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

func UserEventMetricsStoreKey(event *record.UserEvent) (json.Marshaler, error) {
	var keys [][]interface{}
	for prop, val := range event.UserIdentifiers {
		keys = append(keys, []interface{}{prop, val})
	}
	jsonKeys, _ := json.Marshal(keys)
	return userMetricsKey{
		Keys: string(jsonKeys),
		IP:   event.IP.String(),
	}, nil
}

type userMetricsKey struct {
	Keys string
	IP   string
}

func (e userMetricsKey) MarshalJSON() ([]byte, error) {
	v := struct {
		Keys json.RawMessage `json:"keys"`
		IP   string          `json:"ip"`
	}{
		Keys: json.RawMessage(e.Keys),
		IP:   e.IP,
	}
	return json.Marshal(&v)
}
