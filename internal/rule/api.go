// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"github.com/sqreen/go-agent/internal/span"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

type staticRuleDescriptors struct {
	spanInstrumentation   []span.EventListener
	nativeInstrumentation hookDescriptorMap
}

var (
	staticRules       []Descriptor
	staticDescriptors *staticRuleDescriptors
)

func Register(r ...Descriptor) {
	staticRules = append(staticRules, r...)
}

type Descriptor struct {
	Name            string
	Instrumentation InstrumentationDescriptor
}

type InstrumentationDescriptor interface {
	isInstrumentationDescriptor()
}

type NativeInstrumentation struct {
	Function string
	Callback sqhook.PrologCallback
	Priority int
}

func (NativeInstrumentation) isInstrumentationDescriptor() {}

type SpanInstrumentation struct {
	EventListener span.EventListener
}

func (SpanInstrumentation) isInstrumentationDescriptor() {}
