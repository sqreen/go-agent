// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqtime

import "time"

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
