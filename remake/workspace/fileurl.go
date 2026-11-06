// Copyright 2026 Hypit.AI
// Translated from Node pathToFileURL, used by @hypit/workspace.
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package workspace

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

func pathToFileURL(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(absolute, `\\`) {
			rest := strings.TrimPrefix(absolute, `\\`)
			host, tail, _ := strings.Cut(rest, `\`)
			return (&url.URL{Scheme: "file", Host: host, Path: "/" + filepath.ToSlash(tail)}).String()
		}
		return (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(absolute)}).String()
	}
	return (&url.URL{Scheme: "file", Path: absolute}).String()
}
