package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func WorkspaceContextDir(workspace, sessionID string) string {
	if workspace == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "context", sessionID)
}

func WorkspaceMCPDir(workspace string) string {
	if workspace == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "mcp")
}

// WriteDiscoverIndex publishes greppable session context into the workspace
// jail: INDEX.md, notes.md, and a pointer at the spill directory.
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

// WriteMCPCatalog dumps extra tool schemas as files so they do not have to
// occupy the advertised tools array (Cursor-style DCD).
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

func mcpServerOf(name string) string {
	parts := strings.Split(name, "__")
	if len(parts) >= 3 && parts[0] == "mcp" {
		return parts[1]
	}
	return "extra"
}
