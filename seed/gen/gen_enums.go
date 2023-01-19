// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	sqlURL      = "https://raw.githubusercontent.com/metabrainz/musicbrainz-server/master/t/sql/initial.sql"
	dstPath     = "enums.go" // called from parent dir
	licensePath = "COPYING-enums.md"
	commentLen  = 80 - 4 // account for "\t// "
)

type enumType struct {
	Name    string   // enum type name
	Type    string   // Go type
	Comment []string // multiline comment before declaration
	Values  []enumValue
	sort    bool // sort values by name
}

func (et *enumType) add(ev enumValue) { et.Values = append(et.Values, ev) }

type enumValue struct {
	Name    string   // enumType.name and underscore will be prepended
	Value   string   // literal value, i.e. quoted if string
	Comment []string // multiline comment before declaration
	EOL     string   // end-of-line comment
}

func main() {
	linkAttrTypes := enumType{
		Name: "LinkAttributeType",
		Type: "int",
		Comment: []string{
			`LinkType is an ID describing an attribute associated with a link between two`,
			`MusicBrainz entities.`,
		},
	}
	linkTypes := enumType{
		Name: "LinkType",
		Type: "int",
		Comment: []string{
			`LinkType is an ID describing a link between two MusicBrainz entities.`,
			`It sadly doesn't appear to enumerate all possible values. There are 170-ish`,
			`additional link types with translations in po/relationships.pot, many of`,
			`which don't appear to be referenced anywhere else in the server repo.`,
		},
		Values: []enumValue{
			// These IDs are listed in https://musicbrainz.org/recording/create and
			// https://musicbrainz.org/release/add, so presumably they're being used.
			// Comments are from po/relationships.pot.
			{
				Name:  "Crowdfunding_Recording_URL",
				Value: "905",
				Comment: wrap(
					"This links a recording to the relevant crowdfunding project at a "+
						"crowdfunding site like Kickstarter or Indiegogo.", commentLen),
			},
			{
				Name:  "Crowdfunding_Release_URL",
				Value: "906",
				Comment: wrap(
					"This links a release to the relevant crowdfunding project at a crowdfunding "+
						"site like Kickstarter or Indiegogo.", commentLen),
			},
			{
				Name:  "StreamingPaid_Recording_URL",
				Value: "979",
				Comment: wrap(
					"This relationship type is used to link a track to a site where the track can "+
						"be legally streamed for a subscription fee, e.g. Tidal. "+
						"If the site allows free streaming, use \"free streaming\" instead.", commentLen),
			},
			{
				Name:  "StreamingPaid_Release_URL",
				Value: "980",
				Comment: wrap(
					"This relationship type is used to link a release to a site where the tracks "+
						"can be legally streamed for a subscription fee, e.g. Tidal.", commentLen),
			},
		},
		sort: true,
	}
	mediumFormats := enumType{
		Name:    "MediumFormat",
		Type:    "string",
		Comment: []string{`MediumFormat describes a medium's format (e.g. CD, cassette, digital media).`},
	}
	releaseGroupTypes := enumType{
		Name: "ReleaseGroupType",
		Type: "string",
		Comment: []string{
			`ReleaseGroupType describes a release group.`,
			`A release group can be assigned a single primary type and multiple secondary types.`,
		},
	}
	releaseStatuses := enumType{
		Name:    "ReleaseStatus",
		Type:    "string",
		Comment: []string{"ReleaseStatus describes a release's status."},
	}
	releasePackagings := enumType{
		Name:    "ReleasePackaging",
		Type:    "string",
		Comment: []string{"ReleasePackaging describes a release's packaging."},
	}

	allEnums := []*enumType{
		&releaseGroupTypes,
		&releaseStatuses,
		&releasePackagings,
		&mediumFormats,
		&linkTypes, // close to the end because it's super-long
		&linkAttrTypes,
	}

	// Support reading from a file to make development easier.
	now := time.Now().UTC()
	var r io.Reader
	var srcPath string
	if len(os.Args) == 2 {
		srcPath = os.Args[1]
		f, err := os.Open(srcPath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		r = f
	} else {
		resp, err := http.Get(sqlURL)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Fatalf("Got %v: %v", resp.StatusCode, resp.Status)
		}
		r = resp.Body
	}

	// Process the SQL statements.
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		// Super-cheesy: change SQL-escaped apostrophes (which appear in some descriptions)
		// to smart apostrophes so they won't confuse my dumb regular expressions.
		ln := sc.Text()
		ln = strings.ReplaceAll(ln, "''", "’")

		if ms := linkAttrTypeRegexp.FindStringSubmatch(ln); ms != nil {
			id, name, desc := ms[1], ms[2], ms[3]
			linkAttrTypes.add(enumValue{
				Name:    clean(name),
				Value:   id,
				Comment: wrap(desc, commentLen),
			})
		} else if ms := linkTypeRegexp.FindStringSubmatch(ln); ms != nil {
			id, type1, type2, name, desc := ms[1], ms[2], ms[3], ms[4], ms[5]
			switch id {
			case "184":
				// 171 and 184 are both named "discography" and map from Artists to URLs.
				// 184 lists 171 as its parent, and some of the translations call it
				// "discography page", so rename it to that to prevent a conflict.
				name = "discography page"
			}
			linkTypes.add(enumValue{
				// If this format is changed, the hardcoded entries in linkTypes need to be updated.
				Name:    fmt.Sprintf("%s_%s_%s", clean(name), clean(type1), clean(type2)),
				Value:   id,
				Comment: wrap(desc, commentLen),
			})
		} else if ms := mediumFormatsRegexp.FindStringSubmatch(ln); ms != nil {
			name, desc := ms[1], ms[2]
			mediumFormats.add(enumValue{
				Name:    clean(name),
				Value:   fmt.Sprintf("%q", name),
				Comment: wrap(desc, commentLen),
			})
		} else if ms := releaseGroupTypeRegexp.FindStringSubmatch(ln); ms != nil {
			typ, name := ms[1], ms[2]
			eol := "secondary"
			if typ == "primary" {
				eol = "primary"
			}
			releaseGroupTypes.add(enumValue{
				Name:  clean(name),
				Value: fmt.Sprintf("%q", name),
				EOL:   eol,
			})
		} else if ms := releaseStatusRegexp.FindStringSubmatch(ln); ms != nil {
			name, desc := ms[1], ms[2]
			releaseStatuses.add(enumValue{
				Name:    clean(name),
				Value:   fmt.Sprintf("%q", name),
				Comment: wrap(desc, commentLen),
			})
		} else if ms := releasePackagingRegexp.FindStringSubmatch(ln); ms != nil {
			name, desc := ms[1], ms[2]
			releasePackagings.add(enumValue{
				Name:    clean(name),
				Value:   fmt.Sprintf("%q", name),
				Comment: wrap(desc, commentLen),
			})
		}
	}
	if sc.Err() != nil {
		log.Fatal(sc.Err())
	}

	// Sort values if requested.
	for _, et := range allEnums {
		if et.sort {
			sort.Slice(et.Values, func(i, j int) bool { return et.Values[i].Name < et.Values[j].Name })
		}
	}

	// Write the file.
	tmpl, err := template.New("").Parse(fileTemplate)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(dstPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := tmpl.Execute(f, struct {
		License string
		Time    string
		URL     string
		Enums   []*enumType
	}{
		License: licensePath,
		Time:    now.Format("2006-01-02 15:04:05 MST"),
		URL:     sqlURL,
		Enums:   allEnums,
	}); err != nil {
		f.Close()
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	// Format the file.
	if err := exec.Command("gofmt", "-w", dstPath).Run(); err != nil {
		log.Fatal(err)
	}
}

const fileTemplate = `
// This file is derived from https://github.com/metabrainz/musicbrainz-server,
// which is licensed under GNU General Public License (GPL) Version 2 or later.
// This license is located at {{.License}}.

package seed

// This file was automatically generated from a copy of
// {{.URL}}
// downloaded at {{.Time}}.
// It can be regenerated by running "go generate".

{{range .Enums}}
{{range .Comment -}}
// {{.}}
{{end -}}
type {{.Name}} {{.Type}}

const (
{{$en := .Name}}{{range .Values -}}
{{range .Comment -}}
// {{.}}
{{end -}}
{{$en}}_{{.Name}} {{$en}} = {{.Value}}{{if .EOL}} // {{.EOL}}{{end}}
{{end -}}
)
{{end}}
`

// wordMap contains words with specialized capitalization.
var wordMap = map[string]string{
	"8cm":          "8cm",
	"allmusic":     "AllMusic",
	"asin":         "ASIN",
	"bookbrainz":   "BookBrainz",
	"cd":           "CD",
	"cdv":          "CDV",
	"ced":          "CED",
	"dat":          "DAT",
	"dcc":          "DCC",
	"dj":           "DJ",
	"dts":          "DTS",
	"dualdisc":     "DualDisc",
	"dvdaudio":     "DVDAudio",
	"dvd":          "DVD",
	"dvdplus":      "DVDplus",
	"dvdvideo":     "DVDVideo",
	"ep":           "EP",
	"hdcd":         "HDCD",
	"hd":           "HD",
	"hqcd":         "HQCD",
	"imdb":         "IMDB",
	"imslp":        "IMSLP",
	"laserdisc":    "LaserDisc",
	"minidisc":     "MiniDisc",
	"releasegroup": "ReleaseGroup",
	"sacd":         "SACD",
	"shm":          "SHM",
	"slotmusic":    "slotMusic",
	"svcd":         "SVCD",
	"umd":          "UMD",
	"url":          "URL",
	"usb":          "USB",
	"vcd":          "VCD",
	"vgmdb":        "VGMdb",
	"vhd":          "VHD",
	"vhs":          "VHS",
	"viaf":         "VIAF",
	"vinyldisc":    "VinylDisc",
	"youtube":      "YouTube",
}

var nonAlnumRegexp = regexp.MustCompile("[^a-z0-9]+")
var splitRegexp = regexp.MustCompile("[-+ /]+")

// https://go.dev/blog/normalization#performing-magic
var normalizer = transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)))

// clean attempts to transform orig into a string that can be used in an identifier.
// Each word is capitalized.
func clean(orig string) string {
	var s string
	for _, w := range splitRegexp.Split(orig, -1) {
		// Normalize characters using NFKD form. Unicode characters are decomposed (runes are broken
		// into their components) and replaced for compatibility equivalence (characters that
		// represent the same characters but have different visual representations, e.g. '9' and
		// '⁹', are equal). Characters are also de-accented.
		b := make([]byte, len(w))
		if _, _, err := normalizer.Transform(b, []byte(w), true); err == nil {
			b = bytes.TrimRight(b, "\x00")
			w = string(b)
		}
		w = strings.ToLower(w)

		w = nonAlnumRegexp.ReplaceAllString(strings.ToLower(w), "")
		if dst, ok := wordMap[w]; ok {
			w = dst
		} else {
			w = cases.Title(language.English, cases.Compact).String(w)
		}
		s += w
	}
	return s
}

const spaceChars = " \t"

// wrap attempts to wrap orig to lines with the supplied maximum length.
func wrap(orig string, max int) []string {
	var lines []string
	rest := strings.TrimSpace(orig)
	for rest != "" {
		if len(rest) <= max {
			lines = append(lines, rest)
			break
		}
		if idx := strings.LastIndexAny(rest[:max+1], spaceChars); idx >= 0 {
			lines = append(lines, strings.TrimSpace(rest[:idx]))
			rest = strings.TrimSpace(rest[idx:])
		} else if idx := strings.IndexAny(rest[max:], spaceChars); idx >= 0 {
			lines = append(lines, rest[:max+idx])
			rest = strings.TrimSpace(rest[max+idx:])
		} else {
			lines = append(lines, rest)
			break
		}
	}
	return lines
}

// The below schema definitions come from
// https://raw.githubusercontent.com/metabrainz/musicbrainz-server/master/admin/sql/CreateTables.sql.

// TODO: These regular expressions are terrible.
// I guess I should write a little query parser.

//  CREATE TABLE link_attribute_type ( -- replicate
//  	id                  SERIAL,
//  	parent              INTEGER, -- references link_attribute_type.id
//  	root                INTEGER NOT NULL, -- references link_attribute_type.id
//  	child_order         INTEGER NOT NULL DEFAULT 0,
//  	gid                 UUID NOT NULL,
//  	name                VARCHAR(255) NOT NULL,
//  	description         TEXT,
//  	last_updated        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//  );
var linkAttrTypeRegexp = regexp.MustCompile(
	`(?i)^\s*INSERT\s+INTO\s+link_attribute_type\s+VALUES\s*\(` +
		`\s*(\d+)\s*,` + // 'id' (group 1)
		`[^,]+,` + // 'parent'
		`[^,]+,` + // 'root'
		`[^,]+,` + // 'child_order'
		`[^,]+,` + // 'gid' (MBID)
		`\s*'([^']*)'\s*,` + // 'name' (group 2)
		`\s*'([^']*)'\s*,` + // 'description' (group 3)
		`.*`)

//  CREATE TABLE link_type ( -- replicate
//  	id                  SERIAL,
//  	parent              INTEGER, -- references link_type.id
//  	child_order         INTEGER NOT NULL DEFAULT 0,
//  	gid                 UUID NOT NULL,
//  	entity_type0        VARCHAR(50) NOT NULL,
//  	entity_type1        VARCHAR(50) NOT NULL,
//  	name                VARCHAR(255) NOT NULL,
//  	description         TEXT,
//  	link_phrase         VARCHAR(255) NOT NULL,
//  	reverse_link_phrase VARCHAR(255) NOT NULL,
//  	long_link_phrase    VARCHAR(255) NOT NULL,
//  	priority            INTEGER NOT NULL DEFAULT 0,
//  	last_updated        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
//  	is_deprecated       BOOLEAN NOT NULL DEFAULT false,
//  	has_dates           BOOLEAN NOT NULL DEFAULT true,
//  	entity0_cardinality SMALLINT NOT NULL DEFAULT 0,
//  	entity1_cardinality SMALLINT NOT NULL DEFAULT 0
//  );
var linkTypeRegexp = regexp.MustCompile(
	`(?i)^\s*INSERT\s+INTO\s+link_type\s+VALUES\s*\(` +
		`\s*(\d+)\s*,` + // 'id' (group 1)
		`[^,]+,` + // 'parent'
		`[^,]+,` + // 'child_order'
		`[^,]+,` + // 'gid' (MBID)
		`\s*'([^']*)'\s*,` + // 'entity_type0' (group 2)
		`\s*'([^']*)'\s*,` + // 'entity_type1' (group 3)
		`\s*'([^']*)'\s*,` + // 'name' (group 4)
		`\s*'([^']*)'\s*,` + // 'description' (group 5)
		`.*`)

//  CREATE TABLE medium_format ( -- replicate
//  	id                  SERIAL,
//  	name                VARCHAR(100) NOT NULL,
//  	parent              INTEGER, -- references medium_format.id
//  	child_order         INTEGER NOT NULL DEFAULT 0,
//  	year                SMALLINT,
//  	has_discids         BOOLEAN NOT NULL DEFAULT FALSE,
//  	description         TEXT,
//  	gid                 uuid NOT NULL
//  );
var mediumFormatsRegexp = regexp.MustCompile(
	`(?i)^\s*INSERT\s+INTO\s+medium_format\s+VALUES\s*\(` +
		`\s*\d+\s*,*` + // 'id'
		`\s*'([^']+)'\s*,` + // 'name' (group 1)
		`[^,]+,` + // 'parent'
		`[^,]+,` + // 'child_order'
		`[^,]+,` + // 'year'
		`[^,]+,` + // 'has_discids'
		`(?:\s*'([^']+)'\s*,)?` + // 'description' (group 2)
		`.*`)

//  CREATE TABLE release_group_primary_type ( -- replicate
//      id                  SERIAL,
//      name                VARCHAR(255) NOT NULL,
//      parent              INTEGER, -- references release_group_primary_type.id
//      child_order         INTEGER NOT NULL DEFAULT 0,
//      description         TEXT,
//      gid                 uuid NOT NULL
//  );
//  CREATE TABLE release_group_secondary_type ( -- replicate
//      id                  SERIAL NOT NULL, -- PK
//      name                TEXT NOT NULL,
//      parent              INTEGER, -- references release_group_secondary_type.id
//      child_order         INTEGER NOT NULL DEFAULT 0,
//      description         TEXT,
//      gid                 uuid NOT NULL
//  );
var releaseGroupTypeRegexp = regexp.MustCompile(
	`(?i)^\s*INSERT\s+INTO\s+release_group_(primary|secondary)_type\s+VALUES\s*\(` + // (group 1)
		`\s*\d+\s*,*` + // 'id'
		`\s*'([^']+)'\s*,` + // 'name' (group 2)
		`.*`)

//  CREATE TABLE release_status ( -- replicate
//  	id                  SERIAL,
//  	name                VARCHAR(255) NOT NULL,
//  	parent              INTEGER, -- references release_status.id
//  	child_order         INTEGER NOT NULL DEFAULT 0,
//  	description         TEXT,
//  	gid                 uuid NOT NULL
//  );
var releaseStatusRegexp = regexp.MustCompile(
	`(?i)^\s*INSERT\s+INTO\s+release_status\s+VALUES\s*\(` +
		`\s*\d+\s*,*` + // 'id'
		`\s*'([^']+)'\s*,` + // 'name' (group 1)
		`[^,]+,` + // 'parent'
		`[^,]+,` + // 'child_order'
		`\s*'([^']+)'\s*,` + // 'description' (group 2)
		`.*`)

//  CREATE TABLE release_packaging ( -- replicate
//  	id                  SERIAL,
//  	name                VARCHAR(255) NOT NULL,
//  	parent              INTEGER, -- references release_packaging.id
//  	child_order         INTEGER NOT NULL DEFAULT 0,
//  	description         TEXT,
//  	gid                 uuid NOT NULL
//  );
var releasePackagingRegexp = regexp.MustCompile(
	`(?i)^\s*INSERT\s+INTO\s+release_packaging\s+VALUES\s*\(` +
		`\s*\d+\s*,*` + // 'id'
		`\s*'([^']+)'\s*,` + // 'name' (group 1)
		`[^,]+,` + // 'parent'
		`[^,]+,` + // 'child_order'
		`(?:\s*'([^']+)'\s*,)?` + // 'description' (group 2)
		`.*`)
