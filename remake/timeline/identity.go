// Copyright 2026 Hypit.AI
// Translated from @hypit/timeline/src/types.ts and identity.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package timeline

import (
	"errors"
	"math"
	"math/big"
	"strings"
)

// Clock is the discrete sampling clock shared by one timeline and its material.
type Clock struct {
	FrameRate FrameRate
}

// FrameRate is a rational frame rate.
type FrameRate struct {
	Numerator   int64
	Denominator int64
}

// Timeline is one complete, content-independent absolute frame domain.
type Timeline struct {
	ID         string
	FrameRate  FrameRate
	FrameCount int64
}

// LocatedFrameSpan is an authored half-open frame interval.
type LocatedFrameSpan struct {
	StartFrame        int64
	EndFrameExclusive int64
}

const maxSafe = int64(9007199254740991)

// SealClock checks the clock and returns a copy.
func SealClock(value Clock) Clock {
	AssertClockIdentity(value)
	return value
}

// AssertClockIdentity throws when the frame rate is not two positive safe integers.
func AssertClockIdentity(clock Clock) {
	if !positiveSafe(clock.FrameRate.Numerator) || !positiveSafe(clock.FrameRate.Denominator) {
		panic(errors.New("Clock is invalid."))
	}
}

// SealTimeline checks the timeline and returns a copy.
func SealTimeline(value Timeline) Timeline {
	AssertTimelineIdentity(value)
	return value
}

// AssertTimelineIdentity throws when the id or frame count is not a timeline identity.
func AssertTimelineIdentity(timeline Timeline) {
	AssertClockIdentity(Clock{FrameRate: timeline.FrameRate})
	if strings.TrimSpace(timeline.ID) == "" || !safeInteger(timeline.FrameCount) || timeline.FrameCount < 1 {
		panic(errors.New("Timeline is invalid."))
	}
}

// FrameCount returns the checked frame count.
func FrameCount(timeline Timeline) int64 {
	AssertTimelineIdentity(timeline)
	return timeline.FrameCount
}

// DurationSeconds returns frameCount * denominator / numerator.
func DurationSeconds(timeline Timeline) float64 {
	AssertTimelineIdentity(timeline)
	return float64(timeline.FrameCount) * float64(timeline.FrameRate.Denominator) / float64(timeline.FrameRate.Numerator)
}

// SampleFrames is the sample boundary at the end of the timeline.
func SampleFrames(timeline Timeline, sampleRate int64) int64 {
	return FrameSampleBoundary(timeline, FrameCount(timeline), sampleRate)
}

// FrameSampleBoundary converts a frame boundary into a sample boundary.
func FrameSampleBoundary(timeline Timeline, frame, sampleRate int64) int64 {
	frames := FrameCount(timeline)
	if !safeInteger(frame) || frame < 0 || frame > frames {
		panic(errors.New("Timeline frame boundary is invalid."))
	}
	return ClockFrameSampleBoundary(Clock{FrameRate: timeline.FrameRate}, frame, sampleRate)
}

// ClockFrameSampleBoundary rounds frame * sampleRate * denominator / numerator to the nearest safe integer.
func ClockFrameSampleBoundary(clock Clock, frame, sampleRate int64) int64 {
	AssertClockIdentity(clock)
	if !safeInteger(frame) || frame < 0 {
		panic(errors.New("Clock frame boundary is invalid."))
	}
	if !safeInteger(sampleRate) || sampleRate <= 0 {
		panic(errors.New("Timeline sample rate is invalid."))
	}
	numerator := new(big.Int).Mul(big.NewInt(frame), big.NewInt(sampleRate))
	numerator.Mul(numerator, big.NewInt(clock.FrameRate.Denominator))
	denominator := big.NewInt(clock.FrameRate.Numerator)
	value := new(big.Int).Mul(numerator, big.NewInt(2))
	value.Add(value, denominator)
	value.Quo(value, new(big.Int).Mul(denominator, big.NewInt(2)))
	if value.Sign() < 0 || value.Cmp(big.NewInt(maxSafe)) > 0 {
		panic(errors.New("Timeline sample domain exceeds safe arithmetic."))
	}
	return value.Int64()
}

func positiveSafe(value int64) bool {
	return safeInteger(value) && value > 0
}

func safeInteger(value int64) bool {
	return value >= 0 && value <= maxSafe && float64(value) <= math.MaxFloat64
}
