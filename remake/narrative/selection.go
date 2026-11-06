// Copyright 2026 Hypit.AI
// Translated from @hypit/narrative/src/selection.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package narrative

import "errors"

// TokenRange is a range in authored token order.
type TokenRange struct {
	NarrativeID       string
	TokenStart        int64
	TokenEndExclusive int64
}

// AnchorTokenBoundary resolves a semantic anchor to a boundary in the authored token sequence.
func AnchorTokenBoundary(narrative Narrative, anchorID string) int64 {
	var anchor Anchor
	found := false
	for _, item := range narrative.Anchors {
		if item.ID == anchorID {
			anchor = item
			found = true
			break
		}
	}
	if !found {
		panic(errors.New("Narrative " + narrative.ID + " has no anchor " + anchorID + "."))
	}
	var segment Segment
	found = false
	for _, item := range narrative.Segments {
		if item.ID == anchor.SegmentID {
			segment = item
			found = true
			break
		}
	}
	if !found {
		panic(errors.New("Anchor " + anchorID + " has no authored Segment."))
	}
	switch anchor.Kind {
	case "segment-start":
		return segment.TokenStart
	case "segment-end":
		return segment.TokenEndExclusive
	}
	index := -1
	for i, item := range narrative.Tokens {
		if item.ID == anchor.TokenID {
			index = i
			break
		}
	}
	if index < 0 {
		panic(errors.New("Anchor " + anchorID + " has no authored token."))
	}
	if anchor.Kind == "token-start" {
		return int64(index)
	}
	return int64(index + 1)
}

// SelectionTokenRange resolves a selection to the token span between its anchors.
// narrativeID is set when the selection object has that field.
func SelectionTokenRange(narrative Narrative, selection Selection, narrativeID *string) TokenRange {
	if narrativeID != nil && *narrativeID != narrative.ID {
		panic(errors.New("Selection " + selection.ID + " belongs to another Narrative."))
	}
	start, end := -1, -1
	for i, item := range narrative.Anchors {
		if item.ID == selection.StartAnchorID && start < 0 {
			start = i
		}
		if item.ID == selection.EndAnchorID && end < 0 {
			end = i
		}
	}
	if start < 0 || end < 0 || end < start {
		panic(errors.New("Selection " + selection.ID + " has invalid authored anchor order."))
	}
	return TokenRange{
		NarrativeID:       narrative.ID,
		TokenStart:        AnchorTokenBoundary(narrative, selection.StartAnchorID),
		TokenEndExclusive: AnchorTokenBoundary(narrative, selection.EndAnchorID),
	}
}

// TokensForSelection selects content without consulting a timeline.
func TokensForSelection(narrative Narrative, selection Selection, narrativeID *string) []Token {
	span := SelectionTokenRange(narrative, selection, narrativeID)
	return append([]Token{}, narrative.Tokens[span.TokenStart:span.TokenEndExclusive]...)
}
