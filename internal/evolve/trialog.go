package evolve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

func (e *Engine) logTrial(hash string, report eval.RunReport, p Proposal, held map[string]bool) {
	if e == nil || e.TrialRoot == "" || hash == "" {
		return
	}
	safe := hash
	if len(safe) > 16 {
		safe = safe[:16]
	}
	safe = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' {
			return '-'
		}
		return r
	}, safe)
	dir := filepath.Join(e.TrialRoot, safe)
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0o755); err != nil {
		return
	}
	metrics, _ := json.MarshalIndent(report.Metrics, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "metrics.json"), metrics, 0o644)
	prop, _ := json.MarshalIndent(p, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "proposal.json"), prop, 0o644)
	for _, r := range report.Results {
		if held[r.ID] {
			continue
		}
		b, _ := json.MarshalIndent(r, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, "tasks", r.ID+".json"), b, 0o644)
	}
}
