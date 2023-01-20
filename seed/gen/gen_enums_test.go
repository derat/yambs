// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"reflect"
	"testing"
)

func TestWrap(t *testing.T) {
	for _, tc := range []struct {
		orig string
		max  int
		want []string
	}{
		{"abcdef", 5, []string{"abcdef"}},
		{"abcdef", 6, []string{"abcdef"}},
		{"abcdef", 7, []string{"abcdef"}},
		{"abc def", 2, []string{"abc", "def"}},
		{"abc def", 3, []string{"abc", "def"}},
		{"abc def", 4, []string{"abc", "def"}},
		{"abc def", 5, []string{"abc", "def"}},
		{"abc def", 6, []string{"abc", "def"}},
		{"abc def", 7, []string{"abc def"}},
		{"abc   def", 2, []string{"abc", "def"}},
		{"abc   def", 3, []string{"abc", "def"}},
		{"abc   def", 4, []string{"abc", "def"}},
		{"abc   def", 8, []string{"abc", "def"}},
		{"abc   def", 9, []string{"abc   def"}},
		{"abc\ndef ghi", 3, []string{"abc", "def", "ghi"}},
	} {
		if got := wrap(tc.orig, tc.max); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("wrap(%q, %d) = %q; want %q", tc.orig, tc.max, got, tc.want)
		}
	}
}

func TestParseInsert(t *testing.T) {
	// This just tests a variety of tricky lines from
	// https://raw.githubusercontent.com/metabrainz/musicbrainz-server/master/t/sql/initial.sql.
	for _, tc := range []struct {
		stmt  string
		table string
		vals  []interface{}
	}{
		{
			`INSERT INTO event_alias_type VALUES (1, 'Event name', NULL, 0, NULL, '412aac48-424b-3052-a314-1f926e8018c8');`,
			"event_alias_type",
			[]interface{}{1, "Event name", nil, 0, nil, "412aac48-424b-3052-a314-1f926e8018c8"},
		},
		{
			`INSERT INTO language VALUES (284, 'mul', 'mul', '', '[Multiple languages]', 2, 'mul');`,
			"language",
			[]interface{}{284, "mul", "mul", "", "[Multiple languages]", 2, "mul"},
		},
		{
			`INSERT INTO link_attribute_type VALUES (194, NULL, 194, 0, 'b3045913-62ac-433e-9211-ac683cdf6b5c', 'guest', 'This attribute indicates a ''guest'' performance where the performer is not usually part of the band.', '2011-09-21 18:29:05.11911+00');`,
			"link_attribute_type",
			[]interface{}{194, nil, 194, 0, "b3045913-62ac-433e-9211-ac683cdf6b5c", "guest",
				"This attribute indicates a 'guest' performance where the performer is not usually part of the band.",
				"2011-09-21 18:29:05.11911+00"},
		},
		{
			`INSERT INTO release_group_primary_type VALUES (1, 'Album', NULL, 1, NULL, 'f529b476-6e62-324f-b0aa-1f3e33d313fc') ON CONFLICT (id) DO NOTHING;`,
			"release_group_primary_type",
			[]interface{}{1, "Album", nil, 1, nil, "f529b476-6e62-324f-b0aa-1f3e33d313fc"},
		},
		{
			`INSERT INTO link_attribute_type VALUES (14, NULL, 14, 3, '0abd7f04-5e28-425b-956f-94789d9bcbe2', 'instrument', E'This attribute describes the possible instruments that can be captured as part of a performance.\n<br/>\nCan''t find an instrument? <a href="http://wiki.musicbrainz.org/Advanced_Instrument_Tree">Request it!</a>', '2011-09-21 18:29:05.11911+00');`,
			"link_attribute_type",
			[]interface{}{14, nil, 14, 3, "0abd7f04-5e28-425b-956f-94789d9bcbe2", "instrument",
				"This attribute describes the possible instruments that can be captured as part of a performance.\n" +
					"<br/>\nCan't find an instrument? <a href=\"http://wiki.musicbrainz.org/Advanced_Instrument_Tree\">Request it!</a>",
				"2011-09-21 18:29:05.11911+00"},
		},
		{
			`INSERT INTO link_type VALUES (270, NULL, 1, '00687ce8-17e1-3343-b6e5-0a91b919fe24', 'url', 'work', 'misc', 'This relationship type is <strong>deprecated</strong>.', 'miscellaneous roles', 'miscellaneous support', 'has a miscellaneous role on', 0, '2014-05-18 11:09:53.284379+00', true, false, 0, 0);`,
			"link_type",
			[]interface{}{270, nil, 1, "00687ce8-17e1-3343-b6e5-0a91b919fe24", "url", "work", "misc",
				"This relationship type is <strong>deprecated</strong>.",
				"miscellaneous roles", "miscellaneous support", "has a miscellaneous role on",
				0, "2014-05-18 11:09:53.284379+00", true, false, 0, 0},
		},
		{
			`INSERT INTO medium_format VALUES (58, 'Pathé disc', 13, 0, 1906, false, '90 rpm, vertical-cut shellac discs, produced by the Pathé label from 1906 to 1932.', '34cc287c-c448-3fe4-90d6-ed3a6fa35fe5');`,
			"medium_format",
			[]interface{}{58, "Pathé disc", 13, 0, 1906, false,
				"90 rpm, vertical-cut shellac discs, produced by the Pathé label from 1906 to 1932.",
				"34cc287c-c448-3fe4-90d6-ed3a6fa35fe5"},
		},
	} {
		if table, vals, err := parseInsert(tc.stmt); err != nil {
			t.Errorf("parseTable(%q) failed: %v", tc.stmt, err)
		} else if table != tc.table || !reflect.DeepEqual(vals, tc.vals) {
			t.Errorf("parseTable(%q) = %q, %v; want %q, %v", tc.stmt, table, vals, tc.table, tc.vals)
		}
	}
}
