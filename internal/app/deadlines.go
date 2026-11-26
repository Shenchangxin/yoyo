// 评测、演进和 overlay 的墙钟上限。桌面、RPC、HTTP 和命令行都用这里的数，避免各写一份以后再分叉。

package app

import "time"

const (
	// EvalTimeout 是单套评测的上限：安全、普通、TB、行为。
	EvalTimeout = 10 * time.Minute
	// EvalSuiteTimeout 是多模型、密封、迁移、索引和基线搜索的上限。
	EvalSuiteTimeout = 20 * time.Minute

	evolveRound = 15 * time.Minute
	evolveCap   = 2 * time.Hour
	overlayStep = 15 * time.Minute
	overlayCap  = time.Hour
)

// EvolveTimeout 是一轮演进的墙钟。一轮及以下是 15 分钟，多轮按轮数相乘，最长 2 小时。
// rounds 小于等于 1 时返回 15 分钟。
func EvolveTimeout(rounds int) time.Duration {
	timeout := evolveRound
	if rounds > 1 {
		timeout = time.Duration(rounds) * evolveRound
		if timeout > evolveCap {
			timeout = evolveCap
		}
	}
	return timeout
}

// OverlayTimeout 是一次会话 overlay 的墙钟。k 夹在 1 到 4，每份 15 分钟。
// wall 比这个更长时改用 wall，最后不超过 1 小时。
// k 小于 1 时按 1，大于 4 时按 4。wall 小于等于 0 时不延长。
func OverlayTimeout(k int, wall time.Duration) time.Duration {
	if k < 1 {
		k = 1
	}
	if k > 4 {
		k = 4
	}
	timeout := overlayStep * time.Duration(k)
	if wall > timeout {
		timeout = wall
	}
	if timeout > overlayCap {
		timeout = overlayCap
	}
	return timeout
}
