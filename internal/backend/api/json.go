// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

var RequestRecordVersion = "20171208"

func (h *RequestRecord_Request_Header) MarshalJSON() ([]byte, error) {
	var kv []string
	if h != nil {
		kv = []string{h.Key, h.Value}
	}
	return json.Marshal(kv)
}

type ListValue []interface{}

func (l ListValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(([]interface{})(l))
}

func (l ListValue) String() string {
	return fmt.Sprintf("%v", ([]interface{})(l))
}

func (l ListValue) Reset() {
	if len(l) == 0 {
		return
	}
	l = (l)[0:0]
}

type Struct struct {
	Value interface{}
}

// Static assert that the type implements the interface.
var (
	_ json.Marshaler   = Struct{}
	_ json.Marshaler   = &Struct{}
	_ json.Unmarshaler = &Struct{}
)

func (s Struct) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Value)
}

func (s *Struct) UnmarshalJSON(buf []byte) error {
	return json.Unmarshal(buf, &s.Value)
}

func (s Struct) String() string {
	return fmt.Sprintf("%+v", s.Value)
}

func (e *BatchRequest_Event) MarshalJSON() ([]byte, error) {
	buf, err := e.Event.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if len(buf) <= 2 {
		return buf, nil
	}
	eventType, err := json.Marshal(e.EventType)
	if err != nil {
		return nil, err
	}
	buf = []byte(`{"event_type":` + string(eventType) + `,` + string(buf[1:]))
	return buf, nil
}

func (e *RequestRecord_Observed_SDKEvent_Args) MarshalJSON() ([]byte, error) {
	var args json.Marshaler
	switch actual := e.Args.(type) {
	case *RequestRecord_Observed_SDKEvent_Args_Track_:
		args = actual.Track
	case *RequestRecord_Observed_SDKEvent_Args_Identify_:
		args = actual.Identify
	}
	return args.MarshalJSON()
}

func (track *RequestRecord_Observed_SDKEvent_Args_Track) MarshalJSON() ([]byte, error) {
	var args ListValue
	if track != nil {
		args = append(args, track.Event)

		if options := track.Options; options != nil {
			args = append(args, options)
		}
	}
	return args.MarshalJSON()
}

func (identify *RequestRecord_Observed_SDKEvent_Args_Identify) MarshalJSON() ([]byte, error) {
	var args ListValue
	if identify != nil {
		args = append(args, identify.UserIdentifiers)
	}
	return args.MarshalJSON()
}

func (v *RuleDataValues) UnmarshalJSON(data []byte) error {
	var asArray []RuleDataEntry
	if err := json.Unmarshal(data, &asArray); err == nil {
		*v = asArray
		return nil
	}

	var asStruct RuleDataEntry
	if err := json.Unmarshal(data, &asStruct); err != nil {
		return err
	}

	*v = []RuleDataEntry{asStruct}
	return nil
}

// UnmarshalJSON parses rules data to their actual type. The actual type is
// (rarely) given by the json structure key `type`.
func (v *RuleDataEntry) UnmarshalJSON(data []byte) error {
	var discriminant struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &discriminant); err != nil {
		// Some rules come with values not discriminated by a `type` key
		// So we try other types
		// TODO: fix this in the API
		var strArray []string
		err = json.Unmarshal(data, &strArray)
		if err != nil {
			return err
		}
		v.Value = strArray
		return nil
	}

	var value interface{}
	switch t := discriminant.Type; t {
	case CustomErrorPageType:
		value = &CustomErrorPageRuleDataEntry{}
	case RedirectionType:
		value = &RedirectionRuleDataEntry{}
	case BindingAccessorWAFType:
		value = &BindingAccessorWAFRuleDataEntry{}
	case ReactiveWAFType:
		value = &ReactiveWAFRuleDataEntry{}
	case CustomType:
		value = &CustomRuleDataEntry{}
	default:
		return sqerrors.Errorf("unexpected type of rule data value `%s`", t)
	}

	if err := json.Unmarshal(data, value); err != nil {
		return sqerrors.Wrap(err, "json unmarshal")
	}

	v.Value = value
	return nil
}

// MarshalJSON serializes the type to the json representation whose type is
// provided by the key `type`.
func (v *RuleDataEntry) MarshalJSON() ([]byte, error) {
	var discriminant interface{}
	switch actual := v.Value.(type) {
	case *CustomErrorPageRuleDataEntry:
		discriminant = struct {
			Type                          string `json:"type"`
			*CustomErrorPageRuleDataEntry        // Inlined
		}{
			Type: CustomErrorPageType, CustomErrorPageRuleDataEntry: actual,
		}
	}
	return json.Marshal(discriminant)
}

func (c *RuleCallbacks) UnmarshalJSON(data []byte) error {
	var discriminant struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &discriminant); err != nil {
		return err
	}

	var v RuleCallbacksNode
	switch discriminant.Type {
	case "function_waf":
		v = &RuleFunctionWAFCallbacks{}
	default:
		v = &RuleJSCallbacks{}
	}

	if err := json.Unmarshal(data, v); err != nil {
		return err
	}

	c.RuleCallbacksNode = v
	return nil
}

func (c *RuleFunctionWAFCallbacks) UnmarshalJSON(data []byte) error {
	type paramBindingAccessorDescr struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	var payload struct {
		Pre  []paramBindingAccessorDescr `json:"pre"`
		Post []paramBindingAccessorDescr `json:"post"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if len(payload.Pre) > 0 {
		c.Pre = make(map[string]string, len(payload.Pre))
		for _, p := range payload.Pre {
			c.Pre[p.Name] = p.Value
		}
	}

	if len(payload.Post) > 0 {
		c.Post = make(map[string]string, len(payload.Post))
		for _, p := range payload.Post {
			c.Post[p.Name] = p.Value
		}
	}

	return nil
}

func (r *Rule) UnmarshalJSON(data []byte) error {
	type rule Rule
	if err := json.Unmarshal(data, (*rule)(r)); err != nil {
		return err
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}

	signature := &r.Signature.ECDSASignature
	kv := make(map[string]interface{}, len(signature.Keys))
	for _, k := range signature.Keys {
		rawValue, exists := keys[k]
		if !exists {
			continue
		}
		var v interface{}
		if err := json.Unmarshal(rawValue, &v); err != nil {
			return err
		}
		kv[k] = v
	}
	message, err := LexicographicalOrderJSONMarshal(kv)
	if err != nil {
		return err
	}
	signature.Message = message
	return nil
}

func unescapedJSONMarshal(v interface{}) ([]byte, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	buf := b.Bytes()
	if l := len(buf); l >= 1 && buf[l-1] == '\n' {
		// Cf. json.Encoder doc: it adds a trailing \n :,(
		buf = buf[:l-1]
	}
	return buf, nil
}

func LexicographicalOrderJSONMarshal(v interface{}) ([]byte, error) {
	switch actual := v.(type) {
	default:
		return unescapedJSONMarshal(v)
	case map[string]interface{}:
		if len(actual) == 0 {
			return []byte(`{}`), nil
		}
		// Get the list of entries
		entries := make([]string, 0, len(actual))
		for k := range actual {
			entries = append(entries, k)
		}
		// Sort the list of keys
		sort.Strings(entries)
		for i, k := range entries {
			v, err := LexicographicalOrderJSONMarshal(actual[k])
			if err != nil {
				return nil, err
			}
			jsonKey, err := unescapedJSONMarshal(k)
			if err != nil {
				return nil, sqerrors.Wrap(err, "map string key marshaling")
			}
			var s strings.Builder
			s.Write(jsonKey)
			s.WriteByte(':')
			s.Write(v)
			entries[i] = s.String()
		}

		var b bytes.Buffer
		b.WriteRune('{')
		b.WriteString(strings.Join(entries, ","))
		b.WriteRune('}')
		return b.Bytes(), nil
	}
}
