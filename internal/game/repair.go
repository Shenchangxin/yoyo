package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

func (e *Engine) Repair(sessionID, id string) (Project, error) {
	p, err := e.resolve(sessionID, id)
	if err != nil {
		return p, err
	}
	dir, err := e.projectDir(p)
	if err != nil {
		return p, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "qa", "verification.json"))
	if err != nil {
		return p, fmt.Errorf("qa/verification.json missing; run game_verify first")
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return p, err
	}
	status, _ := doc["status"].(string)
	if strings.EqualFold(status, "PASS") {
		p.Stage = StageChecked
		p.Status = "verified"
		p.Error = ""
		return e.Update(p)
	}
	failed := failedChecks(doc)
	var corr []ir.Correspondence
	_ = e.readIR(p, "correspondence.json", &corr)
	touched := localize(failed, corr)
	switch {
	case containsAny(failed, "render", "launch"):
		if _, err := e.AnalyzeScene(sessionID, p.ID, p.Intent); err != nil {
			return p, err
		}
		if _, err := e.ExtractCandidates(sessionID, p.ID); err != nil {
			return p, err
		}
		if _, err := e.Place(sessionID, p.ID); err != nil {
			return p, err
		}
		if _, err := e.RealizeAssets(sessionID, p.ID); err != nil {
			return p, err
		}
	case containsAny(failed, "outcome", "coreLoop"):
		if _, err := e.PlanRules(sessionID, p.ID, p.Intent); err != nil {
			return p, err
		}
		if _, err := e.Place(sessionID, p.ID); err != nil {
			return p, err
		}
		if _, err := e.RealizeAssets(sessionID, p.ID); err != nil {
			return p, err
		}
	case containsAny(failed, "input", "restart"):
		// runtime contract only — recompile
	default:
		if len(touched) == 0 {
			if _, err := e.Place(sessionID, p.ID); err != nil {
				return p, err
			}
		}
	}
	p, err = e.Compile(sessionID, p.ID)
	if err != nil {
		p.Stage = StageFailed
		p.Status = "failed"
		p.Error = err.Error()
		_, _ = e.Update(p)
		return p, err
	}
	p.Status = "repaired"
	p.Error = strings.Join(failed, ",")
	return e.Update(p)
}

func failedChecks(doc map[string]any) []string {
	var out []string
	checks, _ := doc["checks"].(map[string]any)
	for name, v := range checks {
		s := strings.ToUpper(fmt.Sprint(v))
		if s == "FAIL" || s == "NOT_RUN" {
			out = append(out, name)
		}
	}
	return out
}

func localize(failed []string, corr []ir.Correspondence) []string {
	var ids []string
	for _, c := range corr {
		if containsAny(failed, "outcome") && (c.Kappa == "goal_area" || c.Kappa == "collectible") {
			ids = append(ids, c.ID)
		}
		if containsAny(failed, "coreLoop") && c.Kappa == "collectible" {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

func containsAny(have []string, keys ...string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[strings.ToLower(h)] = true
	}
	for _, k := range keys {
		if set[strings.ToLower(k)] {
			return true
		}
	}
	return false
}
