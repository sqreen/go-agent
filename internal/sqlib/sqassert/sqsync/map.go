// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsync

import (
	"sync"
	"sync/atomic"
)

type UInt64Map struct {
	sync.Map
}

func (m *UInt64Map) Add(key interface{}, delta uint64) {
	atomic.AddUint64(m.Get(key), delta)
}

func (m *UInt64Map) Get(key interface{}) *uint64 {
	v, loaded := m.Load(key)
	if !loaded {
		v, _ = m.LoadOrStore(key, new(uint64))
	}
	return v.(*uint64)
}
