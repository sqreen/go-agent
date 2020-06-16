// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"encoding/json"

	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	errors "golang.org/x/xerrors"
)

func (a *AgentType) addUserEvent(e event.UserEventFace) {
	var (
		store  *metrics.Store
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

func (a *AgentType) addIPPasslistEvent(matchedPasslistEntry string) {
	err := a.addPasslistEvent(a.staticMetrics.allowedIP, matchedPasslistEntry)
	if err != nil {
		a.Logger().Error(sqerrors.Wrap(err, "passlist event: could not update the ip passlist metrics store"))
	}
}
func (a *AgentType) addPathPasslistEvent(matchedPasslistEntry string) {
	err := a.addPasslistEvent(a.staticMetrics.allowedPath, matchedPasslistEntry)
	if err != nil {
		a.Logger().Error(sqerrors.Wrap(err, "passlist event: could not update the path passlist metrics store"))
	}
}

func (a *AgentType) addPasslistEvent(store *metrics.Store, matchedPasslistEntry string) error {
	err := store.Add(matchedPasslistEntry, 1)

	var maxStoreLenErr metrics.MaxMetricsStoreLengthError
	if errors.As(err, &maxStoreLenErr) {
		if err := a.staticMetrics.errors.Add(maxStoreLenErr, 1); err != nil {
			a.logger.Debugf("could not update the error metrics store: %v", err)
		}
	}

	return err
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
