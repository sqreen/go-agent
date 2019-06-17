// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package api

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
)

var RequestRecordVersion = "20171208"

func (h *RequestRecord_Request_Header) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("[]"), nil
	}
	return []byte(`["` + h.Key + `", "` + h.Value + `"]`), nil
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

// UnmarshalJSON parses rules data to their actual type. The actual type is
// given by the json structure key `type`.
func (v *RuleDataEntry) UnmarshalJSON(data []byte) error {
	var discriminant struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &discriminant); err != nil {
		return sqerrors.Wrap(err, "json unmarshal")
	}

	var value interface{}
	switch t := discriminant.Type; t {
	case CustomErrorPageType:
		value = &CustomErrorPageRuleDataEntry{}
	default:
		return sqerrors.Wrap(errors.Errorf("unexpected type of rule data value `%s`", t), "json unmarshal")
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
			Type string `json:"type"`
			*CustomErrorPageRuleDataEntry // Inlined
		}{
			Type: CustomErrorPageType, CustomErrorPageRuleDataEntry: actual,
		}
	}
	return json.Marshal(discriminant)
}
