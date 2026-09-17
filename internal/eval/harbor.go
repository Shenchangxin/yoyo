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

type Task struct {
	ID          string
	Dir         string
	Instruction string
	Timeout     time.Duration
	VerifierCmd []string
}

type TaskResult struct {
	ID          string `json:"id"`
	Pass        bool   `json:"pass"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
	Repeats     int    `json:"repeats,omitempty"`
	RepeatsPass int    `json:"repeats_pass,omitempty"`
	Isolate     string `json:"isolate,omitempty"`
	Kind        string `json:"kind,omitempty"` // held_in | held_out | safety
}

type RunReport struct {
	Suite    string          `json:"suite"`
	Snapshot string          `json:"snapshot"`
	Metrics  Metrics         `json:"metrics"`
	Results  []TaskResult    `json:"results"`
	HeldIn   map[string]bool `json:"held_in"`
	HeldOut  map[string]bool `json:"held_out"`
}

type Engine struct {
	SuitesRoot string
}

func NewEngine(suitesRoot string) *Engine {
	return &Engine{SuitesRoot: suitesRoot}
}

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
	// IsolateRoot, when a git repo, makes each task run in a detached
	// worktree of that repo with the Harbor task overlaid. Eval copies
	// remain the default when this is empty.
	IsolateRoot string
}

func (e *Engine) Run(ctx context.Context, opts RunOpts) (RunReport, error) {
	rep := RunReport{
		Suite:    opts.Suite.ID,
		Snapshot: opts.Hash,
		HeldIn:   map[string]bool{},
		HeldOut:  map[string]bool{},
	}
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
		for i := 0; i < repeats; i++ {
			last = e.runTask(ctx, task, timeout, opts)
			if last.Pass {
				passCount++
			}
		}
		last.Pass = passCount > repeats/2 || (repeats == 1 && last.Pass)
		last.Repeats = repeats
		last.RepeatsPass = passCount
		last.Kind = taskKind(id, heldOutSet, safetySet, opts.Suite.HeldIn)
		rep.Results = append(rep.Results, last)
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
	return rep, nil
}

func (e *Engine) runTask(ctx context.Context, task Task, timeout time.Duration, opts RunOpts) TaskResult {
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
		Workspace: work, SessionID: opts.SessionID, Caps: caps, Skills: skillBodies,
		Advertised: advertised,
	}
	loop := opts.Loop
	loop.AllowLLMCompact = false
	_, runErr := rt.Run(tctx, rt.RunRequest{
		SessionID:        opts.SessionID,
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

func runVerifier(work string, task Task) (bool, string, error) {
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
		for _, root := range []string{filepath.Join(work, "tests"), filepath.Join(task.Dir, "tests")} {
			c := filepath.Join(root, name)
			if _, err := os.Stat(c); err == nil {
				script = c
				break
			}
		}
		if script != "" {
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
	if df := dockerfileOf(task.Dir); df != "" && dockerAvailable() {
		if ok, out, err := runVerifierDocker(work, task, script); err == nil {
			return ok, out, err
		}
	}
	out, err := cmd.CombinedOutput()
	return err == nil, string(out), err
}

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

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
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

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

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
