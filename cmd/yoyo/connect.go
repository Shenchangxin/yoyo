// 本文件是 connect 子命令：check。包注释在 main.go。

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// connectCmd 是 connect，目前只有 check。
func connectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Check connected mail, calendar and IM accounts with their real credentials",
	}
	cmd.AddCommand(connectCheckCmd())
	return cmd
}

// connectCheckCmd 是 connect check。每个连接器账号用真实令牌读一次，IM 账号报启用、缺的密钥和最近状态，再看主通道绑没绑。
// --id 只测一个连接器账号；--im-test 往主通道发一条测试消息；--json 打印原始结构。有任何一项没过时退出码不为 0。
func connectCheckCmd() *cobra.Command {
	var only string
	var imTest, asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Read once from every account (no sample fallback); optionally send an IM home test message",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := openApp()
			if err != nil {
				return err
			}
			defer a.Close()
			rep := a.ConnectCheck(strings.TrimSpace(only), imTest)
			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return err
				}
			} else {
				printConnectReport(rep)
			}
			if !rep.OK {
				// 静默掉 cobra 的用法提示：这不是用错了命令，是有账号没连上。
				cmd.SilenceUsage = true
				return fmt.Errorf("some connections failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&only, "id", "", "check only this connector account id")
	cmd.Flags().BoolVar(&imTest, "im-test", false, "send a test message to the IM home channel")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw report")
	return cmd
}

// printConnectReport 一行一项地打印自检结果：OK / FAIL / OFF（停用的 IM 账号），后面跟账号和读到了什么或错在哪。
func printConnectReport(rep app.ConnectReport) {
	fmt.Println("connectors:")
	if len(rep.Connectors) == 0 {
		fmt.Println("  (none connected)")
	}
	for _, r := range rep.Connectors {
		name := fmt.Sprintf("%s [%s/%s] %s", r.ID, r.Provider, r.Kind, r.Label)
		switch {
		case !r.OK:
			fmt.Printf("  FAIL %s\n       %s\n", name, r.Error)
		case r.Note != "" && r.Items == 0:
			fmt.Printf("  OK   %s — %s\n", name, r.Note)
		default:
			fmt.Printf("  OK   %s — %d items\n", name, r.Items)
		}
	}
	fmt.Println("im:")
	if len(rep.IM) == 0 {
		fmt.Println("  (no bot accounts)")
	}
	for _, c := range rep.IM {
		name := fmt.Sprintf("%s [%s] %s", c.ID, c.Platform, c.Label)
		state := c.State
		if state == "" {
			// 运行状态只在跑网关的进程内存里，命令行看不到；这里只核对了启用和密钥。
			state = "secrets present; live status is only visible in the process running the IM gateway (use --im-test)"
		}
		switch {
		case !c.Enabled:
			fmt.Printf("  OFF  %s\n", name)
		case !c.OK && len(c.Missing) > 0:
			fmt.Printf("  FAIL %s\n       missing secrets: %s\n", name, strings.Join(c.Missing, ", "))
		case !c.OK:
			fmt.Printf("  FAIL %s\n       %s %s\n", name, state, c.Detail)
		default:
			fmt.Printf("  OK   %s — %s\n", name, state)
		}
	}
	fmt.Println("im home:")
	switch {
	case !rep.Home.Bound:
		fmt.Println("  not bound (Settings · Proactive → pair code, then send /绑定 <code> from the chat)")
	case !rep.Home.Tested:
		fmt.Printf("  bound to %s %s (add --im-test to send a test message)\n", rep.Home.Platform, rep.Home.Label)
	case rep.Home.TestError != "":
		fmt.Printf("  FAIL test message to %s %s: %s\n", rep.Home.Platform, rep.Home.Label, rep.Home.TestError)
	default:
		fmt.Printf("  OK   test message sent to %s %s\n", rep.Home.Platform, rep.Home.Label)
	}
}
