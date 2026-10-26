// 把会话上下文、额外工具 schema 和延后工具名单写进工作区，供 grep 和 read_file 查看。
// 不把这些正文放进提示词。循环进度和工具列表组装时调用。

package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/tool"
)

// WorkspaceContextDir 返回工作区 .yoyo/context/<sessionID>。workspace 或 sessionID 为空时返回空串。
func WorkspaceContextDir(workspace, sessionID string) string {
	if workspace == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "context", sessionID)
}

// WorkspaceMCPDir 返回工作区 .yoyo/mcp。workspace 为空时返回空串。
func WorkspaceMCPDir(workspace string) string {
	if workspace == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "mcp")
}

// WriteDiscoverIndex 在工作区监狱里写下 INDEX.md 和 notes.md，并列出 spill id。
// 索引正文让模型用 recall_context 的 offset 和 limit 取回，或直接 grep、read_file，而不是把 spill 再倒进提示词。终端输出在 terminals/ 下。
func WriteDiscoverIndex(workspace, sessionID string, spill *Spill) {
	dir := WorkspaceContextDir(workspace, sessionID)
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	var b strings.Builder
	b.WriteString("# Context index\n\n")
	b.WriteString("Full tool bytes live in `spill/` (mirrored into this jail) and in the session spill store. Recover with recall_context (offset/limit), or grep/read_file these paths instead of re-dumping. Terminal output is under `terminals/`.\n\n")
	if spill != nil {
		ids := spill.ListIDs()
		sort.Strings(ids)
		for _, id := range ids {
			b.WriteString("- ")
			b.WriteString(id)
			b.WriteByte('\n')
		}
		if notes := ReadNotes(spill); notes != "" {
			_ = os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notes), 0o644)
		}
	}
	_ = os.WriteFile(filepath.Join(dir, "INDEX.md"), []byte(b.String()), 0o644)
}

// WriteMCPCatalog 把额外工具的 schema 按服务器写成 JSON。文件里的说明写明要先 tool_search，schema 到下一次检查点才进入 tools 数组。本函数只写文件。
func WriteMCPCatalog(workspace string, extras map[string]ExtraTool) {
	dir := WorkspaceMCPDir(workspace)
	if dir == "" || len(extras) == 0 {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	byServer := map[string][]string{}
	for name, extra := range extras {
		server := mcpServerOf(name)
		sdir := filepath.Join(dir, sanitizeID(server))
		_ = os.MkdirAll(sdir, 0o755)
		raw, err := json.MarshalIndent(extra.JSON, "", "  ")
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(sdir, sanitizeID(name)+".json"), raw, 0o644)
		byServer[server] = append(byServer[server], name)
	}
	servers := make([]string, 0, len(byServer))
	for s := range byServer {
		servers = append(servers, s)
	}
	sort.Strings(servers)
	var root strings.Builder
	root.WriteString("# MCP / extra tool catalog\n\n")
	root.WriteString("Each server is a folder. Read INDEX.md then the schema JSON. Use tool_search to enable one; schema joins the tools array at the next checkpoint.\n\n")
	for _, server := range servers {
		names := byServer[server]
		sort.Strings(names)
		fmt.Fprintf(&root, "- %s/ (%d tools)\n", server, len(names))
		var idx strings.Builder
		idx.WriteString("# " + server + "\n\n")
		for _, name := range names {
			idx.WriteString("- ")
			idx.WriteString(name)
			idx.WriteByte('\n')
		}
		_ = os.WriteFile(filepath.Join(dir, sanitizeID(server), "INDEX.md"), []byte(idx.String()), 0o644)
	}
	_ = os.WriteFile(filepath.Join(dir, "INDEX.md"), []byte(root.String()), 0o644)
}

// WriteHostToolIndex 把延后工具的名字和一句话说明写到 .yoyo/tools/INDEX.md。同一组名字写出的文本稳定。名字不写进 tool_search 的描述。
func WriteHostToolIndex(workspace string, names []string) {
	if workspace == "" || len(names) == 0 {
		return
	}
	dir := filepath.Join(workspace, ".yoyo", "tools")
	_ = os.MkdirAll(dir, 0o755)
	specs := map[string]string{}
	for _, s := range tool.HostSpecs() {
		specs[s.Name] = firstSentence(s.Description)
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	var b strings.Builder
	b.WriteString("# Deferred tools\n\n")
	b.WriteString("These tools are not in the tools array yet. tool_search with the name returns the schema. It joins the tools array at the next checkpoint.\n\n")
	seen := map[string]bool{}
	for _, name := range sorted {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		desc := specs[name]
		if desc == "" {
			desc = "extra tool"
		}
		b.WriteString("- ")
		b.WriteString(name)
		b.WriteString(": ")
		b.WriteString(desc)
		b.WriteByte('\n')
	}
	_ = os.WriteFile(filepath.Join(dir, "INDEX.md"), []byte(b.String()), 0o644)
}

// firstSentence 压成一行。180 个字节以内的第一个句号处截断，否则保留 160 个 rune。
func firstSentence(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, ".。"); i >= 0 && i < 180 {
		return s[:i+1]
	}
	return capRunes(s, 160)
}

// mcpServerOf 从 mcp__服务器__工具 取出服务器名。格式不符时返回 extra。
func mcpServerOf(name string) string {
	parts := strings.Split(name, "__")
	if len(parts) >= 3 && parts[0] == "mcp" {
		return parts[1]
	}
	return "extra"
}
