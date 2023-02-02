// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package main implements a command-line program for generating seeded edits.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/render"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/mp3"
	"github.com/derat/yambs/sources/online"
	"github.com/derat/yambs/sources/text"
	"github.com/derat/yambs/web"
)

var version = "[non-release]"

const (
	actionOpen  = "open"  // open the page from a temp file
	actionPrint = "print" // print URLs
	actionServe = "serve" // open the page from a local HTTP server
	actionWrite = "write" // write the page to stdout
)

func main() {
	action := enumFlag{
		val:     defaultAction(),
		allowed: []string{actionOpen, actionPrint, actionServe, actionWrite},
	}
	var entity enumFlag // empty default
	for _, t := range seed.EntityTypes {
		entity.allowed = append(entity.allowed, string(t))
	}
	format := enumFlag{
		val:     string(text.TSV),
		allowed: []string{string(text.CSV), string(text.KeyVal), string(text.TSV)},
	}
	var setCmds repeatedFlag

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage %v: [flag]... <FILE/URL>\n"+
			"Seeds MusicBrainz edits.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Var(&action, "action", fmt.Sprintf("Action to perform with seed URLs (%v)", action.allowedList()))
	addr := flag.String("addr", "localhost:8999", `Address to listen on for -action=serve`)
	extractTrackArtists := flag.Bool("extract-track-artists", false, `Extract artist names from track titles in Bandcamp pages`)
	fields := flag.String("fields", "", `Comma-separated fields for CSV/TSV columns (e.g. "artist,name,length")`)
	flag.Var(&format, "format", fmt.Sprintf("Format for text input (%v)", format.allowedList()))
	listFields := flag.Bool("list-fields", false, "Print available fields for -type and exit")
	server := flag.String("server", "musicbrainz.org", "MusicBrainz server hostname")
	flag.Var(&setCmds, "set", `Set a field for all entities (e.g. "edit_note=from https://www.example.org")`)
	flag.Var(&entity, "type", fmt.Sprintf("Entity type for text or MP3 input (%v)", entity.allowedList()))
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	printVersion := flag.Bool("version", false, "Print the version and exit")
	flag.Parse()

	os.Exit(func() int {
		ctx := context.Background()

		if *printVersion {
			fmt.Println("yambs " + version)
			return 0
		}

		if *listFields {
			if entity.val == "" {
				fmt.Fprintln(os.Stderr, "Must specify entity type via -type")
				return 2
			}
			var list [][2]string // name, desc
			var max int
			for name, desc := range text.ListFields(seed.Entity(entity.val), false /* html */) {
				list = append(list, [2]string{name, desc})
				if len(name) > max {
					max = len(name)
				}
			}
			sort.Slice(list, func(i, j int) bool { return list[i][0] < list[j][0] })
			for _, f := range list {
				fmt.Printf("%-"+strconv.Itoa(max)+"s  %s\n", f[0], f[1])
			}
			return 0
		}

		if !*verbose {
			log.SetOutput(io.Discard)
		}

		var r io.Reader
		var srcURL string
		switch flag.NArg() {
		case 0:
			r = os.Stdin
		case 1:
			if arg := flag.Arg(0); urlRegexp.MatchString(arg) {
				srcURL = arg
			} else {
				f, err := os.Open(arg)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
				defer f.Close()
				r = f
			}
		default:
			flag.Usage()
			return 2
		}

		db := db.NewDB(db.Server(*server), db.Version(version))
		web.SetUserAgent(fmt.Sprintf("yambs/%s (+https://github.com/derat/yambs)", version))

		var edits []seed.Edit
		if srcURL != "" {
			var err error
			cfg := online.Config{ExtractTrackArtists: *extractTrackArtists}
			if edits, err = online.Fetch(ctx, srcURL, setCmds, db, &cfg); err != nil {
				fmt.Fprintln(os.Stderr, "Failed fetching page:", err)
				return 1
			}
		} else {
			if entity.val == "" {
				fmt.Fprintln(os.Stderr, "Must specify entity type via -type")
				return 2
			}
			var err error
			if f, ok := r.(*os.File); ok && strings.HasSuffix(strings.ToLower(f.Name()), ".mp3") {
				if edits, err = mp3.ReadFile(f, seed.Entity(entity.val), setCmds); err != nil {
					fmt.Fprintln(os.Stderr, "Failed reading MP3 file:", err)
					return 1
				}
			} else {
				if edits, err = text.Read(ctx, r, text.Format(format.val), seed.Entity(entity.val),
					strings.Split(*fields, ","), setCmds, db); err != nil {
					fmt.Fprintln(os.Stderr, "Failed reading edits:", err)
					return 1
				}
			}
		}

		opts := []render.Option{
			render.Server(*server),
			render.Version(version), // not actually displayed
		}

		switch action.val {
		case actionOpen:
			if err := render.OpenFile(edits, opts...); err != nil {
				fmt.Fprintln(os.Stderr, "Failed opening page:", err)
				return 1
			}
		case actionPrint:
			for _, ed := range edits {
				if ed.Method() != http.MethodGet {
					fmt.Fprintf(os.Stderr, "Can't print bare URL; %s edit requires %s request\n",
						ed.Entity(), ed.Method())
					return 1
				}
				u, err := url.Parse(ed.URL(*server))
				if err != nil {
					fmt.Fprintln(os.Stderr, "Failed parsing URL:", err)
					return 1
				}
				u.RawQuery = ed.Params().Encode()
				fmt.Println(u.String())
			}
		case actionServe:
			if err := render.OpenHTTP(ctx, *addr, edits, opts...); err != nil {
				fmt.Fprintln(os.Stderr, "Failed serving page:", err)
				return 1
			}
		case actionWrite:
			if err := render.Write(os.Stdout, edits, opts...); err != nil {
				fmt.Fprintln(os.Stderr, "Failed writing page:", err)
				return 1
			}
		}

		return 0
	}())
}

func defaultAction() string {
	// If we're running in a Chrome OS Crostini container, the external Chrome process won't
	// be able to access files that we write to /tmp, so start a web server instead.
	// TODO: Is there a better way to detect this case?
	if _, err := exec.LookPath("garcon-url-handler"); err == nil && os.Getenv("BROWSER") == "" {
		return actionServe
	}
	return actionOpen
}

var urlRegexp = regexp.MustCompile("(?i)^https?://")
