// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	dumpVar    = "MBDUMP_SAMPLE" // env var pointing at extracted dump
	dumpURL    = "https://data.metabrainz.org/pub/musicbrainz/data/sample/"
	dstPath    = "enums.go" // this program is run from 'seed' dir
	mdPath     = "full_enums.md"
	mdURL      = "https://github.com/derat/yambs/blob/main/seed/" + mdPath
	commentLen = 80 - 4 // account for "\t// "

	instLinkAttrTypeID       = "14" // ID for root "instrument" link attribute type
	minInstLinkAttrTypeCount = 10   // minimum count for instrument to be included
)

type enumTypes struct {
	types []*enumType
}

func (ets *enumTypes) add(et *enumType) *enumType {
	ets.types = append(ets.types, et)
	return et
}

func (ets *enumTypes) finish() {
	sort.Slice(ets.types, func(i, j int) bool { return ets.types[i].Name < ets.types[j].Name })
	for _, et := range ets.types {
		if et.sort {
			sort.Slice(et.Values, func(i, j int) bool {
				return strings.ToLower(et.Values[i].Name) < strings.ToLower(et.Values[j].Name)
			})
		}
	}
}

type enumType struct {
	Name    string // enum type name
	Type    string // Go type
	Comment string // comment before declaration
	Values  []enumValue
	sort    bool // sort values by name
}

func (et *enumType) add(ev enumValue) { et.Values = append(et.Values, ev) }

type enumValue struct {
	Name    string // enumType.name and underscore will be prepended
	Value   string // literal value, i.e. quoted if string
	Comment string // comment before declaration
	EOL     string // end-of-line comment
}

func main() {
	if os.Getenv(dumpVar) == "" {
		fmt.Fprintf(os.Stderr, "Set %v to extracted dump from %v\n", dumpVar, dumpURL)
		os.Exit(2)
	}

	tf := openFile("TIMESTAMP")
	defer tf.Close()
	ts, err := io.ReadAll(tf)
	if err != nil {
		log.Fatal("Failed reading timestamp: ", err)
	}

	var enums enumTypes
	var fullEnums enumTypes // unabridged versions of large enums

	langs := enums.add(&enumType{
		Name: "Language",
		Type: "int",
		Comment: `Language represents a human language. ` +
			`These values correspond to integer IDs in the database; ` +
			`note that some fields (most notably Release.Language) ` +
			`confusingly use ISO 639-3 codes instead. Roughly 7400 ` +
			`languages marked as being low-frequency are excluded, ` +
			`but all languages are listed in ` + mdURL + `.`,
		sort: true,
	})
	fullLangs := fullEnums.add(&enumType{
		Name: "Language",
		sort: true,
	})
	readTable("language", func(row []string) {
		id, name, freq := row[0], row[4], row[5]
		if freq != "0" {
			langs.add(enumValue{
				Name:  clean(langCleanRegexp.ReplaceAllString(name, "")),
				Value: id,
				EOL:   name,
			})
		}
		fullLangs.add(enumValue{Name: name, Value: id})
	})

	// Count the frequency of different link attribute types in the sample.
	linkAttrTypeCounts := make(map[string]int)
	readTable("link_attribute", func(row []string) { linkAttrTypeCounts[row[1]]++ })

	linkAttrTypes := enums.add(&enumType{
		Name: "LinkAttributeType",
		Type: "int",
		Comment: `LinkAttributeType is an ID describing an attribute ` +
			`associated with a link between two MusicBrainz entities. ` +
			`Roughly 700 infrequently-appearing musical instruments ` +
			`are excluded, but all types are listed in ` + mdURL + `.`,
		sort: true,
	})
	fullLinkAttrTypes := fullEnums.add(&enumType{
		Name: "LinkAttributeType",
		sort: true,
	})
	readTable("link_attribute_type", func(row []string) {
		id, root, name, desc := row[0], row[2], row[5], row[6]
		// There are a bit over a thousand types corresponding to instruments, so only include ones
		// that show up fairly often in the sample. This seems sub-optimal if the included
		// instruments change across dumps, but I'm not sure what else to do.
		if root != instLinkAttrTypeID || linkAttrTypeCounts[id] >= minInstLinkAttrTypeCount {
			cleaned := clean(name)
			if v, ok := linkAttrTypeMappings[id]; ok {
				cleaned = clean(v)
			}
			linkAttrTypes.add(enumValue{
				Name:    cleaned,
				Value:   id,
				Comment: desc,
				EOL:     name,
			})
		}
		fullLinkAttrTypes.add(enumValue{
			Name:    name,
			Value:   id,
			Comment: desc,
		})
	})

	linkTypes := enums.add(&enumType{
		Name: "LinkType",
		Type: "int",
		Comment: `LinkType is an ID describing a link between two MusicBrainz entities. ` +
			`Only link types relating to entity types that can be seeded by yambs are included.`,
		sort: true,
	})
	readTable("link_type", func(row []string) {
		id, type0, type1, name, desc := row[0], row[4], row[5], row[6], row[7]
		if seedEntityTypes[type0] || seedEntityTypes[type1] {
			linkTypes.add(enumValue{
				Name:    fmt.Sprintf("%s_%s_%s", clean(name), clean(type0), clean(type1)),
				Value:   id,
				Comment: desc,
			})
		}
	})

	mediumFormats := enums.add(&enumType{
		Name:    "MediumFormat",
		Type:    "string",
		Comment: `MediumFormat describes a medium's format (e.g. CD, cassette, digital media).`,
	})
	readTable("medium_format", func(row []string) {
		name, desc := row[1], row[6]
		mediumFormats.add(enumValue{
			Name:    clean(name),
			Value:   fmt.Sprintf("%q", name),
			Comment: desc,
		})
	})

	releaseGroupTypes := enums.add(&enumType{
		Name: "ReleaseGroupType",
		Type: "string",
		Comment: `ReleaseGroupType describes a release group. ` +
			`A release group can be assigned a single primary type and multiple secondary types.`,
	})
	readTable("release_group_primary_type", func(row []string) {
		name := row[1]
		releaseGroupTypes.add(enumValue{
			Name:  clean(name),
			Value: fmt.Sprintf("%q", name),
			EOL:   "primary",
		})
	})
	readTable("release_group_secondary_type", func(row []string) {
		name := row[1]
		releaseGroupTypes.add(enumValue{
			Name:  clean(name),
			Value: fmt.Sprintf("%q", name),
			EOL:   "secondary",
		})
	})

	releasePackagings := enums.add(&enumType{
		Name:    "ReleasePackaging",
		Type:    "string",
		Comment: "ReleasePackaging describes a release's packaging.",
	})
	readTable("release_packaging", func(row []string) {
		name, desc := row[1], row[4]
		releasePackagings.add(enumValue{
			Name:    clean(name),
			Value:   fmt.Sprintf("%q", name),
			Comment: desc,
		})
	})

	releaseStatuses := enums.add(&enumType{
		Name:    "ReleaseStatus",
		Type:    "string",
		Comment: "ReleaseStatus describes a release's status.",
	})
	readTable("release_status", func(row []string) {
		name, desc := row[1], row[4]
		releaseStatuses.add(enumValue{
			Name:    clean(name),
			Value:   fmt.Sprintf("%q", name),
			Comment: desc,
		})
	})

	workAttrTypes := enums.add(&enumType{
		Name:    "WorkAttributeType",
		Type:    "int",
		Comment: `WorkAttributeType describes an attribute attached to a work.`,
		sort:    true,
	})
	readTable("work_attribute_type", func(row []string) {
		id, orig, desc := row[0], row[1], row[6]
		// Most names are full of acronyms like "AGADU ID" or "AKKA/LAA ID".
		// Preserve these in names ending in " ID".
		var name string
		if strings.HasSuffix(orig, " ID") {
			name = normalize(orig) // needed for "MÜST ID"
			name = nonAlnumRegexp.ReplaceAllString(name, "_")
		} else {
			name = clean(orig)
		}
		workAttrTypes.add(enumValue{
			Name:    name,
			Value:   id,
			Comment: desc,
			EOL:     orig,
		})
	})

	workTypes := enums.add(&enumType{
		Name:    "WorkType",
		Type:    "int",
		Comment: "WorkType describes a work's type.",
		sort:    true,
	})
	readTable("work_type", func(row []string) {
		id, name, desc := row[0], row[1], row[4]
		workTypes.add(enumValue{
			Name:    clean(name),
			Value:   id,
			Comment: desc,
		})
	})

	enums.finish()
	fullEnums.finish()

	// Write the file.
	funcMap := map[string]interface{}{
		"wrap": func(s string) []string { return wrap(s, commentLen) },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).Parse(fileTemplate))
	f, err := os.Create(dstPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := tmpl.Execute(f, struct {
		Time  string
		Enums []*enumType
	}{
		Time:  strings.TrimSpace(string(ts)),
		Enums: enums.types,
	}); err != nil {
		f.Close()
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	// Format the file.
	if err := exec.Command("gofmt", "-w", dstPath).Run(); err != nil {
		log.Fatalf("gofmt failed on %v: %v", dstPath, err)
	}

	// Also write the MarkDown file with full definitions.
	mdTmpl := template.Must(template.New("").Funcs(funcMap).Parse(mdTemplate))
	mf, err := os.Create(mdPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := mdTmpl.Execute(mf, struct{ Enums []*enumType }{Enums: fullEnums.types}); err != nil {
		mf.Close()
		log.Fatal(err)
	}
	if err := mf.Close(); err != nil {
		log.Fatal(err)
	}
}

// openFile opens the named relative path under the dump directory.
// It crashes if an error is encountered.
func openFile(rel string) *os.File {
	p := filepath.Join(os.Getenv(dumpVar), rel)
	f, err := os.Open(p)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

// readTable opens the named table from the dump and passes each row to fn.
func readTable(table string, fn func([]string)) {
	f := openFile(filepath.Join("mbdump", table))
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		row := strings.Split(sc.Text(), "\t")
		for i, v := range row {
			if v == `\N` {
				row[i] = ""
			} else {
				v = strings.ReplaceAll(v, `\r\n`, " ")
				v = strings.ReplaceAll(v, `<br/>`, " ")
				row[i] = strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
			}
		}
		fn(row)
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("Failed reading %v: %v", table, err)
	}
}

// fileTemplate is used to generate dstPath.
const fileTemplate = `
package seed

// This file was generated from a dump of the MusicBrainz database
// (https://musicbrainz.org/doc/MusicBrainz_Database/Download)
// initiated at {{.Time}}.
//
// MusicBrainz database dumps are distributed under the CC0 license:
// https://creativecommons.org/publicdomain/zero/1.0/
//
// This file can be regenerated by running "go generate".

{{range .Enums}}
{{range wrap .Comment -}}
// {{.}}
{{end -}}
type {{.Name}} {{.Type}}

const (
{{$en := .Name}}{{range .Values -}}
{{range wrap .Comment -}}
// {{.}}
{{end -}}
{{$en}}_{{.Name}} {{$en}} = {{.Value}}{{if .EOL}} // {{.EOL}}{{end}}
{{end -}}
)
{{end}}
`

// mdTemplate is used to generate mdPath.
const mdTemplate = `# Full MusicBrainz enums

This file contains full definitions of large MusicBrainz enums
that are abridged in [enums.go](./enums.go).

MusicBrainz database dumps are distributed under the CC0 license:
https://creativecommons.org/publicdomain/zero/1.0/

{{range .Enums -}}
## {{.Name}}

| Name | Value |
| :--  | :--   |
{{range .Values -}}
| {{.Name}} | {{.Value}} |
{{end}}
{{end -}}
`

// Matches trailing parentheticals containing numbers.
var langCleanRegexp = regexp.MustCompile(`\s*\([^)]*\d[^)]*\)$`)

// linkAttrTypeMappings remaps duplicate names in the link_attribute_type table.
var linkAttrTypeMappings = map[string]string{
	"560":  "tar lute",      // "tar", conflicts with 752
	"752":  "tar drum",      // "tar", conflicts with 560
	"1032": "number opera",  // "number", conflicts with 788
	"1128": "other subject", // "other", conflicts with 1225
	"1225": "other level",   // "other", conflicts with 1128
}

// seedEntityTypes contains entity types that can be seeded by yambs.
// This is used to prune the link_type table based on its entity_type0 and
// entity_type1 columns.
var seedEntityTypes = map[string]bool{
	"recording":     true,
	"release":       true,
	"release_group": true,
	"work":          true,
}

// wordMap contains words with specialized capitalization.
var wordMap = map[string]string{
	"8cm":          "8cm",
	"allmusic":     "AllMusic",
	"asin":         "ASIN",
	"bookbrainz":   "BookBrainz",
	"cd":           "CD",
	"cdv":          "CDV",
	"ced":          "CED",
	"dataplay":     "DataPlay",
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
	"hipac":        "HiPac",
	"hqcd":         "HQCD",
	"imdb":         "IMDB",
	"imslp":        "IMSLP",
	"laserdisc":    "LaserDisc",
	"minidisc":     "MiniDisc",
	"playtape":     "PlayTape",
	"prs":          "PRS",
	"releasegroup": "ReleaseGroup",
	"sacd":         "SACD",
	"sd":           "SD",
	"shm":          "SHM",
	"slotmusic":    "slotMusic",
	"snappack":     "SnapPack",
	"sp":           "SP",
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

var nonAlnumRegexp = regexp.MustCompile("[^a-zA-Z0-9]+")
var splitRegexp = regexp.MustCompile("[-+ /]+")

// https://go.dev/blog/normalization#performing-magic
var normalizer = transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)))

// normalize normalizes characters using NFKD form.
// Unicode characters are decomposed (runes are broken into their components) and replaced for
// compatibility equivalence (characters that represent the same characters but have different
// visual representations, e.g. '9' and '⁹', are equal). Characters are also de-accented.
func normalize(orig string) string {
	b := make([]byte, len(orig))
	if _, _, err := normalizer.Transform(b, []byte(orig), true); err != nil {
		return orig
	}
	return string(bytes.TrimRight(b, "\x00"))
}

// clean attempts to transform orig into a string that can be used in an identifier.
// Each word is capitalized.
func clean(orig string) string {
	var s string
	for _, w := range splitRegexp.Split(orig, -1) {
		w = strings.ToLower(normalize(w))
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
	for _, rest := range strings.Split(strings.TrimSpace(orig), "\n") {
		rest = strings.TrimSpace(rest)
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
	}
	return lines
}
