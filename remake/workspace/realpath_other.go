// Copyright 2026 Hypit.AI
// Translated from Node fs.realpath, used by @hypit/workspace.
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

//go:build !windows

package workspace

import "path/filepath"

func realPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}
