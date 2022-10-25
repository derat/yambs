// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/text"
)

const (
	actionPage  = "page"
	actionPrint = "print"

	typeRecording = "recording"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage %v: [flag]... <FILE>\n"+
			"Seeds MusicBrainz edits.\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	action := enumFlag{val: actionPage, allowed: []string{actionPage, actionPrint}}
	entType := enumFlag{val: typeRecording, allowed: []string{typeRecording}}
	format := enumFlag{val: string(text.TSV), allowed: []string{string(text.CSV), string(text.TSV)}}
	var setCmds repeatedFlag

	flag.Var(&action, "action", fmt.Sprintf("Action to perform with seed URLs (%v)", action.allowedList()))
	fields := flag.String("fields", "", `Comma-separated fields for text input columns (e.g. "artist,title,length")`)
	flag.Var(&format, "format", fmt.Sprintf("Format for text input (%v)", format.allowedList()))
	listFields := flag.Bool("list-fields", false, "Print available fields for -type and exit")
	flag.Var(&setCmds, "set", `Set a field for all entities (e.g. "artist=The Beatles")`)
	flag.Var(&entType, "type", fmt.Sprintf("Type of entity to create (%v)", entType.allowedList()))
	flag.Parse()

	os.Exit(func() int {
		if *listFields {
			var names []string
			switch entType.val {
			case typeRecording:
				names = text.RecordingFields()
			}
			for _, n := range names {
				fmt.Println(n)
			}
			return 0
		}

		var r io.Reader
		switch flag.NArg() {
		case 0:
			r = os.Stdin
		case 1:
			f, err := os.Open(flag.Arg(0))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			defer f.Close()
			r = f
		default:
			flag.Usage()
			return 2
		}

		var edits []seed.Edit
		switch entType.val {
		case typeRecording:
			recs, err := text.ReadRecordings(r, text.Format(format.val), *fields, setCmds)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Failed reading recordings:", err)
				return 1
			}
			for i := range recs {
				edits = append(edits, &recs[i])
			}
		}

		switch action.val {
		case actionPage:
			if err := openPage(edits); err != nil {
				fmt.Fprintln(os.Stderr, "Failed opening page:", err)
				return 1
			}
		case actionPrint:
			for _, ed := range edits {
				fmt.Println(ed.URL())
			}
		}

		return 0
	}())
}
