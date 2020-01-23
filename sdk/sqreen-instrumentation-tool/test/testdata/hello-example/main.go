// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"fmt"

	"github.com/sqreen/go-agent/sdk/instrumentation/sqreen/test/testdata/helpers"
)

// no params nor results
func f1() { defer helpers.TraceCall()() }

// named param
func f2(p0 string) { defer helpers.TraceCall()() }

// ignored param - using _ - single
func f3(_ string) { defer helpers.TraceCall()() }

// ignored param - using _ - multiple
func f4(_ string, _ int) { defer helpers.TraceCall()() }

// ignored param - using _ - multiple
func f5(_, _ string) { defer helpers.TraceCall()() }

// ignored param - unnamed - single
func f6(string) { defer helpers.TraceCall()() }

// ignored param - unnamed - multiple
func f7(string, int) { defer helpers.TraceCall()() }

// ignored param - numbering - first
func f8(_, _ string, p2 int) { defer helpers.TraceCall()() }

// ignored param: last
func f9(p0 string, _, _ string) { defer helpers.TraceCall()() }

// ignored param: in between
func f10(p0 string, _, _ string, p3 int) { defer helpers.TraceCall()() }

// variadic function
func f11(p ...interface{}) { defer helpers.TraceCall()() }

// named result
func f12() (r0 string) {
	defer helpers.TraceCall()()
	return "f12r0"
}

// ignored result - using _ - single
func f13() (_ string) {
	defer helpers.TraceCall()()
	return "f13r0"
}

// ignored param - using _ - multiple
func f14() (_ string, _ int) {
	defer helpers.TraceCall()()
	return "f14r0", 14
}

// ignored param - using _ - multiple
func f15() (_, _ string) {
	defer helpers.TraceCall()()
	return "f15r0", "f15r1"
}

// ignored param - unnamed - single
func f16() string {
	defer helpers.TraceCall()()
	return "f16r0"
}

// ignored param - unnamed - multiple
func f17() (string, int) {
	defer helpers.TraceCall()()
	return "f17r0", 17
}

// ignored param - numbering - first
func f18() (_, _ string, r2 int) {
	defer helpers.TraceCall()()
	return "f18r0", "f18r1", 18
}

// ignored param: last
func f19() (r0 string, _, _ string) {
	defer helpers.TraceCall()()
	return "f19r0", "f19r1", "f19r2"
}

// ignored param: in between
func f20() (r0 string, _, _ string, r3 int) {
	defer helpers.TraceCall()()
	return "f20r0", "f20r1", "f20r2", 20
}

// param name hides param type name
func f21(int int) int {
	defer helpers.TraceCall()()
	return 21
}

// result name hides type name
func f22() (int int) {
	defer helpers.TraceCall()()
	return 22
}

// result name hides type name
func f23(p0 string, p1 int) (r0 bool, int int, r2 rune) {
	defer helpers.TraceCall()()
	return true, 23, 'r'
}

// not instrumentation: nosplit pragma
//go:nosplit
func f24() { defer helpers.TraceCall()() }

// not instrumented: ignored function
//go:nosplit
func _() { defer helpers.TraceCall()() }

// not instrumentation: sqreen:ignore pragma
//sqreen:ignore
func f25() { defer helpers.TraceCall()() }

type s struct{}

func (r s) m0()    { defer helpers.TraceCall()() }
func (r s) m1(int) { defer helpers.TraceCall()() }
func (r *s) m2()   { defer helpers.TraceCall()() }
func (*s) m3()     { defer helpers.TraceCall()() }
func (s) m4()      { defer helpers.TraceCall()() }
func (s) m5(_ int) { defer helpers.TraceCall()() }
func (s) m6(int)   { defer helpers.TraceCall()() }

func main() {
	defer helpers.TraceCall()()
	fmt.Println("Hello, Go!")

	f1()

	f2("f2 string")
	f3("f2 string")
	f4("f4 string", 4)
	f5("f5str1", "f5str2")
	f6("f6 string")
	f7("f7 string", 7)
	f8("f8str1", "f8str2", 8)
	f9("f9str1", "f9str2", "f9str3")
	f10("f10str1", "f10str2", "f10str3", 10)
	f11(11, "f11str2", true, []int{1, 2, 3})

	fmt.Println("f12 =", f12())
	fmt.Println("f13 =", f13())
	f14r0, f14r1 := f14()
	fmt.Println("f14 =", f14r0, f14r1)
	f15r0, f15r1 := f15()
	fmt.Println("f15 =", f15r0, f15r1)
	fmt.Println("f16 =", f16())
	f17r0, f17r1 := f17()
	fmt.Println("f17 =", f17r0, f17r1)
	f18r0, f18r1, f18r2 := f18()
	fmt.Println("f18 =", f18r0, f18r1, f18r2)
	f19r0, f19r1, f19r2 := f19()
	fmt.Println("f19 =", f19r0, f19r1, f19r2)
	f20r0, f20r1, f20r2, f20r3 := f20()
	fmt.Println("f20 =", f20r0, f20r1, f20r2, f20r3)
	fmt.Println("f21 =", f21(21))
	fmt.Println("f22 =", f22())
	f23r0, f23r1, f23r2 := f23("f23str", 23)
	fmt.Println("f23 =", f23r0, f23r1, f23r2)

	f24()
	f25()

	s := s{}
	s.m0()
	s.m2()
	s.m3()
	s.m4()
	s.m5(5)
	s.m6(6)

	fmt.Println("Bye, Go!")
}
