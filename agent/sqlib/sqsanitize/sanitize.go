// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsanitize

import (
	"reflect"
	"regexp"

	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqsafe"
)

// Scrubber scrubs values according to the key and value regular expressions
// given to `NewScrubber()`. Field names and map keys of type string will be
// checked against the regular expression for keys, while string values will be
// checked against the regular expression for values. Scrubbed values are
// replaced by the given redaction string. Cf. `NewScrubber()` for details.
type Scrubber struct {
	// keyRegexp is the regular expression matching keys that need to be
	// scrubbed. Their values are completely replaced by `redactedValueMask`.
	keyRegexp regex
	// valueRegexp is the regular expression matching values that need to be
	// scrubbed. Only the matching part is replaced by `redactedValueMask`
	valueRegexp regex
	// redactValueMask is the string replacing a scrubbed value.
	redactedValueMask string
	// scrubEveryString is true when every string of a value must be scrubbed.
	// It is used when a key matches keyRegexp.
	scrubEveryString bool
}

// NewScrubber returns a new scrubber configured to redact values matching the
// given regular expressions.
//   - A string value matching the `valueRegexp` is replaced by
//     `redactedValueMask`.
//   - A map key of type string or struct field name matching `keyRegexp` is
//     scrubbed regardless of `valueRegexp` - any string in the associated
//     value is replaced by `redactedValue`.
// An error can be returned if the regular expressions cannot be compiled.
func NewScrubber(keyRegexp, valueRegexp, redactedValueMask string) (*Scrubber, error) {
	keyRE, err := compile(keyRegexp)
	if err != nil {
		return nil, err
	}

	valueRE, err := compile(valueRegexp)
	if err != nil {
		return nil, err
	}

	return &Scrubber{
		keyRegexp:         keyRE,
		valueRegexp:       valueRE,
		redactedValueMask: redactedValueMask,
	}, nil
}

// Scrub a given value. Since it is based on `reflect`, unexported struct
// fields are ignored. This function cannot panic and an error is returned if
// an unexpected panic occurs.
func (s *Scrubber) Scrub(v interface{}) (scrubbed bool, err error) {
	err = sqsafe.Call(func() error {
		scrubbed = s.scrubValue(reflect.ValueOf(v))
		return nil
	})
	return
}

func (s *Scrubber) scrubValue(v reflect.Value) bool {
walk:
	switch v.Kind() {
	case reflect.Ptr:
		v = v.Elem()
		goto walk

	case reflect.Array:
		fallthrough
	case reflect.Slice:
		return s.scrubSlice(v)

	case reflect.Map:
		return s.scrubMap(v)

	case reflect.Struct:
		return s.scrubStruct(v)

	case reflect.String:
		return s.scrubString(v)
	}
	return false
}

func (s *Scrubber) scrubString(v reflect.Value) bool {
	// No need to scrub empty strings
	if v.Len() == 0 {
		return false
	}

	// If scrubEveryString is true, scrub everything regardless of the value
	// regexp
	if s.scrubEveryString {
		v.SetString(s.redactedValueMask)
		return true
	}

	// Scrub the substrings matching the value regular expression
	str := v.String()
	redacted := s.valueRegexp.ReplaceAllString(str, s.redactedValueMask)
	if str == redacted {
		return false
	}
	v.SetString(redacted)
	return true
}

func (s *Scrubber) scrubSlice(v reflect.Value) (scrubbed bool) {
	l := v.Len()
	hasInterfaceElementType := v.Type().Elem().Kind() == reflect.Interface
	for i := 0; i < l; i++ {
		ix := v.Index(i)
		if !hasInterfaceElementType {
			// Not an interface value, scrub the current element.
			if scrubbedElement := s.scrubValue(ix); scrubbedElement {
				scrubbed = true
			}
		} else {
			// Interface value, scrub its element and set the current element to it.
			if newVal, scrubbedElement := s.scrubInterface(ix); scrubbedElement {
				ix.Set(newVal)
				scrubbed = true
			}
		}
	}
	return scrubbed
}

func (s *Scrubber) scrubMap(v reflect.Value) (scrubbed bool) {
	vt := v.Type().Elem()
	hasInterfaceValueType := vt.Kind() == reflect.Interface
	hasStringKeyType := v.Type().Key().Kind() == reflect.String
	for iter := v.MapRange(); iter.Next(); {
		scrubber := s

		// Check if the map key is a string matching the key regular expression.
		// When it does, every string sub-value must be scrubbed regardless of the
		// value regular expression.
		key := iter.Key()
		if hasStringKeyType && !s.scrubEveryString && s.keyRegexp.MatchString(key.String()) {
			scrubber = new(Scrubber)
			*scrubber = *s
			scrubber.scrubEveryString = true
		}

		// Map entries cannot be set. We therefore create a new value in order
		// that can be set by the scrubber.
		val := iter.Value()
		valT := vt

		// When the current value is an interface value, we scrub its underlying
		// value.
		if hasInterfaceValueType {
			val = val.Elem()
			valT = val.Type()
		}

		// Create a new pointer value to the current map value that can be set by
		// the scrubber.
		newVal := reflect.New(valT).Elem()
		newVal.Set(val)
		// Scrub it
		if scrubbedElement := scrubber.scrubValue(newVal); scrubbedElement {
			// Set it
			v.SetMapIndex(key, newVal)
			scrubbed = true
		}
	}
	return scrubbed
}

func (s *Scrubber) scrubStruct(v reflect.Value) (scrubbed bool) {
	l := v.NumField()
	vt := v.Type()
	for i := 0; i < l; i++ {
		ft := vt.Field(i)
		if isUnexportedField(&ft) {
			// Ignore unexported fields
			continue
		}

		scrubber := s
		if !s.scrubEveryString && s.keyRegexp.MatchString(ft.Name) {
			scrubber = new(Scrubber)
			*scrubber = *s
			scrubber.scrubEveryString = true
		}

		f := v.Field(i)
		if f.Kind() == reflect.Interface {
			// Interface value, scrub its element and set it.
			newVal, scrubbed := s.scrubInterface(f)
			if scrubbed {
				f.Set(newVal)
			}
		} else {
			// Not and interface value, scrub the field.
			if scrubbedElement := scrubber.scrubValue(f); scrubbedElement {
				scrubbed = true
			}
		}
	}
	return scrubbed
}

// scrubInterface has a different signature than other `scrubT()` functions
// because interface values cannot be modified. Hence the creation of a new
// value of the underlying interface type, that is set with the given value and
// scrubbed. The scrubbed new value is returned and can be set to the original
// value (map entry, array index, etc.).
func (s *Scrubber) scrubInterface(v reflect.Value) (reflect.Value, bool) {
	// The current element is an interface value which cannot be set, we
	// therefore need to create a new value that can be set by the scrubber.
	if v.IsZero() {
		return reflect.Value{}, false
	}
	val := v.Elem()
	newVal := reflect.New(val.Type()).Elem()
	newVal.Set(val)
	scrubbed := s.scrubValue(newVal)
	return newVal, scrubbed
}

// isUnexportedField returns true is a field is unexported.
func isUnexportedField(f *reflect.StructField) bool {
	// Based on `reflect` documentation, PkgPath is an empty string when exported.
	return f.PkgPath != ""
}

// regex is a helper structure wrapping a regexp and handling when the regexp is
// disabled by matching nothing.
type regex struct {
	re *regexp.Regexp
}

func compile(r string) (regex, error) {
	if r == "" {
		return regex{}, nil
	}

	re, err := regexp.Compile(r)
	if err != nil {
		return regex{}, sqerrors.Wrapf(err, "could not compile regular expression `%q`", r)
	}
	return regex{re: re}, nil
}

func (r regex) MatchString(s string) bool {
	if r.re == nil {
		return false
	}
	return r.re.MatchString(s)
}

func (r regex) ReplaceAllString(src, repl string) string {
	if r.re == nil {
		return src
	}
	return r.re.ReplaceAllString(src, repl)
}
