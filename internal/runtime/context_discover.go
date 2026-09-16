package runtime

import (
	"encoding/json"
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
	b.WriteString("Full tool bytes live in `spill/` (mirrored into this jail) and in the session spill store. Recover with recall_context, or grep/read_file these paths instead of re-dumping.\n\n")
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
	names := make([]string, 0, len(extras))
	for name, extra := range extras {
		names = append(names, name)
		raw, err := json.MarshalIndent(extra.JSON, "", "  ")
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(dir, sanitizeID(name)+".json"), raw, 0o644)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("# MCP / extra tool catalog\n\n")
	b.WriteString("Schemas here are files, not always advertised tools. Use tool_search to enable one.\n\n")
	for _, name := range names {
		b.WriteString("- ")
		b.WriteString(name)
		b.WriteByte('\n')
	}
	_ = os.WriteFile(filepath.Join(dir, "INDEX.md"), []byte(b.String()), 0o644)
}
