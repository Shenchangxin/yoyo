package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

const (
	maxBrowserVerifySteps = 48
	qaEvidenceRel         = "qa/evidence/run.json"
	qaVerificationRel     = "qa/verification.json"
)

var qaCheckNames = []string{"launch", "render", "input", "coreLoop", "outcome", "restart"}

// VerifyPlan is the agent-supplied path. Status is never taken from the model.
type VerifyPlan struct {
	Open         string            `json:"open"`
	Root         string            `json:"root"`
	Terminal     string            `json:"terminal"`
	Restart      string            `json:"restart"`
	CleanContext *bool             `json:"cleanContext,omitempty"`
	Limitations  []qaLimitation    `json:"limitations,omitempty"`
	Steps        []BrowserStep     `json:"steps"`
	Status       string            `json:"status,omitempty"`
	Checks       map[string]string `json:"checks,omitempty"`
}

type qaLimitation struct {
	Scope  string `json:"scope"`
	Reason string `json:"reason"`
}

func parseVerifyPlan(args map[string]any, readFile func(rel string) ([]byte, error)) (VerifyPlan, error) {
	var plan VerifyPlan
	path := strings.TrimSpace(str(args["path"]))
	if path != "" && readFile != nil {
		raw, err := readFile(path)
		if err != nil {
			return plan, fmt.Errorf("browser_verify: read %s: %w", path, err)
		}
		if err := json.Unmarshal(raw, &plan); err != nil {
			return plan, fmt.Errorf("browser_verify: %s is not a JSON plan: %w", path, err)
		}
	}
	if v, ok := args["open"]; ok && str(v) != "" {
		plan.Open = str(v)
	}
	if v, ok := args["root"]; ok && str(v) != "" {
		plan.Root = str(v)
	}
	if v, ok := args["terminal"]; ok && str(v) != "" {
		plan.Terminal = str(v)
	}
	if v, ok := args["restart"]; ok && str(v) != "" {
		plan.Restart = str(v)
	}
	if v, ok := args["steps"]; ok && v != nil {
		steps, err := parseBrowserStepsN(v, maxBrowserVerifySteps)
		if err != nil {
			return plan, fmt.Errorf("browser_verify: %w", err)
		}
		plan.Steps = steps
	} else if len(plan.Steps) > maxBrowserVerifySteps {
		return plan, fmt.Errorf("browser_verify: %d steps exceeds the %d-step cap", len(plan.Steps), maxBrowserVerifySteps)
	} else if len(plan.Steps) > 0 {
		norm := make([]BrowserStep, 0, len(plan.Steps))
		for i, st := range plan.Steps {
			parsed, err := parseBrowserStep(st)
			if err != nil {
				return plan, fmt.Errorf("browser_verify: step %d: %w", i+1, err)
			}
			norm = append(norm, parsed)
		}
		plan.Steps = norm
	}
	if lim, ok := args["limitations"]; ok {
		plan.Limitations = parseLimitations(lim)
	}
	plan.Open = strings.TrimSpace(plan.Open)
	plan.Root = strings.TrimSpace(plan.Root)
	plan.Terminal = strings.TrimSpace(plan.Terminal)
	plan.Restart = strings.TrimSpace(plan.Restart)
	// Agent-supplied verdicts are ignored. The host computes them.
	plan.Status = ""
	plan.Checks = nil
	if looksLikeCSSSelector(plan.Root) {
		plan.Root = ""
	}
	if looksLikeCSSSelector(plan.Terminal) {
		plan.Terminal = ""
	}
	if looksLikeCSSSelector(plan.Restart) {
		plan.Restart = ""
	}
	if plan.Terminal == "" {
		plan.Terminal = "designed-outcome"
	}
	if plan.Restart == "" {
		plan.Restart = "initial-state"
	}
	if len(plan.Steps) == 0 && plan.Open == "" {
		return plan, fmt.Errorf("browser_verify: need path to a JSON plan, or open + steps")
	}
	return plan, nil
}

func parseLimitations(v any) []qaLimitation {
	raw, err := coerceJSONArray(v)
	if err != nil {
		return nil
	}
	var out []qaLimitation
	for _, item := range raw {
		switch t := item.(type) {
		case map[string]any:
			scope := strings.TrimSpace(str(t["scope"]))
			reason := strings.TrimSpace(str(t["reason"]))
			if scope == "" && reason == "" {
				continue
			}
			out = append(out, qaLimitation{Scope: scope, Reason: reason})
		case string:
			scope, reason, ok := strings.Cut(t, "=")
			if !ok {
				continue
			}
			out = append(out, qaLimitation{Scope: strings.TrimSpace(scope), Reason: strings.TrimSpace(reason)})
		}
	}
	return out
}

func (t *WorkspaceTools) browserVerify(args map[string]any) ToolResult {
	if msg, ok := t.repeatVerifyBlocked(); ok {
		return ToolResult{Content: msg}
	}
	plan, err := parseVerifyPlan(args, func(rel string) ([]byte, error) {
		p, err := t.resolve(rel)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(p)
	})
	if err != nil {
		return ToolResult{Err: err}
	}
	rootRel := normalizeVerifyRoot(t, plan.Root, plan.Open)
	plan.Root = rootRel
	rootAbs := t.Workspace
	if rootRel != "" && t != nil {
		if p, err := t.resolve(rootRel); err == nil {
			rootAbs = p
		}
	}
	if err := t.check(capability.WriteWorkspace, "browser_verify", filepath.Join(rootAbs, qaVerificationRel), ""); err != nil {
		return ToolResult{Err: err}
	}

	var results []browserStepResult
	chromeOK := false
	clean := true
	if plan.CleanContext != nil {
		clean = *plan.CleanContext
	}
	if plan.Open != "" {
		if t.Browser == nil {
			results = append(results, browserStepResult{Index: 0, Op: "open", Check: "launch", Err: "no isolated browser", Detail: "no isolated browser"})
		} else {
			if err := t.check(capability.Browser, "browser_verify", "", plan.Open); err != nil {
				return ToolResult{Err: err}
			}
			got := t.browserOpen(plan.Open)
			res := browserStepResult{Index: 0, Op: "open", Check: "launch"}
			if got.Err != nil {
				res.Err = got.Err.Error()
				res.Detail = res.Err
			} else {
				res.OK = true
				res.Detail = got.Content
				chromeOK = !strings.Contains(got.Content, "Isolated Chrome did not start") && !strings.Contains(got.Content, "Isolated Chrome is unavailable")
			}
			results = append(results, res)
		}
	} else if t.Browser != nil {
		if err := t.Browser.EnsurePreview(); err == nil {
			chromeOK = true
		}
		clean = false
	}

	if chromeOK && len(plan.Steps) > 0 {
		stepRes, _ := t.runBrowserSteps(plan.Steps, true)
		// Reindex after the optional open.
		base := len(results)
		for i := range stepRes {
			stepRes[i].Index = base + i + 1
			if stepRes[i].Check == "" && stepRes[i].Op == "screenshot" && stepRes[i].OK {
				stepRes[i].Check = "render"
			}
		}
		results = append(results, stepRes...)
	}

	runDoc, verifyDoc := deriveVerification(rootRel, plan, results, chromeOK, clean)
	if err := writeQAFiles(rootAbs, runDoc, verifyDoc); err != nil {
		return ToolResult{Err: err}
	}
	status, _ := verifyDoc["status"].(string)
	paths := []string{
		qaPath(rootRel, qaEvidenceRel),
		qaPath(rootRel, qaVerificationRel),
	}
	content := formatVerifyResult(status, verifyDoc, results, paths[0], paths[1])
	if rawRoot := str(args["root"]); looksLikeCSSSelector(rawRoot) {
		content = fmt.Sprintf("ignored root %q (CSS selector, not a directory); wrote under %s\n%s", rawRoot, rootRel, content)
	}
	return ToolResult{
		Content: content,
		FileChange: &FileChange{
			Paths:   paths,
			Created: true,
		},
	}
}

// looksLikeCSSSelector reports a CSS selector the model mistook for a
// filesystem path (session 9e69d099da9e65e6 passed root="#stage").
func looksLikeCSSSelector(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, `/\:`) {
		return false
	}
	switch s[0] {
	case '#', '.', '[':
		return true
	}
	return strings.ContainsAny(s, "[]:>+~")
}

func normalizeVerifyRoot(t *WorkspaceTools, root, open string) string {
	inferred := inferVerifyRoot(open)
	if looksLikeCSSSelector(root) {
		root = ""
	}
	root = strings.TrimSpace(strings.ReplaceAll(root, "\\", "/"))
	if root == "" {
		return inferred
	}
	if t == nil {
		return root
	}
	p, err := t.resolve(root)
	if err != nil {
		if inferred != "" {
			return inferred
		}
		return root
	}
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return root
	}
	if strings.Contains(root, "/") {
		return root
	}
	if inferred != "" {
		return inferred
	}
	return root
}

func inferVerifyRoot(open string) string {
	open = strings.ReplaceAll(strings.TrimSpace(open), "\\", "/")
	if open == "" {
		return ""
	}
	dir := pathDir(open)
	for dir != "" && dir != "." {
		base := filepath.Base(dir)
		parent := filepath.Dir(dir)
		if base == "app" && filepath.Base(parent) == "build" {
			return filepath.ToSlash(filepath.Dir(parent))
		}
		if base == "build" {
			return filepath.ToSlash(parent)
		}
		if base == "playable" {
			return filepath.ToSlash(parent)
		}
		if dir == parent {
			break
		}
		dir = parent
	}
	return ""
}

func pathDir(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	i := strings.LastIndex(p, "/")
	if i <= 0 {
		return ""
	}
	return p[:i]
}

func qaPath(rootRel, rel string) string {
	if rootRel == "" {
		return rel
	}
	return filepath.ToSlash(filepath.Join(rootRel, rel))
}

func deriveVerification(rootRel string, plan VerifyPlan, results []browserStepResult, chromeOK, clean bool) (runDoc, verifyDoc map[string]any) {
	observations := map[string]any{}
	inputsByCheck := map[string][]string{}
	visualByCheck := map[string]string{}
	okByCheck := map[string]bool{}
	sawCheck := map[string]bool{}
	var trace []string
	failed := false
	for _, r := range results {
		line := fmt.Sprintf("%s:%s", r.Op, firstLine(r.Detail))
		if r.OK {
			line = "ok " + line
		} else {
			line = "FAIL " + line
			failed = true
		}
		trace = append(trace, line)
		check := r.Check
		if !isQACheck(check) {
			continue
		}
		sawCheck[check] = true
		inputsByCheck[check] = append(inputsByCheck[check], line)
		if r.Visual != "" {
			visualByCheck[check] = r.Visual
		}
		if r.OK {
			okByCheck[check] = true
		}
	}
	if _, ok := sawCheck["launch"]; !ok && plan.Open != "" && chromeOK {
		sawCheck["launch"] = true
		if len(results) > 0 && results[0].Op == "open" && results[0].OK {
			okByCheck["launch"] = true
			inputsByCheck["launch"] = append(inputsByCheck["launch"], "ok open:"+plan.Open)
		}
	}

	checks := map[string]string{}
	reasons := map[string]string{}
	for _, name := range qaCheckNames {
		status, reason := judgeQACheck(name, sawCheck[name], okByCheck[name], inputsByCheck[name], visualByCheck[name], plan)
		checks[name] = status
		if reason != "" {
			reasons[name] = reason
		}
		if status == "NOT_RUN" {
			continue
		}
		state := map[string]any{"ok": status == "PASS"}
		if name == "outcome" {
			state["terminal"] = plan.Terminal
			state[plan.Terminal] = true
		}
		if name == "restart" {
			state["restart"] = plan.Restart
			state[plan.Restart] = true
		}
		if reason != "" {
			state["reason"] = reason
		}
		obs := map[string]any{
			"id":     name,
			"inputs": nonEmpty(inputsByCheck[name], name+" not observed"),
			"state":  state,
		}
		if vis := visualByCheck[name]; vis != "" {
			obs["visual"] = vis
		}
		observations[name] = obs
	}

	env := map[string]any{
		"runtime":   "yoyo-isolated-chrome",
		"open":      plan.Open,
		"root":      rootRel,
		"chrome_ok": chromeOK,
		"recorded":  time.Now().UTC().Format(time.RFC3339),
	}
	if len(trace) == 0 {
		trace = []string{"no steps ran"}
	}
	runDoc = map[string]any{
		"schemaVersion": 1,
		"runId":         "main-path",
		"environment":   env,
		"inputTrace":    trace,
		"observations":  observations,
	}

	limitations := append([]qaLimitation(nil), plan.Limitations...)
	overall := "PASS"
	exitCode := any(0)
	command := "browser_verify"
	if !chromeOK {
		overall = "NOT_RUN"
		exitCode = nil
		command = "NONE"
		limitations = append(limitations, qaLimitation{Scope: "verify", Reason: "isolated Chrome did not run; do not spawn Chrome or connect CDP 9333/9222"})
	} else if failed {
		overall = "FAIL"
	}
	if !clean {
		if overall == "PASS" {
			overall = "NOT_RUN"
		}
		limitations = append(limitations, qaLimitation{Scope: "completeRun", Reason: "run did not start from a clean context"})
	}
	for _, name := range qaCheckNames {
		if checks[name] != "PASS" {
			if chromeOK {
				if overall == "PASS" {
					if checks[name] == "FAIL" {
						overall = "FAIL"
					} else {
						overall = "NOT_RUN"
					}
				} else if checks[name] == "FAIL" {
					overall = "FAIL"
				}
			}
			reason := reasons[name]
			if reason == "" {
				reason = "not run"
			}
			limitations = append(limitations, qaLimitation{Scope: name, Reason: checks[name] + ": " + reason})
		}
	}

	limOut := make([]any, 0, len(limitations))
	for _, l := range limitations {
		limOut = append(limOut, map[string]any{"scope": l.Scope, "reason": l.Reason})
	}
	verifyDoc = map[string]any{
		"schemaVersion": 3,
		"status":        overall,
		"verify":        map[string]any{"command": command, "exitCode": exitCode},
		"completeRun": map[string]any{
			"id":           "main-path",
			"cleanContext": clean,
			"terminal":     plan.Terminal,
			"restart":      plan.Restart,
			"evidence":     qaEvidenceRel,
		},
		"checks":      checks,
		"limitations": limOut,
	}
	return runDoc, verifyDoc
}

func judgeQACheck(name string, saw, ok bool, inputs []string, visual string, plan VerifyPlan) (status, reason string) {
	if !saw {
		return "NOT_RUN", "no observation recorded"
	}
	if !ok {
		return "FAIL", "instrumented step failed"
	}
	if len(inputs) == 0 {
		return "NOT_RUN", "inputs must list the steps actually performed"
	}
	if name == "render" && strings.TrimSpace(visual) == "" {
		return "NOT_RUN", "render.visual is required"
	}
	if name == "outcome" && strings.TrimSpace(plan.Terminal) == "" {
		return "FAIL", "outcome state does not record the terminal marker"
	}
	if name == "restart" && strings.TrimSpace(plan.Restart) == "" {
		return "FAIL", "restart state does not record the restart marker"
	}
	return "PASS", ""
}

func nonEmpty(in []string, fallback string) []string {
	if len(in) == 0 {
		return []string{fallback}
	}
	return in
}

func writeQAFiles(rootAbs string, runDoc, verifyDoc map[string]any) error {
	if err := atomicWriteJSON(filepath.Join(rootAbs, filepath.FromSlash(qaEvidenceRel)), runDoc); err != nil {
		return err
	}
	return atomicWriteJSON(filepath.Join(rootAbs, filepath.FromSlash(qaVerificationRel)), verifyDoc)
}

func atomicWriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".qa-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr != nil {
		_ = os.Remove(tmpName)
		return werr
	}
	if cerr != nil {
		_ = os.Remove(tmpName)
		return cerr
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if err = os.Rename(tmpName, path); err != nil {
			_ = os.Remove(tmpName)
			return err
		}
	}
	return nil
}

func formatVerifyResult(status string, verifyDoc map[string]any, results []browserStepResult, evidenceRel, verifyRel string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "browser_verify status=%s (host-computed; do not rewrite qa/verification.json)\n", status)
	allPass := true
	if checks, ok := verifyDoc["checks"].(map[string]string); ok {
		for _, name := range qaCheckNames {
			fmt.Fprintf(&b, "  %s: %s\n", name, checks[name])
			if checks[name] != "PASS" {
				allPass = false
			}
		}
	}
	fmt.Fprintf(&b, "wrote %s and %s\n", evidenceRel, verifyRel)
	fail := 0
	for _, r := range results {
		if !r.OK {
			fail++
			fmt.Fprintf(&b, "failed step %d %s: %s\n", r.Index, r.Op, firstLine(r.Detail))
		}
	}
	if fail == 0 {
		b.WriteString("instrumented run completed without step failures\n")
	} else if allPass {
		b.WriteString("six checks PASS but a later step failed; overall is FAIL until the full path completes\n")
	}
	b.WriteString("Playthrough and QA share this run. Do not replay clicks just to fill JSON. Do not start python -m http.server or connect CDP 9333/9222.")
	return strings.TrimSpace(b.String())
}
