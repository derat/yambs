// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
)

// FieldDescriptions returns a map from the names of fields that can be passed
// to Read for typ to human-readable descriptions.
func ListFields(typ seed.Type) map[string]string {
	m, ok := typeFields[typ]
	if !ok {
		return nil
	}
	fields := make(map[string]string)
	iter := reflect.ValueOf(m).MapRange()
	for iter.Next() {
		fields[iter.Key().String()] = iter.Value().FieldByName("Desc").String()
	}
	return fields
}

// fieldInfo contains information about a field that can be set by the user.
// If struct fields are renamed, the code that accesses them via reflection
// must also be updated.
type fieldInfo struct {
	Desc string
	Fn   interface{}
}

var typeFields = map[seed.Type]map[string]fieldInfo{
	seed.RecordingType: recordingFields,
	seed.ReleaseType:   releaseFields,
}

// setField sets the named field in edit.
func setField(edit seed.Edit, field, val string) error {
	// TODO: Maybe try to rewrite all of this code to use generics at some point.
	// The function casts below will panic if a function has the wrong signature.
	fn, err := findFieldFunc(edit.Type(), field)
	if err != nil {
		return err
	}
	switch tedit := edit.(type) {
	case *seed.Recording:
		return fn.(func(*seed.Recording, string, string) error)(tedit, field, val)
	case *seed.Release:
		return fn.(func(*seed.Release, string, string) error)(tedit, field, val)
	default:
		return fmt.Errorf("unknown edit type %q", edit.Type())
	}
}

// findFieldFunc looks for a function in typeFields corresponding to the supplied field name.
// It returns an error if the field name is invalid or ambiguous.
func findFieldFunc(typ seed.Type, field string) (interface{}, error) {
	m, ok := typeFields[typ]
	if !ok {
		return nil, fmt.Errorf("unknown edit type %q", typ)
	}
	if field == "" {
		return nil, &fieldNameError{"missing field name"}
	}
	mv := reflect.ValueOf(m)
	if v := mv.MapIndex(reflect.ValueOf(field)); v.IsValid() {
		return v.FieldByName("Fn").Interface(), nil
	}
	var fn interface{}
	for _, kv := range mv.MapKeys() {
		if sv := kv.String(); patternMatches(sv, field) {
			if fn != nil {
				return nil, &fieldNameError{fmt.Sprintf("multiple fields matched by %q", field)}
			}
			fn = mv.MapIndex(kv).FieldByName("Fn").Interface()
		}
	}
	if fn == nil {
		return nil, &fieldNameError{fmt.Sprintf("unknown field %q", field)}
	}
	return fn, nil
}

// patternMatches returns true if s is a prefix of pattern or if
// pattern contains asterisks and matches s when asterisks are
// treated as zero or more digits.
func patternMatches(pattern, s string) bool {
	if !strings.ContainsRune(pattern, '*') {
		// TODO: Get rid of prefix matching? It's weird that it works here but not for wildcard patterns.
		return strings.HasPrefix(pattern, s)
	}
	re := regexp.MustCompile("^" + strings.ReplaceAll(pattern, "*", `\d*`) + "$")
	return re.MatchString(s)
}

// fieldNameError describes a problem with a field name.
type fieldNameError struct{ msg string }

func (err *fieldNameError) Error() string { return err.msg }

func setString(dst *string, val string) error {
	*dst = val
	return nil
}

func setInt(dst *int, val string) error {
	var err error
	*dst, err = strconv.Atoi(val)
	return err
}

func setStringSlice(dst *[]string, val, sep string) error {
	*dst = strings.Split(val, sep)
	return nil
}

func setMBID(dst *string, val string) error {
	if !db.IsMBID(val) {
		return errors.New("not MBID")
	}
	return setString(dst, val)
}

func setBool(dst *bool, val string) error {
	switch strings.ToLower(val) {
	case "1", "true", "t":
		*dst = true
	case "0", "false", "f", "":
		*dst = false
	default:
		return errors.New("invalid value")
	}
	return nil
}

func setDuration(dst *time.Duration, val string) error {
	var err error
	*dst, err = parseDuration(val)
	return err
}

// parseDate parses string dates in a variety of formats.
func parseDate(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02",
		"2006-01",
		"2006",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}

var durationRegexp = regexp.MustCompile(`^` +
	`(?:(\d+):)?` + // optional hours followed by ':'
	`(\d+)?` + // optional minutes
	`:(\d\d(?:\.\d+)?)` + // ':' followed by seconds (fractional part optional)
	`$`)

// parseDuration parses a floating-point number of seconds or a variety of string formats
// including ":43", ":43.051", "5:34", or "1:23:45".
func parseDuration(s string) (time.Duration, error) {
	if ms, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(ms * float64(time.Millisecond)), nil
	}

	matches := durationRegexp.FindStringSubmatch(s)
	if matches == nil {
		return 0, errors.New("unknown format")
	}
	sec, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return 0, err
	}
	if matches[2] != "" {
		min, err := strconv.Atoi(matches[2])
		if err != nil {
			return 0, err
		}
		sec += float64(min) * 60
	}
	if matches[1] != "" {
		hours, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		sec += float64(hours) * 3600
	}
	return time.Duration(sec * float64(time.Second)), nil
}

var indexRegexp = regexp.MustCompile(`^(\d+)`)

// getFieldIndex extracts an integer index from field[prefix:] and calls fn
// with the corresponding item from items, reallocating if necessary.
// items should be of type "*[]T" and fn should be "func(*T) error".
// If prefix starts with "^", it is interpreted as a regular expression.
// If the integer is missing, index 0 is used.
func indexedField(items interface{}, field, prefix string, fn interface{}) error {
	// Strip off the part before the index.
	if strings.HasPrefix(prefix, "^") {
		if re, err := regexp.Compile(prefix); err != nil {
			return err
		} else if match := re.FindString(field); match == "" {
			return &fieldNameError{fmt.Sprintf("field %q not matched by %q", field, prefix)}
		} else {
			field = field[len(match):]
		}
	} else {
		if !strings.HasPrefix(field, prefix) {
			return &fieldNameError{fmt.Sprintf("field %q doesn't start with %q", field, prefix)}
		}
		field = field[len(prefix):]
	}

	var idx int
	var err error
	if ms := indexRegexp.FindStringSubmatch(field); ms != nil {
		if idx, err = strconv.Atoi(ms[1]); err != nil {
			return err
		}
	}

	// This horrendous reflection code exists because the App Engine team is seemingly
	// incapable of supporting a Go runtime modern enough to support generics.
	slice := reflect.Indirect(reflect.ValueOf(items))
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("got %s instead of pointer to slice", slice.Type())
	}

	// Forcing indexed fields to be used in-order is maybe a bit restrictive, but
	// it seems like an easy way to avoid blowing up memory if the user provides
	// e.g. "artist999999999_name".
	//
	// TODO: It seems like the server should have limits on things like the number of artist
	// credits, but if those limits exist (I couldn't find them in the code), they don't seem
	// to be enforced in the frontend: when I hack the seeding code to pass an index like 500,
	// the UI (slowly) adds 500 rows for artist credits. :-/
	if idx > slice.Len() {
		return &fieldNameError{fmt.Sprintf("field %q has index %d but %d wasn't previously used", field, idx, idx-1)}
	}
	if idx == slice.Len() {
		item := reflect.Zero(slice.Type().Elem())
		slice.Set(reflect.Append(slice, item))
	}

	args := []reflect.Value{slice.Index(idx).Addr()}
	if fv := reflect.ValueOf(fn); fv.Kind() != reflect.Func {
		return fmt.Errorf("got %s instead of function", fv.Type())
	} else if out := fv.Call(args); len(out) != 1 {
		return fmt.Errorf("function returned %d values instead of 1", len(out))
	} else if out[0].IsNil() {
		return nil
	} else if err, ok := out[0].Interface().(error); !ok {
		return errors.New("function returned non-error type")
	} else {
		return err
	}
}
