// Copyright 2026 Hypit.AI
// Translated from @hypit/atomic-file (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package atomicfile publishes a complete file without changing the bytes held by existing readers.
package atomicfile

import "os"

// ReplaceFile publishes from at to.
// On Windows this is FileRenameInfoEx with POSIX semantics, not MoveFileEx.
// Elsewhere it is rename(2), which already keeps that reader contract.
func ReplaceFile(from, to string) error {
	return replaceFile(from, to)
}

func replacePOSIX(from, to string) error {
	return os.Rename(from, to)
}
