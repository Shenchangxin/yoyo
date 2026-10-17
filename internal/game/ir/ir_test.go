package ir

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeAndSpacing(t *testing.T) {
	c := Candidate{ID: "c_1", WorldXYZ: Vec3{9, 1, 2}, GridCell: "center"}
	raw, _ := json.Marshal(SanitizeForModel(c))
	if strings.Contains(string(raw), "world_xyz") {
		t.Fatal(string(raw))
	}
	els := []Placement{
		{PlacementID: "a", ElementType: "collectible", WorldXYZ: Vec3{0, 0, 0}},
		{PlacementID: "b", ElementType: "collectible", WorldXYZ: Vec3{0.1, 0, 0}},
	}
	if err := ValidateSpacing(els); err == nil {
		t.Fatal("expected spacing error")
	}
	els[1].WorldXYZ = Vec3{8, 0, 0}
	if err := ValidateSpacing(els); err != nil {
		t.Fatal(err)
	}
}

func TestLogicalMeshRejected(t *testing.T) {
	if !LogicalOnly("player_spawn") {
		t.Fatal("spawn should be logical")
	}
	if err := ValidateAsset(Asset{Role: "player_spawn"}); err == nil {
		t.Fatal("mesh on spawn")
	}
	if err := ValidateAsset(Asset{Role: "player_spawn", SkipMesh: true}); err != nil {
		t.Fatal(err)
	}
}
