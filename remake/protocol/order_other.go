// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

//go:build !windows

package protocol

import (
	"os"
	"strings"

	"golang.org/x/text/language"
)

func defaultLanguage() language.Tag {
	for _, key := range []string{"LC_ALL", "LC_COLLATE", "LANG"} {
		value := os.Getenv(key)
		if value == "" || value == "C" || value == "POSIX" {
			continue
		}
		if i := strings.IndexByte(value, '.'); i >= 0 {
			value = value[:i]
		}
		return parseLanguage(strings.ReplaceAll(value, "_", "-"))
	}
	return language.Und
}
