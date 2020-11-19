// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"fmt"
	"net"

	"github.com/sqreen/go-agent/internal/actor"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	sdk_types "github.com/sqreen/go-agent/sdk/types"
)

func NewIPDenyListCallback(r RuleContext, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	sqassert.NotNil(r)
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
	return newIPDenyListPrologCallback(r, denylist), nil
}

type IPDenyListPrologCallbackType = http_protection.BlockingPrologCallbackType
type IPDenyListEpilogCallbackType = http_protection.BlockingEpilogCallbackType

type IPDenyListError struct {
	DeniedIP      net.IP
	DenyListEntry string
}

func (e IPDenyListError) Error() string {
	return fmt.Sprintf("ip address `%s` matched the denylist entry `%s` and was blocked", e.DeniedIP.String(), e.DenyListEntry)
}

func newIPDenyListPrologCallback(r RuleContext, denylist *actor.CIDRIPListStore) IPDenyListPrologCallbackType {
	return func(**http_protection.ProtectionContext) (epilog IPDenyListEpilogCallbackType, prologErr error) {
		r.Pre(func(c CallbackContext) error {
			ip := c.ProtectionContext().ClientIP()
			exists, matched, err := denylist.Find(ip)
			if err != nil {
				type errKey struct{}
				return sqerrors.WithKey(sqerrors.Wrapf(err, "unexpected error while searching IP address `%#+v` in the IP denylist", ip), errKey{})
			}

			if !exists {
				return nil
			}

			c.HandleAttack(true)

			_ = c.AddMetricsValue(matched, 1)

			epilog = func(e *error) {
				sqassert.NotNil(e)
				err := sdk_types.SqreenError{
					Err: IPDenyListError{
						DeniedIP:      ip,
						DenyListEntry: matched,
					},
				}
				// Display the error message explaining why the request is denied.
				c.Logger().Debug(err.Error())
				*e = err
			}
			return nil
		})
		return
	}
}
