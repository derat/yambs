// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package render generates HTML pages listing edits.
package render

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
	"os"
	"sort"
	"strings"
	"time"

	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/text"
	"github.com/pkg/browser"
)

const defaultServerURL = "https://musicbrainz.org"

// OpenFile writes an HTML page containing edits to a temporary file
// and opens it in a browser.
func OpenFile(edits []seed.Edit, opts ...Option) error {
	tf, err := ioutil.TempFile("",
		fmt.Sprintf("yambs-%s-*.html", time.Now().Format("20060102-150405")))
	if err != nil {
		return err
	}
	log.Print("Writing page to ", tf.Name())
	if err := Write(tf, edits, opts...); err != nil {
		return err
	}
	return browser.OpenFile(tf.Name())
}

// OpenHTTP starts a local HTTP server at addr and opens an HTML page containing
// edits in a browser. This is fairly complicated but it can be convenient if the
// browser doesn't have direct filesystem access (e.g. the server is running in a
// Chrome OS VM), and I think that a fixed host:port may be needed in order to
// permanently tell Chrome to avoid blocking popups.
func OpenHTTP(ctx context.Context, addr string, edits []seed.Edit, opts ...Option) error {
	// Bind to the port first.
	ls, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ls.Close()

	// Get the real address in case the port wasn't specified.
	listenURL := fmt.Sprintf("http://%s/", ls.Addr().String())
	log.Print("Listening at ", listenURL)

	// If there are seed.Info edits with file:// URLs, rewrite them to point at the web server.
	filePaths := make(map[string]struct{})
	for i, ed := range edits {
		us := ed.URL("") // don't need server
		if _, ok := ed.(*seed.Info); ok && strings.HasPrefix(us, "file://") {
			u, err := url.Parse(us)
			if err != nil {
				return err
			}
			u.Scheme = "http"
			u.Host = ls.Addr().String()
			if edits[i], err = seed.NewInfo(ed.Description(), u.String()); err != nil {
				return err
			}
			filePaths[u.Path] = struct{}{}
		}
	}

	var b bytes.Buffer
	if err := Write(&b, edits, opts...); err != nil {
		return err
	}

	// Launch the browser before we start handling requests.
	if err := browser.OpenURL(listenURL); err != nil {
		return err
	}

	// Report that we're done after we've served the page a single time,
	// but only if there aren't any rewritten file:// URLs that we need to serve.
	done := make(chan struct{})
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write(b.Bytes())
			if len(filePaths) == 0 {
				close(done)
			}
		} else if _, ok := filePaths[req.URL.Path]; ok {
			f, err := os.Open(req.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()
			// TODO: Stat the file so we can pass modtime?
			http.ServeContent(w, req, f.Name(), time.Time{}, f)
		} else {
			http.NotFound(w, req)
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

// Write writes an HTML page containing the supplied edits to w.
func Write(w io.Writer, edits []seed.Edit, opts ...Option) error {
	cfg := getConfig(opts...)
	tmpl, err := template.New("").Parse(pageTmpl)
	if err != nil {
		return err
	}
	editInfos, err := NewEditInfos(edits, cfg.serverURL)
	if err != nil {
		return err
	}
	typeInfos := make([]typeInfo, len(seed.EntityTypes))
	for i, t := range seed.EntityTypes {
		typeInfos[i] = newTypeInfo(t)
	}
	return tmpl.Execute(w, struct {
		IconSVG  template.HTML
		Version  string
		TypeInfo []typeInfo
		Edits    []*EditInfo
	}{
		IconSVG:  template.HTML(iconSVG),
		Version:  cfg.version,
		TypeInfo: typeInfos,
		Edits:    editInfos,
	})
}

// Option can be passed to configure the page.
type Option func(*config)

// ServerURL sets the base MusicBrainz server URL,
// e.g. "https://musicbrainz.org" or "https://test.musicbrainz.org".
func ServerURL(u string) Option { return func(cfg *config) { cfg.serverURL = u } }

// Version sets an optional yambs version to include in the page.
func Version(v string) Option { return func(cfg *config) { cfg.version = v } }

type config struct {
	version   string // yambs version
	serverURL string // base MusicBrainz server URL
}

func getConfig(opts ...Option) config {
	cfg := config{serverURL: defaultServerURL}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

//go:embed page.tmpl
var pageTmpl string

//go:embed icon.svg
var iconSVG string

// EditInfo is a version of seed.Edit used in HTML pages.
// It's used both for passing edits to pageTmpl in CLI mode
// and for returning edits via XHRs when running in server mode.
type EditInfo struct {
	Desc   string      `json:"desc"`
	URL    string      `json:"url"`    // includes params iff GET
	Params []paramInfo `json:"params"` // includes params iff POST
}

// paramInfo describes a POST query parameter.
type paramInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NewEditInfo converts a seed.Edit into an EditInfo struct.
func NewEditInfo(edit seed.Edit, serverURL string) (*EditInfo, error) {
	info := EditInfo{Desc: edit.Description()}

	// Use a different approach depending on whether the edit requires a POST or not.
	switch edit.Method() {
	case "GET":
		// If we can use GET, construct a URL including any parameters since <form method="GET">
		// adds an annoying question mark even if there aren't any parameters.
		u, err := url.Parse(edit.URL(serverURL))
		if err != nil {
			return nil, err
		}
		u.RawQuery = edit.Params().Encode()
		info.URL = u.String()
	case "POST":
		// If we need to use POST, keep the parameters separate since <form> annoyingly
		// clears the URL's query string.
		info.URL = edit.URL(serverURL)
		for name, vals := range edit.Params() {
			for _, val := range vals {
				info.Params = append(info.Params, paramInfo{Name: name, Value: val})
			}
		}
	default:
		return nil, fmt.Errorf("unsupported HTTP method %q", edit.Method())
	}

	return &info, nil
}

// NewEditInfos calls NewEditInfo for each of the supplied edits.
func NewEditInfos(edits []seed.Edit, serverURL string) ([]*EditInfo, error) {
	infos := make([]*EditInfo, len(edits))
	for i, edit := range edits {
		var err error
		if infos[i], err = NewEditInfo(edit, serverURL); err != nil {
			return nil, err
		}
	}
	return infos, nil
}

// typeInfo describes the fields that can be set for a given type.
// It's passed to pageTmpl.
type typeInfo struct {
	Type   string      // seed.Entity
	Name   string      // title case human-readable name
	Fields []fieldInfo // fields that can be set for the type

	SetExample    string            // e.g. "field1=val\nfield2=val"
	FieldsExample string            // e.g. "field1,field2"
	InputExamples map[string]string // keyed by text.Format
}

// fieldInfo describes an individual field.
type fieldInfo struct {
	Name string
	Desc template.HTML
}

// newTypeInfo creates a typeInfo for typ.
func newTypeInfo(typ seed.Entity) typeInfo {
	var fields []fieldInfo
	for field, desc := range text.ListFields(typ, true /* html */) {
		fields = append(fields, fieldInfo{
			Name: field,
			Desc: template.HTML(desc),
		})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

	return typeInfo{
		Type:          string(typ),
		Name:          strings.Title(string(typ)),
		Fields:        fields,
		SetExample:    text.SetExample(typ),
		FieldsExample: text.FieldsExample(typ),
		InputExamples: map[string]string{
			string(text.CSV):    text.InputExample(typ, text.CSV),
			string(text.KeyVal): text.InputExample(typ, text.KeyVal),
			string(text.TSV):    text.InputExample(typ, text.TSV),
		},
	}
}
