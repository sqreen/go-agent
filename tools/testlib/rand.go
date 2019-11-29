// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testlib

import (
	"math/rand"

	fuzz "github.com/google/gofuzz"
)

func RandString(size ...int) string {
	fuzzer := fuzz.New().NumElements(2, 2048).NilChance(0)
	if len(size) == 1 {
		n := size[0]
		fuzzer.NumElements(n, n)
	} else if len(size) == 2 {
		from := size[0]
		to := size[1]
		fuzzer.NumElements(from, to)
	}
	var buf []byte
	fuzzer.Fuzz(&buf)
	return string(buf)
}

func RandUint32(boundaries ...uint32) uint32 {
	rand := rand.Uint32()

	switch len(boundaries) {
	case 0:
		return rand

	case 1:
		// At least
		return boundaries[0] + rand

	case 2:
		// Between boundaries
		min := boundaries[0]
		max := boundaries[1]
		return min + (rand % (max - min))

	default:
		panic("unexpected arguments")
	}
}
