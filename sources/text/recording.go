// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
)

const maxArtistCredits = 100

// ReadRecordings reads lines describing recordings from r in the specified format.
// rawFields is a comma-separated list specifying the field associated with each column.
// rawSets contains "field=value" directives describing values to set for all recordings.
func ReadRecordings(r io.Reader, format Format, rawFields string, rawSetCmds []string) ([]seed.Recording, error) {
	setPairs, err := readSetCommands(rawSetCmds)
	if err != nil {
		return nil, err
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

// RecordingFields returns the names of fields that can be passed to ReadRecordings.
func RecordingFields() []string {
	v := make([]string, 0, len(recordingFields))
	for n := range recordingFields {
		v = append(v, n)
	}
	sort.Strings(v)
	return v
}
