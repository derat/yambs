// Copyright 2022 Daniel Erat.
// All rights reserved.

package page

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/derat/yambs/seed"
	"golang.org/x/net/html"
)

func TestWrite_Edits(t *testing.T) {
	info, err := seed.NewInfo("Info Description", "https://www.example.org/")
	if err != nil {
		t.Fatal("NewInfo failed:", err)
	}
	edits := []seed.Edit{
		&seed.Release{
			Title:   "Release Title",
			Artists: []seed.ArtistCredit{{Name: "Release Artist"}},
		},
		&seed.Recording{
			Name:    "Recording Name",
			Artists: []seed.ArtistCredit{{Name: "Recording Artist"}},
		},
		info,
	}

	const srv = "test.musicbrainz.org"
	var b bytes.Buffer
	if err := Write(&b, edits, Server(srv)); err != nil {
		t.Fatal("Write failed:", err)
	}

	// Just perform some basic tests that the edit descriptions and URLs were included
	// and that the page is parseable HTML.
	for _, ed := range edits {
		if desc := template.JSEscapeString(ed.Description()); !strings.Contains(b.String(), desc) {
			t.Errorf("Write didn't include edit description %q", desc)
		}
		if url := template.JSEscapeString(ed.URL(srv)); !strings.Contains(b.String(), url) {
			t.Errorf("Write didn't include edit URL %q", url)
		}
	}
	if _, err := html.Parse(&b); err != nil {
		t.Error("Write wrote invalid HTML:", err)
	}
}

func TestWrite_Form(t *testing.T) {
	// Also check that we can write the no-edits version of the page with the form.
	const version = "20221105-deadbeef"
	var b bytes.Buffer
	if err := Write(&b, nil, Version(version)); err != nil {
		t.Fatal("Write failed:", err)
	}
	if !strings.Contains(b.String(), version) {
		t.Errorf("Write didn't include version %q", version)
	}
	if _, err := html.Parse(&b); err != nil {
		t.Error("Write wrote invalid HTML:", err)
	}
}
