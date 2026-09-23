package video

import (
	"strings"
	"unicode/utf8"
)

const (
	CharsPerMinute = 500
	SecondsPerShot = 12
	CharsPerSecond = 4.5
	PerformancePad = 2
	MinSegment     = 8
	MaxSegment     = 15
)

type DurationPlan struct {
	Chars         int
	TargetSeconds int
	SegmentCount  int
	MinTotal      int
	MaxTotal      int
}

func PlanDuration(script string) DurationPlan {
	n := utf8.RuneCountInString(strings.TrimSpace(script))
	if n < 1 {
		n = 1
	}
	sec := n * 60 / CharsPerMinute
	if sec < MinSegment {
		sec = MinSegment
	}
	count := (sec + SecondsPerShot/2) / SecondsPerShot
	if count < 1 {
		count = 1
	}
	return DurationPlan{
		Chars:         n,
		TargetSeconds: sec,
		SegmentCount:  count,
		MinTotal:      int(float64(sec) * 0.8),
		MaxTotal:      int(float64(sec) * 1.2),
	}
}

func DialogueFloor(dialogueChars int) int {
	if dialogueChars <= 0 {
		return MinSegment
	}
	sec := int(float64(dialogueChars)/CharsPerSecond + 0.5)
	sec += PerformancePad
	if sec < MinSegment {
		return MinSegment
	}
	if sec > MaxSegment {
		return MaxSegment
	}
	return sec
}

func ClampSegment(seconds int, kind string) int {
	lo, hi := MinSegment, MaxSegment
	switch kind {
	case "transition":
		lo, hi = 8, 10
	case "narrative":
		lo, hi = 10, 15
	case "beat":
		lo, hi = 12, 15
	}
	if seconds < lo {
		return lo
	}
	if seconds > hi {
		return hi
	}
	return seconds
}

func ClampProviderDuration(seconds int, provider string) int {
	switch strings.ToLower(provider) {
	case "aliyun", "wan":
		if seconds < 2 {
			return 2
		}
		if seconds > 30 {
			return 30
		}
		return seconds
	default:
		if seconds < 4 {
			return 4
		}
		if seconds > 15 {
			return 15
		}
		return seconds
	}
}
