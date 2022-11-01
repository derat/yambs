// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/derat/yambs/seed"
	"github.com/pkg/browser"
)

// openPage writes an HTML page containing edits to a temporary file
// and opens it in a browser.
func openPage(edits []seed.Edit) error {
	tf, err := ioutil.TempFile("",
		fmt.Sprintf("yambs-%s-*.html", time.Now().Format("20060102-150405")))
	if err != nil {
		return err
	}
	log.Print("Writing page to ", tf.Name())
	if err := writePage(tf, edits); err != nil {
		return err
	}
	return browser.OpenFile(tf.Name())
}

// servePage starts a local HTTP server at addr and opens an HTML page containing
// edits in a browser. This is fairly complicated but it can be convenient if the
// browser doesn't have direct filesystem access (e.g. the server is running in a
// Chrome OS VM), and I think that a fixed host:port may be needed in order to
// permanently tell Chrome to avoid blocking popups.
func servePage(ctx context.Context, addr string, edits []seed.Edit) error {
	var b bytes.Buffer
	if err := writePage(&b, edits); err != nil {
		return err
	}

	// Bind to the port first.
	ls, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ls.Close()

	// Get the real address in case the port wasn't specified and launch the browser.
	url := fmt.Sprintf("http://%s/", ls.Addr().String())
	log.Print("Listening at ", url)
	if err := browser.OpenURL(url); err != nil {
		return err
	}

	// Report that we're done after we've served the page a single time.
	done := make(chan struct{})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write(b.Bytes())
			close(done)
		} else {
			http.NotFound(w, r)
		}
	})

	// Run the server in a goroutine.
	var srv http.Server
	start := make(chan error)
	go func() { start <- srv.Serve(ls) }()
	for {
		select {
		case err := <-start:
			// Serve immediately returns ErrServerClosed after Shutdown is called,
			// but we also need to handle earlier errors.
			if err != nil && err != http.ErrServerClosed {
				return err
			}
		case <-done:
			// Shutdown blocks until all connections are closed.
			log.Print("Shutting down after serving page")
			return srv.Shutdown(ctx)
		}
	}
}

// writePage writes an HTML page containing the supplied edits to w.
func writePage(w io.Writer, edits []seed.Edit) error {
	tmpl, err := template.New("").Parse(pageTmpl)
	if err != nil {
		return err
	}
	type param struct{ Name, Value string }
	type editInfo struct {
		Desc   string
		URL    string  // includes params iff GET
		Params []param // includes params iff POST
	}
	infos := make([]editInfo, len(edits))
	for i, ed := range edits {
		info := editInfo{Desc: ed.Description()}

		// Use a different approach depending on whether the edit requires a POST or not.
		if ed.CanGet() {
			// If we can use GET, construct a URL including any parameters since <form method="GET">
			// adds an annoying question mark even if there aren't any parameters.
			u, err := url.Parse(ed.URL())
			if err != nil {
				return err
			}
			u.RawQuery = ed.Params().Encode()
			info.URL = u.String()
		} else {
			// If we need to use POST, keep the parameters separate since <form> annoyingly
			// clears the URL's query string.
			info.URL = ed.URL()
			for name, vals := range ed.Params() {
				for _, val := range vals {
					info.Params = append(info.Params, param{name, val})
				}
			}
		}

		infos[i] = info
	}
	return tmpl.Execute(w, struct{ Edits []editInfo }{infos})
}

//go:embed page.tmpl
var pageTmpl string
