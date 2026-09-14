package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
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
	cmd.AddCommand(versionCmd(), initCmd(), runCmd(), harnessCmd(), evalCmd(), evolveCmd(), replayCmd(), serveCmd())
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
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "checkout",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			return a.Checkout(args[0])
		},
	})
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

func evalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "eval",
		Short: "Run the active harness against the sealed eval suite",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			rep, err := a.RunEval(ctx, runtime.HeuristicSolver{})
			if err != nil {
				return err
			}
			fmt.Printf("held-in %d/%d held-out %d/%d\n", rep.Metrics.HeldInPass, rep.Metrics.HeldInTotal, rep.Metrics.HeldOutPass, rep.Metrics.HeldOutTotal)
			for _, r := range rep.Results {
				fmt.Printf("  %s pass=%v %s\n", r.ID, r.Pass, r.Error)
			}
			return nil
		},
	}
}

func evolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "evolve",
		Short: "Run one Self-Harness cycle (L1 materials)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			res, err := a.EvolveOnce(ctx, runtime.PromptSensitiveSolver{}, map[string]string{"write-hello": "missing_artifact"})
			if err != nil {
				return err
			}
			fmt.Printf("clusters=%d proposals=%d promoted=%s\n", len(res.Evidence.Clusters), len(res.Proposals), res.Promoted)
			for _, t := range res.Tried {
				fmt.Printf("  %s accepted=%v %s\n", t.Proposal.ID, t.Accepted, t.Reason)
			}
			return nil
		},
	}
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

func serveCmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the local HTTP API and UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
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
	return cmd
}

func mustExe() string {
	e, err := os.Executable()
	if err != nil {
		return "."
	}
	return e
}
