// Package eval 按 Harbor 布局加载任务、在隔离工作区里跑智能体，并用验证器打分。
// copyDir 跳过 tests/，智能体工作区拿不到评分脚本。runTask 只把循环副本的 AllowLLMCompact 写成 false；RunRequest 不填 ModelWindow，所以是 0；CompactionTokens 保持调用方循环上的原值。
// ShouldPromote 要求 held-in 与 held-out 的差值都大于等于 0，且至少一边大于 0，不读 SafetyFail。Promote 在 SafetyFail 大于 0 时先拒绝，然后再调用 ShouldPromote。
package eval

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	rt "github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Task 是一道 Harbor 任务：目录名、说明和可选的 task.toml 时限。
type Task struct {
	ID          string
	Dir         string
	Instruction string
	// Timeout 来自 task.toml 的 timeout_minutes，缺省 2 分钟。Engine.Run 不用它，而用套件的 TimeoutSec。
	Timeout     time.Duration
	VerifierCmd []string
}

// RepeatAttempt 记录一次重复的通过、会话编号和这一次相对开跑前的用量。
type RepeatAttempt struct {
	Pass      bool    `json:"pass"`
	SessionID string  `json:"session_id,omitempty"`
	TokensIn  int     `json:"tokens_in,omitempty"`
	TokensOut int     `json:"tokens_out,omitempty"`
	USD       float64 `json:"usd,omitempty"`
	WallMs    int64   `json:"wall_ms,omitempty"`
}

// TaskResult 是一道题在全部重复之后的汇总。Pass 由通过次数决定，不由运行错误单独决定。
type TaskResult struct {
	ID          string `json:"id"`
	Pass        bool   `json:"pass"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
	Repeats     int    `json:"repeats,omitempty"`
	RepeatsPass int    `json:"repeats_pass,omitempty"`
	Isolate     string `json:"isolate,omitempty"`
	// Kind 按 safety、held_out、held_in 的顺序取最先命中的一种，都不属于时为空。
	Kind      string          `json:"kind,omitempty"`
	TokensIn  int             `json:"tokens_in,omitempty"`
	TokensOut int             `json:"tokens_out,omitempty"`
	USD       float64         `json:"usd,omitempty"`
	WallMs    int64           `json:"wall_ms,omitempty"`
	Attempts  []RepeatAttempt `json:"attempts,omitempty"`
}

// RunReport 是一套题的结果。单题失败写在 Results 里，Run 返回的 error 仍是 nil。
type RunReport struct {
	Suite            string          `json:"suite"`
	Snapshot         string          `json:"snapshot"`
	ModelFingerprint string          `json:"model_fingerprint,omitempty"`
	Metrics          Metrics         `json:"metrics"`
	Results          []TaskResult    `json:"results"`
	HeldIn           map[string]bool `json:"held_in"`
	HeldOut          map[string]bool `json:"held_out"`
	TokensIn         int             `json:"tokens_in,omitempty"`
	TokensOut        int             `json:"tokens_out,omitempty"`
	USD              float64         `json:"usd,omitempty"`
	WallMs           int64           `json:"wall_ms,omitempty"`
}

// Engine 从 SuitesRoot 找任务目录。套件自带 TaskDir 时以套件为准。
type Engine struct {
	SuitesRoot string
}

// NewEngine 记下套件根目录。
func NewEngine(suitesRoot string) *Engine {
	return &Engine{SuitesRoot: suitesRoot}
}

// RunOpts 是一次套件运行的材料。AllowShell 和 RevealHeldOut 本包不读。
type RunOpts struct {
	Suite         artifact.EvalSuite
	Snapshot      artifact.HarnessSnapshot
	Hash          string
	Client        rt.Client
	Loop          artifact.LoopPreset
	Fragments     []artifact.PromptFragment
	Playbook      artifact.Playbook
	Skills        []artifact.Skill
	Trace         *trace.Store
	SessionID     string
	Model         string
	AllowShell    bool
	RevealHeldOut bool
	// IsolateRoot 非空时，每个任务经 runtime.IsolateWorkspace 进入分离工作区，再盖上 Harbor 任务，隔离标记为 git-worktree。
	// 为空时用临时复制目录，标记为 copy。两条路径都经 copyDir，智能体都拿不到 tests/。
	IsolateRoot string
	Meter       *rt.Meter
}

// Run 按 held-in、held-out，再补上尚未出现的 safety 编号跑题。
// Repeats 小于等于 0 时按 1 次。TimeoutSec 小于等于 0 时用 2 分钟，不读 Task.Timeout。
// 通过次数必须大于 Repeats/2；Repeats 为 1 时还接受最后一轮本身通过。
// 安全题加载失败或不通过时 SafetyFail 加一。Meter 为 nil 时换成空表。
func (e *Engine) Run(ctx context.Context, opts RunOpts) (RunReport, error) {
	rep := RunReport{
		Suite:            opts.Suite.ID,
		Snapshot:         opts.Hash,
		ModelFingerprint: opts.Snapshot.ModelFingerprint,
		HeldIn:           map[string]bool{},
		HeldOut:          map[string]bool{},
	}
	if opts.Meter == nil {
		opts.Meter = &rt.Meter{}
	}
	runStart := time.Now()
	heldOutSet := map[string]bool{}
	for _, id := range opts.Suite.HeldOut {
		heldOutSet[id] = true
	}
	ids := append([]string{}, opts.Suite.HeldIn...)
	ids = append(ids, opts.Suite.HeldOut...)
	safetySet := map[string]bool{}
	for _, id := range opts.Suite.Safety {
		safetySet[id] = true
		already := false
		for _, x := range ids {
			if x == id {
				already = true
				break
			}
		}
		if !already {
			ids = append(ids, id)
		}
	}
	repeats := opts.Suite.Repeats
	if repeats <= 0 {
		repeats = 1
	}
	timeout := time.Duration(opts.Suite.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	root := opts.Suite.TaskDir
	if root == "" {
		root = e.SuitesRoot
	}
	for _, id := range ids {
		task, err := LoadHarborTask(e.resolveTaskDir(root, id))
		if err != nil {
			rep.Results = append(rep.Results, TaskResult{ID: id, Error: err.Error(), Kind: taskKind(id, heldOutSet, safetySet, opts.Suite.HeldIn)})
			if safetySet[id] {
				rep.Metrics.SafetyFail++
			}
			continue
		}
		passCount := 0
		var last TaskResult
		var attempts []RepeatAttempt
		for i := 0; i < repeats; i++ {
			beforeIn, beforeOut, beforeUSD := opts.Meter.Totals()
			t0 := time.Now()
			sid := repeatSession(opts.SessionID, id, i)
			last = e.runTask(ctx, task, timeout, opts, sid)
			in, out, usd := opts.Meter.Totals()
			att := RepeatAttempt{
				Pass: last.Pass, SessionID: sid,
				TokensIn: in - beforeIn, TokensOut: out - beforeOut, USD: usd - beforeUSD,
				WallMs: time.Since(t0).Milliseconds(),
			}
			attempts = append(attempts, att)
			last.TokensIn += att.TokensIn
			last.TokensOut += att.TokensOut
			last.USD += att.USD
			last.WallMs += att.WallMs
			if last.Pass {
				passCount++
			}
		}
		last.Pass = passCount > repeats/2 || (repeats == 1 && last.Pass)
		last.Repeats = repeats
		last.RepeatsPass = passCount
		last.Kind = taskKind(id, heldOutSet, safetySet, opts.Suite.HeldIn)
		last.Attempts = attempts
		rep.Results = append(rep.Results, last)
		rep.TokensIn += last.TokensIn
		rep.TokensOut += last.TokensOut
		rep.USD += last.USD
		if safetySet[id] && !last.Pass {
			rep.Metrics.SafetyFail++
		}
		if heldOutSet[id] {
			rep.HeldOut[id] = last.Pass
			rep.Metrics.HeldOutTotal++
			if last.Pass {
				rep.Metrics.HeldOutPass++
			}
		} else if containsID(opts.Suite.HeldIn, id) {
			rep.HeldIn[id] = last.Pass
			rep.Metrics.HeldInTotal++
			if last.Pass {
				rep.Metrics.HeldInPass++
			}
		}
	}
	rep.WallMs = time.Since(runStart).Milliseconds()
	return rep, nil
}

// repeatSession 把基会话、题号和重复下标拼成会话编号。基会话为空时用 eval。
func repeatSession(base, taskID string, i int) string {
	if base == "" {
		base = "eval"
	}
	return fmt.Sprintf("%s~%s~r%d", base, taskID, i)
}

// runTask 在准备好的工作区里跑一轮，再用验证器决定 Pass。
// 关掉模型摘要：只把这份循环副本的 AllowLLMCompact 写成 false。下面的 RunRequest 不填 ModelWindow，所以是 0。CompactionTokens 不在这里赋值，调用方循环上是多少就传多少。
func (e *Engine) runTask(ctx context.Context, task Task, timeout time.Duration, opts RunOpts, sessionID string) TaskResult {
	work, cleanup, isolate, err := prepareWork(task.Dir, opts.IsolateRoot)
	if err != nil {
		return TaskResult{ID: task.ID, Error: err.Error()}
	}
	defer cleanup()
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	caps := capability.NewBroker(capability.AutoPolicy{
		Allow: []capability.Level{capability.ReadWorkspace, capability.WriteWorkspace, capability.Shell},
	}, func(context.Context, capability.Request) (capability.Decision, error) {
		return capability.Always, nil
	})
	skillBodies := map[string]string{}
	for _, s := range opts.Skills {
		skillBodies[s.Name] = s.Body
	}
	advertised := append([]string(nil), opts.Snapshot.Tools...)
	if len(opts.Snapshot.EvalTools) > 0 {
		advertised = append([]string(nil), opts.Snapshot.EvalTools...)
	}
	tools := &rt.WorkspaceTools{
		Workspace: work, Home: work, SessionID: sessionID, Caps: caps, Skills: skillBodies,
		Advertised: advertised,
	}
	loop := opts.Loop
	loop.AllowLLMCompact = false
	_, runErr := rt.Run(tctx, rt.RunRequest{
		SessionID:        sessionID,
		TaskID:           task.ID,
		User:             task.Instruction,
		Workspace:        work,
		Harness:          opts.Snapshot,
		HarnessHash:      opts.Hash,
		ModelFingerprint: opts.Snapshot.ModelFingerprint,
		Model:            opts.Model,
		Loop:             loop,
		Fragments:        opts.Fragments,
		Playbook:         opts.Playbook,
		Skills:           opts.Skills,
		Tools:            tools,
		Client:           opts.Client,
		Trace:            opts.Trace,
		Meter:            opts.Meter,
	})
	res := TaskResult{ID: task.ID, Output: "", Isolate: isolate}
	if runErr != nil {
		res.Error = runErr.Error()
	}
	ok, out, verr := runVerifier(work, task)
	res.Output = out
	if verr != nil {
		res.Error = strings.TrimSpace(res.Error + " " + verr.Error())
	}
	res.Pass = ok
	return res
}

// LoadHarborTask 读 instruction.md，题号取目录名。
// task.toml 的 timeout_minutes 大于 0 时写入 Task.Timeout，否则为 2 分钟。Engine.Run 不读这个字段。
func LoadHarborTask(dir string) (Task, error) {
	inst, err := os.ReadFile(filepath.Join(dir, "instruction.md"))
	if err != nil {
		return Task{}, err
	}
	task := Task{
		ID:          filepath.Base(dir),
		Dir:         dir,
		Instruction: string(inst),
		Timeout:     2 * time.Minute,
	}
	if b, err := os.ReadFile(filepath.Join(dir, "task.toml")); err == nil {
		var cfg struct {
			TimeoutMin float64 `toml:"timeout_minutes"`
		}
		_ = toml.Unmarshal(b, &cfg)
		if cfg.TimeoutMin > 0 {
			task.Timeout = time.Duration(cfg.TimeoutMin * float64(time.Minute))
		}
	}
	return task, nil
}

// runVerifier 先问个人验证器。未接手时读任务目录 tests/expect.toml，再按平台顺序找测试脚本。
// 脚本在智能体工作区里执行，但文件来自任务目录的 tests/，不来自工作区。
// 存在 environment/Dockerfile 且本机有 docker 时先试容器；容器命令 err 非 nil 时退回本机脚本。
func runVerifier(work string, task Task) (bool, string, error) {
	if ok, out, err, handled := runPersonalVerifier(work, task); handled {
		return ok, out, err
	}
	expect := filepath.Join(task.Dir, "tests", "expect.toml")
	if _, err := os.Stat(expect); err == nil {
		return runExpect(work, expect)
	}
	order := []string{"test.ps1", "test.bat", "test.sh"}
	if runtime.GOOS != "windows" {
		order = []string{"test.sh", "test.ps1", "test.bat"}
	}
	var script string
	for _, name := range order {
		c := filepath.Join(task.Dir, "tests", name)
		if _, err := os.Stat(c); err == nil {
			script = c
			break
		}
	}
	if script == "" {
		return false, "", fmt.Errorf("no verifier")
	}
	var cmd *exec.Cmd
	switch strings.ToLower(filepath.Ext(script)) {
	case ".ps1":
		cmd = exec.Command("powershell", "-NoProfile", "-File", script)
	case ".bat", ".cmd":
		cmd = exec.Command(script)
	default:
		if runtime.GOOS == "windows" {
			cmd = exec.Command("bash", script)
			if _, err := exec.LookPath("bash"); err != nil {
				cmd = exec.Command("powershell", "-NoProfile", "-File", strings.TrimSuffix(script, ".sh")+".ps1")
				if _, err := os.Stat(strings.TrimSuffix(script, ".sh") + ".ps1"); err != nil {
					return runGoishVerifier(work, script)
				}
			}
		} else {
			cmd = exec.Command("bash", script)
		}
	}
	cmd.Dir = work
	quiet(cmd)
	if df := dockerfileOf(task.Dir); df != "" && dockerAvailable() {
		if ok, out, err := runVerifierDocker(work, task, script); err == nil {
			return ok, out, err
		}
	}
	out, err := cmd.CombinedOutput()
	return err == nil, string(out), err
}

// taskKind 给题目贴种类：安全优先，其次 held-out，再其次 held-in。
func taskKind(id string, heldOut, safety map[string]bool, heldIn []string) string {
	if safety[id] {
		return "safety"
	}
	if heldOut[id] {
		return "held_out"
	}
	if containsID(heldIn, id) {
		return "held_in"
	}
	return ""
}

// runGoishVerifier 在 Windows 上没有 bash、也没有同名 ps1 时，只认脚本里的 test -f 或 [ -f。
// 命中的相对路径相对工作区检查是否存在。没有这类检查时返回无法在本平台执行。
func runGoishVerifier(work, script string) (bool, string, error) {
	b, err := os.ReadFile(script)
	if err != nil {
		return false, "", err
	}
	content := string(b)
	if strings.Contains(content, "test -f") || strings.Contains(content, "[ -f") {
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "-f ") {
				parts := strings.Fields(line)
				for i, p := range parts {
					if p == "-f" && i+1 < len(parts) {
						path := strings.Trim(parts[i+1], "`\"'")
						path = strings.TrimSuffix(path, "]")
						if !filepath.IsAbs(path) {
							path = filepath.Join(work, path)
						}
						if _, err := os.Stat(path); err != nil {
							return false, "missing " + path, err
						}
					}
				}
			}
		}
		return true, "ok", nil
	}
	return false, content, fmt.Errorf("cannot execute verifier on this platform")
}

// copyDir 把任务目录拷进智能体工作区，遇到 tests 目录或其下的路径就跳过。工作区里因此没有 tests/。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "tests" || strings.HasPrefix(rel, "tests"+string(os.PathSeparator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode())
	})
}

// containsID 报告编号是否在列表里。
func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// prepareWork 准备智能体工作区。IsolateRoot 非空时调用 runtime.IsolateWorkspace，再盖上任务文件，标记为 git-worktree。
// git worktree 失败时 IsolateWorkspace 会改成复制该根目录，标记仍然是 git-worktree。
// IsolateRoot 为空时创建前缀为 yoyo-eval- 的临时目录，标记为 copy。两条路径都经 copyDir，都不会带上 tests/。
func prepareWork(taskDir, isolateRoot string) (string, func(), string, error) {
	nop := func() {}
	if isolateRoot != "" {
		work, cleanup, err := rt.IsolateWorkspace(isolateRoot)
		if err != nil {
			return "", nop, "", err
		}
		if err := copyDir(taskDir, work); err != nil {
			cleanup()
			return "", nop, "", err
		}
		return work, cleanup, "git-worktree", nil
	}
	work, err := os.MkdirTemp("", "yoyo-eval-*")
	if err != nil {
		return "", nop, "", err
	}
	cleanup := func() { _ = os.RemoveAll(work) }
	if err := copyDir(taskDir, work); err != nil {
		cleanup()
		return "", nop, "", err
	}
	return work, cleanup, "copy", nil
}
