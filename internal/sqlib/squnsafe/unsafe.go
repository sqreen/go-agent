// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package squnsafe

import "unsafe"

// StringToBytes returns the given string as an slice of bytes without copying
// it into a new slice. The empty string "" returns a nil slice. The returned
// slice points to the same string and it mustn't be modified to keep the
// original string immutable.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
