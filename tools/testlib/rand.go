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
