// Package main 是 yoyo 命令行。root 挂上这些 cobra 子命令：version、init、run、harness、eval、evolve、session、replay、trace、serve、daemon、update、doctor、logs、pack。
// harness 再挂 list、show、lineage、reveal、checkout、rollback、diff。session 再挂 list、dump。update 再挂 check、apply。pack 再挂 list、install、uninstall、enable、update。
// bundledEvals 按这个顺序找评测目录：相对路径 evals，可执行文件目录下的 evals，该目录上一级的 evals，当前工作目录下的 evals。第一个是目录的转成绝对路径返回；都不是目录就返回字符串 evals。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/remote"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/update"
	"github.com/Shenchangxin/yoyo/internal/version"
)

// main 先处理 NSIS 辅助进程。诊断进程名还空、且不是 worker 时，写成命令行进程。然后执行 root，失败时把错误打到标准错误并以状态 1 退出。
func main() {
	if update.HandleNSISHelper() {
		return
	}
	if os.Getenv(diaglog.EnvProcess) == "" && os.Getenv(diaglog.EnvWorker) != "1" {
		_ = os.Setenv(diaglog.EnvProcess, string(diaglog.ProcessCLI))
	}
	if err := root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// openApp 用环境变量 YOYO_HOME 和 bundledEvals 的目录打开应用。
func openApp() (*app.App, error) {
	evals := bundledEvals()
	return app.Open(os.Getenv("YOYO_HOME"), evals)
}

// bundledEvals 按这个顺序找评测目录：相对路径 evals；能拿到可执行文件时，再加上它所在目录的 evals，以及该目录上一级的 evals；能拿到工作目录时，再加上工作目录下的 evals。
// 第一个 Stat 成功且是目录的候选用 Abs 后返回。Abs 的错误被丢掉。一个都不是目录时返回字符串 evals。
func bundledEvals() string {
	candidates := []string{"evals"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "evals"))
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "evals"))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "evals"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "evals"
}

// root 建立 yoyo 命令，并按这个顺序挂上 version、init、run、harness、eval、evolve、session、replay、trace、serve、daemon、update、doctor、logs、pack。
func root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yoyo",
		Short: "Yoyo self-harnessing local agent",
	}
	cmd.AddCommand(versionCmd(), initCmd(), runCmd(), harnessCmd(), evalCmd(), evolveCmd(), sessionCmd(), replayCmd(), traceCmd(), serveCmd(), daemonCmd(), updateCmd(), doctorCmd(), logsCmd(), packCmd())
	return cmd
}

// versionCmd 是 version，打印 version.Version。
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Version)
		},
	}
}

// initCmd 是 init。打开应用后打印 home 和 active。
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create local Yoyo home and seed the default harness",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			fmt.Println("home", a.Home.Root)
			fmt.Println("active", a.ActiveHash())
			return nil
		},
	}
}

// runCmd 是 run，至少要一个参数，只把 args[0] 交给 Send。--workspace 非空时写入配置。--session 为空时新建会话。会话 id 打到标准错误。
func runCmd() *cobra.Command {
	var workspace, session string
	cmd := &cobra.Command{
		Use:   "run [message]",
		Short: "Run one agent turn in a workspace",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			if workspace != "" {
				a.Config.Workspace = workspace
			}
			if session == "" {
				m, err := a.NewSession(a.Workspace())
				if err != nil {
					return err
				}
				session = m.ID
			}
			out, err := a.Send(cmd.Context(), session, args[0], nil, nil)
			if err != nil {
				return err
			}
			fmt.Println(out)
			fmt.Fprintln(os.Stderr, "session", session)
			return nil
		},
	}
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace directory")
	cmd.Flags().StringVar(&session, "session", "", "existing session id")
	return cmd
}

// harnessCmd 是 harness，子命令按挂上的顺序是 list、show、lineage、reveal、checkout、rollback、diff。
// show 最多一个参数，空则用当前 active。reveal 最多一个参数，空哈希交给 RevealHarness。checkout 正好一个参数，--l3 为真时确认 L3。diff 正好两个参数。
func harnessCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "harness", Short: "Inspect and switch harness snapshots"}
	cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			refs, err := a.ListHarnesses()
			if err != nil {
				return err
			}
			for k, v := range refs {
				fmt.Printf("%s\t%s\n", k, v)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "show",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			hash := a.ActiveHash()
			if len(args) == 1 {
				hash = args[0]
			}
			snap, err := a.LoadSnapshot(hash)
			if err != nil {
				return err
			}
			fmt.Printf("%s model=%s parent=%s note=%s\n", hash, snap.ModelFingerprint, snap.Parent, snap.Note)
			if snap.Parent == "" {
				return nil
			}
			d, err := a.Diff(snap.Parent, hash)
			if err != nil {
				return err
			}
			fmt.Print(d)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "lineage",
		Short: "Print reachable snapshots and what each evolved vs its parent",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			lin, err := a.HarnessLineage()
			if err != nil {
				return err
			}
			if len(lin.Nodes) == 0 {
				fmt.Println("no snapshots")
				return nil
			}
			for _, n := range lin.Nodes {
				fmt.Printf("%s\tparent=%s\trefs=%s\tnote=%s\n", n.Hash, n.Parent, strings.Join(n.Refs, ","), n.Note)
				for _, c := range n.Changes {
					detail := c.Detail
					if detail == "" && (c.From != "" || c.To != "") {
						detail = c.From + " -> " + c.To
					}
					l3 := ""
					if c.L3 {
						l3 = "\tL3"
					}
					fmt.Printf("  %s\t%s\t%s\t%s%s\n", c.Surface, c.Op, c.ID, detail, l3)
				}
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "reveal [hash]",
		Short: "Open the harness CAS store, or a snapshot's decoded artifacts, in the file manager",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			hash := ""
			if len(args) == 1 {
				hash = args[0]
			}
			return a.RevealHarness(hash)
		},
	})
	checkout := &cobra.Command{
		Use:  "checkout",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			l3, _ := cmd.Flags().GetBool("l3")
			return a.CheckoutOpts(args[0], app.CheckoutOpts{ConfirmL3: l3})
		},
	}
	checkout.Flags().Bool("l3", false, "confirm L3 loop/policy change")
	cmd.AddCommand(checkout)
	cmd.AddCommand(&cobra.Command{
		Use: "rollback",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			return a.Rollback()
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "diff",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			d, err := a.Diff(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Print(d)
			return nil
		},
	})
	return cmd
}

// labClient 在 solver 去掉空白并小写后是 heuristic 或 fixture 时返回 HeuristicSolver，是 prompt-sensitive 或 prompt_sensitive 时返回 PromptSensitiveSolver。其他值，包括 live，返回 nil。
func labClient(solver string) runtime.Client {
	switch strings.ToLower(strings.TrimSpace(solver)) {
	case "heuristic", "fixture":
		return runtime.HeuristicSolver{}
	case "prompt-sensitive", "prompt_sensitive":
		return runtime.PromptSensitiveSolver{}
	default:
		return nil
	}
}

// evalCmd 是 eval。整次运行限时 20 分钟。--models 非空时走 BestOfModels，不再看后面的套件开关。
// 否则在 --snapshot 非空且 --best 小于等于 1 时走 RunEvalHash。--best 大于 1 时走 BestOfNMut。剩下走 RunEvalMut。
// 套件开关的优先顺序在 labEvalMut。--solver 默认 live。--snapshot 不移动 refs/active。
func evalCmd() *cobra.Command {
	var best int
	var safety, tb, sealed, transfer, index, behavior bool
	var models []string
	var solver string
	var snapshot string
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Run the active harness against the smoke or sealed eval suite",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()
			client := labClient(solver)
			if len(models) > 0 {
				clients := map[string]runtime.Client{}
				if client != nil {
					clients["*"] = client
				}
				rep, err := a.BestOfModels(ctx, models, clients)
				if err != nil {
					return err
				}
				fmt.Printf("best-of-models winner=%s in %d/%d out %d/%d usd=%.4f\n", rep.BestModel, rep.Best.Metrics.HeldInPass, rep.Best.Metrics.HeldInTotal, rep.Best.Metrics.HeldOutPass, rep.Best.Metrics.HeldOutTotal, rep.Best.USD)
				return nil
			}
			kind, mut := labEvalMut(index, transfer, sealed, tb, safety, behavior)
			if snapshot != "" && best <= 1 {
				out, err := a.RunEvalHash(ctx, client, snapshot, mut)
				if err != nil {
					return err
				}
				printEval(kind, out)
				return nil
			}
			if best > 1 {
				rep, err := a.BestOfNMut(ctx, best, client, mut)
				if err != nil {
					return err
				}
				printEval(fmt.Sprintf("best-of-%d %s", best, kind), rep.Best)
				for i, r := range rep.Reports {
					fmt.Printf("  run %d in=%d/%d out=%d/%d usd=%.4f\n", i+1, r.Metrics.HeldInPass, r.Metrics.HeldInTotal, r.Metrics.HeldOutPass, r.Metrics.HeldOutTotal, r.USD)
				}
				return nil
			}
			out, err := a.RunEvalMut(ctx, client, mut)
			if err != nil {
				return err
			}
			printEval(kind, out)
			return nil
		},
	}
	cmd.Flags().IntVar(&best, "best", 0, "repeat the suite N times and keep the best score")
	cmd.Flags().BoolVar(&safety, "safety", false, "also run the no-escape safety task if the suite omitted it")
	cmd.Flags().BoolVar(&tb, "tb", false, "opt-in Terminal-Bench-style subset (mkdir-note, copy-seed) with repeats=2")
	cmd.Flags().BoolVar(&sealed, "sealed", false, "private 12/8/10 catalog (repeats=2); evolve-set stays off the promote gate")
	cmd.Flags().BoolVar(&transfer, "transfer", false, "run the sealed transfer split only (never fed to Propose)")
	cmd.Flags().BoolVar(&index, "index", false, "Harbor-Index stand-in as transfer only; ids never enter Propose")
	cmd.Flags().BoolVar(&behavior, "behavior", false, "cheap behavior probes (not the default promote gate)")
	cmd.Flags().StringSliceVar(&models, "models", nil, "parallel-model best-of-N (does not change the sealed seed suite)")
	cmd.Flags().StringVar(&solver, "solver", "live", "live model client, or heuristic/prompt-sensitive for fixtures")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "evaluate this harness hash without moving refs/active")
	return cmd
}

// labEvalMut 按参数顺序选套件，先命中的生效：index、transfer、sealed、tb、behavior、safety，都没有则是 smoke 且变换为 nil。
// index 用 eval.ApplyIndexTransferOnly，名字是 index-transfer。transfer 先 ApplySealed，再把 HeldIn 清空、HeldOut 换成原来的 Transfer、Transfer 和 Safety 清空。
// sealed 用 eval.ApplySealed。tb 把 mkdir-note 加进 HeldIn、copy-seed 加进 HeldOut，Repeats 小于 2 时改成 2。
// behavior 把 ID 写成 behavior-probes，HeldIn 换成 eval.BehaviorIDs，HeldOut、Transfer、Safety、EvolveIn 清空，Repeats 为 1。
// safety 在 Safety 里还没有 no-escape 时追加它。
func labEvalMut(index, transfer, sealed, tb, safety, behavior bool) (string, func(*artifact.EvalSuite)) {
	switch {
	case index:
		return "index-transfer", eval.ApplyIndexTransferOnly
	case transfer:
		return "transfer", func(suite *artifact.EvalSuite) {
			eval.ApplySealed(suite)
			suite.HeldIn = nil
			suite.HeldOut = append([]string{}, suite.Transfer...)
			suite.Transfer = nil
			suite.Safety = nil
		}
	case sealed:
		return "sealed", eval.ApplySealed
	case tb:
		return "tb", func(suite *artifact.EvalSuite) {
			suite.HeldIn = append(append([]string{}, suite.HeldIn...), "mkdir-note")
			suite.HeldOut = append(append([]string{}, suite.HeldOut...), "copy-seed")
			if suite.Repeats < 2 {
				suite.Repeats = 2
			}
		}
	case behavior:
		return "behavior", func(suite *artifact.EvalSuite) {
			suite.ID = "behavior-probes"
			suite.HeldIn = append([]string{}, eval.BehaviorIDs...)
			suite.HeldOut = nil
			suite.Transfer = nil
			suite.Safety = nil
			suite.EvolveIn = nil
			suite.Repeats = 1
		}
	case safety:
		return "safety", func(suite *artifact.EvalSuite) {
			already := false
			for _, id := range suite.Safety {
				if id == "no-escape" {
					already = true
					break
				}
			}
			if !already {
				suite.Safety = append(append([]string{}, suite.Safety...), "no-escape")
			}
		}
	default:
		return "smoke", nil
	}
}

// printEval 打印套件名、held-in、held-out、safety_fail、美元、令牌、墙钟、快照前 12 位和指纹，再逐条打印结果。Kind 为空时印成 -。
func printEval(kind string, rep eval.RunReport) {
	fmt.Printf("%s held-in %d/%d held-out %d/%d safety_fail=%d usd=%.4f tokens=%d/%d wall_ms=%d snap=%s fp=%s\n",
		kind, rep.Metrics.HeldInPass, rep.Metrics.HeldInTotal, rep.Metrics.HeldOutPass, rep.Metrics.HeldOutTotal, rep.Metrics.SafetyFail,
		rep.USD, rep.TokensIn, rep.TokensOut, rep.WallMs, shortHash(rep.Snapshot), rep.ModelFingerprint)
	for _, r := range rep.Results {
		k := r.Kind
		if k == "" {
			k = "-"
		}
		fmt.Printf("  %s kind=%s pass=%v usd=%.4f %s\n", r.ID, k, r.Pass, r.USD, r.Error)
	}
}

// shortHash 在长度大于 12 时只留前 12 个字节，否则原样返回。
func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

// updateCmd 是 update，子命令是 check 和 apply。check 把 CheckUpdate 的结果缩进打印。apply 对当前可执行文件调用 ApplyStagedUpdate，ErrQuitNeeded 和 ErrOpenedPackage 不当成失败。
func updateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update", Short: "Check and apply signed updates (never from the agent loop)"}
	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Verify the signed channel JSON and download a newer package if present",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(a.CheckUpdate())
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "apply",
		Short: "Install the downloaded package (human/L3 action; CLI replaces argv[0])",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			exe, _ := os.Executable()
			if err := a.ApplyStagedUpdate(exe); err != nil {
				if !errors.Is(err, update.ErrQuitNeeded) && !errors.Is(err, update.ErrOpenedPackage) {
					return err
				}
			}
			fmt.Println("applied", a.StagingPath())
			return nil
		},
	})
	return cmd
}

// doctorCmd 是 doctor。--export 非空时把脱敏支持包写到该路径并打印路径，否则缩进打印 Doctor。
func doctorCmd() *cobra.Command {
	var dest string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Print workstation health for humans (vault, workspace, harness)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			if dest != "" {
				path, err := a.ExportDiagnostics(dest)
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(a.Doctor())
		},
	}
	cmd.Flags().StringVar(&dest, "export", "", "write a redacted support zip")
	return cmd
}

// logsCmd 是 logs。没有参数或参数不是 path 时按 tail 处理。path 只打印日志文件路径。--limit 默认 80。可按 --level、--component、--session 过滤。
func logsCmd() *cobra.Command {
	var level, component, session string
	var limit int
	cmd := &cobra.Command{
		Use:   "logs [path|tail]",
		Short: "Diagnostic log path or tail (not the journal)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			op := "tail"
			if len(args) > 0 {
				op = strings.ToLower(args[0])
			}
			if op == "path" {
				if a.Log == nil {
					return fmt.Errorf("logger unavailable")
				}
				fmt.Println(a.Log.Path())
				return nil
			}
			if a.Log == nil {
				return fmt.Errorf("logger unavailable")
			}
			recs, err := a.Log.Tail(limit, diaglog.Filter{Level: level, Component: component, SessionID: session})
			if err != nil {
				return err
			}
			fmt.Println(diaglog.FormatRecords(recs))
			return nil
		},
	}
	cmd.Flags().StringVar(&level, "level", "", "filter level")
	cmd.Flags().StringVar(&component, "component", "", "filter component")
	cmd.Flags().StringVar(&session, "session", "", "filter session_id")
	cmd.Flags().IntVar(&limit, "limit", 80, "lines")
	return cmd
}

// evolveCmd 是 evolve。--overlay 为真时交给 runOverlayEvolve，不跑套件循环。否则 rounds 小于等于 0 时改成 1，超时是轮数乘 15 分钟。
// 默认只写 canary。--promote 才移动 refs/active。--k 默认 3。--overlay-k 默认 1。--overlay-mode 默认 agg。--reuse-turn 默认 true。
func evolveCmd() *cobra.Command {
	var k, rounds, baselines, overlayK int
	var promote, sealed, behavior, index, overlay, reuseTurn, overlayLib bool
	var solver, sessionID, task, overlayMode string
	var maxUSD float64
	var maxWall time.Duration
	cmd := &cobra.Command{
		Use:   "evolve",
		Short: "Run Self-Harness cycles (L1 materials). Default writes refs/canary only.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if overlay {
				return runOverlayEvolve(cmd, sessionID, task, solver, reuseTurn, overlayK, overlayMode, overlayLib, promote, sealed, behavior, index, baselines, maxUSD, maxWall)
			}
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			if rounds <= 0 {
				rounds = 1
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rounds)*15*time.Minute)
			defer cancel()
			res, err := a.EvolveWith(ctx, labClient(solver), nil, app.EvolveRun{
				K: k, Rounds: rounds, PromoteActive: promote, Sealed: sealed,
				MaxUSD: maxUSD, MaxWall: maxWall, Behavior: behavior, IndexTransfer: index, Baselines: baselines,
			})
			if err != nil {
				return err
			}
			fmt.Printf("clusters=%d proposals=%d promoted=%s merged=%v active_moved=%v usd=%.4f tokens=%d/%d wall_ms=%d\n",
				len(res.Evidence.Clusters), len(res.Proposals), res.Promoted, res.Merged, res.ActiveMoved,
				res.Spend.USD, res.Spend.TokensIn, res.Spend.TokensOut, res.Spend.WallMs)
			for _, tr := range res.Tried {
				fmt.Printf("  %s accepted=%v hit=%d miss=%d %s\n", tr.Proposal.ID, tr.Accepted, tr.ManifestoHit, tr.ManifestoMiss, tr.Reason)
			}
			if res.Compare != nil {
				fmt.Printf("lab compare usd=%.4f bon=%d iid=%d scs=%d snap=%s fp=%s\n",
					res.Compare.Spend.USD, len(res.Compare.BestOfN), len(res.Compare.IID), len(res.Compare.SCS),
					shortHash(res.Compare.Snapshot), res.Compare.ModelFingerprint)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&k, "k", 3, "proposals per cycle")
	cmd.Flags().IntVar(&rounds, "rounds", 1, "outer Propose→Prove rounds (still canary-only unless --promote)")
	cmd.Flags().BoolVar(&promote, "promote", false, "move refs/active (default: canary + archive only)")
	cmd.Flags().BoolVar(&sealed, "sealed", false, "use the private 12/8/10 catalog")
	cmd.Flags().BoolVar(&behavior, "behavior", false, "add cheap behavior probes as safety")
	cmd.Flags().BoolVar(&index, "index", false, "append Harbor-Index stand-in ids to transfer only")
	cmd.Flags().IntVar(&baselines, "baselines", 0, "also run matched-budget BoN / IID / SCS with N repeats")
	cmd.Flags().Float64Var(&maxUSD, "max-usd", 0, "stop the lab when estimated USD reaches this cap")
	cmd.Flags().DurationVar(&maxWall, "max-wall", 0, "stop the lab at this wall clock budget")
	cmd.Flags().StringVar(&solver, "solver", "live", "live model client, or heuristic/prompt-sensitive for fixture tests")
	cmd.Flags().BoolVar(&overlay, "overlay", false, "session Propose–Probe–Compose; pins this chat, never moves active")
	cmd.Flags().StringVar(&sessionID, "session", "", "session id (required with --overlay)")
	cmd.Flags().StringVar(&task, "task", "", "overlay task; default is the last user message")
	cmd.Flags().BoolVar(&reuseTurn, "reuse-turn", true, "reuse the last finished turn as the overlay base rollout")
	cmd.Flags().IntVar(&overlayK, "overlay-k", 1, "session PPC width or depth; R=4k rollouts, P=3k improver sessions")
	cmd.Flags().StringVar(&overlayMode, "overlay-mode", "agg", "agg (parallel width) or seq (nested depth)")
	cmd.Flags().BoolVar(&overlayLib, "overlay-library", false, "print the per-domain experience INDEX and exit")
	return cmd
}

// runOverlayEvolve 要求 --session 非空，否则错误文案含 evolve --overlay requires --session。
// --promote、--sealed、--behavior、--index、--baselines 大于 0，或命令行改过 --k、--rounds，会交给 OverlayFlagConflict。
// --overlay-library 为真时只打印领域索引。overlayK 先收到 1 到 4。超时是该值乘 15 分钟，maxWall 更大时用 maxWall，再超过 1 小时就截成 1 小时。
func runOverlayEvolve(cmd *cobra.Command, sessionID, task, solver string, reuseTurn bool, overlayK int, overlayMode string, overlayLib, promote, sealed, behavior, index bool, baselines int, maxUSD float64, maxWall time.Duration) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("evolve --overlay requires --session")
	}
	var mix []string
	if promote {
		mix = append(mix, "--promote")
	}
	if sealed {
		mix = append(mix, "--sealed")
	}
	if behavior {
		mix = append(mix, "--behavior")
	}
	if index {
		mix = append(mix, "--index")
	}
	if baselines > 0 {
		mix = append(mix, "--baselines")
	}
	if cmd.Flags().Changed("k") {
		mix = append(mix, "--k")
	}
	if cmd.Flags().Changed("rounds") {
		mix = append(mix, "--rounds")
	}
	if err := app.OverlayFlagConflict(mix...); err != nil {
		return err
	}
	a, err := openApp()
	if err != nil {
		return err
	}
	defer a.Close()
	if overlayLib {
		view, err := a.OverlayLibrary(sessionID)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "domain %s entries %d\n", view.Domain, len(view.Entries))
		if view.Index != "" {
			fmt.Fprintln(cmd.OutOrStdout(), view.Index)
		}
		return nil
	}
	ok := overlayK
	if ok < 1 {
		ok = 1
	}
	if ok > 4 {
		ok = 4
	}
	timeout := 15 * time.Minute * time.Duration(ok)
	if maxWall > timeout {
		timeout = maxWall
	}
	if timeout > time.Hour {
		timeout = time.Hour
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	res, err := a.OverlaySession(ctx, app.OverlayRun{
		SessionID: sessionID,
		Task:      task,
		ReuseTurn: reuseTurn,
		K:         overlayK,
		Mode:      overlayMode,
		MaxUSD:    maxUSD,
		MaxWall:   maxWall,
		Client:    labClient(solver),
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "overlay adopted=%v kept_base=%v hash=%s parent=%s compose=%s k=%d R=%d P=%d usd=%.4f library=%s\n",
		res.Adopted, res.KeptBase, shortHash(res.Hash), shortHash(res.Parent), res.Compose, res.K, res.R, res.P, res.Spend.USD, res.LibraryStatus)
	if res.Adopted {
		fmt.Fprintf(cmd.ErrOrStderr(), "This chat is pinned to %s. Checkout still owns production.\n", shortHash(res.Hash))
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), "No contrast. Active unchanged.")
	}
	return nil
}

// sessionCmd 是 session，子命令是 list 和 dump。
func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Look up a session by id (full trajectory + artifacts)",
	}
	cmd.AddCommand(sessionListCmd(), sessionDumpCmd())
	return cmd
}

// sessionListCmd 是 session list。每行打印 id、事件数、溢出制品数、标题和轨迹路径。标题为空时印成 -。
func sessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List session ids and on-disk trajectory/spill paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			for _, e := range a.ListSessionIndex() {
				title := e.Title
				if title == "" {
					title = "-"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%d\t%s\t%s\n", e.ID, e.Events, e.SpillArtifacts, title, e.TrajectoryPath)
			}
			return nil
		},
	}
}

// sessionDumpCmd 是 session dump，正好一个会话参数。--out 非空时把缩进 JSON 写到该路径，父目录按 0o755 创建，文件权限 0o644。--json 为真时再把同一份打到标准输出，且不转义 HTML。
func sessionDumpCmd() *cobra.Command {
	var asJSON bool
	var outPath string
	cmd := &cobra.Command{
		Use:   "dump [session]",
		Short: "Write the full trajectory and artifact bodies for a session id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			dump, err := a.DumpSession(args[0])
			if err != nil {
				return err
			}
			if outPath != "" {
				b, err := json.MarshalIndent(dump, "", "  ")
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(outPath, b, 0o644); err != nil {
					return err
				}
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				enc.SetEscapeHTML(false)
				return enc.Encode(dump)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "session_id: %s\n", dump.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "home: %s\n", dump.Home)
			fmt.Fprintf(cmd.OutOrStdout(), "dump: %s\n", dump.Paths.Dump)
			fmt.Fprintf(cmd.OutOrStdout(), "trajectory: %s\n", dump.Paths.Trajectory)
			if dump.Paths.Meta != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "meta: %s\n", dump.Paths.Meta)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "spill_dir: %s\n", dump.Paths.SpillDir)
			if dump.Paths.ContextDir != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "context_dir: %s\n", dump.Paths.ContextDir)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "events: %d\n", len(dump.Trajectory))
			fmt.Fprintf(cmd.OutOrStdout(), "artifacts: %d\n", len(dump.Artifacts))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the full dump on stdout")
	cmd.Flags().StringVar(&outPath, "out", "", "also write the dump JSON to this path")
	return cmd
}

// replayCmd 是 replay，正好一个参数。先 ResolveSessionID，再按时间、类型、来源和载荷逐行打印轨迹。
func replayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "replay [session]",
		Short: "Print a recorded trajectory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			id, err := a.ResolveSessionID(args[0])
			if err != nil {
				return err
			}
			evs, err := a.ReadTrace(id)
			if err != nil {
				return err
			}
			for _, ev := range evs {
				fmt.Printf("%s %s %s %v\n", ev.TS.Format(time.RFC3339), ev.Type, ev.Source, ev.Payload)
			}
			return nil
		},
	}
}

// traceCmd 是 trace，正好一个参数。ResolveSessionID 之后把 SessionTrace 缩进打印成 JSON。
func traceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trace [session]",
		Short: "Print a session trace (calls, trajectory, artifacts)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			id, err := a.ResolveSessionID(args[0])
			if err != nil {
				return err
			}
			tr, err := a.SessionTrace(id)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(tr)
		},
	}
}

// serveCmd 是 serve。--addr 默认 127.0.0.1:3080。--stdio 为真时走 api.ServeRPC，不再 ListenAndServe。静态文件来自 frontendStatic。remote.Bind 成功且节点非空时把静态目录交给它。
func serveCmd() *cobra.Command {
	var addr string
	var stdio bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the local HTTP API and UI (or JSON-RPC on stdio)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			static := frontendStatic()
			if node, err := remote.Bind(a); err == nil && node != nil {
				node.UseStatic(static)
				defer node.Close()
			}
			if stdio {
				return api.ServeRPC(cmd.Context(), a, os.Stdin, os.Stdout)
			}
			fmt.Println("listening", addr, "active", a.ActiveHash())
			return http.ListenAndServe(addr, api.Handler(a, static))
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:3080", "listen address")
	cmd.Flags().BoolVar(&stdio, "stdio", false, "JSON-RPC on stdin/stdout instead of HTTP")
	return cmd
}

// daemonCmd 是 daemon，和 serve 用同一套 HTTP 处理。--addr 默认 127.0.0.1:3080。没有 --stdio。
func daemonCmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Always-on local daemon (same JSON-RPC/HTTP as serve; sleeps skip jobs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			static := frontendStatic()
			if node, err := remote.Bind(a); err == nil && node != nil {
				node.UseStatic(static)
				defer node.Close()
			}
			fmt.Println("yoyo daemon", addr, "active", a.ActiveHash())
			fmt.Println("honest always-on: this machine must stay awake; sleep skips scheduled jobs")
			return http.ListenAndServe(addr, api.Handler(a, static))
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:3080", "listen address")
	return cmd
}

// mustExe 返回当前可执行文件路径。os.Executable 失败时返回 .。
func mustExe() string {
	e, err := os.Executable()
	if err != nil {
		return "."
	}
	return e
}

// frontendStatic 先看相对路径 frontend/dist，再看可执行文件目录下的 frontend/dist。第一个是目录的交给 FileServer。都不是目录时返回 nil。
func frontendStatic() http.Handler {
	for _, dir := range []string{"frontend/dist", filepath.Join(filepath.Dir(mustExe()), "frontend", "dist")} {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return http.FileServer(http.Dir(dir))
		}
	}
	return nil
}
