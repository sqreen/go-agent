// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqatomic

import "sync/atomic"

// AtomicUint32 is a wrapper type of a uint32 providing convenience methods of
// commonly used atomic operations.
type AtomicUint32 uint32

func (i *AtomicUint32) unwrap() *uint32 { return (*uint32)(i) }

func (i *AtomicUint32) Load() uint32 {
	return atomic.LoadUint32(i.unwrap())
}

func (i *AtomicUint32) Increment() uint32 {
	return i.Add(1)
}

func (i *AtomicUint32) Decrement() uint32 {
	return i.Add(-1)
}

func (i *AtomicUint32) Add(delta uint32) uint32 {
	return atomic.AddUint32(i.unwrap(), delta)
}
