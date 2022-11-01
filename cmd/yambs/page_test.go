// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/derat/yambs/seed"
	"golang.org/x/net/html"
)

func TestWritePage(t *testing.T) {
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

	var b bytes.Buffer
	if err := writePage(&b, edits); err != nil {
		t.Fatal("writePage failed:", err)
	}

	// Just perform some basic tests that the edit descriptions and URLs were included
	// and that the page is parseable HTML.
	for _, ed := range edits {
		if desc := template.JSEscapeString(ed.Description()); !strings.Contains(b.String(), desc) {
			t.Errorf("writePage didn't include edit description %q", desc)
		}
		if url := template.JSEscapeString(ed.URL()); !strings.Contains(b.String(), url) {
			t.Errorf("writePage didn't include edit URL %q", url)
		}
	}
	if _, err := html.Parse(&b); err != nil {
		t.Error("writePage wrote invalid HTML:", err)
	}
}
