// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import "unicode"

// detectScriptLocal attempts to guess the main script used in the supplied titles.
// An empty string is returned if the script can't be detected.
func detectScriptLocal(titles []string) string {
	var known, total int
	counts := make(map[string]int)
	for _, title := range titles {
		for _, ch := range title {
			total++
			for rng, script := range rangeScripts {
				if unicode.Is(rng, ch) {
					counts[script]++
					known++
					break
				}
			}
		}
	}
	if total == 0 {
		return ""
	} else if pct := float32(known) / float32(total); pct < minKnownScriptPct {
		return ""
	}
	pcts := make(map[string]float32, len(counts))
	for s, n := range counts {
		pcts[s] = float32(n) / float32(known)
	}

	// Merge some languages with multiple scripts into single scripts.
	// Delete the old entries to make sure that we don't introduce ties.
	if pcts["Hira"] > 0 || pcts["Kana"] > 0 {
		pcts["Jpan"] = pcts["Hani"] + pcts["Hira"] + pcts["Kana"]
		delete(pcts, "Hira")
		delete(pcts, "Kana")
	}
	if pcts["Hang"] > 0 {
		pcts["Kore"] = pcts["Hani"] + pcts["Hang"]
		delete(pcts, "Hang")
	}

	var bestScript string
	var bestPct float32
	for s, p := range pcts {
		// Choose the earlier script to break ties (since Go map iteration order is undefined).
		if p > bestPct || p == bestPct && s < bestScript {
			bestScript = s
			bestPct = p
		}
	}

	if bestPct < minScriptPct {
		return ""
	}
	return bestScript
}

const (
	minKnownScriptPct = 0.5  // at least this many chars must be in a known script
	minScriptPct      = 0.75 // at least this many known chars must be in the dominant script
)

// Per https://www.britannica.com/list/the-worlds-5-most-commonly-used-writing-systems, the 5
// most-commonly-used writing systems are Latin, Chinese, Arabic, Devanagari, and Bengali.
// https://unicode.org/iso15924/iso15924-codes.html lists ISO 15924 codes.
// TODO: This is woefully incomplete and probably also incorrect.
var rangeScripts = map[*unicode.RangeTable]string{
	unicode.Arabic:     "Arab",
	unicode.Armenian:   "Armn",
	unicode.Bengali:    "Beng",
	unicode.Bopomofo:   "Bopo",
	unicode.Cyrillic:   "Cyrl",
	unicode.Devanagari: "Deva",
	unicode.Greek:      "Grek",
	// TODO: Is using "Hani" (Hanzi, Kanji, Hanja) here reasonable? There's also "Hans" (simplified)
	// and "Hant" (traditional), but I don't think I can easily determine which is being used based
	// on code tables.
	unicode.Han:      "Hani",
	unicode.Hangul:   "Hang",
	unicode.Hebrew:   "Hebr",
	unicode.Hiragana: "Hira",
	unicode.Katakana: "Kana",
	unicode.Latin:    "Latn",
	unicode.Thai:     "Thai",
}
