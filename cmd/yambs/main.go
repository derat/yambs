// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/bandcamp"
	"github.com/derat/yambs/sources/text"
)

var version = "[non-release]"

const (
	actionOpen  = "open"  // open the page
	actionPrint = "print" // print URLs
	actionServe = "serve" // serve the page locally over HTTP
	actionWrite = "write" // write the page to stdout
)

func main() {
	action := enumFlag{
		val:     actionOpen,
		allowed: []string{actionOpen, actionPrint, actionServe, actionWrite},
	}
	editType := enumFlag{
		// TODO: Get rid of the default value?
		val:     string(seed.RecordingType),
		allowed: []string{string(seed.RecordingType), string(seed.ReleaseType)},
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
	addr := flag.String("addr", "localhost:8999", "Address to listen on for HTTP requests")
	fields := flag.String("fields", "", `Comma-separated fields for CSV/TSV columns (e.g. "artist,name,length")`)
	flag.Var(&format, "format", fmt.Sprintf("Format for text input (%v)", format.allowedList()))
	listFields := flag.Bool("list-fields", false, "Print available fields for -type and exit")
	server := flag.Bool("server", false, "Run a web server at -addr with a form for generating seed URLs")
	flag.Var(&setCmds, "set", `Set a field for all entities (e.g. "artist=The Beatles")`)
	flag.Var(&editType, "type", fmt.Sprintf("Entity type of text input (%v)", editType.allowedList()))
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
			var list [][2]string // name, desc
			var max int
			for name, desc := range text.ListFields(seed.Type(editType.val)) {
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

		if *server {
			if err := runServer(ctx, *addr); err != nil {
				fmt.Fprintln(os.Stderr, "Failed serving:", err)
				return 1
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

		db := db.NewDB(db.Version(version))
		var edits []seed.Edit
		if srcURL != "" {
			var err error
			if edits, err = bandcamp.Fetch(ctx, srcURL); err != nil {
				fmt.Fprintln(os.Stderr, "Failed fetching page:", err)
				return 1
			}
		} else {
			var err error
			if edits, err = text.Read(ctx, r, text.Format(format.val), seed.Type(editType.val),
				strings.Split(*fields, ","), setCmds, db); err != nil {
				fmt.Fprintln(os.Stderr, "Failed reading edits:", err)
				return 1
			}
		}

		switch action.val {
		case actionOpen:
			if err := openPage(edits); err != nil {
				fmt.Fprintln(os.Stderr, "Failed opening page:", err)
				return 1
			}
		case actionPrint:
			for _, ed := range edits {
				if !ed.CanGet() {
					fmt.Fprintf(os.Stderr, "Can't print bare URL; %s edit requires POST request\n",
						ed.Type())
					return 1
				}
				u, err := url.Parse(ed.URL())
				if err != nil {
					fmt.Fprintln(os.Stderr, "Failed parsing URL:", err)
					return 1
				}
				u.RawQuery = ed.Params().Encode()
				fmt.Println(u.String())
			}
		case actionServe:
			if err := servePage(ctx, *addr, edits); err != nil {
				fmt.Fprintln(os.Stderr, "Failed serving page:", err)
				return 1
			}
		case actionWrite:
			if err := writePage(os.Stdout, edits); err != nil {
				fmt.Fprintln(os.Stderr, "Failed writing page:", err)
				return 1
			}
		}

		return 0
	}())
}

var urlRegexp = regexp.MustCompile("(?i)^https?://")
