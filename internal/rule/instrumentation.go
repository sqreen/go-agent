// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package rule

import (
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

type InstrumentationFace interface {
	Find(symbol string) (HookFace, error)
	Health(expectedVersion string) error
}

type HookFace interface {
	Attach(prologs ...sqhook.PrologCallback) error
}

type defaultInstrumentationImpl struct{}

var defaultInstrumentationEngine defaultInstrumentationImpl

func (defaultInstrumentationImpl) Health(expectedVersion string) error {
	return sqhook.Health(expectedVersion)
}

func (defaultInstrumentationImpl) Find(symbol string) (HookFace, error) {
	hook, err := sqhook.Find(symbol)
	if err != nil {
		return nil, err
	}
	if hook == nil {
		return nil, nil
	}
	return defaultHook{hook}, nil
}

type defaultHook struct{ *sqhook.Hook }
