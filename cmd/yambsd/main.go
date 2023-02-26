// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package main implements a web server for generating seeded edits.
package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/render"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/online"
	"github.com/derat/yambs/sources/online/tidal"
	"github.com/derat/yambs/sources/text"
	"github.com/derat/yambs/web"
)

const (
	// TODO: Figure out reasonable values for these.
	maxReqBytes        = 128 * 1024
	textEditsTimeout   = 10 * time.Second
	onlineEditsTimeout = time.Minute // can take longer due to MB API rate limit
	maxEdits           = 200
	maxFields          = 1000

	editsDelay       = 3 * time.Second
	editsRateMapSize = 256
)

var version string

func init() {
	// When deploying to App Engine, app.yaml passes the version string via an environment variable.
	if v := os.Getenv("APP_VERSION"); v != "" {
		version = v
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %v [flag]...\n"+
			"Runs a web server for seeding MusicBrainz edits.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	addr := flag.String("addr", "localhost:8999", `Address to listen on for HTTP requests`)
	server := flag.String("server", "musicbrainz.org", "MusicBrainz server hostname")
	flag.Parse()

	// Just generate the page once.
	var b bytes.Buffer
	if err := render.Write(&b, nil, render.Version(version)); err != nil {
		log.Fatal("Failed generating page: ", err)
	}
	form := b.Bytes()

	serverURL := "https://" + *server
	db := mbdb.NewDB(mbdb.Version(version))
	web.SetUserAgent(fmt.Sprintf("yambs/%s (+https://github.com/derat/yambs)", version))
	rm := newRateMap(editsDelay, editsRateMapSize)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		if req.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(form); err != nil {
			log.Print("Failed writing page: ", err)
		}
	})

	// Generate edits requested via the form.
	http.HandleFunc("/edits", func(w http.ResponseWriter, req *http.Request) {
		caddr := clientAddr(req)
		infos, err := getEditsForRequest(w, req, serverURL, rm, db)
		if err != nil {
			var msg string
			code := http.StatusInternalServerError
			if herr, ok := err.(*httpError); ok {
				code = herr.code
				msg = herr.msg
			}
			if msg == "" {
				msg = http.StatusText(code)
			}
			log.Printf("Sending %d to %s: %v", code, caddr, err)
			http.Error(w, msg, code)
			return
		}
		log.Printf("Returning %d edit(s) to %s", len(infos), caddr)
		if err := json.NewEncoder(w).Encode(infos); err != nil {
			log.Printf("Failed sending edits to %s: %v", caddr, err)
		}
	})

	// Perform redirects for seed.AddCoverArtRedirectURI.
	http.HandleFunc("/redirect-add-cover-art", func(w http.ResponseWriter, req *http.Request) {
		mbid := req.FormValue("release_mbid")
		if !mbdb.IsMBID(mbid) {
			http.Error(w, "Invalid release_mbid parameter", http.StatusBadRequest)
			return
		}
		dst, err := url.Parse(req.Referer())
		if err != nil || !mbSrvRegexp.MatchString(dst.Host) {
			http.Error(w, "Bad referrer", http.StatusBadRequest)
			return
		}
		dst.Path = "/release/" + mbid + "/add-cover-art"
		dst.RawQuery = ""
		dst.Fragment = ""
		http.Redirect(w, req, dst.String(), http.StatusFound)
	})

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(faviconData)
	})
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "User-agent: *\nAllow: /\n")
	})

	// Handle App Engine specifying the port to listen on.
	if port := os.Getenv("PORT"); port != "" {
		*addr = ":" + port
	}
	log.Print("Listening on ", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("Failed listening: ", err)
	}
}

// mbSrvRegexp matches a hostname under musicbrainz.org.
var mbSrvRegexp = regexp.MustCompile(`(?i)(?:^|\.)musicbrainz\.org$`)

// httpError implements the error interface but also wraps an HTTP status code
// and message that should be rutrned to the user.
type httpError struct {
	code int    // HTTP status code
	msg  string // message to display to user; if empty, generated from code
	err  error  // actual underlying error to log
}

func (e *httpError) Error() string { return e.err.Error() }

// httpErrorf returns an *httpError with the supplied status code and an err
// field constructed from format and args. The user-visible message will just
// be generated from code.
func httpErrorf(code int, format string, args ...interface{}) *httpError {
	return &httpError{code: code, err: fmt.Errorf(format, args...)}
}

// getEditsForRequest generates render.EditInfo objects in response to an /edits request to the server.
func getEditsForRequest(w http.ResponseWriter, req *http.Request,
	serverURL string, rm *rateMap, db *mbdb.DB) ([]*render.EditInfo, error) {
	if req.Method != http.MethodPost {
		return nil, httpErrorf(http.StatusMethodNotAllowed, "bad method %q", req.Method)
	}

	// TODO: Check referrer?

	now := time.Now()
	caddr := clientAddr(req)

	ip, _, err := net.SplitHostPort(caddr)
	if err != nil {
		ip = caddr
	}
	if !rm.attempt(ip, now) {
		return nil, &httpError{
			code: http.StatusTooManyRequests,
			msg:  "Please wait a few seconds and try again",
			err:  errors.New("too many requests"),
		}
	}

	req.Body = http.MaxBytesReader(w, req.Body, maxReqBytes)
	if err := req.ParseMultipartForm(maxReqBytes); err != nil {
		return nil, &httpError{http.StatusBadRequest, "", err}
	}

	src := req.FormValue("source")
	log.Printf("Handling %v-byte %q request from %s", req.ContentLength, src, caddr)

	var edits []seed.Edit
	switch src {
	case "online":
		ctx, cancel := context.WithTimeout(req.Context(), onlineEditsTimeout)
		defer cancel()
		cfg := online.Config{
			CountryCode:         strings.ToUpper(strings.TrimSpace(req.FormValue("country"))),
			ExtractTrackArtists: req.FormValue("extractTrackArtists") == "1",
		}
		if !countryCodeRegexp.MatchString(cfg.CountryCode) {
			return nil, &httpError{
				code: http.StatusBadRequest,
				msg:  "Invalid country code (should be two letters)",
				err:  fmt.Errorf("invalid country %q", cfg.CountryCode),
			}
		} else if cfg.CountryCode == tidal.AllCountriesCode {
			return nil, &httpError{
				code: http.StatusBadRequest,
				msg:  "Use yambs command-line program to query all countries",
				err:  errors.New("not querying all countries"),
			}
		}
		if url, err := online.CleanURL(req.FormValue("url")); err != nil {
			return nil, &httpError{
				code: http.StatusBadRequest,
				msg:  fmt.Sprintf("Unsupported URL (%v)", strings.Join(online.ExampleURLs, ", ")),
				err:  fmt.Errorf("%q: %v", req.FormValue("onlineUrl"), err),
			}
		} else if edits, err = online.Fetch(ctx, url, req.Form["set"], db, &cfg); err != nil {
			return nil, &httpError{
				code: http.StatusInternalServerError,
				msg:  fmt.Sprint("Failed getting edits: ", err),
				err:  err,
			}
		}

	case "text":
		ctx, cancel := context.WithTimeout(req.Context(), textEditsTimeout)
		defer cancel()
		// Sigh, so dumb.
		allowedTypes := make([]interface{}, len(seed.EntityTypes))
		for i, t := range seed.EntityTypes {
			allowedTypes[i] = t
		}
		typ := seed.Entity(req.FormValue("type"))
		if !checkEnum(typ, allowedTypes...) {
			return nil, httpErrorf(http.StatusBadRequest, "bad type %q", string(typ))
		}
		format := text.Format(req.FormValue("format"))
		if !checkEnum(format, text.CSV, text.KeyVal, text.TSV) {
			return nil, httpErrorf(http.StatusBadRequest, "bad format %q", string(format))
		}
		var err error
		if edits, err = text.Read(ctx, strings.NewReader(req.FormValue("input")),
			format, typ, req.Form["field"], req.Form["set"], db,
			text.MaxEdits(maxEdits), text.MaxFields(maxFields)); err != nil {
			return nil, &httpError{
				code: http.StatusInternalServerError,
				msg:  fmt.Sprint("Failed getting edits: ", err),
				err:  err,
			}
		}

	default:
		return nil, httpErrorf(http.StatusBadRequest, "bad source %q", req.FormValue("source"))
	}
	return render.NewEditInfos(edits, serverURL)
}

// countryCodeRegexp is used to validate the optional "country" query parameter.
var countryCodeRegexp = regexp.MustCompile(`^(?:|[A-Z][A-Z])$`)

// clientAddr returns the client's address (which may be either "ip" or "ip:port").
func clientAddr(req *http.Request) string {
	// When running under App Engine, connections come from 127.0.0.1,
	// so get the client IP from the X-Forwarded-For header.
	if os.Getenv("GAE_ENV") != "" {
		if hdr := req.Header.Get("X-Forwarded-For"); hdr != "" {
			// X-Forwarded-For: <client>, <proxy1>, <proxy2>
			return strings.SplitN(hdr, ", ", 2)[0]
		}
	}
	return req.RemoteAddr
}

// checkEnum returns true if input appears in valid.
func checkEnum(input interface{}, valid ...interface{}) bool {
	for _, v := range valid {
		if input == v {
			return true
		}
	}
	return false
}

//go:embed favicon.ico
var faviconData []byte
