// Copyright 2026 Hypit.AI
// Translated from @hypit/blob (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package blob

import "github.com/Shenchangxin/yoyo/remake/protocol"

// Package is the hypit.package@1 activation exported by @hypit/blob.
type Package struct {
	Format  string                    `json:"format"`
	Modules []protocol.ResolvedModule `json:"modules"`
}

var HypitPackage = Package{
	Format:  "hypit.package@1",
	Modules: []protocol.ResolvedModule{{Manifest: BlobManifest}},
}
