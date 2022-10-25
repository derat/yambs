// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
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
	if err := writePage(tf, edits); err != nil {
		return err
	}
	return browser.OpenFile(tf.Name())
}

// writePage writes an HTML page containing the supplied edits to w.
func writePage(w io.Writer, edits []seed.Edit) error {
	tmpl, err := template.New("").Parse(strings.TrimLeft(pageTmpl, "\n"))
	if err != nil {
		return err
	}
	type editInfo struct {
		Desc   string
		URL    template.URL
		Method string
		Params map[string]string
	}
	infos := make([]editInfo, len(edits))
	for i, ed := range edits {
		info := editInfo{
			Desc:   ed.Description(),
			URL:    template.URL(ed.URL()),
			Method: "post",
		}
		if ed.CanGet() {
			info.Method = "get"
		}
		params := ed.Params()
		info.Params = make(map[string]string, len(params))
		for k := range params {
			info.Params[k] = params.Get(k)
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
            <form action="{{.URL}}" method="{{.Method}}" target="_blank">
              {{- range $k, $v := .Params}}
              <input type="hidden" name="{{$k}}" value="{{$v}}" />
              {{- end}}
            </form>
            <a>{{.Desc}}</a>
          </td>
        </tr>
        {{- end}}
      </tbody>
    </table>
    <div id="button-container">
      <button id="open-all-button">Open all</button>
      <button id="open-selected-button">Open selected</button>
    </div>
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
    const forms = rows.map((r) => r.querySelector('form'));
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
      // If there's a single edit, just open it.
      if (forms.length === 1) {
        forms[0].target = '_self';
        forms[0].submit();
      }

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

      rows
        .map((r) => r.querySelector('a'))
        .forEach((a, idx) => {
          a.addEventListener('click', () => {
            forms[idx].submit();
          });
        });

      headerCheckbox.addEventListener('click', () => {
        lastClickIndex = -1;
        const empty = getNumSelected() === 0;
        checkboxes.forEach((cb) => (cb.checked = empty));
        updateState();
      });

      openSelectedButton.addEventListener('click', () => {
        forms
          .filter((_, i) => checkboxes[i].checked)
          .forEach((f) => f.submit());
        advanceSelection();
      });

      openAllButton.addEventListener('click', () => {
        for (const f of forms) f.submit();
      });
    })();
  </script>
</html>
`
