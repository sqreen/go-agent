// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule_test

import (
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/metrics"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/internal/rule"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
	"github.com/stretchr/testify/require"
)

func func1(_ http.ResponseWriter, _ *http.Request, _ http.Header, _ int, _ []byte) {}
func func2(_ http.ResponseWriter, _ *http.Request, _ http.Header, _ int, _ []byte) {}

type empty struct{}

func TestEngineUsage(t *testing.T) {
	logger := plog.NewLogger(plog.Debug, os.Stderr, 0)
	engine := rule.NewEngine(logger, metrics.NewEngine(plog.NewLogger(plog.Debug, os.Stderr, 0)))
	hookFunc1 := sqhook.New(func1)
	require.NotNil(t, hookFunc1)
	hookFunc2 := sqhook.New(func2)
	require.NotNil(t, hookFunc2)

	t.Run("empty state", func(t *testing.T) {
		require.Empty(t, engine.PackID())
		engine.SetRules("my pack id", nil)
		require.Equal(t, engine.PackID(), "my pack id")
		// No problem enabling/disabling the engine
		engine.Enable()
		engine.Disable()
		engine.Enable()
		engine.SetRules("my other pack id", []api.Rule{})
		require.Equal(t, engine.PackID(), "my other pack id")
	})

	t.Run("multiple rules", func(t *testing.T) {
		engine.Disable()
		engine.SetRules("yet another pack id", []api.Rule{
			{
				Name: "a valid rule",
				Hookpoint: api.Hookpoint{
					Class:    reflect.TypeOf(empty{}).PkgPath(),
					Method:   "func1",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
			},
			{
				Name: "another valid rule",
				Hookpoint: api.Hookpoint{
					Class:    reflect.TypeOf(empty{}).PkgPath(),
					Method:   "func2",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
			},
		})

		t.Run("callbacks are not attached when disabled", func(t *testing.T) {
			// Check the callbacks were not attached because rules are disabled
			prologFunc1, epilogFunc1 := hookFunc1.Callbacks()
			require.Nil(t, prologFunc1)
			require.Nil(t, epilogFunc1)
			prologFunc2, epilogFunc2 := hookFunc2.Callbacks()
			require.Nil(t, prologFunc2)
			require.Nil(t, epilogFunc2)
		})

		t.Run("enabling the rules attaches the callbacks", func(t *testing.T) {
			// Enable the rules
			engine.Enable()
			// Check the callbacks were now attached
			prologFunc1, epilogFunc1 := hookFunc1.Callbacks()
			require.NotNil(t, prologFunc1)
			require.Nil(t, epilogFunc1)
			prologFunc2, epilogFunc2 := hookFunc2.Callbacks()
			require.NotNil(t, prologFunc2)
			require.Nil(t, epilogFunc2)
		})

		t.Run("disabling the rules removes the callbacks", func(t *testing.T) {
			// Disable the rules
			engine.Disable()
			// Check the callbacks were all removed for func1 and not func2
			prologFunc1, epilogFunc1 := hookFunc1.Callbacks()
			require.Nil(t, prologFunc1)
			require.Nil(t, epilogFunc1)
			prologFunc2, epilogFunc2 := hookFunc2.Callbacks()
			require.Nil(t, prologFunc2)
			require.Nil(t, epilogFunc2)
		})

		t.Run("enabling the rules again sets back the callbacks", func(t *testing.T) {
			// Enable again the rules
			engine.Enable()
			// Check the callbacks are attached again
			prologFunc1, epilogFunc1 := hookFunc1.Callbacks()
			require.NotNil(t, prologFunc1)
			require.Nil(t, epilogFunc1)
			prologFunc2, epilogFunc2 := hookFunc2.Callbacks()
			require.NotNil(t, prologFunc2)
			require.Nil(t, epilogFunc2)
		})
	})

	t.Run("modify enabled rules", func(t *testing.T) {
		// Modify the rules while enabled
		engine.SetRules("a pack id", []api.Rule{
			{
				Name: "another valid rule",
				Hookpoint: api.Hookpoint{
					Class:    reflect.TypeOf(empty{}).PkgPath(),
					Method:   "func2",
					Callback: "WriteCustomErrorPage",
				},
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
			},
		})
		// Check the callbacks were removed for func1 and not func2
		prologFunc1, epilogFunc1 := hookFunc1.Callbacks()
		require.Nil(t, prologFunc1)
		require.Nil(t, epilogFunc1)
		prologFunc2, epilogFunc2 := hookFunc2.Callbacks()
		require.NotNil(t, prologFunc2)
		require.Nil(t, epilogFunc2)
	})

	t.Run("replace the enabled rules with an empty array of rules", func(t *testing.T) {
		// Set the rules with an empty array while enabled
		engine.SetRules("yet another pack id", []api.Rule{})
		// Check the callbacks were all removed for func1 and not func2
		prologFunc1, epilogFunc1 := hookFunc1.Callbacks()
		require.Nil(t, prologFunc1)
		require.Nil(t, epilogFunc1)
		prologFunc2, epilogFunc2 := hookFunc2.Callbacks()
		require.Nil(t, prologFunc2)
		require.Nil(t, epilogFunc2)
	})
}
