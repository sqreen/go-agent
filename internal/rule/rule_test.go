// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/rule"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type instrumentationMockup struct{ mock.Mock }

func (i *instrumentationMockup) Health(expectedVersion string) error {
	return i.Mock.Called(expectedVersion).Error(0)
}

type hookMockup struct{ mock.Mock }

var _ rule.HookFace = &hookMockup{}

func (i *instrumentationMockup) Find(symbol string) (rule.HookFace, error) {
	res := i.Called(symbol)
	err := res.Error(1)
	if h := res.Get(0); h != nil {
		return h.(rule.HookFace), err
	}
	return nil, err
}

func (i *instrumentationMockup) ExpectFind(symbol string) *mock.Call {
	return i.On("Find", symbol)
}

func (h *hookMockup) Attach(prologs ...sqhook.PrologCallback) error {
	return h.Called(prologs).Error(0)
}

func (h *hookMockup) ExpectAttach(prologs ...interface{}) *mock.Call {
	var args interface{}
	if l := len(prologs); l == 1 && prologs[0] == mock.Anything {
		args = prologs[0]
	} else {
		prologArgs := make([]sqhook.PrologCallback, l)
		for i, p := range prologs {
			prologArgs[i] = p
		}
		args = prologArgs
	}
	return h.On("Attach", args)
}

func (h *hookMockup) PrologFuncType() reflect.Type {
	return h.Called().Get(0).(reflect.Type)
}

type empty struct{}

var thisPkgPath = reflect.TypeOf(empty{}).PkgPath()

func TestEngineUsage(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	logger := plog.NewLogger(plog.Debug, os.Stderr, 0)
	metrics := metrics.NewEngine(plog.NewLogger(plog.Debug, os.Stderr, 0), 100000000)

	t.Run("empty state", func(t *testing.T) {
		instrumentation := &instrumentationMockup{}
		defer instrumentation.AssertExpectations(t)
		engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)

		// No problem using the engine without rules
		require.Empty(t, engine.PackID())
		engine.SetRules("my pack id", nil)
		require.Equal(t, engine.PackID(), "my pack id")
		engine.Enable()
		engine.Disable()
		engine.Enable()
		engine.SetRules("my other pack id", []api.Rule{})
		require.Equal(t, engine.PackID(), "my other pack id")
	})

	t.Run("setting multiple rules", func(t *testing.T) {
		rules := []api.Rule{
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"a valid rule"}`),
			},
			{
				Name: "another valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func2",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"another valid rule"}`),
			},
			{
				Name: "valid rule but no hookpoint",
				Hookpoint: api.Hookpoint{
					Method:   "main.main",
					Callback: "WriteCustomErrorPage",
				},
				Signature: MakeSignature(privateKey, `{"name":"my rule"}`),
			},
			{
				Name: "valid rule but unknown callback",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func2",
					Callback: "don't exist",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"another valid rule"}`),
			},
		}

		t.Run("when disabled", func(t *testing.T) {
			instrumentation := &instrumentationMockup{}
			defer instrumentation.AssertExpectations(t)

			hook1 := &hookMockup{}
			defer hook1.AssertExpectations(t)

			hook2 := &hookMockup{}
			defer hook2.AssertExpectations(t)

			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func1")).Return(hook1, nil).Once()
			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func2")).Return(hook2, nil).Twice()
			instrumentation.ExpectFind("main.main").Return(rule.HookFace(nil), nil).Once()

			engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)
			engine.Disable()
			engine.SetRules("yet another pack id", rules)
		})

		t.Run("enabling the rules attaches the callbacks", func(t *testing.T) {
			hook1 := &hookMockup{}
			defer hook1.AssertExpectations(t)
			hook1.ExpectAttach(mock.Anything).Return(nil).Once()

			hook2 := &hookMockup{}
			defer hook2.AssertExpectations(t)
			hook2.ExpectAttach(mock.Anything).Return(nil).Once()

			instrumentation := &instrumentationMockup{}
			defer instrumentation.AssertExpectations(t)
			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func1")).Return(hook1, nil).Once()
			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func2")).Return(hook2, nil).Twice()
			instrumentation.ExpectFind("main.main").Return(rule.HookFace(nil), nil).Once()

			engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)
			engine.SetRules("my pack id", rules)
			// Enable the rules: callbacks should be attached
			engine.Enable()
		})

		t.Run("disabling the rules removes the callbacks", func(t *testing.T) {
			instrumentation := &instrumentationMockup{}
			hook1 := &hookMockup{}
			hook2 := &hookMockup{}

			engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)
			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func1")).Return(hook1, nil).Once()
			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func2")).Return(hook2, nil).Twice()
			instrumentation.ExpectFind("main.main").Return(rule.HookFace(nil), nil).Once()
			engine.SetRules("my pack id", rules)
			instrumentation.AssertExpectations(t)

			// Enable the rules: callbacks should be attached
			hook1.ExpectAttach(mock.Anything).Return(nil).Once()
			hook2.ExpectAttach(mock.Anything).Return(nil).Once()
			engine.Enable()
			hook1.AssertExpectations(t)
			hook2.AssertExpectations(t)

			// Disable the rules: callbacks should be removed
			hook2.ExpectAttach(nil).Return(nil).Once()
			hook1.ExpectAttach(nil).Return(nil).Once()
			engine.Disable()
			hook1.AssertExpectations(t)
			hook2.AssertExpectations(t)
		})

		t.Run("enabling the rules again sets back the callbacks", func(t *testing.T) {
			instrumentation := &instrumentationMockup{}
			hook1 := &hookMockup{}
			hook2 := &hookMockup{}

			engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)
			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func1")).Return(hook1, nil).Once()
			instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func2")).Return(hook2, nil).Twice()
			instrumentation.ExpectFind("main.main").Return(rule.HookFace(nil), nil).Once()
			engine.SetRules("my pack id", rules)
			instrumentation.AssertExpectations(t)

			// Enable the rules: callbacks should be attached
			hook1.ExpectAttach(mock.Anything).Return(nil).Once()
			hook2.ExpectAttach(mock.Anything).Return(nil).Once()
			engine.Enable()
			hook1.AssertExpectations(t)
			hook2.AssertExpectations(t)

			// Disable the rules: callbacks should be removed
			hook1.ExpectAttach(nil).Return(nil).Once()
			hook2.ExpectAttach(nil).Return(nil).Once()
			engine.Disable()
			hook1.AssertExpectations(t)
			hook2.AssertExpectations(t)

			// Re-enable the rules: callbacks should be re-attached
			hook1.ExpectAttach(mock.Anything).Return(nil).Once()
			hook2.ExpectAttach(mock.Anything).Return(nil).Once()
			engine.Enable()
			hook1.AssertExpectations(t)
			hook2.AssertExpectations(t)
		})
	})

	t.Run("modify enabled rules", func(t *testing.T) {
		rules1 := []api.Rule{
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"a valid rule"}`),
			},
			{
				Name: "another valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func2",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"another valid rule"}`),
			},
			{
				Name: "valid rule but no hookpoint",
				Hookpoint: api.Hookpoint{
					Method:   "main.main",
					Callback: "WriteCustomErrorPage",
				},
				Signature: MakeSignature(privateKey, `{"name":"my rule"}`),
			},
			{
				Name: "valid rule but unknown callback",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func2",
					Callback: "don't exist",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"another valid rule"}`),
			},
		}
		rules2 := []api.Rule{
			{
				Name: "another valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func3",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"another valid rule"}`),
			},
		}

		instrumentation := &instrumentationMockup{}
		hook1 := &hookMockup{}
		hook2 := &hookMockup{}
		hook3 := &hookMockup{}

		engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)
		engine.Enable()

		instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func1")).Return(hook1, nil).Once()
		instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func2")).Return(hook2, nil).Twice()
		instrumentation.ExpectFind("main.main").Return(rule.HookFace(nil), nil).Once()
		hook1.ExpectAttach(mock.Anything).Return(nil).Once()
		hook2.ExpectAttach(mock.Anything).Return(nil).Once()
		engine.SetRules("a pack id", rules1)
		instrumentation.AssertExpectations(t)
		hook1.AssertExpectations(t)
		hook2.AssertExpectations(t)

		// Modify the rules while enabled: hooks no longer used should be disabled
		hook1.ExpectAttach(nil).Return(nil).Once()           // disabled
		hook2.ExpectAttach(nil).Return(nil).Once()           // disabled
		hook3.ExpectAttach(mock.Anything).Return(nil).Once() // enabled
		instrumentation.ExpectFind(fmt.Sprintf("%s.%s", thisPkgPath, "func3")).Return(hook3, nil).Twice()
		engine.SetRules("another pack id", rules2)
		hook1.AssertExpectations(t)
		hook2.AssertExpectations(t)
		hook3.AssertExpectations(t)
	})

	t.Run("replace the enabled rules with an empty array of rules", func(t *testing.T) {
		rules := []api.Rule{
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"a valid rule"}`),
			},
			{
				Name: "another valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func2",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"another valid rule"}`),
			},
			{
				Name: "valid rule but no hookpoint",
				Hookpoint: api.Hookpoint{
					Method:   "main.main",
					Callback: "WriteCustomErrorPage",
				},
				Signature: MakeSignature(privateKey, `{"name":"my rule"}`),
			},
			{
				Name: "valid rule but unknown callback",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func2",
					Callback: "don't exist",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: MakeSignature(privateKey, `{"name":"another valid rule"}`),
			},
		}

		instrumentation := &instrumentationMockup{}
		hook1 := &hookMockup{}
		hook2 := &hookMockup{}

		engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)
		engine.Enable()

		instrumentation.ExpectFind(thisPkgPath+".func1").Return(hook1, nil).Once()
		instrumentation.ExpectFind(thisPkgPath+".func2").Return(hook2, nil).Twice()
		instrumentation.ExpectFind("main.main").Return(rule.HookFace(nil), nil).Once()
		hook1.ExpectAttach(mock.Anything).Return(nil).Once()
		hook2.ExpectAttach(mock.Anything).Return(nil).Once()
		engine.SetRules("a pack id", rules)
		instrumentation.AssertExpectations(t)
		hook1.AssertExpectations(t)
		hook2.AssertExpectations(t)

		// Set the rules with an empty array while enabled: hooks should be disabled
		hook1.ExpectAttach(nil).Return(nil).Once()
		hook2.ExpectAttach(nil).Return(nil).Once()
		engine.SetRules("yet another pack id", []api.Rule{})
		hook1.AssertExpectations(t)
		hook2.AssertExpectations(t)
	})

	t.Run("add rules having signature issues", func(t *testing.T) {
		validSignature := MakeSignature(privateKey, `{"name":"a valid rule"}`).ECDSASignature

		instrumentation := &instrumentationMockup{}
		hook1 := &hookMockup{}

		engine := rule.NewEngine(logger, instrumentation, metrics, nil, publicKey)
		engine.Enable()

		// Set rules having signature errors
		engine.SetRules("a pack id", []api.Rule{
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: api.RuleSignature{ /*zero value*/ },
			},
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: api.RuleSignature{
					ECDSASignature: api.ECDSASignature{
						Message: validSignature.Message,
						/* zero signature value */
					},
				},
			},
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: api.RuleSignature{
					ECDSASignature: api.ECDSASignature{
						Value: validSignature.Value,
						/* zero message value */
					},
				},
			},
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Method:   thisPkgPath + ".func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
				Signature: api.RuleSignature{
					ECDSASignature: api.ECDSASignature{
						Value:   validSignature.Value,
						Message: []byte(`wrong message`),
					},
				},
			},
		})
		instrumentation.AssertExpectations(t)
		hook1.AssertExpectations(t)
	})
}

func MakeSignature(privateKey *ecdsa.PrivateKey, message string) api.RuleSignature {
	hash := sha512.Sum512([]byte(message))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		panic(err)
	}
	signature, err := asn1.Marshal(struct{ R, S *big.Int }{R: r, S: s})
	if err != nil {
		panic(err)
	}
	return api.RuleSignature{
		ECDSASignature: api.ECDSASignature{
			Message: []byte(message),
			Value:   base64.StdEncoding.EncodeToString(signature),
		},
	}
}
