// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package web interacts with web pages.
package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// Page represents a parsed HTML page.
type Page struct {
	Root *html.Node
}

// FetchPage fetches and parses the HTML page at the supplied URL.
func FetchPage(ctx context.Context, url string) (*Page, error) {
	log.Print("Fetching ", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %v: %v", resp.StatusCode, resp.Status)
	}

	size := "unknown"
	if resp.ContentLength >= 0 {
		size = fmt.Sprint(resp.ContentLength)
	}
	log.Printf("Parsing %s-byte response from %v", size, url)
	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Page{root}, nil
}

// Query returns the first node matched by the supplied CSS selector.
// The returned result has a non-nil Err field if no node was matched.
func (p *Page) Query(query string) QueryResult {
	sel, err := cascadia.Parse(query)
	if err != nil {
		return QueryResult{nil, err}
	}
	node := cascadia.Query(p.Root, sel)
	if node == nil {
		return QueryResult{nil, errors.New("node not found")}
	}
	return QueryResult{node, nil}
}

// QueryResult contains the result of a call to Query.
type QueryResult struct {
	Node *html.Node
	Err  error
}

// Attr returns the first occurrence of the named attribute.
// An error is returned if the attribute isn't present.
func (res QueryResult) Attr(attr string) (string, error) {
	if res.Err != nil {
		return "", res.Err
	}
	for _, a := range res.Node.Attr {
		if a.Key == attr {
			return a.Val, nil
		}
	}
	return "", errors.New("attribute not found")
}

// Text recursively concatenates the contens of all child text nodes.
// TODO: Add more control over formatting, e.g. trimming.
func (res QueryResult) Text(addSpaces bool) (string, error) {
	if res.Err != nil {
		return "", res.Err
	}
	return getText(res.Node, addSpaces), nil
}

// getText concatenates all text content in and under n.
func getText(n *html.Node, addSpaces bool) string {
	if n == nil {
		return ""
	}
	var text string
	if n.Type == html.TextNode {
		text = n.Data
		if addSpaces {
			text = strings.TrimSpace(text)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s := getText(c, addSpaces)
		if addSpaces {
			if s = strings.TrimSpace(s); s != "" {
				if text != "" {
					text += " "
				}
				text += s
			}
		} else {
			text += s
		}
	}
	return text
}
