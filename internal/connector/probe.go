// probe.go 是连接自检：用账号上真实的令牌和地址读一次，看连不连得上。
// 和 Read 不同，这里拿不到令牌、或者读失败时不退回本地样例，失败就原样报出来，免得「读到了样例」被当成「连上了」。
// 给 yoyo connect check 和以后界面上的「测试连接」用。只读，不发信、不写日程。

package connector

import (
	"fmt"
	"strings"
)

// ProbeResult 是一个账号的自检结果。
// Items 是读到的真实条目数（不含「收件箱是空的」这类说明行）；Note 是说明行的内容；Error 为空时 OK 为真。
type ProbeResult struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	OK       bool   `json:"ok"`
	Items    int    `json:"items"`
	Note     string `json:"note,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Probe 用真实令牌读一次 id 这个账号。找不到账号、没有令牌、登录失败、接口报错都写进 Error。
// 令牌快过期时照常先换票（liveRead 里的 freshToken），所以自检通过也说明换票这条路是通的。
func (b *Broker) Probe(id string) ProbeResult {
	b.mu.Lock()
	a, ok := b.find(id)
	b.mu.Unlock()
	if !ok {
		return ProbeResult{ID: id, Error: fmt.Sprintf("unknown account %s", id)}
	}
	out := ProbeResult{ID: a.ID, Provider: a.Provider, Kind: string(a.Kind), Label: a.Label}
	rows, err := b.liveRead(a, "")
	if err != nil {
		out.Error = err.Error()
		if missingToken(err) {
			out.Error = "没有令牌：在「连接」里完成授权，或填好授权码 / 应用专用密码"
		}
		return out
	}
	for _, row := range rows {
		if note := strings.TrimSpace(row["note"]); note != "" {
			if out.Note == "" {
				out.Note = note
			}
			continue
		}
		out.Items++
	}
	out.OK = true
	return out
}

// ProbeAll 依次自检所有账号；only 非空时只测这一个。按 List 的顺序返回。
func (b *Broker) ProbeAll(only string) []ProbeResult {
	var out []ProbeResult
	for _, a := range b.List() {
		if only != "" && a.ID != only {
			continue
		}
		out = append(out, b.Probe(a.ID))
	}
	if only != "" && len(out) == 0 {
		out = append(out, ProbeResult{ID: only, Error: fmt.Sprintf("unknown account %s", only)})
	}
	return out
}
