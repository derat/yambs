// Copyright 2022 Daniel Erat.
// All rights reserved.

//go:build nogcp

package seed

import "context"

// detectLangNetwork is a no-op implementation used in builds with the nogcp tag to
// avoid a bulky (7 MB!) dependency on the GCP libraries.
func detectLangNetwork(ctx context.Context, titles []string) (lang, script string, err error) {
	return "", "", nil
}
