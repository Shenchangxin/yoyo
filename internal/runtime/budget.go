package runtime

import (
	"fmt"
	"sync"
)

// Meter tracks estimated spend. Harbor evals leave MaxBudgetUSD at 0
// (unlimited). The agent loop stops with a typed error instead of hanging.
type Meter struct {
	mu         sync.Mutex
	InputTok   int
	OutputTok  int
	USD        float64
	USDPerMTok float64
}

type ErrBudget struct {
	Used, Cap float64
}

func (e ErrBudget) Error() string {
	return fmt.Sprintf("budget exceeded: $%.4f / $%.4f", e.Used, e.Cap)
}

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
		rate = 0.40
	}
	m.USD += float64(in+out) * rate / 1_000_000
}

func (m *Meter) Check(cap float64) error {
	if m == nil || cap <= 0 {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.USD > cap {
		return ErrBudget{Used: m.USD, Cap: cap}
	}
	return nil
}

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
