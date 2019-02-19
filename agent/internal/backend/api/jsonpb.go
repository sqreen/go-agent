package api

import (
	"encoding/json"
	fmt "fmt"

	"github.com/gogo/protobuf/jsonpb"
)

// DefaultJSONPBMarshaler is the default JSON to Protocol-Buffer marsahler. It
// uses the fields original names and includes zero values of empty fields.
var DefaultJSONPBMarshaler = jsonpb.Marshaler{
	OrigName:     true,
	EmitDefaults: true,
}

var RequestRecordVersion = "20171208"

func (h *RequestRecord_Request_Header) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return h.MarshalJSON()
}

func (h *RequestRecord_Request_Header) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("[]"), nil
	}
	return []byte(`["` + h.Key + `", "` + h.Value + `"]`), nil
}

type ListValue []interface{}

func NewPopulatedListValue(_ randyApi) *ListValue {
	return (*ListValue)(&[]interface{}{})
}

func (l ListValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(([]interface{})(l))
}

func (l ListValue) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return l.MarshalJSON()
}

func (l ListValue) ProtoMessage() {}

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

func NewPopulatedStruct(_ randyApi) *Struct {
	return &Struct{}
}

func (s *Struct) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return s.MarshalJSON()
}

func (s *Struct) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Value)
}

func (s *Struct) ProtoMessage() {}

func (s *Struct) String() string {
	return fmt.Sprintf("%+v", s.Value)
}

func (s *Struct) Reset() {
	s.Value = nil
}

func (e *BatchRequest_Event) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return e.MarshalJSON()
}

func (e *BatchRequest_Event) MarshalJSON() ([]byte, error) {
	buf, err := e.Event.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if len(buf) <= 2 {
		return buf, nil
	}
	buf = []byte(`{"event_type":"` + e.EventType + `",` + string(buf[1:]))
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

func (e *RequestRecord_Observed_SDKEvent_Args) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return e.MarshalJSON()
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

func (e *RequestRecord_Observed_SDKEvent_Args_Track) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return e.MarshalJSON()
}

func (identify *RequestRecord_Observed_SDKEvent_Args_Identify) MarshalJSON() ([]byte, error) {
	var args ListValue
	if identify != nil {
		args = append(args, identify.GetUserIdentifiers())
	}
	return args.MarshalJSON()
}

func (e *RequestRecord_Observed_SDKEvent_Args_Identify) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return e.MarshalJSON()
}
