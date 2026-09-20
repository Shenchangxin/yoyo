package main

import (
	"context"
	"encoding/json"
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
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/version"
)

func main() {
	if err := root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func openApp() (*app.App, error) {
	evals := bundledEvals()
	return app.Open(os.Getenv("YOYO_HOME"), evals)
}

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

func root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yoyo",
		Short: "Yoyo self-harnessing local agent",
	}
	cmd.AddCommand(versionCmd(), initCmd(), runCmd(), harnessCmd(), evalCmd(), evolveCmd(), replayCmd(), traceCmd(), serveCmd(), daemonCmd(), updateCmd(), doctorCmd())
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Version)
		},
	}
}

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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			out, err := a.Send(ctx, session, args[0], nil, nil)
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

func evalCmd() *cobra.Command {
	var best int
	var safety, tb, sealed, transfer, index, behavior bool
	var models []string
	var solver string
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
	return cmd
}

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

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update", Short: "Stage and apply signed binary updates (never from the agent loop)"}
	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Verify the update channel and download into staging if newer",
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
		Short: "Replace the current binary with updates/yoyo.staging (human/L3 action)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			exe, _ := os.Executable()
			if err := a.ApplyStagedUpdate(exe); err != nil {
				return err
			}
			fmt.Println("applied", a.StagingPath())
			return nil
		},
	})
	return cmd
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Print workstation health for humans (vault, workspace, harness)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(a.Doctor())
		},
	}
}

func evolveCmd() *cobra.Command {
	var k, rounds, baselines int
	var promote, sealed, behavior, index bool
	var solver string
	var maxUSD float64
	var maxWall time.Duration
	cmd := &cobra.Command{
		Use:   "evolve",
		Short: "Run Self-Harness cycles (L1 materials). Default writes refs/canary only.",
		RunE: func(cmd *cobra.Command, args []string) error {
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
	return cmd
}

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
			evs, err := a.Trajectory(args[0])
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
			tr, err := a.SessionTrace(args[0])
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(tr)
		},
	}
}

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
			if stdio {
				return api.ServeRPC(cmd.Context(), a, os.Stdin, os.Stdout)
			}
			var static http.Handler
			for _, dir := range []string{"frontend/dist", filepath.Join(filepath.Dir(mustExe()), "frontend", "dist")} {
				if st, err := os.Stat(dir); err == nil && st.IsDir() {
					static = http.FileServer(http.Dir(dir))
					break
				}
			}
			fmt.Println("listening", addr, "active", a.ActiveHash())
			return http.ListenAndServe(addr, api.Handler(a, static))
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:3080", "listen address")
	cmd.Flags().BoolVar(&stdio, "stdio", false, "JSON-RPC on stdin/stdout instead of HTTP")
	return cmd
}

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
			fmt.Println("yoyo daemon", addr, "active", a.ActiveHash())
			fmt.Println("honest always-on: this machine must stay awake; sleep skips scheduled jobs")
			var static http.Handler
			for _, dir := range []string{"frontend/dist", filepath.Join(filepath.Dir(mustExe()), "frontend", "dist")} {
				if st, err := os.Stat(dir); err == nil && st.IsDir() {
					static = http.FileServer(http.Dir(dir))
					break
				}
			}
			return http.ListenAndServe(addr, api.Handler(a, static))
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:3080", "listen address")
	return cmd
}

func mustExe() string {
	e, err := os.Executable()
	if err != nil {
		return "."
	}
	return e
}
