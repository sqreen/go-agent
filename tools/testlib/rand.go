// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testlib

import "math/rand"

func RandString(size ...int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	var n int
	if len(size) == 1 {
		n = size[0]
	} else {
		from := size[0]
		to := size[1]
		n = from + rand.Intn(to-from)
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
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
