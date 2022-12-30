// Copyright 2022 Daniel Erat.
// All rights reserved.

//go:build !nogcp

package seed

import (
	"context"
	"errors"
	"log"
	"strings"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
)

// detectLangNetwork attempts to detect the language and script of a release with the supplied
// release and track titles using the Google Cloud Translation API. The returned values are
// appropriate for the Language and Script fields in seed.Release: lang is an ISO 639-3 code (e.g.
// "eng") and script is an ISO 15924 code (e.g. "Latn"). Empty strings are returned if the language
// and/or script can't be detected.
func detectLangNetwork(ctx context.Context, titles []string) (lang, script string, err error) {
	client, err := translate.NewClient(ctx)
	if err != nil {
		return "", "", err
	}
	defer client.Close()

	// From https://wiki.musicbrainz.org/Style/Release:
	//
	//   The language attribute should be used for the language used for the release title and track
	//   titles. It should not be used for the language the lyrics are written in, nor for the
	//   language used for other extra information on the cover.
	//
	//   If several languages are used in the titles, choose the most common language. For releases
	//   where there's an equal mix of two or more languages and hence no obvious answer, 'Multiple
	//   Languages' may be the best choice. But remember that it is quite common for languages to
	//   borrow words and phrases, and so "Je ne sais quoi" in an English title does not make
	//   something multiple languages, nor do a few English words in a foreign language title. (Some
	//   languages borrow quite extensively, and especially for Japanese, unless most of the titles
	//   are in other languages, Japanese is probably the best choice.)
	//
	//   If several scripts are used in the titles, choose the most common script. For releases
	//   where there's an equal mix of two or more scripts and hence no obvious answer, 'Multiple
	//   Scripts' may be the best choice. However, as the Latin script is common in many languages
	//   that primarily use another script, Latin should only be chosen if there are no more than
	//   one or two titles (or a few characters) in other scripts. For example, a Japanese release
	//   with a mix of English and Japanese titles should normally use 'Japanese' as the script."
	//
	// The reasonable thing to do would be to send the release and track titles as separate strings
	// and then try to fit the results to the MB guidelines. Unfortunately, the Translate API seems
	// to be junk. Even though the response can hold multiple Detection objects (i.e. hypotheses)
	// for each input string, I'm only getting one for each. This is problematic when a string is
	// valid in multiple languages (which seems to be common for e.g. Norwegian/Danish/Swedish),
	// since GCP will only return one language for each, and not necessarily the same language for
	// all tracks in an album. Sending all the titles in a single string seems to result in a better
	// guess in the single-language case, but it makes it hard to identify mixed-language/-script
	// cases.

	full := strings.Join(titles, "\n")
	log.Printf("Detecting language for %q (%d bytes)", truncate(full, 40, true), len(full))
	res, err := client.DetectLanguage(ctx, []string{full})
	if err != nil {
		return "", "", err
	} else if len(res) != 1 || len(res[0]) == 0 {
		return "", "", errors.New("empty result")
	}

	det := res[0][0]
	tag := det.Language
	if det.Confidence < minLangConfidence {
		log.Printf("Discarding language %q with confidence %0.2f", tag, det.Confidence)
		return "", "", nil
	}

	if base, conf := tag.Base(); conf >= language.Low {
		lang = base.ISO3()
	}
	if scr, conf := tag.Script(); conf >= language.Low && script != "Zzzz" {
		script = scr.String()
	}

	// TODO: Inspect individual runes to detect cases where there are mixed scripts?

	log.Printf("Detected language %q and script %q with confidence %0.2f",
		lang, script, det.Confidence)
	return lang, script, nil
}

const minLangConfidence = 0.5
