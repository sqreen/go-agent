package api_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/sqlib/sqsanitize"
	"github.com/stretchr/testify/require"
)

func TestWAFAttackInfo_Scrub(t *testing.T) {
	t.Run("AGO-137", func(t *testing.T) {
		// Test case covering issue https://sqreen.atlassian.net/browse/AGO-137

		// The idea of the following mess is to mock a WAF attack with a lowercase
		// transformation of the request parameters. The fix should correctly scrub
		// the request parameters in the attack.

		// Set of PII values used in this test. They are uppercase here while they
		// are lowercased in the attack information.
		pii := map[string]string{
			"access_token":  "PASSWORD_1",
			"api_key":       "PASSWORD_2",
			"apikey":        "PASSWORD_3",
			"authorization": "PASSWORD_4",
		}

		// Create test data including the resolved values of the WAF using lowercase
		// parameters and the request parameters including a separate attack
		// parameter.
		resolvedValues := make(map[string]string, len(pii))
		params := make(map[string][]interface{}, len(pii))
		for k, v := range pii {
			resolvedValues[k] = strings.ToLower(v)
			params[k] = []interface{}{v}
		}
		resolvedValues["attack"] = "java.lang.processbuilder"

		// Prepare the WAF data json string
		resolvedValuesJSON, err := json.Marshal(resolvedValues)
		require.NoError(t, err)
		resolvedValuesJSONStr, err := json.Marshal(string(resolvedValuesJSON))
		require.NoError(t, err)
		wafData := []byte(`[
                         {
                             "ret_code": 1,
                             "flow": "shell_injection-monitoring",
                             "step": "start",
                             "rule": "rule_944100",
                             "filter": [
                                 {
                                     "operator": "@rx",
                                     "operator_value": "java\\.lang\\.(?:runtime|processbuilder)",
                                     "binding_accessor": "#.Request.Body.String",
                                     "resolved_value": ` + string(resolvedValuesJSONStr) + `,
                                     "match_status": "java.lang.processbuilder"
                                 }
                             ]
                         }
                     ]`)

		// Create a fake request record with the interesting parts for this test
		// only
		record := api.RequestRecord{
			Request: api.RequestRecord_Request{
				Parameters: api.RequestRecord_Request_Parameters{
					Params: params,
				},
			},
			Observed: api.RequestRecord_Observed{
				Attacks: []*api.RequestRecord_Observed_Attack{
					{Info: api.WAFAttackInfo{WAFData: wafData}},
				},
			},
		}

		// Create a scrubber of the PII values
		keyRE := regexp.MustCompile(`(?i)(passw(((or)?d))|(phrase))|(secret)|(authorization)|(api_?key)|((access_?)?token)`)
		valueRE := regexp.MustCompile(`(?:\d[ -]*?){13,16}`)
		redactionString := "Redacted by Test"
		scrubber := sqsanitize.NewScrubber(keyRE, valueRE, redactionString)

		// Scrub the request record
		info := sqsanitize.Info{}
		scrubbed, err := record.Scrub(scrubber, info)

		// It shouldn't fail and it should have scrubbed
		require.NoError(t, err)
		require.True(t, scrubbed)


		scrubbedWAFData := string(record.Observed.Attacks[0].Info.(api.WAFAttackInfo).WAFData)

		// Check that the count of redactions in the WAF info string is correct:
		// one per PII value.
		require.Equal(t, len(pii), strings.Count(scrubbedWAFData, redactionString))

		// For each PII value
		for k, v := range pii {
			// Check that the returned scrubbed values contain the PII value
			require.Contains(t, info, v)
			// Check that the WAF data has been scrubbed
			require.NotContains(t, scrubbedWAFData, v)
			// Check that the request parameter has been scrubbed
			require.Equal(t, redactionString, record.Request.Parameters.Params[k][0].(string))
		}
	})
}
