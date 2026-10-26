// duration.go 按字数估算成片时长，并把单段秒数夹进各供应商接受的范围。

package video

import (
	"strings"
	"unicode/utf8"
)

// 这些常量是口播时长和单段秒数的估算参数。
const (
	// CharsPerMinute 是估算口播时长时每分钟按 500 个字符计。
	CharsPerMinute = 500
	// SecondsPerShot 是估算段数时每段按 12 秒计。
	SecondsPerShot = 12
	// CharsPerSecond 是对白下限用的每秒 4.5 个字符。
	CharsPerSecond = 4.5
	// PerformancePad 是对白秒数上再加的 2 秒。
	PerformancePad = 2
	// MinSegment 是单段最短 8 秒。
	MinSegment = 8
	// MaxSegment 是单段最长 15 秒。
	MaxSegment = 15
)

// DurationPlan 是字数、目标秒数、段数，以及目标秒数的 0.8 倍和 1.2 倍。
type DurationPlan struct {
	Chars         int `json:"chars"`
	TargetSeconds int `json:"target_seconds"`
	SegmentCount  int `json:"segment_count"`
	MinTotal      int `json:"min_total"`
	MaxTotal      int `json:"max_total"`
}

// PlanDuration 按字符数除以每分钟 500 字得到秒数，短于 MinSegment 时抬到 MinSegment。段数用秒数除以 SecondsPerShot 四舍五入，至少 1 段。
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

// DialogueFloor 用字符数除以 CharsPerSecond 四舍五入，再加上 PerformancePad，然后夹进 MinSegment 和 MaxSegment。字符数小于等于 0 时返回 MinSegment。
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

// ClampSegment 按种类夹秒数：transition 是 8 到 10，narrative 是 10 到 15，beat 是 12 到 15，其余是 MinSegment 到 MaxSegment。
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

// ClampProviderDuration 把 aliyun 和 wan 夹进 2 到 30 秒，其余供应商夹进 4 到 15 秒。
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
