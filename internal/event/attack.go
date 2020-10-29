// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package event

import "runtime"

type AttackEventOption func(*AttackEvent)

func WithAttackInfo(info interface{}) AttackEventOption {
	return func(e *AttackEvent) {
		e.Info = info
	}
}

func WithTest(t bool) AttackEventOption {
	return func(e *AttackEvent) {
		e.Test = t
	}
}

func WithStackTrace() AttackEventOption {
	return func(e *AttackEvent) {
		e.StackTrace = callers()
	}
}

func callers() []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	return pcs[0:n]
}
