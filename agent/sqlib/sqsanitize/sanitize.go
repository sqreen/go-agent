// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsanitize

import (
	"reflect"
	"regexp"

	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
)

// Scrubber scrubs values according to the key and value regular expressions
// given to `NewScrubber()`. Field names and map keys of type string will be
// checked against the regular expression for keys, while string values will be
// checked against the regular expression for values. A matching value is
// replaced by the given redaction string, while
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
// given regular expressions:
//   - Values matching `valueRegexp` are replaced by `redactedValueMask` (only
//     the matching part).
//   - Map values with keys matching `keyRegexp` are replaced by
//     `redactedValueMask`.
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

func (s *Scrubber) Scrub(v interface{}) error {
	return s.scrubValue(reflect.ValueOf(v))
}

func (s *Scrubber) scrubValue(v reflect.Value) error {
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
	return nil
}

func (s *Scrubber) scrubString(v reflect.Value) error {
	// No need to scrub empty strings
	if v.Len() == 0 {
		return nil
	}

	// If scrubEveryString is true, scrub everything regardless of the value
	// regexp
	if s.scrubEveryString {
		v.SetString(s.redactedValueMask)
		return nil
	}

	// Scrub the substrings matching the value regular expression
	str := v.String()
	redacted := s.valueRegexp.ReplaceAllString(str, s.redactedValueMask)
	if str != redacted {
		v.SetString(redacted)
	}
	return nil
}

func (s *Scrubber) scrubSlice(v reflect.Value) error {
	l := v.Len()
	for i := 0; i < l; i++ {
		e := v.Index(i)
		if !e.CanSet() {
			//Scrub &v[i]
			e = e.Addr()
		}
		if err := s.scrubValue(e); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scrubber) scrubMap(v reflect.Value) error {
	vt := v.Type().Elem()

	// TODO: hasInterfaceValueType := vt.Kind() == reflect.Interface
	hasStringKeyType := v.Type().Key().Kind() == reflect.String
	for iter := v.MapRange(); iter.Next(); {
		scrubber := s

		// Check if the map key is string matching the key regular expression.
		// When it does, every string sub-value must be scrubbed.
		key := iter.Key()
		if hasStringKeyType && !s.scrubEveryString && s.keyRegexp.MatchString(key.String()) {
			scrubber = new(Scrubber)
			*scrubber = *s
			scrubber.scrubEveryString = true

			// TODO
			//if hasInterfaceValueType {
			//v.SetMapIndex(key, reflect.ValueOf(s.redactedValueMask))
			//continue
			//}
		}

		val := iter.Value()
		if val.CanSet() {
			return scrubber.scrubValue(val)
		}

		// Map entries are not addressable. Hence, we create a new value in order
		// to scrub it and set the map index to the scrubbed value.
		newVal := reflect.New(vt).Elem()
		newVal.Set(val)
		if err := scrubber.scrubValue(newVal); err != nil {
			return err
		}
		v.SetMapIndex(key, newVal)
	}
	return nil
}

func (s *Scrubber) scrubStruct(v reflect.Value) error {
	l := v.NumField()
	vt := v.Type()
	for i := 0; i < l; i++ {
		ft := vt.Field(i)
		if ft.PkgPath != "" { // TODO: isExportedField() + test go assumption
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
		if !f.CanSet() {
			// Scrub &v.Field
			f = f.Addr().Elem()
		}

		if err := scrubber.scrubValue(f); err != nil {
			return err
		}
	}
	return nil
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
