// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

// A binding accessor is an expression allowing to get data from a given
// context. It is used in order to dynamically get data from, for example,
// requests. Rules the agent receives can therefore contain the binding
// accessor expressions to evaluate and pass to the callbacks.
//
// The compiled expression is evaluated upon a Go value and returns the
// resulting Go value.
package bindingaccessor

import (
	"errors"
	"strconv"
	"strings"

	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
)

// Context is the data type a binding accessor expression is executed with.
// This data can be therefore used using the binding accessor.
type Context interface{}

// BindingAccessorFunc is a compiled binding accessor expression function.
// Errors or panics during the binding accessor execution are returned as an
// error.
type BindingAccessorFunc func(ctx Context) (value interface{}, err error)

// A valueFunc value is a compiled binding accessor expression. Executing it
// returns the result of the expression with the given context `ctx`. They can
// be combined together in order to represent a whole binding accessor
// expression through different function closures (field access, array access,
// map access, etc.).
type valueFunc func(ctx Context, maxDepth int) (value interface{}, err error)

// A transformationFunc value transforms an input value to another output value.
// The traversal of the input value cannot be deeper than maxDepth, and the
// output value cannot exceed maxElements.
type transformationFunc func(ctx Context, valueIn interface{}, maxDepth, maxElements int) (valueOut interface{})

// Maximum binding accessor execution depth. The binding accessor execution
// traverses Go values. It cannot go deeper than this value.
const MaxExecutionDepth = 10

// ErrMaxExecutionDepth is returned by the BindingAccessorFunc when the
// binding accessor execution reached the maximum depth `MaxExecutionDepth`.
var ErrMaxExecutionDepth = errors.New("maximum binding accessor execution depth reached")

// Compile returns the compiled binding accessor expression function.
func Compile(expr string) (program BindingAccessorFunc, err error) {
	defer func() {
		if err != nil {
			err = sqerrors.Wrap(err, "binding accessor compilation error")
		}
	}()

	exprFn, err := compileExpr(expr)
	if err != nil {
		return nil, err
	}

	// Wrap into a safe function that cannot panic
	return func(ctx Context) (value interface{}, err error) {
		// Panics are how `reflect` returns forbidden and unexpected accesses.
		// We need to catch any panic and return it as an error.
		err = sqsafe.Call(func() error {
			var err error
			value, err = exprFn(ctx, MaxExecutionDepth)
			return err
		})
		if err != nil {
			return nil, sqerrors.Wrap(err, "binding accessor execution error")
		}
		return value, nil
	}, nil
}

func compileExpr(expr string) (valueFunc, error) {
	buf := strings.TrimSpace(expr)
	// Get the first operand
	valueFn, buf, err := compileOperand(buf)
	if err != nil {
		return nil, err
	}

	for len(buf) > 0 {
		switch buf[0] {
		case '(':
			valueFn, buf, err = compileCall(valueFn, buf[1:])
			if err != nil {
				return nil, err
			}
		case '.':
			valueFn, buf, err = compileField(valueFn, buf[1:])
			if err != nil {
				return nil, err
			}
		case '[':
			valueFn, buf, err = compileIndex(valueFn, buf[1:])
			if err != nil {
				return nil, err
			}
		case '|':
			return compileTransformations(valueFn, buf[1:])
		default:
			return nil, sqerrors.Errorf("undefined operation `%c` in `%s`", buf[0], expr)
		}
	}
	return valueFn, nil
}

const (
	newValueMaxDepth    = 10
	NewValueMaxElements = 150
)

func compileTransformations(valueFn valueFunc, buf string) (valueFunc, error) {
	pipeline := strings.Split(buf, "|")
	for _, tr := range pipeline {
		trFn, err := compileTransformation(tr)
		if err != nil {
			return nil, err
		}
		lastValueFn := valueFn
		valueFn = func(ctx Context, depth int) (value interface{}, err error) {
			v, err := lastValueFn(ctx, depth)
			if err != nil {
				return nil, err
			}
			return trFn(ctx, v, newValueMaxDepth, NewValueMaxElements), nil
		}
	}
	return valueFn, nil
}

func compileCall(valueFn valueFunc, buf string) (valueFunc, string, error) {
	close := strings.IndexByte(buf, ')')
	if close == -1 {
		return nil, buf, sqerrors.Errorf("missing closing index bracket `)` in `%s`", buf)
	}

	// Compile the list of arguments
	var args []valueFunc
	if argList := buf[:close]; len(argList) > 0 {
		argv := strings.Split(argList, ",")
		args = make([]valueFunc, len(argv))
		for i, a := range argv {
			arg, err := compileExpr(a)
			if err != nil {
				return nil, buf, err
			}
			args[i] = arg
		}
	}

	buf = buf[close:]
	if len(buf) == 1 {
		buf = buf[:0]
	} else {
		buf = buf[1:]
	}

	return func(ctx Context, depth int) (interface{}, error) {
		if depth == 0 {
			return nil, ErrMaxExecutionDepth
		}
		v, err := valueFn(ctx, depth-1)
		if err != nil {
			return nil, err
		}
		argValues := make([]interface{}, len(args))
		for i, arg := range args {
			a, err := arg(ctx, depth-1)
			if err != nil {
				return nil, err
			}
			argValues[i] = a
		}
		return execCall(v, argValues...)
	}, buf, nil
}

func compileField(valueFn valueFunc, buf string) (valueFunc, string, error) {
	field, buf := parseIdentifier(buf)
	field = strings.TrimRight(field, " ")
	if len(field) == 0 {
		return nil, buf, sqerrors.New("unexpected empty field name")
	}

	return func(ctx Context, depth int) (value interface{}, err error) {
		if depth == 0 {
			return nil, ErrMaxExecutionDepth
		}
		v, err := valueFn(ctx, depth-1)
		if err != nil {
			return nil, err
		}
		return execFieldAccess(v, field)
	}, buf, nil
}

func parseIndex(s string) (interface{}, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil, sqerrors.New("empty index value")
	}
	if l := len(s); s[0] == '\'' && s[l-1] == '\'' {
		return s[1 : l-1], nil
	}

	n, err := strconv.Atoi(s)
	if err == nil {
		return n, nil
	}

	return nil, sqerrors.Errorf("unexpected index value `%s`", s)
}

func compileOperand(buf string) (valueFunc, string, error) {
	return compileIdentifier(buf)
}

func parseIdentifier(buf string) (identifier string, newBuf string) {
	switch separator := strings.IndexAny(buf, "([.|"); separator {
	case -1:
		return buf, buf[:0]
	default:
		return buf[:separator], buf[separator:]
	}
}

func compileIdentifier(buf string) (valueFunc, string, error) {
	identifier, buf := parseIdentifier(buf)
	identifier = strings.TrimSpace(identifier)

	var valueFn valueFunc
	switch identifier {
	default:
		// Try match string value
		if l := len(identifier); l >= 2 && identifier[0] == '\'' && identifier[l-1] == '\'' {
			str := identifier[1 : l-1]
			valueFn = func(ctx Context, depth int) (interface{}, error) {
				return str, nil
			}
			break
		}
		return nil, buf, sqerrors.Errorf("unknown identifier `%s`", identifier)
	case "#":
		valueFn = func(ctx Context, depth int) (interface{}, error) {
			return ctx, nil
		}
	case "nil":
		fallthrough
	case "null":
		valueFn = func(ctx Context, depth int) (interface{}, error) {
			return nil, nil
		}
	}

	return valueFn, buf, nil
}

func compileIndex(valueFn valueFunc, buf string) (valueFunc, string, error) {
	close := strings.IndexByte(buf, ']')
	if close == -1 {
		return nil, buf, sqerrors.Errorf("missing closing index bracket `]` in `%s`", buf)
	}

	index, err := parseIndex(buf[:close])
	if err != nil {
		return nil, buf, err
	}

	buf = buf[close:]
	if len(buf) == 1 {
		buf = buf[:0]
	} else {
		buf = buf[1:]
	}

	return func(ctx Context, depth int) (interface{}, error) {
		if depth == 0 {
			return nil, ErrMaxExecutionDepth
		}
		v, err := valueFn(ctx, depth-1)
		if err != nil {
			return nil, err
		}
		return execIndexAccess(v, index)
	}, buf, nil
}

func compileTransformation(buf string) (transformationFunc, error) {
	buf = strings.TrimSpace(buf)
	switch buf {
	case "flat_values":
		return execFlatValues, nil
	case "flat_keys":
		return execFlatKeys, nil
	default:
		return nil, sqerrors.Errorf("unexpected transformation function `%s`", buf)
	}
}
