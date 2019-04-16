package api

import (
	"encoding/json"
	fmt "fmt"
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

func (s *Struct) Reset() {
	s.Value = nil
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
	switch actual := e.GetArgs().(type) {
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
		args = append(args, track.GetEvent())

		if options := track.GetOptions(); options != nil {
			args = append(args, options)
		}
	}
	return args.MarshalJSON()
}

func (identify *RequestRecord_Observed_SDKEvent_Args_Identify) MarshalJSON() ([]byte, error) {
	var args ListValue
	if identify != nil {
		args = append(args, identify.GetUserIdentifiers())
	}
	return args.MarshalJSON()
}
