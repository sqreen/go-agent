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

func (p *Scrubber) Scrub(v interface{}) error {
	return p.scrubValue(reflect.ValueOf(v))
}

func (p *Scrubber) scrubValue(v reflect.Value) error {
walk:
	switch v.Kind() {
	case reflect.Ptr:
		v = v.Elem()
		goto walk

	case reflect.Array:
		fallthrough
	case reflect.Slice:
		return p.scrubSlice(v)

	case reflect.Map:
		return p.scrubMap(v)

	case reflect.Struct:
		return p.scrubStruct(v)

	case reflect.String:
		return p.scrubString(v)
	}
	return nil
}

func (p *Scrubber) scrubString(v reflect.Value) error {
	str := v.String()
	redacted := p.valueRegexp.ReplaceAllString(str, p.redactedValueMask)
	if str != redacted {
		v.SetString(redacted)
	}
	return nil
}

func (s *Scrubber) scrubSlice(v reflect.Value) error {
	// TODO: ignore when unsupported type or return false to stop the recursion?
	// if _, ok := supported[v.Type().Kind]; !ok { ///// }
	l := v.Len()
	for i := 0; i < l; i++ {
		// Scrub &v[i]
		if err := s.scrubValue(v.Index(i).Addr()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scrubber) scrubMap(v reflect.Value) error {
	vt := v.Type().Elem()
	//hasStringKeyType := v.Type().Key().Kind() == reflect.String
	for iter := v.MapRange(); iter.Next(); {
		key := iter.Key()
		//if hasStringKeyType && s.keyRegexp.MatchString(key.String()) {
		//	// TODO: scrub every string?
		//	v.SetMapIndex(key, reflect.Value{})
		//}

		val := iter.Value()
		if val.CanSet() {
			return s.scrubValue(val)
		}

		newVal := reflect.New(vt).Elem()
		newVal.Set(val)
		if err := s.scrubValue(newVal); err != nil {
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
		if ft.PkgPath != "" {
			// Ignore unexported fields
			continue
		}
		if ft.Type.Kind() == reflect.String && s.keyRegexp.MatchString(ft.Name) { // TODO: recursive key check
			// TODO: scrub every string?
			v.Field(i).Set(reflect.Zero(vt))
		}
		// Scrub &v.Field
		if err := s.scrubValue(v.Field(i).Addr().Elem()); err != nil {
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
