// prompt.go 拥有交给外部 agent 的会话参数和提示正文：控制面 MCP 条目、系统说明放在哪、额外目录，以及新会话前的延续说明。
// 不拥有说明的内容（superagent.externalWorkerPrompt），也不拥有连接和回合（worker.go）。

package acpworker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// mcpName 是控制面在 agent 那边的服务器名。工具名去前缀时也认它。
const mcpName = "yoyo"

// mcpServers 返回交给 agent 的控制面。对方支持 HTTP MCP 时直连，token 放在 Authorization 头里；
// 否则经 yoyo 自己的 stdio 桥，token 放在桥进程的环境变量里。两种都不进命令行参数。url 或 token 为空时不给。
func mcpServers(caps acp.AgentCapabilities, url, token, exe string) []acp.McpServer {
	if url == "" || token == "" {
		return []acp.McpServer{}
	}
	if caps.McpCapabilities.HTTP {
		return []acp.McpServer{{
			Type: "http", Name: mcpName, URL: url,
			Headers: []acp.HTTPHeader{{Name: "Authorization", Value: "Bearer " + token}},
		}}
	}
	return []acp.McpServer{{
		Name: mcpName, Command: exe, Args: []string{},
		Env: []acp.EnvVariable{
			{Name: "YOYO_MCP_BRIDGE", Value: "1"},
			{Name: "YOYO_MCP_URL", Value: url},
			{Name: "YOYO_MCP_TOKEN", Value: token},
		},
	}}
}

// systemPrompt 在对方支持时把说明以 append 方式放进 session/new。不支持时返回 nil，说明改放进第一条消息。
func systemPrompt(caps acp.AgentCapabilities, text string) *acp.SystemPrompt {
	if !caps.SessionCapabilities.SystemPrompt || strings.TrimSpace(text) == "" {
		return nil
	}
	return &acp.SystemPrompt{Mode: "append", Text: text}
}

// extraDirs 在对方支持时返回可写目录加只读目录。
func extraDirs(caps acp.AgentCapabilities, spec superagent.WorkerSpec) []string {
	if !caps.SessionCapabilities.AdditionalDirectories {
		return nil
	}
	var out []string
	out = append(out, spec.Roots...)
	out = append(out, spec.ReadRoots...)
	return out
}

// instructionsBlock 把说明包成第一条消息开头的 <yoyo-instructions> 块。
func instructionsBlock(text string) string {
	return "<yoyo-instructions>\n" + strings.TrimSpace(text) + "\n</yoyo-instructions>"
}

// continuationNote 是续接不上、只能新开会话时放在第一条消息前的说明：上次的摘要加最后三行记录。events 只含这一轮之前的事件，没有可说的时返回空串。
// 交接那一行不算记录：刚接手的工人没有做过什么，交接说明已经在派工正文里。
func continuationNote(events []dsh.Event) string {
	summary := lastReportSummary(events)
	var lines []superagent.TranscriptLine
	for _, line := range superagent.TranscriptLines(events) {
		if line.Source != "handover" {
			lines = append(lines, line)
		}
	}
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	if summary == "" && len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("这张工单之前已经做过一部分，原来的会话接不上了。接着往下做，不要重做已经完成的部分，也不要重复已经做成的外部动作。\n")
	if summary != "" {
		fmt.Fprintf(&b, "上次的摘要：%s\n", clipRunes(summary, 300))
	}
	if len(lines) > 0 {
		b.WriteString("最后几行记录：\n")
		for _, line := range lines {
			who := line.Role
			if line.Name != "" {
				who += " " + line.Name
			}
			fmt.Fprintf(&b, "- %s：%s\n", who, clipRunes(oneLine(line.Text), 300))
		}
	}
	return strings.TrimSpace(b.String())
}

// lastReportSummary 返回日志里最后一次 orchestrator_report 的摘要。没有报告时用最后一段助手正文。
func lastReportSummary(events []dsh.Event) string {
	for i := len(events) - 1; i >= 0; i-- {
		msg, ok := events[i].Data.(dsh.Assistant)
		if events[i].Type != dsh.EventAssistant || !ok {
			continue
		}
		for j := len(msg.ToolCalls) - 1; j >= 0; j-- {
			call := msg.ToolCalls[j]
			if call.Name != "orchestrator_report" {
				continue
			}
			var args struct {
				Summary string `json:"summary"`
			}
			if json.Unmarshal([]byte(call.Arguments), &args) == nil && strings.TrimSpace(args.Summary) != "" {
				return strings.TrimSpace(args.Summary)
			}
		}
	}
	for i := len(events) - 1; i >= 0; i-- {
		if msg, ok := events[i].Data.(dsh.Assistant); ok && events[i].Type == dsh.EventAssistant && strings.TrimSpace(msg.Text) != "" {
			return strings.TrimSpace(msg.Text)
		}
	}
	return ""
}

// promptBlocks 把前言和这一轮领到的消息拼成一段文字。多条消息之间空一行。
func promptBlocks(preface string, msgs []dsh.UserMessage) []acp.ContentBlock {
	var parts []string
	if p := strings.TrimSpace(preface); p != "" {
		parts = append(parts, p)
	}
	for _, msg := range msgs {
		if text := strings.TrimSpace(msg.Text); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "继续。")
	}
	return []acp.ContentBlock{acp.TextBlock(strings.Join(parts, "\n\n"))}
}

// oneLine 把换行收成空格。
func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// clipRunes 把 text 收成最多 n 个字符。
func clipRunes(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n])
}

// tailRunes 取 text 最后 n 个字符。
func tailRunes(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[len(runes)-n:])
}
