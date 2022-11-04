// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/bandcamp"
)

const (
	// TODO: Figure out reasonable values for these.
	serverMaxReqBytes = 128 * 1024
	serverReqTimeout  = 10 * time.Second
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

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var reqSize string
		if req.ContentLength > 0 {
			reqSize = fmt.Sprintf(" (%d)", req.ContentLength)
		}
		log.Printf("%v %q%s from %s", req.Method, req.URL, reqSize, req.RemoteAddr)

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
			infos, err := getEditsForRequest(ctx, w, req)
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

// getEditsForRequest generates editInfo objects in response to a /edits request to the server.
func getEditsForRequest(ctx context.Context, w http.ResponseWriter, req *http.Request) ([]*editInfo, error) {
	if req.Method != http.MethodPost {
		return nil, &httpError{
			code: http.StatusMethodNotAllowed,
			err:  fmt.Errorf("bad method %q", req.Method),
		}
	}

	// TODO: Check referrer?

	req.Body = http.MaxBytesReader(w, req.Body, serverMaxReqBytes)
	if err := req.ParseMultipartForm(serverMaxReqBytes); err != nil {
		return nil, &httpError{http.StatusBadRequest, "", err}
	}

	var edits []seed.Edit
	switch req.FormValue("source") {
	case "bandcamp":
		if u, err := bandcamp.CleanURL(req.FormValue("bandcampUrl")); err != nil {
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
		// TODO: Implement this.
		return nil, errors.New("unimplemented")
	default:
		return nil, &httpError{
			code: http.StatusBadRequest,
			err:  fmt.Errorf("bad source %q", req.FormValue("source")),
		}
	}
	return newEditInfos(edits)
}
