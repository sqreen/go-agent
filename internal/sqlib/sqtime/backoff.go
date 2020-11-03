// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqtime

import (
	"sync/atomic"
	"time"
)

type Backoff struct {
	current, max time.Duration
	rate         int64
}

func NewBackoff(min, max time.Duration, rate int64) *Backoff {
	return &Backoff{
		current: min,
		max:     max,
		rate:    rate,
	}
}

func (b *Backoff) Next() (duration time.Duration, max bool) {
	if b.current >= b.max {
		b.current = b.max
		return b.current, true
	}
	b.current *= time.Duration(b.rate)
	return b.current, false
}

type BackoffCounter uint64

// Do atomically increments backoff counter and calls function `f` along with
// the incremented counter value when the new count is a power of two.
func (c *BackoffCounter) Do(f func(count uint64)) {
	v := atomic.AddUint64((*uint64)(c), 1)
	// Is power of two? (0 included)
	if (v & (v - 1)) == 0 {
		f(v)
	}
}
