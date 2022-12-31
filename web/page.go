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
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			log.Print("Got redirect to ", req.URL)
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status %v: %v", res.StatusCode, res.Status)
	}

	var size string
	if res.ContentLength >= 0 {
		size = fmt.Sprintf(" (%d bytes)", res.ContentLength)
	}
	log.Print("Parsing ", url, size)
	root, err := html.Parse(res.Body)
	if err != nil {
		return nil, err
	}
	return &Page{root}, nil
}

// Query calls QueryNode using p.Root.
func (p *Page) Query(query string) QueryResult { return QueryNode(p.Root, query) }

// QueryNode returns the first node matched by the supplied CSS selector.
// The returned result has a non-nil Err field if no node was matched.
func QueryNode(root *html.Node, query string) QueryResult {
	sel, err := cascadia.Parse(query)
	if err != nil {
		return QueryResult{nil, err}
	}
	node := cascadia.Query(root, sel)
	if node == nil {
		return QueryResult{nil, errors.New("node not found")}
	}
	return QueryResult{node, nil}
}

// QueryResult contains the result of a call to Query or QueryNode.
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

// Text recursively concatenates the contents of all child text nodes.
// TODO: Add more control over formatting, e.g. trimming.
func (res QueryResult) Text(addSpaces bool) (string, error) {
	if res.Err != nil {
		return "", res.Err
	}
	return GetText(res.Node, addSpaces), nil
}

// QueryAll returns all nodes matched by the supplied CSS selector.
// Unlike Query/QueryNode, an error is not returned if no nodes are matched.
func (p *Page) QueryAll(query string) QueryAllResult {
	sel, err := cascadia.Parse(query)
	if err != nil {
		return QueryAllResult{nil, err}
	}
	return QueryAllResult{cascadia.QueryAll(p.Root, sel), nil}
}

// QueryAllResult contains the result of a call to QueryAll.
type QueryAllResult struct {
	Nodes []*html.Node
	Err   error
}

// Text returns the contents of all text nodes under res.Nodes.
func (res QueryAllResult) Text(addSpaces bool) ([]string, error) {
	if res.Err != nil || len(res.Nodes) == 0 {
		return nil, res.Err
	}
	text := make([]string, len(res.Nodes))
	for i, n := range res.Nodes {
		text[i] = GetText(n, addSpaces)
	}
	return text, nil
}

// GetText concatenates all text content in and under n.
func GetText(n *html.Node, addSpaces bool) string {
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
		s := GetText(c, addSpaces)
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

// SetUserAgent sets a value for the "User-Agent" header to be sent
// in all future HTTP requests.
func SetUserAgent(ua string) { userAgent = ua }

var userAgent string
