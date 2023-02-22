// Copyright 2023 Daniel Erat.
// All rights reserved.

package mbdb

// edits holds information about edits needed to transform one string into another.
type edits struct{ ins, dels, subs int }

// dist returns the Levenshtein distance (i.e. total number of edits).
func (e edits) dist() int { return e.ins + e.dels + e.subs }

// levenshtein computes the Levenshtein distance between a and b using the Wagner–Fischer algorithm.
// It's based on pseudocode from https://en.wikipedia.org/wiki/Wagner%E2%80%93Fischer_algorithm.
func levenshtein(as, bs string) edits {
	a, b := []rune(as), []rune(bs)

	// For all i and j, es[i][j] will hold the Levenshtein distance
	// between the first i runes of a and the first j runes of b.
	es := make([][]edits, len(a)+1)
	for i := range es {
		es[i] = make([]edits, len(b)+1)
	}

	// Source prefixes can be transformed into empty sequences by dropping all runes.
	for i := 1; i <= len(a); i++ {
		es[i][0] = edits{dels: i}
	}
	// Target prefixes can be reached from empty source prefix by inserting every rune.
	for j := 1; j <= len(b); j++ {
		es[0][j] = edits{ins: j}
	}
	for j := 1; j <= len(b); j++ {
		for i := 1; i <= len(a); i++ {
			// Deletion.
			e := es[i-1][j]
			e.dels++

			// Insertion.
			if es[i][j-1].dist()+1 < e.dist() {
				e = es[i][j-1]
				e.ins++
			}

			// Substitution.
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			if es[i-1][j-1].dist()+cost < e.dist() {
				e = es[i-1][j-1]
				e.subs += cost
			}

			es[i][j] = e
		}
	}

	return es[len(a)][len(b)]
}
