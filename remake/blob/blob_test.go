// Copyright 2026 Hypit.AI
// Translated from @hypit/blob (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package blob

import "testing"

func TestBlobModuleDeclaresOneType(t *testing.T) {
	if BlobManifest.Format != "hypit.module@1" || BlobManifest.Name != "@hypit/blob" || BlobManifest.Version != "1" {
		t.Fatalf("%+v", BlobManifest)
	}
	if len(BlobManifest.Types) != 1 || BlobManifest.Types[0].Name != "Blob" {
		t.Fatalf("%+v", BlobManifest.Types)
	}
	if len(BlobManifest.Dependencies) != 0 || len(BlobManifest.Capabilities) != 0 || len(BlobManifest.Producers) != 0 {
		t.Fatal("blob declares no dependencies, capabilities, or producers")
	}
	if BlobTypes.Blob.Name != "Blob" || BlobDependency.Module != BlobModuleRef {
		t.Fatal("blob type or dependency drifted")
	}
	if HypitPackage.Format != "hypit.package@1" || len(HypitPackage.Modules) != 1 {
		t.Fatal("activation package")
	}
	if HypitPackage.Modules[0].Manifest.Name != BlobManifest.Name {
		t.Fatal("activation does not carry the blob manifest")
	}
}
