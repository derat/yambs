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
	"regexp"
	"strconv"
	"strings"
	"time"
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
)

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

// fieldNameError describes a problem with a field name.
type fieldNameError struct{ msg string }

func (err *fieldNameError) Error() string { return err.msg }

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
