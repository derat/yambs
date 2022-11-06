// Copyright 2022 Daniel Erat.
// All rights reserved.

package web

import (
	"context"
	"net/http"
)

type clientKey struct{}

// WithClient returns a context derived from ctx with the supplied *http.Client.
// This is a disgusting hack to make it easier to use App Engine urlfetch clients
// without needing to explicitly pass them around.
func WithClient(ctx context.Context, cl *http.Client) context.Context {
	return context.WithValue(ctx, clientKey{}, cl)
}

// GetClient returns the *http.Client previously attached to ctx via WithClient.
// If no client was attached, http.DefaultClient is returned.
func GetClient(ctx context.Context) *http.Client {
	if cl := ctx.Value(clientKey{}); cl != nil {
		return cl.(*http.Client)
	}
	return http.DefaultClient
}
