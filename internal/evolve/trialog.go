// 试用日志：把指标、提议和非 held 题的结果写到 TrialRoot。held 题的正文不落盘。不参与晋升判断。

package evolve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

// logTrial 在引擎、TrialRoot 或哈希为空时直接返回。目录名取哈希前 16 个字节，并把斜杠换成连字符。held 里的题号不写 tasks 文件。
// hash 为空时直接返回；否则目录名取前 16 个字节，斜杠和反斜杠换成连字符。
// report 的 Metrics 写成 metrics.json；Results 里题号不在 held 中的写成 tasks 下的 json。
// p 整份写成 proposal.json。
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
