// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package span

import (
	"sync"

	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

type (
	eventEmitter struct {
		mu          sync.RWMutex
		onNewChild  []OnNewChildEventListenerFunc
		onEnd       []OnEndEventListenerFunc
		onChildData []OnChildDataEventListenerFunc
		onNamedSpan map[string][]OnNewChildEventListenerFunc
	}

	OnNewChildEventListenerFunc  func(s EmergingSpan) error
	OnChildDataEventListenerFunc func(s Span, data AttributeGetter) error
	OnEndEventListenerFunc       func(results AttributeGetter) error

	EventRegister interface {
		Register(l EventListener)
		OnNewChild(l OnNewChildEventListenerFunc)
		OnNewNamedChild(name string, l OnNewChildEventListenerFunc)
		OnChildData(l OnChildDataEventListenerFunc)
		OnEnd(l OnEndEventListenerFunc)
	}

	EmergingSpan interface {
		Span
		EventRegister
	}
)

type EventListener interface{ applyTo(*eventEmitter) }

func (e *eventEmitter) Register(l EventListener)               { l.applyTo(e) }
func (l OnNewChildEventListenerFunc) applyTo(e *eventEmitter)  { e.OnNewChild(l) }
func (l OnEndEventListenerFunc) applyTo(e *eventEmitter)       { e.OnEnd(l) }
func (l OnChildDataEventListenerFunc) applyTo(e *eventEmitter) { e.OnChildData(l) }

type newNamedChildSpanEventListener struct {
	name string
	l    OnNewChildEventListenerFunc
}

func NewNamedChildSpanEventListener(name string, l OnNewChildEventListenerFunc) EventListener {
	return newNamedChildSpanEventListener{
		name: name,
		l:    l,
	}
}

func (l newNamedChildSpanEventListener) applyTo(e *eventEmitter) {
	e.OnNewNamedChild(l.name, l.l)
}

func (e *eventEmitter) OnNewNamedChild(name string, l OnNewChildEventListenerFunc) {
	if e.onNamedSpan == nil {
		e.onNamedSpan = make(map[string][]OnNewChildEventListenerFunc)
		e.OnNewChild(func(s EmergingSpan) error {
			v, exists := s.Get("span.name")
			if !exists {
				return nil
			}
			spanName, ok := v.(string)
			if !ok {
				return nil
			}
			listeners, exists := e.onNamedSpan[spanName]
			if !exists {
				return nil
			}
			for _, l := range listeners {
				if err := l(s); err != nil {
					return err
				}
			}
			return nil
		})
	}

	e.onNamedSpan[name] = append(e.onNamedSpan[name], l)
}

func (e *eventEmitter) OnNewChild(l OnNewChildEventListenerFunc) {
	e.onNewChild = append(e.onNewChild, l)
}

func (e *eventEmitter) OnEnd(l OnEndEventListenerFunc) {
	e.onEnd = append(e.onEnd, l)
}

func (e *eventEmitter) OnChildData(l OnChildDataEventListenerFunc) {
	e.onChildData = append(e.onChildData, l)
}

func (e *eventEmitter) EmitNewChildEvent(s EmergingSpan) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, l := range e.onNewChild {
		if err := l(s); err != nil {
			return err
		}
	}
	return nil
}

func (e *eventEmitter) EmitEndEvent(results AttributeGetter) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var errs sqerrors.ErrorCollection
	for _, l := range e.onEnd {
		if err := l(results); err != nil {
			errs.Add(err)
		}
	}

	return errs.ToError()
}

func (e *eventEmitter) EmitChildDataEvent(s Span, data AttributeGetter) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, l := range e.onChildData {
		if err := l(s, data); err != nil {
			return err
		}
	}
	return nil
}

func (e *eventEmitter) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onNewChild = nil
	e.onEnd = nil
	e.onNamedSpan = nil
	e.onChildData = nil
}

func forEachRunningParent(s Span, do func(Span) error) error {
	for s = s.Parent(); s != nil; s = s.Parent() {
		if s.State() != RunningState {
			continue
		}
		if err := do(s); err != nil {
			return err
		}
	}
	return nil
}

type NewChildEventEmitter interface {
	EmitNewChildEvent(s EmergingSpan) error
}

func emitNewChildEvent(s *span) error {
	return forEachRunningParent(s, func(parent Span) error {
		emitter, ok := parent.(NewChildEventEmitter)
		if !ok {
			return nil
		}
		return emitter.EmitNewChildEvent(s)
	})
}

type ChildDataEventEmitter interface {
	EmitChildDataEvent(s Span, data AttributeGetter) error
}

func emitChildDataEvent(s Span, data AttributeGetter) error {
	return forEachRunningParent(s, func(parent Span) error {
		emitter, ok := parent.(ChildDataEventEmitter)
		if !ok {
			return nil
		}
		return emitter.EmitChildDataEvent(s, data)
	})
}
