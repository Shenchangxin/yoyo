// Copyright 2026 Hypit.AI
// Translated from @hypit/studio-companion (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package studiocompanion

import (
	"testing"

	"github.com/Shenchangxin/yoyo/remake/facet"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestTypedChildValuesRecoverAnOmittedAuthoredIDWithoutOrdinalGuessing(t *testing.T) {
	itemType := protocol.TypeRef{Module: protocol.ModuleRef{Name: "@example/media", Version: "1"}, Name: "ItemSpec"}
	child := StudioPlacementChild{
		SourcePath: "main.svml", Tag: "Item", Range: Range{Start: 10, End: 40},
		Attributes: map[string]string{}, AttributeValueRanges: map[string]Range{}, References: []string{},
		ReferenceAttributes: map[string]string{}, ReferenceTypes: map[string]string{},
		Values: []StudioObservedValue{{ID: "track.item.0001.spec", Type: itemType, Value: map[string]any{"id": "track.item.0001"}}},
	}
	trackID := "track"
	placement := &StudioPlacement{
		SourcePath: "main.svml", Tag: "Track", Module: itemType.Module, Surface: "track", ID: &trackID,
		Range: Range{Start: 0, End: 50}, Records: []string{}, Values: []StudioObservedValue{}, Outputs: []string{},
		OutputPorts: []StudioPlacementOutputPort{}, Children: []StudioPlacementChild{child},
		Attributes: map[string]string{}, AttributeValueRanges: map[string]Range{}, References: []string{},
		ReferenceAttributes: map[string]string{}, ReferenceTypes: map[string]string{},
	}
	context := StudioTrackCompanionContext{Placement: placement}
	got := AuthoredChildFor(context, "track.item.0001", []protocol.TypeRef{itemType})
	if got != &placement.Children[0] {
		t.Fatalf("%+v", got)
	}
	wrong := itemType
	wrong.Module.Version = "2"
	if AuthoredChildFor(context, "track.item.0001", []protocol.TypeRef{wrong}) != nil {
		t.Fatal("version mismatch recovered a child")
	}
}

func TestTemporalRelationIdentitiesAreQualifiedByTheSelectedPhysicalPackage(t *testing.T) {
	producer := protocol.ProducerRef{Module: protocol.ModuleRef{Name: "@example/author", Version: "1"}, Name: "point"}
	contribution := StudioContributionFromPackage("@project/author", []facet.Facet{CreateStudioCompanionFacet(StudioCompanionFacetInput{
		TemporalRelations: []StudioTemporalRelationCompanion{{
			ID:    "point",
			Match: StudioTemporalRelationMatch{Producer: producer, Output: "point"},
			Invert: func(StudioTemporalRelationInvertInput) []StudioTemporalRelationInversePlan {
				return []StudioTemporalRelationInversePlan{}
			},
		}},
	})})
	if len(contribution.TemporalRelations) != 1 {
		t.Fatalf("%+v", contribution.TemporalRelations)
	}
	if contribution.TemporalRelations[0].ID != "@project/author#point" {
		t.Fatal(contribution.TemporalRelations[0].ID)
	}
	if contribution.TemporalRelations[0].Match != (StudioTemporalRelationMatch{Producer: producer, Output: "point"}) {
		t.Fatalf("%+v", contribution.TemporalRelations[0].Match)
	}
}
