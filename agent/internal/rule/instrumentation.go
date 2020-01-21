// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package rule

import "github.com/sqreen/go-agent/agent/sqlib/sqhook"

type InstrumentationFace interface {
	Find(symbol string) (HookFace, error)
}

type HookFace interface {
	Attach(prolog sqhook.PrologCallback) error
}

type defaultInstrumentation struct{}

var defaultInstrumentationEngine defaultInstrumentation

func (defaultInstrumentation) Find(symbol string) (HookFace, error) {
	hook, err := sqhook.Find(symbol)
	if err != nil {
		return nil, err
	}
	return defaultHook{hook}, nil
}

type defaultHook struct{ *sqhook.Hook }

func (h defaultHook) Attach(prolog sqhook.PrologCallback) error {
	return h.Hook.Attach(prolog)
}
