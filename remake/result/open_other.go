// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/node/store.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

//go:build !windows

package result

import "os"

func openShared(path string) (*os.File, error) {
	return os.Open(path)
}
