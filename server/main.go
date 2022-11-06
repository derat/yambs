// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package main implements an App Engine server for generating seeded edits.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/page"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/bandcamp"
	"github.com/derat/yambs/sources/text"
	"github.com/derat/yambs/web"

	"google.golang.org/appengine/v2"
	"google.golang.org/appengine/v2/urlfetch"
)

const (
	// TODO: Figure out reasonable values for these.
	maxReqBytes  = 128 * 1024
	editsTimeout = 10 * time.Second
	editsDelay   = 3 * time.Second
	maxEdits     = 200
	maxFields    = 1000
)

func main() {
	// Just generate the page once.
	var b bytes.Buffer
	if err := page.Write(&b, nil); err != nil {
		panic(fmt.Sprint("Failed generating page: ", err))
	}
	form := b.Bytes()

	// TODO: Consider switching to App Engine logging so messages are associated with requests.
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

	db := db.NewDB()
	rm := newRateMap()

	http.HandleFunc("/edits", func(w http.ResponseWriter, req *http.Request) {
		ctx := appengine.NewContext(req)
		ctx, cancel := context.WithTimeout(ctx, editsTimeout)
		defer cancel()

		// Attach the urlfetch client to the context so other packages can use it.
		ctx = web.WithClient(ctx, urlfetch.Client(ctx))

		infos, err := getEditsForRequest(ctx, w, req, rm, db)
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
			log.Printf("Sending %d to %s: %v", code, req.RemoteAddr, err)
			http.Error(w, msg, code)
			return
		}
		log.Printf("Returning %d edit(s) to %s", len(infos), req.RemoteAddr)
		if err := json.NewEncoder(w).Encode(infos); err != nil {
			log.Printf("Failed sending edits to %s: %v", req.RemoteAddr, err)
		}
	})

	appengine.Main()
}

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
func httpErrorf(code int, format string, args ...any) *httpError {
	return &httpError{code: code, err: fmt.Errorf(format, args...)}
}

// getEditsForRequest generates page.EditInfo objects in response to a /edits request to the server.
func getEditsForRequest(ctx context.Context, w http.ResponseWriter, req *http.Request,
	rm *rateMap, db *db.DB) (
	[]*page.EditInfo, error) {
	if req.Method != http.MethodPost {
		return nil, httpErrorf(http.StatusMethodNotAllowed, "bad method %q", req.Method)
	}

	// TODO: Check referrer?

	if ip, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if !rm.update(ip, time.Now(), editsDelay) {
			return nil, &httpError{
				code: http.StatusTooManyRequests,
				msg:  "Please wait a few seconds and try again",
				err:  errors.New("too many requests"),
			}
		}
	}

	req.Body = http.MaxBytesReader(w, req.Body, maxReqBytes)
	if err := req.ParseMultipartForm(maxReqBytes); err != nil {
		return nil, &httpError{http.StatusBadRequest, "", err}
	}

	src := req.FormValue("source")
	log.Printf("Handling %v-byte %q request from %s", req.ContentLength, src, req.RemoteAddr)

	var edits []seed.Edit
	switch src {
	case "bandcamp":
		if url, err := bandcamp.CleanURL(req.FormValue("url")); err != nil {
			return nil, &httpError{
				code: http.StatusBadRequest,
				msg:  fmt.Sprint("Server only accepts bandcamp.com album URLs: ", err),
				err:  fmt.Errorf("%q: %v", req.FormValue("bandcampUrl"), err),
			}
		} else if edits, err = bandcamp.Fetch(ctx, url); err != nil {
			return nil, &httpError{
				code: http.StatusInternalServerError,
				msg:  fmt.Sprint("Failed getting edits: ", err),
				err:  err,
			}
		}

	case "text":
		typ := seed.Type(req.FormValue("type"))
		if !checkEnum(typ, seed.RecordingType, seed.ReleaseType) {
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
	return page.NewEditInfos(edits)
}

// checkEnum returns true if input appears in valid.
func checkEnum[T comparable](input T, valid ...T) bool {
	for _, v := range valid {
		if input == v {
			return true
		}
	}
	return false
}
