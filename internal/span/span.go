// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package span

import (
	"sync/atomic"

	protection_types "github.com/sqreen/go-agent/internal/protection/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
)

type (
	Span interface {
		Parent() Span
		State() State
		AttributeGetter
		EmitData(data AttributeGetter) error
	}

	SpanEnder interface {
		Span
		End(results AttributeGetter) error
	}

	AttributeGetter interface {
		Get(key string) (value interface{}, exists bool)
	}

	ProtectionContextGetter interface {
		ProtectionContext() protection_types.ProtectionContext
	}

	span struct {
		parent     Span
		state      State
		protection protection_types.ProtectionContext
		AttributeGetter
		eventEmitter
	}

	State uint32

	AttributeMap map[string]interface{}
)

const (
	RunningState State = iota + 1
	EndedState
)

var RootSpan Span = &span{state: RunningState}

func NewSpan(options ...Option) (s SpanEnder, err error) {
	sp := &span{
		state: RunningState,
	}

	for _, o := range options {
		o.apply(sp)
	}

	if sp.parent == nil {
		glsPush(sp)
	}

	defer func() {
		if err != nil {
			// Call End() in case some event listeners have set some finish callbacks
			// for example to free their memory.
			// We keep the first error error and ignore the one End() might return.
			_ = sp.End(nil)
		}
	}()

	if err := emitNewChildEvent(sp); err != nil {
		return nil, err
	}
	return sp, nil
}

func (s *span) Parent() Span {
	return s.parent
}

func (s *span) State() State {
	return s.state.Get()
}

func (s *span) Get(key string) (value interface{}, exists bool) {
	if s.AttributeGetter == nil {
		return nil, false
	}
	return s.AttributeGetter.Get(key)
}

func (s *span) ProtectionContext() protection_types.ProtectionContext {
	return s.protection
}

func (s *span) End(results AttributeGetter) error {
	defer func() {
		s.eventEmitter.Clear()
		s.state.Set(EndedState)
		glsPop(s)
	}()
	return s.EmitEndEvent(results)
}

func (s *span) EmitData(data AttributeGetter) error {
	return emitChildDataEvent(s, data)
}

type (
	Option interface {
		apply(s *span)
	}

	optionFunc func(s *span)
)

func (f optionFunc) apply(s *span) {
	f(s)
}

func WithProtectionContext(p protection_types.ProtectionContext) Option {
	return optionFunc(func(s *span) {
		s.protection = p
	})
}

func WithParent(parent Span) Option {
	return optionFunc(func(s *span) {
		s.parent = parent
	})
}

func WithAttributes(attributes AttributeGetter) Option {
	return optionFunc(func(s *span) {
		s.AttributeGetter = attributes
	})
}

func WithEventListeners(l ...EventListener) Option {
	return optionFunc(func(s *span) {
		for _, l := range l {
			s.eventEmitter.Register(l)
		}
	})
}

func Current() Span {
	return fromGLS()
}

func fromGLS() Span {
	s, _ := sqgls.Get().(Span)
	return s
}

func glsPush(s *span) {
	s.parent = Current()
	sqgls.Set(s)
	current := Current()
	sqassert.True(current == s)
}

func glsPop(s *span) {
	current := Current()
	sqassert.True(current == s)
	parent := s.Parent()
	sqgls.Set(parent)
	current = Current()
	sqassert.True(current == parent)
}

func ProtectionContext(span Span) protection_types.ProtectionContext {
	for span := span; span != nil; span = span.Parent() {
		if span.State() != RunningState {
			continue
		}
		if p, ok := span.(ProtectionContextGetter); ok {
			if p := p.ProtectionContext(); p != nil {
				return p
			}
		}
	}
	return nil
}

func (m AttributeMap) Get(key string) (value interface{}, exists bool) {
	if len(m) == 0 {
		return nil, false
	}
	value, exists = m[key]
	return
}

func GetAttribute(span Span, key string) (value interface{}, exists bool) {
	for span := span; span != nil; span = span.Parent() {
		if v, exists := span.Get(key); exists {
			return v, exists
		}
	}
	return nil, false
}

func (s *State) Get() State {
	return State(atomic.LoadUint32((*uint32)(s)))
}

func (s *State) Set(state State) {
	atomic.StoreUint32((*uint32)(s), uint32(state))
}
