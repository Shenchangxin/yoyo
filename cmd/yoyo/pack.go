// 本文件是 pack 子命令：list、install、uninstall、enable、update。包注释在 main.go。

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Shenchangxin/yoyo/internal/skillpack"
)

// packCmd 是 pack，子命令按顺序是 list、install、uninstall、enable、update。
func packCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Install and enable skill packs (Superpowers, NovelToGame, and later catalogs)",
	}
	cmd.AddCommand(packListCmd(), packInstallCmd(), packUninstallCmd(), packEnableCmd(), packUpdateCmd())
	return cmd
}

// packListCmd 是 pack list，把当前工作区的 ListPacks 缩进打印。
func packListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List known and installed skill packs",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(a.ListPacks(a.Workspace()))
		},
	}
}

// packInstallCmd 是 pack install，正好一个 id。--path 传给 InstallPack 当本地检出目录，空则走目录安装。
func packInstallCmd() *cobra.Command {
	var local string
	cmd := &cobra.Command{
		Use:   "install [id]",
		Short: "Install a pack (GitHub catalog or --path local checkout)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			m, err := a.InstallPack(args[0], local)
			if err != nil {
				return err
			}
			fmt.Printf("installed %s skills=%d origin=%s\n", m.ID, m.SkillCount, m.Origin.Kind)
			return nil
		},
	}
	cmd.Flags().StringVar(&local, "path", "", "local checkout (repo root or skills/)")
	return cmd
}

// packUninstallCmd 是 pack uninstall，正好一个 id，交给 UninstallPack。
func packUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [id]",
		Short: "Remove an installed pack from ~/.yoyo/packs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			return a.UninstallPack(args[0])
		},
	}
}

// packEnableCmd 是 pack enable，正好一个 id。--scope 默认是工作区；空字符串在运行时也改成工作区。--off 为真时关闭。
func packEnableCmd() *cobra.Command {
	var scope string
	var off bool
	cmd := &cobra.Command{
		Use:   "enable [id]",
		Short: "Enable (or --off) a pack for this workspace or globally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			if scope == "" {
				scope = skillpack.ScopeWorkspace
			}
			return a.EnablePack(args[0], scope, !off)
		},
	}
	cmd.Flags().StringVar(&scope, "scope", skillpack.ScopeWorkspace, "workspace | global | inherit")
	cmd.Flags().BoolVar(&off, "off", false, "disable instead of enable")
	return cmd
}

// packUpdateCmd 是 pack update，正好一个 id，按记录的来源重新拉取。
func packUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [id]",
		Short: "Re-fetch an installed pack from its recorded origin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			m, err := a.UpdatePack(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("updated %s commit=%s skills=%d\n", m.ID, m.Commit, m.SkillCount)
			return nil
		},
	}
}
