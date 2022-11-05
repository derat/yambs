// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/bandcamp"
	"github.com/derat/yambs/sources/text"
)

const (
	// TODO: Figure out reasonable values for these.
	serverMaxReqBytes = 128 * 1024
	serverReqTimeout  = 10 * time.Second
	serverMaxEdits    = 200
	serverMaxFields   = 1000
)

// runServer starts an HTTP server at addr to serve a page that lets users
// generate seeded edits. This method never returns (unless serving fails).
func runServer(ctx context.Context, addr string) error {
	// Just generate the page once.
	var b bytes.Buffer
	if err := writePage(&b, nil); err != nil {
		return err
	}
	page := b.Bytes()

	db := db.NewDB()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var reqSize string
		if req.ContentLength > 0 {
			reqSize = fmt.Sprintf(" (%d)", req.ContentLength)
		}
		log.Printf("%v %s%s from %s", req.Method, req.URL, reqSize, req.RemoteAddr)

		switch req.URL.Path {
		case "/":
			if req.Method != http.MethodGet {
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				break
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if _, err := w.Write(page); err != nil {
				log.Print("Failed writing page: ", err)
			}

		case "/edits":
			ctx, cancel := context.WithTimeout(ctx, serverReqTimeout)
			defer cancel()
			infos, err := getEditsForRequest(ctx, w, req, db)
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
			log.Printf("Sending %d edit(s) to %s", len(infos), req.RemoteAddr)
			if err := json.NewEncoder(w).Encode(infos); err != nil {
				log.Printf("Failed sending edits to %s: %v", req.RemoteAddr, err)
			}

		default:
			http.NotFound(w, req)
		}
	})

	log.Println("Listening on", addr)
	srv := http.Server{Addr: addr}
	return srv.ListenAndServe()
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

// getEditsForRequest generates editInfo objects in response to a /edits request to the server.
func getEditsForRequest(ctx context.Context, w http.ResponseWriter, req *http.Request, db *db.DB) (
	[]*editInfo, error) {
	if req.Method != http.MethodPost {
		return nil, httpErrorf(http.StatusMethodNotAllowed, "bad method %q", req.Method)
	}

	// TODO: Check referrer?

	req.Body = http.MaxBytesReader(w, req.Body, serverMaxReqBytes)
	if err := req.ParseMultipartForm(serverMaxReqBytes); err != nil {
		return nil, &httpError{http.StatusBadRequest, "", err}
	}

	var edits []seed.Edit
	switch req.FormValue("source") {
	case "bandcamp":
		if u, err := bandcamp.CleanURL(req.FormValue("url")); err != nil {
			return nil, &httpError{
				code: http.StatusBadRequest,
				msg:  fmt.Sprint("Server only accepts bandcamp.com album URLs: ", err),
				err:  fmt.Errorf("%q: %v", req.FormValue("bandcampUrl"), err),
			}
		} else if edits, err = bandcamp.Fetch(ctx, u); err != nil {
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
			text.MaxEdits(serverMaxEdits), text.MaxFields(serverMaxFields)); err != nil {
			return nil, &httpError{
				code: http.StatusInternalServerError,
				msg:  fmt.Sprint("Failed getting edits: ", err),
				err:  err,
			}
		}

	default:
		return nil, httpErrorf(http.StatusBadRequest, "bad source %q", req.FormValue("source"))
	}
	return newEditInfos(edits)
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
