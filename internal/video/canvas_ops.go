package video

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/canvas/capability"
)

func (e *Engine) applyCanvasOps(canvasID string, ops []any) (string, error) {
	if len(ops) == 0 {
		return "", fmt.Errorf("ops required")
	}
	if len(ops) > 40 {
		return "", fmt.Errorf("too many ops")
	}
	row, err := e.getCanvasProject(canvasID)
	if err != nil {
		return "", err
	}
	doc := e.projectDocFromRow(row)
	nodes := asSlice(doc["nodes"])
	conns := asSlice(doc["connections"])
	reg := capability.BuiltinRegistry()
	changed := 0
	for _, raw := range ops {
		op := asMap(raw)
		kind := firstNonEmpty(strAnyMap(op, "op"), strAnyMap(op, "type"))
		switch kind {
		case "add_node", "create":
			n := asMap(op["node"])
			if len(n) == 0 {
				n = op
			}
			typ := strAnyMap(n, "type")
			if typ == "" {
				typ = "text"
			}
			if d, ok := reg.Resolve(typ); ok {
				if strAnyMap(n, "id") == "" {
					n["id"] = NewID()
				}
				if n["width"] == nil {
					n["width"] = d.DefaultWidth
				}
				if n["height"] == nil {
					n["height"] = d.DefaultHeight
				}
				if n["title"] == nil {
					n["title"] = d.Label
				}
				if n["metadata"] == nil && d.CreateMetadata != nil {
					n["metadata"] = d.CreateMetadata(strAnyMap(n, "content", "prompt"))
				}
			}
			if strAnyMap(n, "id") == "" {
				n["id"] = NewID()
			}
			n["type"] = typ
			nodes = append(nodes, n)
			changed++
		case "update_node", "patch":
			id := firstNonEmpty(strAnyMap(op, "id"), strAnyMap(op, "node_id"))
			patch := asMap(op["patch"])
			if len(patch) == 0 {
				patch = op
			}
			for i, n := range nodes {
				m := asMap(n)
				if strAnyMap(m, "id") != id {
					continue
				}
				for k, v := range patch {
					if k == "op" || k == "type" || k == "id" || k == "node_id" {
						continue
					}
					if k == "metadata" {
						meta := asMap(m["metadata"])
						for mk, mv := range asMap(v) {
							meta[mk] = mv
						}
						m["metadata"] = meta
						continue
					}
					m[k] = v
				}
				nodes[i] = m
				changed++
			}
		case "delete_node", "remove":
			id := firstNonEmpty(strAnyMap(op, "id"), strAnyMap(op, "node_id"))
			next := []any{}
			for _, n := range nodes {
				if strAnyMap(asMap(n), "id") == id {
					changed++
					continue
				}
				next = append(next, n)
			}
			nodes = next
			nextC := []any{}
			for _, c := range conns {
				m := asMap(c)
				if strAnyMap(m, "from", "source") == id || strAnyMap(m, "to", "target") == id {
					continue
				}
				nextC = append(nextC, c)
			}
			conns = nextC
		case "connect":
			conn := map[string]any{
				"id": firstNonEmpty(strAnyMap(op, "id"), NewID()),
				"from": firstNonEmpty(strAnyMap(op, "from"), strAnyMap(op, "source")),
				"to":   firstNonEmpty(strAnyMap(op, "to"), strAnyMap(op, "target")),
				"relation": strAnyMap(op, "relation"),
			}
			conns = append(conns, conn)
			changed++
		case "disconnect":
			id := strAnyMap(op, "id")
			next := []any{}
			for _, c := range conns {
				m := asMap(c)
				if strAnyMap(m, "id") == id {
					changed++
					continue
				}
				next = append(next, c)
			}
			conns = next
		case "move":
			id := firstNonEmpty(strAnyMap(op, "id"), strAnyMap(op, "node_id"))
			for i, n := range nodes {
				m := asMap(n)
				if strAnyMap(m, "id") != id {
					continue
				}
				if _, ok := op["x"]; ok {
					m["x"] = op["x"]
				}
				if _, ok := op["y"]; ok {
					m["y"] = op["y"]
				}
				nodes[i] = m
				changed++
			}
		}
	}
	doc["nodes"] = nodes
	doc["connections"] = conns
	payload, _ := json.Marshal(doc)
	if err := e.saveCanvasPayload(canvasID, payload, row.Title); err != nil {
		return "", err
	}
	e.hub("canvas_patch", map[string]any{"canvas_id": canvasID, "ops": len(ops), "changed": changed})
	return marshalJSON(map[string]any{"ok": true, "changed": changed, "revision": row.Revision + 1}), nil
}

func (e *Engine) promptCompile(operation string, vars map[string]any) string {
	switch operation {
	case "chapter_assets", "characters":
		return firstNonEmpty(strAnyMap(vars, "source"), "") + "\n\nExtract characters, scenes, and props as structured lists."
	case "storyboard", "storyboard_plan":
		return "Plan a storyboard from:\n" + firstNonEmpty(strAnyMap(vars, "source"), strAnyMap(vars, "script"))
	case "first_frame":
		return "Describe a first-frame still for: " + strAnyMap(vars, "shot")
	case "character_card":
		return "Write a character card for: " + strAnyMap(vars, "name")
	case "three_view":
		return "Generate a three-view turnaround prompt for: " + strAnyMap(vars, "name")
	case "short_drama_outline":
		return "Write a short-drama outline from: " + strAnyMap(vars, "premise")
	default:
		b, _ := json.Marshal(vars)
		return strings.TrimSpace(operation + "\n" + string(b))
	}
}
