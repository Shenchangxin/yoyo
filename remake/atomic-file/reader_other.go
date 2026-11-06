// Copyright 2026 Hypit.AI
// Translated from @hypit/atomic-file (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

//go:build !windows

package atomicfile

import "os"

func openReader(path string) (*os.File, error) {
	return os.Open(path)
}
