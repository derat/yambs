// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package text parses entities from textual input like CSV or TSV files.
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

// config is passed to Options to configure Read's behavior.
type config struct {
	maxEdits  int
	maxFields int
}

// Read reads one or more edits of the specified type from r in the specified format.
// fields specifies the field associated with each column (unused for the KeyVal format).
// Multiple fields can be associated with a single column by separating their names with
// slashes, and empty field names indicate that the column should be ignored.
// rawSets contains "field=value" directives describing values to set for all edits.
func Read(ctx context.Context, r io.Reader, format Format, typ seed.Entity,
	fields []string, rawSetCmds []string, db *db.DB, opts ...Option) ([]seed.Edit, error) {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}

	setPairs, err := ParseSetCommands(rawSetCmds, typ)
	if err != nil {
		return nil, err
	}
	rr, fields, err := newRowReader(r, format, fields)
	if err != nil {
		return nil, err
	}

	// Count fields, including slash-separated names.
	// Empty fields are included but that's arguably safer.
	var nfields int
	for _, f := range fields {
		nfields += len(strings.Split(f, "/"))
	}
	if nfields == 0 {
		return nil, errors.New("no fields specified")
	} else if cfg.maxFields > 0 && len(setPairs)+nfields > cfg.maxFields {
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

		edit := newEdit(typ)
		if edit == nil {
			return nil, fmt.Errorf("unknown edit type %q", typ)
		}

		for _, pair := range setPairs {
			if err := SetField(edit, pair[0], pair[1]); err != nil {
				return nil, fmt.Errorf("failed setting %q: %v", pair[0]+"="+pair[1], err)
			}
		}
		for j, field := range fields {
			// Skip setting anything if the field name is empty.
			// This is handy if the input file contains additional columns that
			// the user doesn't want to use.
			if field == "" {
				continue
			}
			for _, fd := range strings.Split(field, "/") {
				val := cols[j]
				err := SetField(edit, fd, val)
				if _, ok := err.(*fieldNameError); ok {
					return nil, fmt.Errorf("%q: %v", fd, err)
				} else if err != nil {
					return nil, fmt.Errorf("bad %v %q: %v", fd, val, err)
				}
			}
		}

		if err := edit.Finish(ctx, db); err != nil {
			return nil, err
		}

		edits = append(edits, edit)
	}
	if len(edits) == 0 {
		return nil, errors.New("empty input")
	}
	return edits, nil
}

// Option can be passed to Read to configure its behavior.
type Option func(*config)

// MaxEdits returns an Option that limits the maximum number of edits to be read.
// If more edits are supplied, an error will be returned.
func MaxEdits(max int) Option { return func(c *config) { c.maxEdits = max } }

// MaxEdits returns an Option that limits the maximum number of fields that can be set.
// "field=value" directives are included in the count.
func MaxFields(max int) Option { return func(c *config) { c.maxFields = max } }

// rowReader is used by Read to read entity data row-by-row.
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

// newRowReader returns a rowReader for reading the named fields from r in format.
// fieldsOut should be used afterward (fields are specified via r for the KeyVal format).
func newRowReader(r io.Reader, format Format, fields []string) (
	rr rowReader, fieldsOut []string, err error) {
	switch format {
	case CSV:
		cr := csv.NewReader(r)
		cr.FieldsPerRecord = len(fields)
		return cr, fields, nil
	case KeyVal:
		// Transform the input into a single row and use it to synthesize the field list.
		// Note that fields isn't used in this case, since the field names are provided
		// in the input.
		var sr singleRowReader
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			parts := strings.SplitN(sc.Text(), "=", 2)
			if len(parts) != 2 {
				sr.err = fmt.Errorf(`line %d (%q) not "field=value" format`, len(fields)+1, sc.Text())
				break
			}
			fieldsOut = append(fieldsOut, parts[0])
			sr.row = append(sr.row, parts[1])
		}
		return &sr, fieldsOut, sc.Err()
	case TSV:
		return &tsvReader{bufio.NewScanner(r), len(fields)}, fields, nil
	default:
		return nil, nil, fmt.Errorf("unknown format %q", format)
	}
}

// newEdit returns a new seed.Edit for the specified entity type.
// nil is returned if the type is unsupported.
func newEdit(typ seed.Entity) seed.Edit {
	switch typ {
	case seed.LabelEntity:
		return &seed.Label{}
	case seed.RecordingEntity:
		return &seed.Recording{}
	case seed.ReleaseEntity:
		return &seed.Release{}
	case seed.WorkEntity:
		return &seed.Work{}
	default:
		return nil
	}
}
