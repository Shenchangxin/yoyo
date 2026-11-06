// Copyright 2026 Hypit.AI
// Translated from @hypit/timeline
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package timeline

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/remake/admission"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestSampleBoundaryMatchesRationalRounding(t *testing.T) {
	timeline := Timeline{ID: "film", FrameRate: FrameRate{Numerator: 30, Denominator: 1}, FrameCount: 30}
	if FrameCount(timeline) != 30 {
		t.Fatal(FrameCount(timeline))
	}
	if DurationSeconds(timeline) != 1 {
		t.Fatal(DurationSeconds(timeline))
	}
	if SampleFrames(timeline, 48000) != 48000 {
		t.Fatal(SampleFrames(timeline, 48000))
	}
	if ClockFrameSampleBoundary(Clock{FrameRate: timeline.FrameRate}, 1, 48000) != 1600 {
		t.Fatal("one frame")
	}
}

func TestInvalidTimelineIdentities(t *testing.T) {
	mustPanic(t, "Clock is invalid.", func() { AssertClockIdentity(Clock{}) })
	mustPanic(t, "Timeline is invalid.", func() {
		AssertTimelineIdentity(Timeline{ID: " ", FrameRate: FrameRate{Numerator: 24, Denominator: 1}, FrameCount: 1})
	})
	mustPanic(t, "Timeline frame boundary is invalid.", func() {
		FrameSampleBoundary(Timeline{ID: "film", FrameRate: FrameRate{Numerator: 24, Denominator: 1}, FrameCount: 2}, 3, 48000)
	})
	mustPanic(t, "Timeline sample rate is invalid.", func() {
		ClockFrameSampleBoundary(Clock{FrameRate: FrameRate{Numerator: 24, Denominator: 1}}, 0, 0)
	})
}

func TestValidatorAcceptsAnInlineTimeline(t *testing.T) {
	closure := protocol.ResolvedModuleClosure{}
	_ = closure
	Component.Validators[0].Handler(admission.TypeValidatorContext{
		Type: Types.Timeline,
		Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{
			"id": "film", "frameCount": float64(12),
			"frameRate": map[string]any{"numerator": float64(24), "denominator": float64(1)},
		}},
	})
	message := recoverMessage(func() {
		Component.Validators[0].Handler(admission.TypeValidatorContext{
			Value: protocol.StoredValue{Kind: "blob"},
		})
	})
	if !strings.Contains(message, "Timeline must be inline.") {
		t.Fatal(message)
	}
	if Package.Format != "hypit.package@1" || len(Package.Facets) != 2 || Manifest.Name != "@hypit/timeline" {
		t.Fatal("activation")
	}
}

func mustPanic(t *testing.T, part string, fn func()) {
	t.Helper()
	message := recoverMessage(fn)
	if !strings.Contains(message, part) {
		t.Fatal(message)
	}
}

func recoverMessage(fn func()) (message string) {
	defer func() {
		rec := recover()
		err, ok := rec.(error)
		if !ok {
			panic(rec)
		}
		message = err.Error()
	}()
	fn()
	return ""
}
