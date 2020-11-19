// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore no need for instrumentation of hook points in this test

package main

import (
	"reflect"
	"sync"

	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
	"github.com/sqreen/go-agent/tools/testlib"
)

type MyGLSType struct {
	s string
	i int
	b bool
	f float32
}

func getMyGLS() *MyGLSType {
	g := sqgls.Get()
	if g == nil {
		return nil
	}
	return g.(*MyGLSType)
}

func setMyGLS(v *MyGLSType) {
	sqgls.Set(v)
}

func main() {
	testGLS(func() {
		testGLS(func() {
			testGLS(func() {
				testGLS(func() {
					testGLS(nil)
				})
			})
		})
	})
}

func testGLS(Go func()) {
	gls := getMyGLS()
	if gls != nil {
		panic("unexpected non-nil gls value")
	}

	myGLS := &MyGLSType{
		s: testlib.RandUTF8String(),
		i: 0,
		b: false,
		f: 0,
	}
	setMyGLS(myGLS)

	gotGLS := getMyGLS()
	if gotGLS == nil {
		panic("unexpected nil gls value")
	}
	if gotGLS != myGLS {
		panic("unexpected different gls pointer value")
	}
	if !reflect.DeepEqual(gotGLS, myGLS) {
		panic("unexpected non equal gls values")
	}

	if Go != nil {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			Go()
			wg.Done()
		}()
		wg.Wait()
	}
}
