// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/sqreen/go-agent/sdk/types"
)

func NewShellshockCallback(rule RuleFace, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	sqassert.NotNil(rule)
	sqassert.NotNil(cfg)

	data, ok := cfg.Data().([]interface{})
	if !ok {
		return nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `%T`", cfg.Data(), data)
	}

	if l := len(data); l == 0 {
		return nil, sqerrors.New("empty list of regular expressions`")
	} else if l > 1 {
		return nil, sqerrors.Errorf("unexpected number of data entries`")
	}

	values, ok := data[0].([]string)
	if !ok {
		return nil, sqerrors.Errorf("unexpected callback data values type: got `%T` instead of `%T`", data[0], values)
	}
	regexps := make([]*regexp.Regexp, len(values))
	for i := range values {
		expr := values[i]
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, sqerrors.Wrapf(err, "could not compile the regular expression `%s`", expr)
		}

		regexps[i] = re
	}

	return newShellshockPrologCallback(rule, cfg.BlockingMode(), regexps), nil
}

type ShellshockPrologCallbackType = func(name *string, argv *[]string, attr **os.ProcAttr) (ShellshockEpilogCallbackType, error)
type ShellshockEpilogCallbackType = func(**os.Process, *error)

type ShellshockAttackInfo struct {
	Found         string `json:"found"`
	VariableName  string `json:"variable_name"`
	VariableValue string `json:"variable_value"`
}

func newShellshockPrologCallback(rule RuleFace, blockingMode bool, regexps []*regexp.Regexp) ShellshockPrologCallbackType {
	return func(_ *string, _ *[]string, attr **os.ProcAttr) (ShellshockEpilogCallbackType, error) {
		ctx := httpprotection.FromGLS()
		if ctx == nil {
			return nil, nil
		}

		rule.MonitorPre()

		env := (*attr).Env
		if env == nil {
			env = os.Environ()
		}
		if len(env) == 0 {
			return nil, nil
		}

		for _, env := range env {
			v := strings.SplitN(env, `=`, 2)
			if l := len(v); l <= 0 || l > 2 {
				ctx.Logger().Error(sqerrors.Errorf("unexpected number of elements split by `=` in `%s`", env))
				return nil, nil
			} else if l == 1 {
				// Skip empty values
				continue
			}

			name, value := v[0], v[1]
			for _, re := range regexps {
				if re.MatchString(value) {
					info := ShellshockAttackInfo{
						Found:         re.String(),
						VariableName:  name,
						VariableValue: value,
					}
					err := errors.WithStack(types.SqreenError{})
					ctx.AddAttackEvent(rule.NewAttackEvent(blockingMode, info, sqerrors.StackTrace(err)))

					if !blockingMode {
						return nil, nil
					}

					defer ctx.CancelHandlerContext()
					ctx.WriteDefaultBlockingResponse()
					return func(_ **os.Process, callErr *error) {
						*callErr = err
					}, sqhook.AbortError
				}
			}
		}

		return nil, nil
	}
}
