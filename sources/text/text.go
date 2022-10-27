// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package text parses entities from textual input (e.g. CSV or TSV).
package text

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/derat/yambs/seed"
)

// Format represents a textual format for supplying seed data.
type Format string

const (
	// CSV corresponds to lines of comma-separated values as described in RFC 4180.
	// See https://pkg.go.dev/encoding/csv.
	CSV Format = "csv"
	// KeyVal corresponds to an individual "field=value" pair on each line.
	// Unlike the other formats, this is used to specify a single edit.
	KeyVal Format = "keyval"
	// TSV corresponds to lines of tab-separated values. No escaping is supported.
	TSV Format = "tsv"

	// TODO: Tune these limits. They just exist to prevent the user from trivially
	// allocating tons of memory by specifying a field like "artist99999999_name".
	maxArtistCredits = 100
	maxReleaseEvents = 100
	maxReleaseLabels = 100
)

var (
	artistIndexRegexp = regexp.MustCompile(`^artist(\d*)_.*`)
	eventIndexRegexp  = regexp.MustCompile(`^event(\d*)_.*`)
	labelIndexRegexp  = regexp.MustCompile(`^label(\d*)_.*`)
)

// ReadEdits reads one or more edits of the specified type from r in the specified format.
// rawFields is a comma-separated list specifying the field associated with each column.
// rawSets contains "field=value" directives describing values to set for all edits.
func ReadEdits(r io.Reader, format Format, typ seed.Type,
	rawFields string, rawSetCmds []string) ([]seed.Edit, error) {
	setPairs, err := readSetCommands(rawSetCmds)
	if err != nil {
		return nil, err
	}
	rows, fields, err := readInput(r, format, rawFields)
	if err != nil {
		return nil, err
	}

	edits := make([]seed.Edit, 0, len(rows))
	for _, cols := range rows {
		var edit seed.Edit
		switch typ {
		case seed.RecordingType:
			edit = &seed.Recording{}
		case seed.ReleaseType:
			edit = &seed.Release{}
		default:
			return nil, fmt.Errorf("unknown edit type %q", typ)
		}

		for _, pair := range setPairs {
			if err := setField(edit, pair[0], pair[1]); err != nil {
				return nil, fmt.Errorf("failed setting %q: %v", pair[0]+"="+pair[1], err)
			}
		}
		for j, field := range fields {
			val := cols[j]
			err := setField(edit, field, val)
			if _, ok := err.(*fieldNameError); ok {
				return nil, err
			} else if err != nil {
				return nil, fmt.Errorf("bad %v %q: %v", field, val, err)
			}
		}
		edits = append(edits, edit)
	}
	return edits, nil
}

// FieldDescriptions returns a map from the names of fields that can be passed
// to ReadEdits for typ to human-readable descriptions.
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

// readSetCommands parses a list of "field=val" commands.
func readSetCommands(cmds []string) ([][2]string, error) {
	pairs := make([][2]string, len(cmds))
	for i, cmd := range cmds {
		parts := strings.SplitN(cmd, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf(`malformed set command %q (want "field=val")`, cmd)
		}
		pairs[i] = [2]string{parts[0], parts[1]}
	}
	return pairs, nil
}

// readInput reads data from r in the specified format and returns one row per
// entity and a list of field names corresponding to the columns in each row.
func readInput(r io.Reader, format Format, rawFields string) (rows [][]string, fields []string, err error) {
	switch format {
	case CSV:
		fields = strings.Split(rawFields, ",")
		cr := csv.NewReader(r)
		cr.FieldsPerRecord = len(fields)
		rows, err = cr.ReadAll()
		return rows, fields, err
	case KeyVal:
		// Transform the input into a single row and use it to synthesize the field list.
		if rawFields != "" {
			return nil, nil, fmt.Errorf("%s format doesn't need field list", KeyVal)
		}
		rows = append(rows, []string{})
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			parts := strings.SplitN(sc.Text(), "=", 2)
			if len(parts) != 2 {
				return nil, nil, fmt.Errorf(`line %d (%q) not "field=value" format`, len(fields)+1, sc.Text())
			}
			fields = append(fields, parts[0])
			rows[0] = append(rows[0], parts[1])
		}
		return rows, fields, sc.Err()
	case TSV:
		fields = strings.Split(rawFields, ",")
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			cols := strings.Split(sc.Text(), "\t")
			if len(cols) != len(fields) {
				return nil, nil, fmt.Errorf("line %d (%q) has %v field(s); want %v",
					len(rows)+1, sc.Text(), len(cols), len(fields))
			}
			rows = append(rows, cols)
		}
		return rows, fields, sc.Err()
	default:
		return nil, nil, fmt.Errorf("unknown format %q", format)
	}
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
		if sv := kv.String(); strings.HasPrefix(sv, field) || globMatches(sv, field) {
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

// globMatches returns true if glob contains '*' and matches name per filepath.Match.
func globMatches(glob, name string) bool {
	if !strings.ContainsRune(glob, '*') {
		return false
	}
	matched, err := filepath.Match(glob, name)
	return err == nil && matched
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

// getFieldIndex extracts an integer index from field via re's first match group
// and returns the corresponding item from items, reallocating if necessary.
// If the match group is empty, index 0 is used.
// If more than max items would be used, an error is returned.
func getIndexedField[T any](items *[]T, field string, re *regexp.Regexp, max int) (*T, error) {
	matches := re.FindStringSubmatch(field)
	if matches == nil {
		return nil, &fieldNameError{fmt.Sprintf(`field not matched by %q`, re)}
	} else if len(matches) != 2 {
		return nil, fmt.Errorf("got %d match(es); want 2", len(matches))
	}
	var idx int
	if matches[1] != "" {
		var err error
		if idx, err = strconv.Atoi(matches[1]); err != nil {
			return nil, err
		} else if idx >= max {
			return nil, &fieldNameError{fmt.Sprintf("invalid index %d", idx)}
		}
	}

	if idx >= len(*items) {
		old := *items
		*items = make([]T, idx+1)
		copy(*items, old)
	}
	return &(*items)[idx], nil
}
