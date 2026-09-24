package video

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/canvas/capability"
	"github.com/Shenchangxin/yoyo/internal/video/canvas/layout"
)

func CanvasToolNames() []string {
	specs := CanvasToolSpecs()
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

func CanvasToolSpecs() []ToolSpec {
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "number"}
	obj := func(props map[string]any, req ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			m["required"] = req
		}
		return m
	}
	arr := map[string]any{"type": "array", "items": map[string]any{"type": "object"}}
	return []ToolSpec{
		{Name: "canvas_list_node_types", ReadOnly: true, Desc: "List built-in canvas node types the agent may create.", Params: obj(map[string]any{})},
		{Name: "canvas_get_state", ReadOnly: true, Desc: "Read a compact summary of the bound canvas: nodes, connections, viewport, revision.", Params: obj(map[string]any{"limit": num})},
		{Name: "canvas_read_storyboard", ReadOnly: true, Desc: "Read storyboard / script node tables from the bound canvas.", Params: obj(map[string]any{"node_id": str})},
		{Name: "canvas_read_batch_table", ReadOnly: true, Desc: "Read a batch-table node.", Params: obj(map[string]any{"node_id": str})},
		{Name: "canvas_inspect_image", ReadOnly: true, Desc: "Describe an image node. The host substitutes resource bytes in memory; they are not stored in the transcript.", Params: obj(map[string]any{"node_id": str, "resource_id": str})},
		{Name: "image_text_detect", ReadOnly: true, Desc: "OCR-style text detection for a canvas image resource (best-effort).", Params: obj(map[string]any{"resource_id": str})},
		{Name: "task_get", ReadOnly: true, Desc: "Get a canvas generation task by id.", Params: obj(map[string]any{"task_id": str, "id": str})},
		{Name: "model_list", ReadOnly: true, Desc: "List available canvas models filtered by mode and reference counts.", Params: obj(map[string]any{"mode": str, "capability": str})},
		{Name: "skill_read_file", ReadOnly: true, Desc: "Read a bundled or installed skill file.", Params: obj(map[string]any{"skill_id": str, "path": str})},
		{Name: "skill_search", ReadOnly: true, Desc: "Search installed skills.", Params: obj(map[string]any{"q": str, "query": str})},
		{Name: "canvas_apply_ops", Desc: "Apply structured canvas operations (add/update/delete/connect/move). Never put blob URLs in the document.", Params: obj(map[string]any{"ops": arr, "operations": arr}, "ops")},
		{Name: "canvas_arrange_nodes", Desc: "Arrange canvas nodes with the same layout algorithm as the editor.", Params: obj(map[string]any{"mode": str, "node_ids": map[string]any{"type": "array", "items": str}})},
		{Name: "canvas_create_storyboard", Desc: "Create a storyboard node with rows.", Params: obj(map[string]any{"title": str, "rows": arr})},
		{Name: "canvas_edit_storyboard", Desc: "Patch storyboard rows.", Params: obj(map[string]any{"node_id": str, "rows": arr}, "node_id")},
		{Name: "canvas_edit_batch_table", Desc: "Patch a batch-table node.", Params: obj(map[string]any{"node_id": str, "rows": arr}, "node_id")},
		{Name: "generate_media", Desc: "Queue image/video/audio generation for a canvas node. Media jobs require user approval.", Params: obj(map[string]any{"mode": str, "prompt": str, "node_id": str, "channel_id": str, "model": str}, "prompt")},
		{Name: "image_layer_split", Desc: "Request AI layer split for an image node (queued as an image edit task).", Params: obj(map[string]any{"node_id": str, "resource_id": str})},
		{Name: "image_annotation_render", Desc: "Render annotations onto an image node (queued).", Params: obj(map[string]any{"node_id": str, "resource_id": str, "prompt": str})},
		{Name: "plan_update", Desc: "Update the session plan bar.", Params: obj(map[string]any{"plan": str, "text": str})},
		{Name: "ask_user", Desc: "Ask the user a clarifying question before continuing.", Params: obj(map[string]any{"question": str, "prompt": str}, "question")},
		{Name: "remember_lesson", Desc: "Store a pending personal lesson for the user to approve.", Params: obj(map[string]any{"topic": str, "lesson": str, "situation": str})},
		{Name: "recall_lessons", ReadOnly: true, Desc: "Recall approved canvas lessons.", Params: obj(map[string]any{"query": str})},
	}
}

func (e *Engine) CanvasTool(sessionID, name string, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}
	canvasID, ok := e.CanvasSessionBind(sessionID)
	if !ok {
		canvasID = strAnyMap(args, "canvas_id", "project_id")
	}
	if canvasID == "" && name != "model_list" && name != "canvas_list_node_types" && name != "recall_lessons" && name != "skill_search" && name != "skill_read_file" && name != "ask_user" && name != "plan_update" && name != "remember_lesson" {
		return "", fmt.Errorf("no canvas bound to this thread — open Infinite canvas first")
	}
	switch name {
	case "canvas_list_node_types":
		reg := capability.BuiltinRegistry()
		out := []map[string]any{}
		for _, d := range reg.List() {
			out = append(out, map[string]any{
				"type": d.Type, "label": d.Label, "version": d.Version, "purpose": d.Purpose,
				"agentSupported": true, "goodFor": d.GoodFor, "notIdealFor": d.NotIdealFor,
			})
		}
		return marshalJSON(out), nil
	case "canvas_get_state":
		row, err := e.getCanvasProject(canvasID)
		if err != nil {
			return "", err
		}
		doc := e.projectDocFromRow(row)
		nodes, _ := doc["nodes"].([]any)
		limit := intAny(args["limit"])
		if limit <= 0 {
			limit = 80
		}
		summaries := []map[string]any{}
		for i, n := range nodes {
			if i >= limit {
				break
			}
			m := asMap(n)
			summaries = append(summaries, map[string]any{
				"id": strAnyMap(m, "id"), "type": strAnyMap(m, "type"), "title": strAnyMap(m, "title"),
				"x": m["x"], "y": m["y"], "width": m["width"], "height": m["height"],
			})
		}
		conns, _ := doc["connections"].([]any)
		return marshalJSON(map[string]any{
			"id": canvasID, "title": row.Title, "revision": row.Revision,
			"nodeCount": len(nodes), "connectionCount": len(conns),
			"nodes": summaries, "updatedAt": row.UpdatedAt,
		}), nil
	case "canvas_read_storyboard", "canvas_read_batch_table":
		return e.readCanvasNode(canvasID, strAnyMap(args, "node_id"))
	case "canvas_inspect_image":
		return e.inspectCanvasImage(canvasID, args)
	case "image_text_detect":
		return marshalJSON(map[string]any{"texts": []any{}, "note": "desktop OCR is best-effort; use the image node itself"}), nil
	case "task_get":
		id := firstNonEmpty(strAnyMap(args, "task_id"), strAnyMap(args, "id"))
		t, err := e.canvasTask(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(t), nil
	case "model_list":
		return marshalJSON(e.modelCatalog()), nil
	case "skill_read_file", "skill_search":
		return marshalJSON(map[string]any{"items": []any{}, "note": "use Yoyo Skills workspace; canvas inserts installed skills"}), nil
	case "canvas_apply_ops":
		ops := asSlice(args["ops"])
		if len(ops) == 0 {
			ops = asSlice(args["operations"])
		}
		return e.applyCanvasOps(canvasID, ops)
	case "canvas_arrange_nodes":
		return e.arrangeCanvasNodes(canvasID, strAnyMap(args, "mode"), args)
	case "canvas_create_storyboard":
		return e.createStoryboardNode(canvasID, args)
	case "canvas_edit_storyboard", "canvas_edit_batch_table":
		return e.patchTableNode(canvasID, strAnyMap(args, "node_id"), asSlice(args["rows"]))
	case "generate_media", "image_layer_split", "image_annotation_render":
		mode := firstNonEmpty(strAnyMap(args, "mode"), "image")
		if name != "generate_media" {
			mode = "image"
		}
		task, err := e.createCanvasTask(map[string]any{
			"type": mode, "prompt": firstNonEmpty(strAnyMap(args, "prompt"), name),
			"projectId": canvasID, "provider": strAnyMap(args, "channel_id"), "model": strAnyMap(args, "model"),
			"clientContext": map[string]any{"nodeId": strAnyMap(args, "node_id")},
			"input":         args,
		})
		if err != nil {
			return "", err
		}
		return marshalJSON(task), nil
	case "plan_update":
		return firstNonEmpty(strAnyMap(args, "plan"), strAnyMap(args, "text"), "plan updated"), nil
	case "ask_user":
		q := firstNonEmpty(strAnyMap(args, "question"), strAnyMap(args, "prompt"))
		if q == "" {
			return "", fmt.Errorf("question is empty")
		}
		return q, nil
	case "remember_lesson":
		return e.rememberLesson(args)
	case "recall_lessons":
		return e.recallLessons(strAnyMap(args, "query"))
	default:
		return "", fmt.Errorf("unknown canvas tool %s", name)
	}
}

func (e *Engine) readCanvasNode(canvasID, nodeID string) (string, error) {
	row, err := e.getCanvasProject(canvasID)
	if err != nil {
		return "", err
	}
	doc := e.projectDocFromRow(row)
	for _, n := range asSlice(doc["nodes"]) {
		m := asMap(n)
		if strAnyMap(m, "id") == nodeID {
			return marshalJSON(m), nil
		}
	}
	return "", fmt.Errorf("node not found")
}

func (e *Engine) inspectCanvasImage(canvasID string, args map[string]any) (string, error) {
	resourceID := strAnyMap(args, "resource_id")
	if resourceID == "" && canvasID != "" {
		raw, err := e.readCanvasNode(canvasID, strAnyMap(args, "node_id"))
		if err == nil {
			var m map[string]any
			_ = json.Unmarshal([]byte(raw), &m)
			meta := asMap(m["metadata"])
			resourceID = strings.TrimPrefix(firstNonEmpty(strAnyMap(meta, "storageKey"), strAnyMap(m, "storageKey")), "resource:")
		}
	}
	if resourceID == "" {
		return "", fmt.Errorf("resource_id required")
	}
	res, err := e.getCanvasResource(resourceID)
	if err != nil {
		return "", err
	}
	return marshalJSON(map[string]any{
		"resourceId": resourceID, "kind": res.Kind, "mime": res.MimeType, "width": res.Width, "height": res.Height,
		"url": res.PublicURL, "note": "bytes are substituted in-memory by the host and are not persisted in chat",
	}), nil
}

func (e *Engine) rememberLesson(args map[string]any) (string, error) {
	now := Now()
	id := NewID()
	_, err := e.DB.Exec(`INSERT INTO canvas_lessons(id, topic, category, situation, lesson, steps_json, status, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, strAnyMap(args, "topic"), firstNonEmpty(strAnyMap(args, "category"), "other"), strAnyMap(args, "situation"), strAnyMap(args, "lesson"), "[]", "pending", now, now)
	if err != nil {
		return "", err
	}
	return marshalJSON(map[string]any{"id": id, "status": "pending"}), nil
}

func (e *Engine) recallLessons(query string) (string, error) {
	rows, err := e.DB.Query(`SELECT id, topic, category, situation, lesson, status FROM canvas_lessons WHERE status = 'approved' ORDER BY updated_at DESC LIMIT 20`)
	if err != nil {
		return marshalJSON([]any{}), nil
	}
	defer rows.Close()
	out := []map[string]any{}
	q := strings.ToLower(query)
	for rows.Next() {
		var id, topic, cat, sit, lesson, status string
		if rows.Scan(&id, &topic, &cat, &sit, &lesson, &status) != nil {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(topic+sit+lesson), q) {
			continue
		}
		out = append(out, map[string]any{"id": id, "topic": topic, "category": cat, "situation": sit, "lesson": lesson, "status": status})
	}
	return marshalJSON(out), nil
}

func (e *Engine) arrangeCanvasNodes(canvasID, mode string, args map[string]any) (string, error) {
	row, err := e.getCanvasProject(canvasID)
	if err != nil {
		return "", err
	}
	doc := e.projectDocFromRow(row)
	var nodes []layout.Node
	rawNodes := asSlice(doc["nodes"])
	ids := map[string]struct{}{}
	for _, v := range asSlice(args["node_ids"]) {
		ids[fmt.Sprint(v)] = struct{}{}
	}
	for _, n := range rawNodes {
		m := asMap(n)
		id := strAnyMap(m, "id")
		if len(ids) > 0 {
			if _, ok := ids[id]; !ok {
				continue
			}
		}
		nodes = append(nodes, layout.Node{
			ID: id, Type: strAnyMap(m, "type"), Lane: strAnyMap(m, "type"),
			X: float64(intAny(m["x"])), Y: float64(intAny(m["y"])),
			Width: float64(intAny(m["width"])), Height: float64(intAny(m["height"])),
			Movable: true,
		})
	}
	var edges []layout.Edge
	for _, c := range asSlice(doc["connections"]) {
		m := asMap(c)
		edges = append(edges, layout.Edge{From: strAnyMap(m, "from", "source"), To: strAnyMap(m, "to", "target")})
	}
	lm := layout.Mode(mode)
	var pos map[string]layout.Position
	switch lm {
	case layout.ModeFlow:
		pos = layout.Flow(nodes, edges)
	case layout.ModeByLane:
		pos = layout.ByLane(nodes)
	case layout.ModeRow, layout.ModeColumn, layout.ModeGrid:
		pos = layout.Linear(nodes, lm)
	default:
		pos = layout.Auto(nodes, edges)
	}
	next := []any{}
	for _, n := range rawNodes {
		m := asMap(n)
		if p, ok := pos[strAnyMap(m, "id")]; ok {
			m["x"] = p.X
			m["y"] = p.Y
		}
		next = append(next, m)
	}
	doc["nodes"] = next
	payload, _ := json.Marshal(doc)
	if err := e.saveCanvasPayload(canvasID, payload, row.Title); err != nil {
		return "", err
	}
	e.hub("canvas_patch", map[string]any{"canvas_id": canvasID, "kind": "arrange"})
	return marshalJSON(map[string]any{"ok": true, "moved": len(pos)}), nil
}

func (e *Engine) createStoryboardNode(canvasID string, args map[string]any) (string, error) {
	row, err := e.getCanvasProject(canvasID)
	if err != nil {
		return "", err
	}
	doc := e.projectDocFromRow(row)
	id := NewID()
	node := map[string]any{
		"id": id, "type": "script", "title": firstNonEmpty(strAnyMap(args, "title"), "Storyboard"),
		"x": 80, "y": 80, "width": 720, "height": 480,
		"metadata": map[string]any{"kind": "storyboard", "rows": asSlice(args["rows"])},
	}
	nodes := asSlice(doc["nodes"])
	nodes = append(nodes, node)
	doc["nodes"] = nodes
	payload, _ := json.Marshal(doc)
	if err := e.saveCanvasPayload(canvasID, payload, row.Title); err != nil {
		return "", err
	}
	return marshalJSON(node), nil
}

func (e *Engine) patchTableNode(canvasID, nodeID string, rows []any) (string, error) {
	ops := []any{map[string]any{"op": "update_node", "id": nodeID, "metadata": map[string]any{"rows": rows}}}
	return e.applyCanvasOps(canvasID, ops)
}
