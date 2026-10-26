// browser_verify 在隔离 Chrome 上跑检查，并把宿主算出的结论写到 qa/evidence/run.json 和 qa/verification.json。
// 模型给出的 status 和 checks 被丢掉。计划文件和 root 经 WorkspaceTools.resolve。不在本文件创建 Windows Job。

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
	// maxBrowserVerifySteps 是验证计划一次最多接受的步数。
	maxBrowserVerifySteps = 48
	// qaEvidenceRel 是运行记录相对项目根的路径。
	qaEvidenceRel = "qa/evidence/run.json"
	// qaVerificationRel 是验证结论相对项目根的路径。
	qaVerificationRel = "qa/verification.json"
)

// qaCheckNames 是六项检查的固定顺序。
var qaCheckNames = []string{"launch", "render", "input", "coreLoop", "outcome", "restart"}

// VerifyPlan 是代理提供的路径。Status 从不采用模型给的值。
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

// qaLimitation 是一条范围和原因，写入 verification.json 的 limitations。
type qaLimitation struct {
	Scope  string `json:"scope"`
	Reason string `json:"reason"`
}

// parseVerifyPlan 从 path 或参数读计划。模型给的 Status 和 Checks 被清空。看起来像 CSS 选择器的 root、terminal、restart 被丢掉。
// terminal 空则用 designed-outcome，restart 空则用 initial-state。既没有步骤也没有 open 则失败。文件步骤超过 48 步失败。
// args 读取 path、open、root、terminal、restart、steps 和 limitations。
// readFile 在 path 去掉空白后非空且自身非 nil 时用来读计划文件。
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
	// 代理给的结论被忽略。由宿主计算。
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

// parseLimitations 解析对象数组，或 scope=reason 形式的字符串。解析失败返回 nil。
// v 交给 coerceJSONArray，失败时返回 nil。
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

// browserVerify 跑验证并写出两份 JSON。计划文件和 root 都走 resolve。CleanContext 未给出时视为干净。没有 open 但预览能起来时，chrome 算可用且 clean 被改成假。
// root 若是 CSS 选择器，结果里会说明已忽略，并写到推断出的项目目录下。
// args 传给 parseVerifyPlan。
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
		// 可选的 open 占了前面的序号，后面的步骤接着编。
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

// looksLikeCSSSelector 判断模型是不是把 CSS 选择器当成了文件系统路径（会话 9e69d099da9e65e6 传过 root="#stage"）。
// 含斜杠、反斜杠或冒号的字符串不是选择器。以 #、. 或 [ 开头，或含 []:>+~ 的才是。
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

// normalizeVerifyRoot 把 root 收成项目目录。CSS 选择器被丢掉。resolve 失败、或解析结果不是目录且路径里没有斜杠时，改用从 open 推断的目录。
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

// inferVerifyRoot 从 open 往上找 build/app、build 或 playable，返回其项目目录。找不到返回空串。
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

// pathDir 返回最后一个斜杠之前的部分。没有斜杠或斜杠在开头则返回空串。
func pathDir(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	i := strings.LastIndex(p, "/")
	if i <= 0 {
		return ""
	}
	return p[:i]
}

// qaPath 把相对路径接到 rootRel 下。rootRel 为空时原样返回。
// rel 在 rootRel 为空时原样返回，否则接到 rootRel 后面。
func qaPath(rootRel, rel string) string {
	if rootRel == "" {
		return rel
	}
	return filepath.ToSlash(filepath.Join(rootRel, rel))
}

// deriveVerification 由宿主计算六项检查和总状态。Chrome 没跑是 NOT_RUN。任一步失败是 FAIL。不是干净上下文时，本可 PASS 的总状态改成 NOT_RUN。任一检查不是 PASS 时，FAIL 优先，否则降为 NOT_RUN。
// rootRel 写入 environment 的 root。
// plan 读取 Open、Limitations、Terminal 和 Restart。
// results 逐条收成轨迹，并按 Check 归到各项检查。
// chromeOK 为假时总状态写成 NOT_RUN；为真时参与补 launch，并在检查不是 PASS 时决定总状态。
// clean 写入 completeRun；为假时，本可 PASS 的总状态改成 NOT_RUN。
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

// judgeQACheck 给一项检查定 PASS、FAIL 或 NOT_RUN。没看到是 NOT_RUN。步骤失败是 FAIL。没有输入记录是 NOT_RUN。render 没有 visual 是 NOT_RUN。outcome 或 restart 缺少标记是 FAIL。
// name 为 render、outcome 或 restart 时额外检查 visual 或 plan 上的标记。
// saw 为假时返回 NOT_RUN。
// ok 为假时返回 FAIL。
// inputs 长度为 0 时返回 NOT_RUN。
// plan 在 outcome 或 restart 上读取 Terminal、Restart，去掉空白后为空则返回 FAIL。
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

// nonEmpty 在列表为空时返回只含 fallback 的一段。
// in 长度为 0 时改成只含 fallback 的一段。
func nonEmpty(in []string, fallback string) []string {
	if len(in) == 0 {
		return []string{fallback}
	}
	return in
}

// writeQAFiles 原子写入 run.json 和 verification.json。
// rootAbs 与 qaEvidenceRel、qaVerificationRel 拼成写入路径。
// runDoc 交给 atomicWriteJSON 写入 run.json。
// verifyDoc 交给 atomicWriteJSON 写入 verification.json。
func writeQAFiles(rootAbs string, runDoc, verifyDoc map[string]any) error {
	if err := atomicWriteJSON(filepath.Join(rootAbs, filepath.FromSlash(qaEvidenceRel)), runDoc); err != nil {
		return err
	}
	return atomicWriteJSON(filepath.Join(rootAbs, filepath.FromSlash(qaVerificationRel)), verifyDoc)
}

// atomicWriteJSON 先写同目录临时文件再改名。改名失败时删掉目标再改一次，仍失败则删掉临时文件。
// path 的目录会先创建，临时文件再改名到这里。
// v 经 MarshalIndent 写成 JSON，缩进是两个空格。
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

// formatVerifyResult 写出宿主计算的状态、六项检查和失败步骤。六项都是 PASS 但后面还有失败步骤时，说明总状态仍是 FAIL。
// status 写入首行的 status=。
// verifyDoc 读取 checks，按 qaCheckNames 逐项写出。
// results 里 OK 为假的步骤逐条写出。
// evidenceRel 写入 wrote 行。
// verifyRel 写入 wrote 行。
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
