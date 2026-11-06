// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

//go:build windows

package protocol

import (
	"golang.org/x/sys/windows"
	"golang.org/x/text/language"
)

func defaultLanguage() language.Tag {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil || len(langs) == 0 {
		return language.Und
	}
	return parseLanguage(langs[0])
}
