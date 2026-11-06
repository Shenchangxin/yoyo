// Copyright 2026 Hypit.AI
// Translated from @hypit/narrative/src/types.ts and identity.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package narrative

import (
	"errors"
	"strings"
)

// Segment is one contiguous authored excerpt.
type Segment struct {
	ID                string
	StartAnchorID     string
	EndAnchorID       string
	TokenStart        int64
	TokenEndExclusive int64
}

// Token is one authored token inside a segment.
type Token struct {
	ID            string
	SegmentID     string
	StartAnchorID string
	EndAnchorID   string
	Text          string
	Normalized    string
}

// Turn is one speaker turn over a token span.
type Turn struct {
	ID                string
	SegmentID         string
	Role              string
	HasRole           bool
	TokenStart        int64
	TokenEndExclusive int64
}

// Selection is an author-selected anchor pair.
type Selection struct {
	ID            string
	StartAnchorID string
	EndAnchorID   string
}

// SelectionRef is a selection exported with its owning narrative id.
type SelectionRef struct {
	Selection
	NarrativeID string
}

// Moment is one authored semantic instant.
type Moment struct {
	ID       string
	AnchorID string
}

// MomentRef is a moment exported with its owning narrative id.
type MomentRef struct {
	Moment
	NarrativeID string
}

// Anchor is a segment or token boundary.
type Anchor struct {
	ID        string
	Kind      string
	SegmentID string
	TokenID   string
}

// Narrative is the complete authored content.
type Narrative struct {
	ID         string
	Segments   []Segment
	Tokens     []Token
	Turns      []Turn
	Selections []Selection
	Moments    []Moment
	Anchors    []Anchor
}

// SegmentRef is one author-selected contiguous excerpt.
type SegmentRef struct {
	Kind              string
	NarrativeID       string
	ID                string
	TokenStart        int64
	TokenEndExclusive int64
}

const maxSafe = int64(9007199254740991)

func nonempty(value, subject string) {
	if len(trimSpace(value)) == 0 {
		panic(errors.New(subject + " must not be empty."))
	}
}

func trimSpace(value string) string {
	return strings.TrimSpace(value)
}

func unique(values []string, subject string) map[string]struct{} {
	found := map[string]struct{}{}
	for _, value := range values {
		nonempty(value, subject)
		if _, ok := found[value]; ok {
			panic(errors.New(subject + " repeats " + value + "."))
		}
		found[value] = struct{}{}
	}
	return found
}

// AssertIdentity checks that segments, tokens, turns, selections, moments, and anchors agree.
func AssertIdentity(value Narrative) {
	nonempty(value.ID, "Narrative id")
	if len(value.Segments) == 0 {
		panic(errors.New("Narrative must contain at least one Segment."))
	}
	segmentIDs := unique(ids(len(value.Segments), func(i int) string { return value.Segments[i].ID }), "Narrative Segment id")
	tokenIDs := unique(ids(len(value.Tokens), func(i int) string { return value.Tokens[i].ID }), "Narrative Token id")
	unique(ids(len(value.Turns), func(i int) string { return value.Turns[i].ID }), "Narrative Turn id")
	unique(ids(len(value.Selections), func(i int) string { return value.Selections[i].ID }), "Narrative Selection id")
	unique(ids(len(value.Moments), func(i int) string { return value.Moments[i].ID }), "Narrative Moment id")
	anchorIDs := unique(ids(len(value.Anchors), func(i int) string { return value.Anchors[i].ID }), "Narrative Anchor id")
	anchorOrder := map[string]int{}
	for index, anchor := range value.Anchors {
		anchorOrder[anchor.ID] = index
	}
	anchorCursor := 0
	var tokenCursor int64
	for _, segment := range value.Segments {
		if !safe(segment.TokenStart) || !safe(segment.TokenEndExclusive) || segment.TokenStart != tokenCursor || segment.TokenEndExclusive < segment.TokenStart || segment.TokenEndExclusive > int64(len(value.Tokens)) || !has(anchorIDs, segment.StartAnchorID) || !has(anchorIDs, segment.EndAnchorID) {
			panic(errors.New("Narrative Segment " + segment.ID + " has invalid token or Anchor boundaries."))
		}
		if anchorCursor >= len(value.Anchors) {
			panic(errors.New("Narrative Segment " + segment.ID + " has no ordered start Anchor."))
		}
		segmentStart := value.Anchors[anchorCursor]
		anchorCursor++
		if segmentStart.ID != segment.StartAnchorID || segmentStart.Kind != "segment-start" || segmentStart.SegmentID != segment.ID {
			panic(errors.New("Narrative Segment " + segment.ID + " has no ordered start Anchor."))
		}
		for index := segment.TokenStart; index < segment.TokenEndExclusive; index++ {
			token := value.Tokens[index]
			if token.SegmentID != segment.ID || !has(tokenIDs, token.ID) || !has(anchorIDs, token.StartAnchorID) || !has(anchorIDs, token.EndAnchorID) || token.Text == "" || token.Normalized == "" {
				panic(errors.New("Narrative Token " + token.ID + " disagrees with Segment " + segment.ID + "."))
			}
			if anchorCursor+1 >= len(value.Anchors) {
				panic(errors.New("Narrative Token " + token.ID + " has no ordered Anchor pair."))
			}
			start := value.Anchors[anchorCursor]
			end := value.Anchors[anchorCursor+1]
			anchorCursor += 2
			if start.ID != token.StartAnchorID || start.Kind != "token-start" || start.SegmentID != segment.ID || start.TokenID != token.ID || end.ID != token.EndAnchorID || end.Kind != "token-end" || end.SegmentID != segment.ID || end.TokenID != token.ID {
				panic(errors.New("Narrative Token " + token.ID + " has no ordered Anchor pair."))
			}
		}
		if anchorCursor >= len(value.Anchors) {
			panic(errors.New("Narrative Segment " + segment.ID + " has no ordered end Anchor."))
		}
		segmentEnd := value.Anchors[anchorCursor]
		anchorCursor++
		if segmentEnd.ID != segment.EndAnchorID || segmentEnd.Kind != "segment-end" || segmentEnd.SegmentID != segment.ID {
			panic(errors.New("Narrative Segment " + segment.ID + " has no ordered end Anchor."))
		}
		tokenCursor = segment.TokenEndExclusive
	}
	if tokenCursor != int64(len(value.Tokens)) {
		panic(errors.New("Narrative Segments must partition Tokens in order."))
	}
	if anchorCursor != len(value.Anchors) {
		panic(errors.New("Narrative semantic anchors must be exactly the Segment and Token boundaries."))
	}
	for _, turn := range value.Turns {
		if !has(segmentIDs, turn.SegmentID) || !safe(turn.TokenStart) || !safe(turn.TokenEndExclusive) || turn.TokenEndExclusive <= turn.TokenStart || turn.TokenStart < 0 || turn.TokenEndExclusive > int64(len(value.Tokens)) || !turnCovers(value.Tokens, turn) {
			panic(errors.New("Narrative Turn " + turn.ID + " has invalid Token coverage."))
		}
	}
	for _, selection := range value.Selections {
		start, startOK := anchorOrder[selection.StartAnchorID]
		end, endOK := anchorOrder[selection.EndAnchorID]
		if !startOK || !endOK || end < start {
			panic(errors.New("Narrative Selection " + selection.ID + " has invalid Anchor order."))
		}
	}
	for _, moment := range value.Moments {
		if !has(anchorIDs, moment.AnchorID) {
			panic(errors.New("Narrative Moment " + moment.ID + " names an unknown Anchor."))
		}
	}
}

// AssertSegmentRefIdentity checks one exported segment reference.
func AssertSegmentRefIdentity(value SegmentRef) {
	if value.Kind != "segment" {
		panic(errors.New("NarrativeSegmentRef must be a Segment."))
	}
	nonempty(value.NarrativeID, "NarrativeSegmentRef narrativeId")
	nonempty(value.ID, "NarrativeSegmentRef id")
	if !safe(value.TokenStart) || !safe(value.TokenEndExclusive) || value.TokenStart < 0 || value.TokenEndExclusive < value.TokenStart {
		panic(errors.New("NarrativeSegmentRef Token coverage is invalid."))
	}
}

// AssertSelectionRefIdentity checks one exported selection reference.
func AssertSelectionRefIdentity(value SelectionRef) {
	nonempty(value.NarrativeID, "NarrativeSelection narrativeId")
	nonempty(value.ID, "NarrativeSelection id")
	nonempty(value.StartAnchorID, "NarrativeSelection startAnchorId")
	nonempty(value.EndAnchorID, "NarrativeSelection endAnchorId")
}

// AssertMomentRefIdentity checks one exported moment reference.
func AssertMomentRefIdentity(value MomentRef) {
	nonempty(value.NarrativeID, "NarrativeMoment narrativeId")
	nonempty(value.ID, "NarrativeMoment id")
	nonempty(value.AnchorID, "NarrativeMoment anchorId")
}

func turnCovers(tokens []Token, turn Turn) bool {
	for index := turn.TokenStart; index < turn.TokenEndExclusive; index++ {
		if tokens[index].SegmentID != turn.SegmentID {
			return false
		}
	}
	return true
}

func ids(n int, at func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = at(i)
	}
	return out
}

func has(values map[string]struct{}, key string) bool {
	_, ok := values[key]
	return ok
}

func safe(value int64) bool {
	return value >= 0 && value <= maxSafe
}
