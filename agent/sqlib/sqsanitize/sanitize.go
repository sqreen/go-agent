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

// CustomScrubber is the interface a type can implement in order to provide
// a custom scrubbing method. For example, it could be used to scrub unexported
// struct fields.
// The method is given the pointer to the scrubber so that it can continue
// scrubbing its underlying values when required. It is the method's
// responsibility to update the scrubbing information.
type CustomScrubber interface {
	Scrub(*Scrubber, Info) (bool, error)
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

// RedactedValueMask returns the configured redactedValueMask
func (s *Scrubber) RedactedValueMask() string {
	return s.redactedValueMask
}

// Scrub a given value. Since it is based on `reflect`, unexported struct
// fields are ignored. When `info` is not nil, it is used to store every
// scrubbed value and provide them to the caller.
// This function cannot panic and an error is returned if an unexpected panic
// occurs.
func (s *Scrubber) Scrub(v interface{}, info Info) (scrubbed bool, err error) {
	if v == nil {
		return false, nil
	}
	err = sqsafe.Call(func() error {
		scrubbed = s.scrubValue(reflect.ValueOf(v), info)
		return nil
	})
	return
}

func (s *Scrubber) scrubValue(v reflect.Value, info Info) bool {
	if ok, scrubbed, err := s.tryCustomScrubberInterface(v, info); ok {
		if err != nil {
			// TODO: change signature in order to bubble it up
			panic(err)
		}
		return scrubbed
	}

walk:
	switch v.Kind() {
	case reflect.Interface:
		_, scrubbed := s.scrubInterface(v, info)
		return scrubbed

	case reflect.Ptr:
		v = v.Elem()
		goto walk

	case reflect.Array:
		fallthrough
	case reflect.Slice:
		return s.scrubSlice(v, info)

	case reflect.Map:
		return s.scrubMap(v, info)

	case reflect.Struct:
		return s.scrubStruct(v, info)

	case reflect.String:
		return s.scrubString(v, info)
	}
	return false
}

// tryCustomScrubberInterface calls tryCustomScrubber on v or retries on its
// address when possible.
func (s *Scrubber) tryCustomScrubberInterface(v reflect.Value, info Info) (ok, scrubbed bool, err error) {
	// Try v
	if v.CanInterface() {
		ok, scrubbed, err = s.tryCustomScrubber(v, info)
		if ok || err != nil {
			return
		}
	}
	// Retry on the address if v is addressable since it could also implement the
	// CustomScrubber interface.
	if v.CanAddr() {
		if v := v.Addr(); v.CanInterface() {
			ok, scrubbed, err = s.tryCustomScrubber(v, info)
		}
	}
	return
}

// tryCustomScrubber call the CustomScrubber method when v implements it.
func (s *Scrubber) tryCustomScrubber(v reflect.Value, info Info) (ok, scrubbed bool, err error) {
	var custom CustomScrubber
	custom, ok = v.Interface().(CustomScrubber)
	if !ok {
		return
	}
	scrubbed, err = custom.Scrub(s, info)
	return
}

func (s *Scrubber) scrubString(v reflect.Value, info Info) (scrubbed bool) {
	// No need to scrub empty strings
	str := v.String()
	if len(str) == 0 {
		return false
	}

	// If scrubEveryString is true, scrub everything regardless of the value
	// regexp
	if s.scrubEveryString {
		v.SetString(s.redactedValueMask)
		scrubbed = true
	} else {
		// Scrub the substrings matching the value regular expression
		redacted := s.valueRegexp.ReplaceAllString(str, s.redactedValueMask)
		if str == redacted {
			return false
		}
		v.SetString(redacted)
		scrubbed = true
	}

	if scrubbed {
		info.Add(str)
	}

	return scrubbed
}

func (s *Scrubber) scrubSlice(v reflect.Value, info Info) (scrubbed bool) {
	l := v.Len()
	hasInterfaceElementType := v.Type().Elem().Kind() == reflect.Interface
	for i := 0; i < l; i++ {
		ix := v.Index(i)
		if !hasInterfaceElementType {
			// Not an interface value, scrub the current element.
			if scrubbedElement := s.scrubValue(ix, info); scrubbedElement {
				scrubbed = true
			}
		} else {
			// Interface value, scrub its element and set the current element to it.
			if newVal, scrubbedElement := s.scrubInterface(ix, info); scrubbedElement {
				ix.Set(newVal)
				scrubbed = true
			}
		}
	}
	return scrubbed
}

func (s *Scrubber) scrubMap(v reflect.Value, info Info) (scrubbed bool) {
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
		if scrubbedElement := scrubber.scrubValue(newVal, info); scrubbedElement {
			// Set it
			v.SetMapIndex(key, newVal)
			scrubbed = true
		}
	}
	return scrubbed
}

func (s *Scrubber) scrubStruct(v reflect.Value, info Info) (scrubbed bool) {
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
			newVal, scrubbedElement := s.scrubInterface(f, info)
			if scrubbedElement {
				f.Set(newVal)
				scrubbed = true
			}
		} else {
			// Not and interface value, scrub the field.
			if scrubbedElement := scrubber.scrubValue(f, info); scrubbedElement {
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
func (s *Scrubber) scrubInterface(v reflect.Value, info Info) (reflect.Value, bool) {
	if v.IsZero() {
		return reflect.Value{}, false
	}
	// The current element is an interface value which cannot be set, we
	// therefore need to create a new value that can be set by the scrubber.
	val := v.Elem()
	newVal := reflect.New(val.Type()).Elem()
	newVal.Set(val)
	// Scrub it.
	scrubbed := s.scrubValue(newVal, info)
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

type Info map[string]struct{}

func (i Info) Add(value string) {
	if i == nil {
		return
	}
	i[value] = struct{}{}
}

func (i Info) Contains(value string) bool {
	if len(i) == 0 {
		return false
	}
	_, exists := i[value]
	return exists
}

func (i Info) Append(info Info) {
	if i == nil || len(info) == 0 {
		return
	}
	for v := range info {
		i.Add(v)
	}
}
