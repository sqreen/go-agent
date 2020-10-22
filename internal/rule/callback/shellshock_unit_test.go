// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

// TODO: higher-level callback API that will allow to avoid the missing GLS
//  during testing, while a callback framework should be completely mockable.

//func TestShellshockCallbacks(t *testing.T) {
//	regexps := []*regexp.Regexp{
//		regexp.MustCompile(`oh my regexp`),
//	}
//	ctx := &RuleMockup{}
//
//	prolog := newShellshockPrologCallback(ctx, true, regexps)
//
//	ctx.ExpectNewAttackEvent(true, ShellshockAttackInfo{
//		Found:         `oh my regexp`,
//		VariableName:  "name",
//		VariableValue: "does it match oh my regexp?",
//	})
//	attr := &os.ProcAttr{
//		Env: []string{"name=does it match oh my regexp?"},
//	}
//	epilog, err := prolog(nil, nil, &attr)
//	require.Error(t, err)
//	require.Equal(t, sqhook.AbortError, err)
//	require.NotNil(t, epilog)
//
//	epilog(nil, &err)
//	require.Error(t, err)
//	require.True(t, xerrors.As(err, &types.SqreenError{}))
//}
//
//type RuleMockup struct {
//	mock.Mock
//}
//
//func (r *RuleMockup) AddMetricsValue(key interface{}, value int64) error {
//	return r.Called(key, value).Error(0)
//}
//
//func (r *RuleMockup) ExpectPushMetricsValue(key interface{}, value int64) *mock.Call {
//	return r.On("AddMetricsValue", key, value)
//}
//
//func (r *RuleMockup) NewAttackEvent(blocked bool, info interface{}, st errors.StackTrace) *event.AttackEvent {
//	return r.Called(blocked, info, st).Get(0).(*event.AttackEvent)
//}
//
//func (r *RuleMockup) ExpectNewAttackEvent(blocked bool, info interface{}) *mock.Call {
//	return r.On("NewAttackEvent", blocked, info)
//}
