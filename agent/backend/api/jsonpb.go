package api

import (
	"github.com/gogo/protobuf/jsonpb"
)

// DefaultJSONPBMarshaler is the default JSON to Protocol-Buffer marsahler. It
// uses the fields original names and includes zero values of empty fields.
var DefaultJSONPBMarshaler = jsonpb.Marshaler{
	OrigName:     true,
	EmitDefaults: true,
}
