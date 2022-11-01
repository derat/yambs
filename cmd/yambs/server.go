// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bytes"
	"log"
	"net/http"
)

// runServer starts an HTTP server at addr with a form that lets users
// generate seeded edits. This method never returns (unless serving fails).
func runServer(addr string) error {
	// Just generate the page once.
	var b bytes.Buffer
	if err := writePage(&b, nil); err != nil {
		return err
	}
	form := b.Bytes()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(form)
	})

	http.HandleFunc("/edits", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement this. Also check method, referrer, etc.
		http.Error(w, "unimplemented", http.StatusInternalServerError)
	})

	log.Println("Listening on", addr)
	srv := http.Server{Addr: addr}
	return srv.ListenAndServe()
}
