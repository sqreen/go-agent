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
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/sqreen/go-agent/sdk/types"
)

func NewShellshockCallback(r RuleContext, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	sqassert.NotNil(r)
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

	return newShellshockPrologCallback(r, regexps), nil
}

type ShellshockPrologCallbackType = func(name *string, argv *[]string, attr **os.ProcAttr) (ShellshockEpilogCallbackType, error)
type ShellshockEpilogCallbackType = func(**os.Process, *error)

type ShellshockAttackInfo struct {
	Found         string `json:"found"`
	VariableName  string `json:"variable_name"`
	VariableValue string `json:"variable_value"`
}

func newShellshockPrologCallback(r RuleContext, regexps []*regexp.Regexp) ShellshockPrologCallbackType {
	return func(_ *string, _ *[]string, attr **os.ProcAttr) (epilog ShellshockEpilogCallbackType, prologErr error) {
		r.Pre(func(c CallbackContext) error {
			env := (*attr).Env
			if env == nil {
				env = os.Environ()
			}
			if len(env) == 0 {
				return nil
			}
			for _, env := range env {
				v := strings.SplitN(env, `=`, 2)
				if l := len(v); l <= 0 || l > 2 {
					type errKey struct{}
					return sqerrors.WithKey(sqerrors.Errorf("unexpected number of elements split by `=` in `%s`", env), errKey{})
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

						if blocked := c.HandleAttack(true, event.WithAttackInfo(info), event.WithStackTrace()); blocked {
							epilog = func(_ **os.Process, callErr *error) {
								*callErr = types.SqreenError{Err: errors.New("shellshock protection")}
							}
							prologErr = sqhook.AbortError
							return nil
						}
					}
				}
			}
			return nil
		})
		return
	}
}
