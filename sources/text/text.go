// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package text parses entities from textual input (e.g. CSV or TSV).
package text

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
)

// Format represents a textual format for supplying seed data.
type Format string

const (
	// CSV corresponds to lines of comma-separated values as described in RFC 4180.
	// See https://pkg.go.dev/encoding/csv.
	CSV Format = "csv"
	// KeyVal corresponds to an individual "field=value" pair on each line.
	// Unlike the other formats, this is used to specify a single entity.
	KeyVal Format = "keyval"
	// TSV corresponds to lines of tab-separated values. No escaping is supported.
	TSV Format = "tsv"

	maxArtistCredits = 100
)

// ReadRecordings reads lines describing recordings from r in the specified format.
// rawFields is a comma-separated list specifying the field associated with each column.
// rawSets contains "field=value" directives describing values to set for all recordings.
func ReadRecordings(r io.Reader, format Format, rawFields string, rawSetCmds []string) ([]seed.Recording, error) {
	setPairs := make([][2]string, len(rawSetCmds))
	for i, cmd := range rawSetCmds {
		parts := strings.SplitN(cmd, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf(`malformed set command %q (want "field=val")`, cmd)
		}
		setPairs[i] = [2]string{parts[0], parts[1]}
	}

	rows, fields, err := readInput(r, format, rawFields)
	if err != nil {
		return nil, err
	}

	recs := make([]seed.Recording, 0, len(rows))
	for _, cols := range rows {
		var rec seed.Recording
		for _, pair := range setPairs {
			if err := setRecordingField(&rec, pair[0], pair[1]); err != nil {
				return nil, fmt.Errorf("failed setting %q: %v", pair[0]+"="+pair[1], err)
			}
		}
		for j, field := range fields {
			val := cols[j]
			err := setRecordingField(&rec, field, val)
			if _, ok := err.(*fieldNameError); ok {
				return nil, err
			} else if err != nil {
				return nil, fmt.Errorf("bad %v %q: %v", field, val, err)
			}
		}
		recs = append(recs, rec)
	}
	return recs, nil
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
		// Transform the input into a single row, and use it to synthesize the field list.
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

// fieldNameError describes a problem with a field name.
type fieldNameError struct{ msg string }

func (err *fieldNameError) Error() string { return err.msg }

// setRecordingField sets rec's named field to the supplied value.
func setRecordingField(rec *seed.Recording, field, val string) error {
	if field == "" {
		return &fieldNameError{"missing field name"}
	}
	fn, ok := recordingFields[field]
	if !ok {
		for k, v := range recordingFields {
			if strings.HasPrefix(k, field) || globMatches(k, field) {
				if fn != nil {
					return &fieldNameError{fmt.Sprintf("multiple fields matched by %q", field)}
				}
				fn = v
			}
		}
	}
	if fn == nil {
		return &fieldNameError{fmt.Sprintf("unknown field %q", field)}
	}
	return fn(rec, field, val)
}

// recordingFields maps from user-supplied field names to functions that set the appropriate
// field in a seed.Recording.
var recordingFields = map[string](func(r *seed.Recording, k, v string) error){
	"artist":         func(r *seed.Recording, k, v string) error { return setString(&r.Artist, v) },
	"disambiguation": func(r *seed.Recording, k, v string) error { return setString(&r.Disambiguation, v) },
	"edit_note":      func(r *seed.Recording, k, v string) error { return setString(&r.EditNote, v) },
	"isrcs":          func(r *seed.Recording, k, v string) error { return setStringSlice(&r.ISRCs, v, ",") },
	"length":         func(r *seed.Recording, k, v string) error { return setDuration(&r.Length, v) },
	"mbid":           func(r *seed.Recording, k, v string) error { return setString(&r.MBID, v) },
	"title":          func(r *seed.Recording, k, v string) error { return setString(&r.Title, v) },
	"video":          func(r *seed.Recording, k, v string) error { return setBool(&r.Video, v) },

	"artist*_mbid": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		// The /recording/create form only seems to accept database IDs, not MBIDs.
		// TODO: Pass a context in, maybe.
		ac.ID, err = db.GetArtistID(context.Background(), v)
		return err
	},
	"artist*_name": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		return setString(&ac.Name, v)
	},
	"artist*_credited": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		return setString(&ac.NameAsCredited, v)
	},
	"artist*_join_phrase": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		return setString(&ac.JoinPhrase, v)
	},
}

// RecordingFields returns the names of fields that can be passed to ReadRecordings.
func RecordingFields() []string {
	v := make([]string, 0, len(recordingFields))
	for n := range recordingFields {
		v = append(v, n)
	}
	sort.Strings(v)
	return v
}

// globMatches returns true if glob contains '*' and matches name per filepath.Match.
func globMatches(glob, name string) bool {
	if !strings.ContainsRune(glob, '*') {
		return false
	}
	matched, err := filepath.Match(glob, name)
	return err == nil && matched
}

var artistFieldRegexp = regexp.MustCompile(`^artist(\d*)_.*`)

// getArtistCredit extracts a zero-based index from field (e.g. "artist3_name") and returns
// a pointer to the corresponding item from rec.ArtistCredits, creating empty items if needed.
// The index 0 is inferred for e.g. "artist_name".
func getArtistCredit(rec *seed.Recording, field string) (*seed.ArtistCredit, error) {
	matches := artistFieldRegexp.FindStringSubmatch(field)
	if matches == nil {
		return nil, &fieldNameError{`field doesn't start with "artist_" or "artist<num>_"`}
	}
	var idx int
	if matches[1] != "" {
		var err error
		if idx, err = strconv.Atoi(matches[1]); err != nil {
			return nil, err
		} else if idx >= maxArtistCredits {
			// Keep the user from shooting themselves in the foot.
			return nil, &fieldNameError{fmt.Sprintf("invalid artist index %d", idx)}
		}
	}
	if idx >= len(rec.ArtistCredits) {
		old := rec.ArtistCredits
		rec.ArtistCredits = make([]seed.ArtistCredit, idx+1)
		copy(rec.ArtistCredits, old)
	}
	return &rec.ArtistCredits[idx], nil
}

func setString(dst *string, val string) error {
	*dst = val
	return nil
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
