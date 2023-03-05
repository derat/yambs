// Copyright 2023 Daniel Erat.
// All rights reserved.

package strutil

import (
	"bytes"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// https://go.dev/blog/normalization#performing-magic
var normalizer = transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)))

// Normalize normalizes characters using NFKD form.
// Unicode characters are decomposed (runes are broken into their components) and replaced for
// compatibility equivalence (characters that represent the same characters but have different
// visual representations, e.g. '9' and '⁹', are equal). Characters are also de-accented.
// TODO: Maybe go farther and e.g. replace smart quotes with dumb quotes?
func Normalize(orig string) string {
	b := make([]byte, len(orig))
	if _, _, err := normalizer.Transform(b, []byte(orig), true); err != nil {
		return orig
	}
	return string(bytes.TrimRight(b, "\x00"))
}
