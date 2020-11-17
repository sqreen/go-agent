// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhttp

import (
	"net/http"
)

func wrapResponseWriter(w http.ResponseWriter) (http.ResponseWriter, *responseWriterObserver) {
	wrapper := &responseWriterObserver{ResponseWriter: w}
	w = adaptResponseWriter(wrapper, w)
	return w, wrapper
}
