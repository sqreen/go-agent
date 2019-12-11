// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testlib

import (
	"math/rand"
	"unicode"

	"golang.org/x/net/http/httpguts"
)

func RandHTTPHeaderValue(length ...int) string {
	str := RandPrintableUSASCIIString(length...)
	if !httpguts.ValidHeaderFieldValue(str) {
		panic("unexpected invalid HTTP header value")
	}
	return str
}

func RandPrintableUSASCIIString(length ...int) string {
	numChars := randStringLength(length...)
	buf := make([]byte, numChars)
	for i := 0; i < numChars; i++ {
		// Any printable USASCII character: between 32 and MaxASCII
		buf[i] = byte(32) + byte(rand.Intn(unicode.MaxASCII-32))
	}
	return string(buf)
}

func RandUTF8String(length ...int) string {
	numChars := randStringLength(length...)
	codePoints := make([]rune, numChars)
	for i := 0; i < numChars; i++ {
		// Get a random utf8 character code point which is any value between 0 and
		// 0x10FFFF, including non-printable and control characters
		codePoints[i] = rune(rand.Intn(unicode.MaxRune))
	}
	return string(codePoints)
}

func randStringLength(size ...int) (length int) {
	var from, to int
	if len(size) == 1 {
		// String length up to the given max length value
		to = size[0]
	} else if len(size) == 2 {
		// String length between the given boundaries
		from = size[0]
		to = size[1]
	} else {
		// String length up to 1024 characters
		to = rand.Intn(1024)
	}
	upTo := to - from
	if upTo == 0 {
		// Avoid Intn(0) which panics
		return from
	}
	return from + rand.Intn(upTo)

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
