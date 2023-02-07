// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"context"
	"net/http"
	"net/url"

	"github.com/derat/yambs/mbdb"
)

// Info wraps a URL containing extra information (e.g. cover art); it's not an actual edit.
type Info struct {
	desc   string // human-readable description
	url    string
	params url.Values
	abs    bool // if false, treat url as path under MB server
}

func NewInfo(desc, rawURL string) (*Info, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	params := u.Query()
	u.RawQuery = ""
	return &Info{
		desc:   desc,
		url:    u.String(),
		params: params,
		abs:    u.IsAbs(),
	}, nil
}

func (in *Info) Entity() Entity      { return InfoEntity }
func (in *Info) Description() string { return in.desc }
func (in *Info) URL(serverURL string) string {
	if in.abs {
		return in.url
	}
	return serverURL + in.url
}
func (in *Info) Params() url.Values                            { return in.params }
func (in *Info) Method() string                                { return http.MethodGet }
func (in *Info) Finish(ctx context.Context, db *mbdb.DB) error { return nil }
