// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol/test/canonical.test.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package protocol

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestCanonicalValuesHaveStableKeyOrderAndPreserveArrayOrder(t *testing.T) {
	if CanonicalStringify(map[string]any{"b": 1, "a": 2}) != `{"a":2,"b":1}` {
		t.Fatalf("%s", CanonicalStringify(map[string]any{"b": 1, "a": 2}))
	}
	if CanonicalStringify([]any{1, 2}) == CanonicalStringify([]any{2, 1}) {
		t.Fatal("array order was not preserved")
	}
}

func TestCanonicalizationNormalizesNegativeZeroAndIsIdempotent(t *testing.T) {
	once := Canonicalize(map[string]any{"z": []any{1, map[string]any{"y": "x"}}, "a": math.Copysign(0, -1)}).(map[string]any)
	again := Canonicalize(once).(map[string]any)
	if CanonicalStringify(again) != CanonicalStringify(once) {
		t.Fatal("canonicalize was not idempotent")
	}
	got := once["a"].(float64)
	if got != 0 || math.Signbit(got) {
		t.Fatalf("negative zero survived: %v", got)
	}
}

func TestValuesWithoutOneJSONMeaningAreRefused(t *testing.T) {
	err := mustSvml(t, func() { Canonicalize(map[string]any{"a": math.NaN()}) })
	if !strings.Contains(err.Message, "must be finite") {
		t.Fatal(err.Message)
	}
	err = mustSvml(t, func() { Canonicalize(map[string]any{"a": Undefined{}}) })
	if !strings.Contains(err.Message, "is undefined") {
		t.Fatal(err.Message)
	}
}

func TestResourceIdsRetainTheirNarrowExecutionReferenceSyntax(t *testing.T) {
	if !IsResourceId("res_fixture-resource") || IsResourceId("fixture-resource") || IsResourceId("md5:abc") {
		t.Fatal("resource id syntax")
	}
}

func TestBuildIdsCannotAddressAParentOrNestedPath(t *testing.T) {
	AssertBuildId("build-19")
	for _, value := range []string{"..", ".", "episode/19", "episode\\19", " padded "} {
		mustMessage(t, "Build id", func() { AssertBuildId(value) })
	}
}

func TestPublicBuildIdsCarrySortableUTCSubmissionTime(t *testing.T) {
	earlier := OrderedBuildId(mustUnixMilli(t, "2026-09-02T10:20:30.123Z"), "0000000001")
	later := OrderedBuildId(mustUnixMilli(t, "2026-09-02T10:20:30.124Z"), "0000000000")
	if earlier != "bld_20260902T102030123Z_0000000001" {
		t.Fatal(earlier)
	}
	got, ok := BuildIdCreatedAt(earlier)
	if !ok || got != mustUnixMilli(t, "2026-09-02T10:20:30.123Z") {
		t.Fatalf("created %d %v", got, ok)
	}
	if earlier >= later {
		t.Fatal("ordered ids did not sort")
	}
	AssertOrderedBuildId(earlier)
	mustMessage(t, "UTC submission time", func() { AssertOrderedBuildId("build-19") })
}

func mustUnixMilli(t *testing.T, iso string) int64 {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.UnixMilli()
}

func mustSvml(t *testing.T, fn func()) *SvmlError {
	t.Helper()
	var got *SvmlError
	func() {
		defer func() {
			rec := recover()
			err, ok := rec.(*SvmlError)
			if !ok {
				t.Fatalf("panic %#v", rec)
			}
			got = err
		}()
		fn()
	}()
	if got == nil {
		t.Fatal("expected SvmlError")
	}
	return got
}

func mustMessage(t *testing.T, part string, fn func()) {
	t.Helper()
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic")
		}
		err, ok := rec.(error)
		if !ok || !strings.Contains(err.Error(), part) {
			t.Fatalf("panic %v", rec)
		}
	}()
	fn()
}
