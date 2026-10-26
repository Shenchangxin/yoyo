// 按 token 估算美元花费。Harbor 评测和聊天挂的是同一块 Meter，两边共用 tokens 与 USD 字段。
// 不发起模型调用。Run 在每轮成功后调用 Add 和 Check。

package runtime

import (
	"fmt"
	"sync"
)

// Meter 累计估算花费。USDPerMTok 小于等于 0 时，Add 不用这个字段，改用 0.40。
type Meter struct {
	mu         sync.Mutex
	InputTok   int
	OutputTok  int
	USD        float64
	USDPerMTok float64 // 每百万 token 的美元费率；小于等于 0 时 Add 改用 0.40
}

// ErrBudget 表示花费超过上限。Used 是已累计美元，Cap 是 Check 收到的上限。
type ErrBudget struct {
	Used, Cap float64
}

// Error 返回 budget exceeded，Used 和 Cap 都保留四位小数。
func (e ErrBudget) Error() string {
	return fmt.Sprintf("budget exceeded: $%.4f / $%.4f", e.Used, e.Cap)
}

// Add 把 in、out 累加进 InputTok、OutputTok。m 为 nil 时直接返回。
// USDPerMTok 小于等于 0 时费率用 0.40。美元增量是 float64(in+out) 乘以该费率，再除以 1_000_000。
func (m *Meter) Add(in, out int) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InputTok += in
	m.OutputTok += out
	rate := m.USDPerMTok
	if rate <= 0 {
		// 费率没设或设成 0 时用 0.40 美元每百万 token。增量是 (in+out) 乘费率再除以 1e6。
		rate = 0.40
	}
	m.USD += float64(in+out) * rate / 1_000_000
}

// Check 在 m 为 nil 或 cap 小于等于 0 时返回 nil，不持锁、不比较。cap 大于 0 且 USD 大于 cap 时返回 ErrBudget；USD 等于 cap 不算超过。
func (m *Meter) Check(cap float64) error {
	if m == nil || cap <= 0 {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.USD > cap {
		// 等于上限不算超过。cap 小于等于 0 时函数开头已经返回，不会进到这里。
		return ErrBudget{Used: m.USD, Cap: cap}
	}
	return nil
}

// Totals 返回已累计的输入 token、输出 token 和美元。m 为 nil 时三个都是 0。
func (m *Meter) Totals() (in, out int, usd float64) {
	if m == nil {
		return 0, 0, 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.InputTok, m.OutputTok, m.USD
}

// Snapshot 返回 usd、input_tokens、output_tokens。m 非 nil 时再带上 usd_per_mtok。m 为 nil 时前三项都是 0，没有 usd_per_mtok。
func (m *Meter) Snapshot() map[string]any {
	if m == nil {
		return map[string]any{"usd": 0, "input_tokens": 0, "output_tokens": 0}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{
		"usd":           m.USD,
		"input_tokens":  m.InputTok,
		"output_tokens": m.OutputTok,
		"usd_per_mtok":  m.USDPerMTok,
	}
}
