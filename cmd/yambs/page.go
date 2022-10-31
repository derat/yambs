// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
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
	tmpl, err := template.New("").Parse(strings.TrimLeft(pageTmpl, "\n"))
	if err != nil {
		return err
	}
	type param struct{ Name, Value string }
	type editInfo struct {
		Desc   string
		URL    template.URL // includes params iff GET
		Params []param      // includes params iff POST
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
			info.URL = template.URL(u.String())
		} else {
			// If we need to use POST, keep the parameters separate since <form> annoyingly
			// clears the URL's query string.
			info.URL = template.URL(ed.URL())
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

const pageTmpl = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta
      name="viewport"
      content="width=device-width, initial-scale=1, minimum-scale=1"
    />
    <title>yambs</title>
    <style>
      :root {
        --border-color: #ccc;
        --header-color: #eee;
        --link-color: #444;
        --margin: 8px;
      }
      body {
        font-family: Roboto, Arial, Helvetica, sans-serif;
        font-size: 14px;
      }
      h1 {
        font-size: 20px;
        margin-bottom: var(--margin);
      }
      h2 {
        font-size: 16px;
        margin-bottom: var(--margin);
      }

      #edits-table {
        border: solid 1px var(--border-color);
        border-collapse: collapse;
      }
      #edits-table th {
        background-color: var(--header-color);
      }
      #edits-table td,
      #edits-table th {
        border: solid 1px var(--border-color);
        text-align: left;
      }
      #edits-table th:nth-child(2),
      #edits-table td:nth-child(2) {
        /* This is hacky; table layout is a disaster. */
        max-width: calc(100vw - 100px);
        overflow: hidden;
        padding: 0 var(--margin);
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      #edits-table a {
        color: var(--link-color);
        cursor: pointer;
        text-decoration: underline;
      }
      #header-checkbox.partial {
        opacity: 0.4;
      }

      #button-container {
        display: flex;
        gap: var(--margin);
        margin-top: var(--margin);
      }

      #opening-overlay {
        align-items: center;
        backdrop-filter: blur(1px);
        background-color: #0002;
        display: none;
        font-size: 20px;
        height: 100vh;
        justify-content: center;
        left: 0;
        position: fixed;
        top: 0;
        width: 100vw;
        z-index: 1;
      }
      #opening-overlay.visible {
        display: flex;
      }
    </style>
  </head>
  <body>
    <h1>yambs</h1>
    <h2>Seeded MusicBrainz edits</h2>
    <table id="edits-table">
      <thead>
        <tr>
          <th><input id="header-checkbox" type="checkbox" /></th>
          <th>Edit</th>
        </tr>
      </thead>
      <tbody>
        {{range .Edits -}}
        <tr>
          <td><input type="checkbox" /></td>
          <td>
            {{if .Params -}}
            <form action="{{.URL}}" method="post" target="_blank">
              {{- range .Params}}
              <input type="hidden" name="{{.Name}}" value="{{.Value}}" />
              {{- end}}
            </form>
            {{end -}}
            <a {{if not .Params}}href="{{.URL}}" {{end}}target="_blank">
              {{.Desc}}
            </a>
          </td>
        </tr>
        {{- end}}
      </tbody>
    </table>
    <div id="button-container">
      <button id="open-all-button">Open all</button>
      <button id="open-selected-button">Open selected</button>
    </div>
    <div id="opening-overlay">Opening edit...</div>
  </body>
  <script>
    const $ = (id) => document.getElementById(id);

    const headerCheckbox = $('header-checkbox');
    const openAllButton = $('open-all-button');
    const openSelectedButton = $('open-selected-button');
    const rows = [...document.querySelectorAll('#edits-table tbody tr')];
    const checkboxes = rows.map((r) =>
      r.querySelector('input[type="checkbox"]')
    );
    const forms = rows.map((r) => r.querySelector('form')); // null for GETs
    const links = rows.map((r) => r.querySelector('a'));
    const defaultSelectionSize = 5;

    let lastClickIndex = -1;

    // Returns a 2-element array with the starting and ending index of the selection range.
    // If there isn't a single range, null is returned.
    function getSelectionRange() {
      let start = -1;
      let end = -1;
      for (const [idx, cb] of checkboxes.entries()) {
        if (!cb.checked) continue;
        if (start < 0) start = end = idx;
        else if (end === idx - 1) end = idx;
        else return null; // not a continuous range
      }
      return start < 0 ? null : [start, end];
    }

    // Returns the number of selected rows.
    const getNumSelected = () => checkboxes.filter((cb) => cb.checked).length;

    // Updates buttons and header checkbox state for the currently-checked checkboxes.
    function updateState() {
      openAllButton.disabled = rows.length === 0;

      // Update the "Open selected" button's text and disabled state.
      const range = getSelectionRange();
      openSelectedButton.innerText =
        range && range[1] < rows.length - 1
          ? 'Open selected and advance'
          : 'Open selected';
      openSelectedButton.disabled = getNumSelected() === 0;

      // Make the header checkbox checked if any rows are selected, and translucent if only some of
      // the rows are selected.
      const count = getNumSelected();
      headerCheckbox.checked = count > 0;
      const partial = count > 0 && count < checkboxes.length;
      if (partial) headerCheckbox.classList.add('partial');
      else headerCheckbox.classList.remove('partial');
    }

    // If a continuous range of n rows is selected, advances the selection to the next n rows.
    function advanceSelection() {
      const range = getSelectionRange();
      if (!range || range[1] === rows.length - 1) return;

      const start = range[1] + 1;
      const end = start + (range[1] - range[0] + 1);
      checkboxes.forEach(
        (cb, idx) => (cb.checked = idx >= start && idx <= end)
      );
      updateState();
    }

    // Initialize the page.
    (() => {
      for (
        let i = 0;
        i < Math.min(defaultSelectionSize, checkboxes.length);
        i++
      ) {
        checkboxes[i].checked = true;
      }
      updateState();

      checkboxes.forEach((cb, idx) =>
        cb.addEventListener('click', (e) => {
          // On shift-click, update the range starting at the last-clicked checkbox.
          if (e.shiftKey && lastClickIndex >= 0 && lastClickIndex != idx) {
            const checked = cb.checked;
            const start = Math.min(lastClickIndex, idx);
            const end = Math.max(lastClickIndex, idx);
            for (let i = start; i <= end; i++) checkboxes[i].checked = checked;
          }
          lastClickIndex = idx;
          updateState();
        })
      );

      links.forEach((a, idx) => {
        a.addEventListener('click', (ev) => {
          // If there's a form (because this edit requires a POST), submit it.
          // Otherwise, just let the link perform its default action.
          const f = forms[idx];
          if (f) {
            f.submit();
            ev.preventDefault();
          }
        });
      });

      headerCheckbox.addEventListener('click', () => {
        lastClickIndex = -1;
        const empty = getNumSelected() === 0;
        checkboxes.forEach((cb) => (cb.checked = empty));
        updateState();
      });

      openSelectedButton.addEventListener('click', () => {
        links.filter((_, i) => checkboxes[i].checked).forEach((a) => a.click());
        advanceSelection();
      });

      openAllButton.addEventListener('click', () => {
        for (const a of links) a.click();
      });

      // If there's a single edit, just open it in the current window.
      if (rows.length === 1) {
        if (forms[0]) forms[0].target = '_self';
        links[0].target = '_self';
        links[0].click();
        $('opening-overlay').classList.add('visible');
      }
    })();
  </script>
</html>
`
