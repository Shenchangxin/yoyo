// connect_check.go 汇总一次连接自检，给 yoyo connect check 用：
//   - 每个连接器账号用真实令牌读一次（connector.Broker.Probe），不退回本地样例；
//   - 每个 IM 机器人账号报是否启用、保险库里缺哪几项密钥、最近一次运行状态；
//   - IM 主通道绑没绑上；imTest 为真时往主通道发一条测试消息，同步等结果。
//
// 命令行是 tool 角色，不领头，IM 网关不在这个进程里跑。运行状态（im.Status）只记在跑网关的那个进程的内存里、不落盘，
// 所以从命令行看 State 一般是空的，这里只能核对「启用了没有、密钥齐不齐」；要看连上没有，看桌面设置里的状态，或者用 --im-test。
// 发测试消息走 HTTP 接口的平台（Telegram、飞书、Slack、Discord 等）在这里也能发；OneBot 这类要靠已有长连接的发不出去。
// 只读，除了 imTest 那一条消息，不发信、不写日程。

package app

import (
	"sort"

	"github.com/Shenchangxin/yoyo/internal/connector"
)

// IMCheck 是一个 IM 机器人账号的自检结果。Missing 是保险库里还没填的密钥名（不带原文）。
type IMCheck struct {
	ID       string   `json:"id"`
	Platform string   `json:"platform"`
	Label    string   `json:"label"`
	Enabled  bool     `json:"enabled"`
	Missing  []string `json:"missing,omitempty"`
	State    string   `json:"state,omitempty"`
	Detail   string   `json:"detail,omitempty"`
	OK       bool     `json:"ok"`
}

// IMHomeCheck 是主通道的自检结果。Tested 为真时 TestError 为空表示测试消息发出去了。
type IMHomeCheck struct {
	Bound     bool   `json:"bound"`
	Platform  string `json:"platform,omitempty"`
	Label     string `json:"label,omitempty"`
	Tested    bool   `json:"tested"`
	TestError string `json:"test_error,omitempty"`
}

// ConnectReport 是一次连接自检。OK 在所有连接器、所有启用的 IM 账号和（要求发测试时的）主通道都通过时为真。
type ConnectReport struct {
	OK         bool                    `json:"ok"`
	Connectors []connector.ProbeResult `json:"connectors"`
	IM         []IMCheck               `json:"im"`
	Home       IMHomeCheck             `json:"im_home"`
}

// ConnectCheck 跑一次连接自检。only 非空时只测这个连接器账号（IM 部分照常报）；imTest 为真时往主通道发一条测试消息。
func (a *App) ConnectCheck(only string, imTest bool) ConnectReport {
	out := ConnectReport{OK: true, Connectors: []connector.ProbeResult{}, IM: []IMCheck{}}
	if a == nil {
		out.OK = false
		return out
	}
	if a.Connectors != nil {
		out.Connectors = a.Connectors.ProbeAll(only)
		for _, r := range out.Connectors {
			if !r.OK {
				out.OK = false
			}
		}
	}
	if a.IM != nil {
		snap := a.IM.Snapshot()
		states := map[string]string{}
		details := map[string]string{}
		for _, st := range snap.Statuses {
			states[st.ID], details[st.ID] = st.State, st.Detail
		}
		for _, acct := range snap.Accounts {
			c := IMCheck{ID: acct.ID, Platform: acct.Platform, Label: acct.Label, Enabled: acct.Enabled, State: states[acct.ID], Detail: details[acct.ID]}
			for name, has := range acct.HasSecret {
				if !has {
					c.Missing = append(c.Missing, name)
				}
			}
			sort.Strings(c.Missing)
			// 停用的账号不算失败；启用的要密钥齐全，且最近状态不是 error。
			c.OK = !c.Enabled || (len(c.Missing) == 0 && c.State != "error")
			if !c.OK {
				out.OK = false
			}
			out.IM = append(out.IM, c)
		}
	}
	h := a.IMHomeState()
	out.Home = IMHomeCheck{Bound: h.Bound, Platform: h.Platform, Label: h.Label}
	if imTest {
		out.Home.Tested = true
		if err := a.IMTest(); err != nil {
			out.Home.TestError = err.Error()
			out.OK = false
		}
	}
	return out
}
