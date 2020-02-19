// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

// Types to sync with the instrumentation tool

type HookTableType = []HookDescriptorFuncType
type HookDescriptorFuncType = func(*HookDescriptorType)
type HookDescriptorType = struct {
	Func, PrologVar interface{}
}
