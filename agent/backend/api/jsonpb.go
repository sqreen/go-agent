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
