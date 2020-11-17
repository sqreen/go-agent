// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhttp

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdaptResponseWriter(t *testing.T) {
	// Dummy test taking some shortcuts but validating the wrapper and wrapped
	// values are correctly adapted:
	//    - the wrapper should be the one called
	//    - the returned wrapped value should implement the same interface as the
	//      wrapped value

	// TODO: generate the tests too so that we cover every possible case

	t.Run("Flusher", func(t *testing.T) {
		wrapper := &myFakeResponseWriter{}
		wrapped := myFakeResponseWriterFlusher{
			myFakeResponseWriter: wrapper,
		}

		w := adaptResponseWriter(wrapper, wrapped)

		// The wrapper should implement the interface
		_, ok := w.(http.Flusher)
		require.True(t, ok)

		// The wrapper method should be the one called
		w.WriteHeader(42)

		require.Equal(t, 42, wrapper.status)
	})

	t.Run("Flusher+StringWriter", func(t *testing.T) {
		wrapper := &myFakeResponseWriter{}
		wrapped := myFakeResponseWriterFlusherStringWriter{
			myFakeResponseWriter: wrapper,
		}

		w := adaptResponseWriter(wrapper, wrapped)

		// The wrapper should implement the interface
		_, ok := w.(interface {
			http.Flusher
			io.StringWriter
		})
		require.True(t, ok)

		// The wrapper method should be the one called
		w.WriteHeader(42)

		require.Equal(t, 42, wrapper.status)
	})
}

type myFakeResponseWriter struct {
	status int
}

func (*myFakeResponseWriter) Header() http.Header       { return nil }
func (*myFakeResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *myFakeResponseWriter) WriteHeader(s int)       { w.status = s }

type myFakeFlusher struct{}

func (myFakeFlusher) Flush() {}

type myFakeResponseWriterFlusher struct {
	*myFakeResponseWriter
	myFakeFlusher
}

type myFakeResponseWriterFlusherStringWriter struct {
	*myFakeResponseWriter
	myFakeFlusher
}

func (myFakeResponseWriterFlusherStringWriter) WriteString(string) (int, error) { return 0, nil }
