// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqerrors

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
)

// Frame represents a program counter inside a stack frame.
// For historical reasons if Frame is interpreted as a uintptr
// its value represents the program counter + 1.
type Frame errors.Frame

// PC returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f Frame) PC() uintptr { return uintptr(f) - 1 }

// File returns the full path to the file that contains the
// function for this Frame's pc.
func (f Frame) File() string {
	pc := f.PC()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}
	file, _ := fn.FileLine(pc)
	return file
}

// Line returns the line number of source code of the
// function for this Frame's pc.
func (f Frame) Line() int {
	pc := f.PC()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(pc)
	return line
}

// name returns the name of this function, if known.
func (f Frame) Name() string {
	pc := f.PC()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return fmt.Sprintf("unknown function name at 0x%x", pc)
	}
	return fn.Name()
}
