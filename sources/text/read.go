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
	"strings"

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
	// Unlike the other formats, this is used to specify a single edit.
	KeyVal Format = "keyval"
	// TSV corresponds to lines of tab-separated values. No escaping is supported.
	TSV Format = "tsv"
)

// config is passed to Options to configure ReadEdit's behavior.
type config struct {
	maxEdits  int
	maxFields int
}

// ReadEdits reads one or more edits of the specified type from r in the specified format.
// rawFields is a comma-separated list specifying the field associated with each column.
// rawSets contains "field=value" directives describing values to set for all edits.
func ReadEdits(ctx context.Context, r io.Reader, format Format, typ seed.Type,
	rawFields string, rawSetCmds []string, db *db.DB, opts ...Option) ([]seed.Edit, error) {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}

	setPairs, err := readSetCommands(rawSetCmds)
	if err != nil {
		return nil, err
	}
	rr, fields, err := newRowReader(r, format, rawFields)
	if err != nil {
		return nil, err
	}
	if cfg.maxFields > 0 && len(setPairs)+len(fields) > cfg.maxFields {
		return nil, errors.New("too many fields")
	}

	var edits []seed.Edit
	for {
		cols, err := rr.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if cfg.maxEdits > 0 && len(edits) == cfg.maxEdits {
			return nil, errors.New("too many edits")
		}

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

		if err := edit.Finish(ctx, db); err != nil {
			return nil, err
		}

		edits = append(edits, edit)
	}
	return edits, nil
}

// Option can be passed to ReadEdits to configure its behavior.
type Option func(*config)

// MaxEdits returns an Option that limits the maximum number of edits to be read.
// If more edits are supplied, an error will be returned.
func MaxEdits(max int) Option { return func(c *config) { c.maxEdits = max } }

// MaxEdits returns an Option that limits the maximum number of fields that can be set.
// "field=value" directives are included in the count.
func MaxFields(max int) Option { return func(c *config) { c.maxFields = max } }

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

// rowReader is used by ReadEdits to read entity data row-by-row.
type rowReader interface {
	Read() ([]string, error)
}

// tsvReader is a rowReader implementation for TSV input.
type tsvReader struct {
	sc      *bufio.Scanner
	nfields int
}

func (tr *tsvReader) Read() ([]string, error) {
	if !tr.sc.Scan() {
		return nil, io.EOF
	}
	cols := strings.Split(tr.sc.Text(), "\t")
	if len(cols) != tr.nfields {
		return nil, fmt.Errorf("line %q has %v field(s); want %v",
			tr.sc.Text(), len(cols), tr.nfields)
	}
	return cols, nil
}

// singleRowReader is a rowReader implementation that just returns a single row (or error)
// and then returns io.EOF. It is used to return KeyVal data (which is read by newRowReader).
type singleRowReader struct {
	row []string
	err error
}

func (sr *singleRowReader) Read() ([]string, error) {
	defer func() {
		sr.row = nil
		sr.err = io.EOF
	}()
	return sr.row, sr.err
}

// newRowReader returns a rowReader for reading from r in format and
// a list of field names corresponding to the columns in each row.
func newRowReader(r io.Reader, format Format, rawFields string) (
	rr rowReader, fields []string, err error) {
	switch format {
	case CSV:
		fields = strings.Split(rawFields, ",")
		cr := csv.NewReader(r)
		cr.FieldsPerRecord = len(fields)
		return cr, fields, nil
	case KeyVal:
		// Transform the input into a single row and use it to synthesize the field list.
		// Note that rawFields isn't used in this case, since the field names are provided
		// in the input.
		var sr singleRowReader
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			parts := strings.SplitN(sc.Text(), "=", 2)
			if len(parts) != 2 {
				sr.err = fmt.Errorf(`line %d (%q) not "field=value" format`, len(fields)+1, sc.Text())
				break
			}
			fields = append(fields, parts[0])
			sr.row = append(sr.row, parts[1])
		}
		return &sr, fields, sc.Err()
	case TSV:
		fields = strings.Split(rawFields, ",")
		return &tsvReader{bufio.NewScanner(r), len(fields)}, fields, nil
	default:
		return nil, nil, fmt.Errorf("unknown format %q", format)
	}
}
