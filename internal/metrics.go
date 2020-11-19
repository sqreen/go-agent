// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"encoding/json"

	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

func (a *AgentType) addUserEvent(e event.UserEventFace) {
	var (
		store  *metrics.TimeHistogram
		logFmt string
	)

	var uevent *event.UserEvent
	switch actual := e.(type) {
	case *event.AuthUserEvent:
		uevent = actual.UserEvent
		if actual.LoginSuccess {
			store = a.staticMetrics.sdkUserLoginSuccess
			logFmt = "user event: user login success `%+v`"
		} else {
			store = a.staticMetrics.sdkUserLoginFailure
			logFmt = "user event: user login failure `%+v`"
		}
	case *event.SignupUserEvent:
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
		type errKey struct{}
		err = sqerrors.WithKey(err, errKey{})
		err = sqerrors.Wrap(err, "user event: could not update the user metrics store")
		a.logger.Error(err)
	}
}

func (a *AgentType) addIPPasslistEvent(matchedPasslistEntry string) {
	if !a.addPasslistEvent(a.staticMetrics.allowedIP, matchedPasslistEntry) {
		a.logger.Debug("passlist event: could not add the ip passlist event")
	}
}
func (a *AgentType) addPathPasslistEvent(matchedPasslistEntry string) {
	if !a.addPasslistEvent(a.staticMetrics.allowedPath, matchedPasslistEntry) {
		a.logger.Debug("passlist event: could not add the path passlist event")
	}
}

func (a *AgentType) addPasslistEvent(store *metrics.TimeHistogram, matchedPasslistEntry string) bool {
	if err := store.Add(matchedPasslistEntry, 1); err != nil {
		type errKey struct{}
		a.logger.Error(sqerrors.WithKey(err, errKey{}))
		return false
	}
	return true
}

func UserEventMetricsStoreKey(e *event.UserEvent) (json.Marshaler, error) {
	var keys [][]interface{}
	for prop, val := range e.UserIdentifiers {
		keys = append(keys, []interface{}{prop, val})
	}
	jsonKeys, _ := json.Marshal(keys)
	return userMetricsKey{
		Keys: string(jsonKeys),
		IP:   e.IP.String(),
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
