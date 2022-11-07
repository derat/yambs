// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"context"
	"net/http"
	"net/url"

	"github.com/derat/yambs/db"
)

// Info wraps a URL containing extra information (e.g. cover art); it's not an actual edit.
type Info struct {
	desc, url string
	params    url.Values
}

func NewInfo(desc, rawURL string) (*Info, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	params := u.Query()
	u.RawQuery = ""
	return &Info{desc: desc, url: u.String(), params: params}, nil
}

func (in *Info) Type() Type                                  { return InfoType }
func (in *Info) Description() string                         { return in.desc }
func (in *Info) URL() string                                 { return in.url }
func (in *Info) Params() url.Values                          { return in.params }
func (in *Info) Method() string                              { return http.MethodGet }
func (in *Info) Finish(ctx context.Context, db *db.DB) error { return nil }
