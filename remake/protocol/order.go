// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

import (
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// localeCompare is String.prototype.localeCompare with no locale argument:
// the process default locale. Canonical object key order uses it, as
// @hypit/protocol canonicalize does.
//
// golang.org/x/text/collate matches Node on the ASCII, digit, and accent
// cases covered by the upstream tests. It does not yet match Node ICU on
// Han-versus-Latin primary order. That gap is recorded on the protocol
// catalog entry and is not papered over.

var (
	collOnce sync.Once
	coll     *collate.Collator
)

func localeCompare(left, right string) int {
	collOnce.Do(func() {
		coll = collate.New(defaultLanguage())
	})
	return coll.CompareString(left, right)
}

// LocaleCompare is String.prototype.localeCompare with no locale argument.
func LocaleCompare(left, right string) int {
	return localeCompare(left, right)
}

func sortKeys(keys []string) {
	for i := 1; i < len(keys); i++ {
		j := i
		for j > 0 && localeCompare(keys[j-1], keys[j]) > 0 {
			keys[j-1], keys[j] = keys[j], keys[j-1]
			j--
		}
	}
}

func parseLanguage(name string) language.Tag {
	tag, err := language.Parse(name)
	if err != nil {
		return language.Und
	}
	return tag
}
