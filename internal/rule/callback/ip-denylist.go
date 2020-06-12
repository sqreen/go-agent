// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"fmt"
	"net"

	"github.com/sqreen/go-agent/internal/actor"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	sdktypes "github.com/sqreen/go-agent/sdk/types"
)

func NewIPDenyListCallback(rule RuleFace, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	sqassert.NotNil(rule)
	sqassert.NotNil(cfg)

	data, ok := cfg.Data().([]interface{})
	if !ok {
		return nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `%T`", cfg.Data(), data)
	}

	if l := len(data); l == 0 {
		return nil, sqerrors.New("empty denylist`")
	} else if l > 1 {
		return nil, sqerrors.Errorf("unexpected number of data entries`")
	}

	d1 := data[0]
	cidrs, ok := d1.([]string)
	if !ok {
		return nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `%T`", d1, cidrs)
	}

	if len(cidrs) == 0 {
		return nil, sqerrors.New("empty denylist`")
	}

	denylist, err := actor.NewCIDRIPListStore(cidrs)
	if err != nil {
		return nil, sqerrors.Wrapf(err, "unexpected error while creating the IP denylist")
	}
	return newIPDenyListPrologCallback(rule, denylist), nil
}

type IPDenyListPrologCallbackType = httpprotection.BlockingPrologCallbackType
type IPDenyListEpilogCallbackType = httpprotection.BlockingEpilogCallbackType

type IPDenyListError struct {
	DeniedIP      net.IP
	DenyListEntry string
}

func (e IPDenyListError) Error() string {
	return fmt.Sprintf("ip address `%s` matched the denylist entry `%s` and was blocked", e.DeniedIP.String(), e.DenyListEntry)
}

func newIPDenyListPrologCallback(rule RuleFace, denylist *actor.CIDRIPListStore) IPDenyListPrologCallbackType {
	return func(p **httpprotection.RequestContext) (IPDenyListEpilogCallbackType, error) {
		ctx := *p
		ip := ctx.RequestReader.ClientIP()
		exists, matched, err := denylist.Find(ip)

		if err != nil {
			ctx.Logger().Error(sqerrors.Wrapf(err, "unexpected error while searching IP address `%#+v` in the IP denylist", ip))
			return nil, nil
		}

		if !exists {
			return nil, nil
		}

		ctx.WriteDefaultBlockingResponse()

		if err := rule.PushMetricsValue(matched, 1); err != nil {
			ctx.Logger().Error(sqerrors.Wrapf(err, "could not update the metrics"))
		}

		return func(e *error) {
			sqassert.NotNil(e)
			err := sdktypes.SqreenError{
				Err: IPDenyListError{
					DeniedIP:      ip,
					DenyListEntry: matched,
				},
			}
			// Display the error message explaining why the request is denied.
			ctx.Logger().Debug(err.Error())
			*e = err
		}, nil
	}
}
