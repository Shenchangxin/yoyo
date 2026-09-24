package video

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const canvasHistoryLimit = 20
const canvasHistoryInterval = 5 * time.Minute

type canvasProjectRow struct {
	ID            string
	Title         string
	PayloadJSON   string
	TimelineJSON  string
	Revision      int64
	WorkspaceMode string
	Theme         string
	ProjectLink   string
	CreatedAt     string
	UpdatedAt     string
}

func (e *Engine) getCanvasProject(id string) (canvasProjectRow, error) {
	var r canvasProjectRow
	err := e.DB.QueryRow(`SELECT id, title, payload_json, timeline_json, revision, workspace_mode, theme, project_link, created_at, updated_at FROM canvas_projects WHERE id = ? AND deleted_at = ''`, id).
		Scan(&r.ID, &r.Title, &r.PayloadJSON, &r.TimelineJSON, &r.Revision, &r.WorkspaceMode, &r.Theme, &r.ProjectLink, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return r, fmt.Errorf("canvas not found")
	}
	return r, nil
}

func (e *Engine) listCanvasProjectRows(search, sort string, page, pageSize int) ([]canvasProjectRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 40
	}
	where := `deleted_at = ''`
	var args []any
	if q := strings.TrimSpace(search); q != "" {
		where += ` AND (title LIKE ? OR id LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	order := `updated_at DESC`
	switch sort {
	case "name":
		order = `title COLLATE NOCASE ASC`
	case "nodes":
		order = `json_array_length(json_extract(payload_json, '$.nodes')) DESC, updated_at DESC`
	}
	var total int
	if err := e.DB.QueryRow(`SELECT COUNT(*) FROM canvas_projects WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := e.DB.Query(`SELECT id, title, payload_json, timeline_json, revision, workspace_mode, theme, project_link, created_at, updated_at FROM canvas_projects WHERE `+where+` ORDER BY `+order+` LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []canvasProjectRow
	for rows.Next() {
		var r canvasProjectRow
		if err := rows.Scan(&r.ID, &r.Title, &r.PayloadJSON, &r.TimelineJSON, &r.Revision, &r.WorkspaceMode, &r.Theme, &r.ProjectLink, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []canvasProjectRow{}
	}
	return out, total, nil
}

func decodeProjectDoc(raw []byte) (map[string]any, error) {
	m := map[string]any{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("画布数据格式错误")
	}
	return m, nil
}

func (e *Engine) projectDocFromRow(r canvasProjectRow) map[string]any {
	doc := map[string]any{}
	_ = json.Unmarshal([]byte(r.PayloadJSON), &doc)
	if doc == nil {
		doc = map[string]any{}
	}
	doc["id"] = r.ID
	doc["title"] = r.Title
	doc["revision"] = r.Revision
	doc["createdAt"] = r.CreatedAt
	doc["updatedAt"] = r.UpdatedAt
	if r.ProjectLink != "" {
		doc["projectId"] = r.ProjectLink
	}
	if _, ok := doc["nodes"]; !ok {
		doc["nodes"] = []any{}
	}
	if _, ok := doc["connections"]; !ok {
		doc["connections"] = []any{}
	}
	if _, ok := doc["chatSessions"]; !ok {
		doc["chatSessions"] = []any{}
	}
	if _, ok := doc["directorScenes"]; !ok {
		doc["directorScenes"] = []any{}
	}
	if _, ok := doc["viewport"]; !ok {
		doc["viewport"] = map[string]any{"x": 0, "y": 0, "k": 1}
	}
	if r.TimelineJSON != "" {
		var tl any
		if json.Unmarshal([]byte(r.TimelineJSON), &tl) == nil {
			doc["timeline"] = tl
		}
	}
	return doc
}

func canvasSummary(r canvasProjectRow) map[string]any {
	doc := map[string]any{}
	_ = json.Unmarshal([]byte(r.PayloadJSON), &doc)
	nodes, _ := doc["nodes"].([]any)
	preview := nodes
	if len(preview) > 4 {
		preview = preview[:4]
	}
	return map[string]any{
		"id":           r.ID,
		"title":        r.Title,
		"revision":     r.Revision,
		"createdAt":    r.CreatedAt,
		"updatedAt":    r.UpdatedAt,
		"projectId":    r.ProjectLink,
		"nodeCount":    len(nodes),
		"previewNodes": preview,
	}
}

func canvasNodeCount(payload string) int {
	var document struct {
		Nodes []json.RawMessage `json:"nodes"`
	}
	_ = json.Unmarshal([]byte(payload), &document)
	return len(document.Nodes)
}

func canvasConnectionCount(payload string) int {
	var document struct {
		Connections []json.RawMessage `json:"connections"`
	}
	_ = json.Unmarshal([]byte(payload), &document)
	return len(document.Connections)
}

func (e *Engine) upsertCanvasProject(raw json.RawMessage, repair bool) (map[string]any, canvasEnv) {
	doc, err := decodeProjectDoc(raw)
	if err != nil {
		return nil, canvasFail(400, err.Error(), "bad_request")
	}
	id := strings.TrimSpace(fmt.Sprint(doc["id"]))
	if id == "" {
		return nil, canvasFail(400, "画布 ID 缺失", "bad_request")
	}
	if utf8.RuneCountInString(id) > 80 {
		return nil, canvasFail(400, "画布 ID 过长", "bad_request")
	}
	revVal, hasRev := doc["revision"]
	if !hasRev {
		return nil, canvasConflict("缺少画布版本，请保留本地草稿后重新加载画布")
	}
	baseRev := int64(intAny(revVal))
	existing, existingErr := e.getCanvasProject(id)
	found := existingErr == nil
	if found && existing.Revision != baseRev {
		return nil, canvasConflict("画布已被更新，请保留本地草稿后重新加载")
	}
	if !found && baseRev != 0 {
		return nil, canvasConflict("画布已被更新，请保留本地草稿后重新加载")
	}
	now := Now()
	title := strings.TrimSpace(fmt.Sprint(doc["title"]))
	if title == "" {
		title = "Untitled canvas"
	}
	delete(doc, "revision")
	delete(doc, "remoteContentHash")
	doc["viewport"] = map[string]any{"x": 0, "y": 0, "k": 1}
	if found {
		prev := map[string]any{}
		_ = json.Unmarshal([]byte(existing.PayloadJSON), &prev)
		if vp, ok := prev["viewport"]; ok {
			doc["viewport"] = vp
		}
	}
	created := now
	if found {
		created = existing.CreatedAt
	}
	doc["createdAt"] = created
	doc["updatedAt"] = now
	projectLink := strAnyMap(doc, "projectId")
	payload, err := json.Marshal(doc)
	if err != nil {
		return nil, canvasFail(400, "画布无法序列化", "bad_request")
	}
	if !repair {
		if err := e.validateCanvasResources(payload); err != nil {
			return nil, canvasFail(400, err.Error(), "bad_request")
		}
	}
	newRev := baseRev + 1
	reason := "automatic"
	if repair {
		reason = "before_resource_repair"
	}
	if found {
		_ = e.maybeSnapshot(existing, reason)
		_, err = e.DB.Exec(`UPDATE canvas_projects SET title=?, payload_json=?, revision=?, project_link=?, updated_at=? WHERE id=?`,
			title, string(payload), newRev, projectLink, now, id)
	} else {
		_, err = e.DB.Exec(`INSERT INTO canvas_projects(id, title, payload_json, timeline_json, revision, workspace_mode, theme, project_link, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			id, title, string(payload), "", newRev, "simple", "", projectLink, created, now)
	}
	if err != nil {
		return nil, canvasFail(500, err.Error(), "internal")
	}
	return map[string]any{
		"id":        id,
		"title":     title,
		"revision":  newRev,
		"createdAt": created,
		"updatedAt": now,
	}, canvasOK(nil)
}

func (e *Engine) validateCanvasResources(payload []byte) error {
	ids := collectResourceIDs(payload)
	for _, id := range ids {
		var n int
		err := e.DB.QueryRow(`SELECT COUNT(*) FROM canvas_resources WHERE id = ? AND deleted_at = ''`, id).Scan(&n)
		if err != nil || n == 0 {
			return fmt.Errorf("画布引用了缺失资源 %s", id)
		}
	}
	return nil
}

func collectResourceIDs(payload []byte) []string {
	s := string(payload)
	seen := map[string]struct{}{}
	var out []string
	const prefix = "resource:"
	for {
		i := strings.Index(s, prefix)
		if i < 0 {
			break
		}
		s = s[i+len(prefix):]
		j := 0
		for j < len(s) {
			c := s[j]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
				j++
				continue
			}
			break
		}
		if j == 0 {
			continue
		}
		id := s[:j]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (e *Engine) maybeSnapshot(existing canvasProjectRow, reason string) error {
	force := reason == "before_restore" || reason == "before_resource_repair"
	if !force {
		var last string
		_ = e.DB.QueryRow(`SELECT created_at FROM canvas_snapshots WHERE canvas_id = ? ORDER BY created_at DESC LIMIT 1`, existing.ID).Scan(&last)
		if last != "" {
			if t, err := time.Parse(time.RFC3339Nano, last); err == nil && time.Since(t) < canvasHistoryInterval {
				return nil
			}
		}
	}
	now := Now()
	_, err := e.DB.Exec(`INSERT INTO canvas_snapshots(id, canvas_id, revision, title, payload_json, reason, node_count, connection_count, created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		NewID(), existing.ID, existing.Revision, existing.Title, existing.PayloadJSON, reason, canvasNodeCount(existing.PayloadJSON), canvasConnectionCount(existing.PayloadJSON), now)
	if err != nil {
		return err
	}
	rows, err := e.DB.Query(`SELECT id FROM canvas_snapshots WHERE canvas_id = ? ORDER BY created_at DESC`, existing.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) > canvasHistoryLimit {
		for _, id := range ids[canvasHistoryLimit:] {
			_, _ = e.DB.Exec(`DELETE FROM canvas_snapshots WHERE id = ?`, id)
		}
	}
	return nil
}

func (e *Engine) listCanvasHistory(id string) (map[string]any, error) {
	row, err := e.getCanvasProject(id)
	if err != nil {
		return nil, err
	}
	rows, err := e.DB.Query(`SELECT id, canvas_id, revision, title, payload_json, reason, node_count, connection_count, created_at FROM canvas_snapshots WHERE canvas_id = ? ORDER BY created_at DESC LIMIT ?`, id, canvasHistoryLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	snaps := []map[string]any{}
	for rows.Next() {
		var sid, cid, title, payload, reason, created string
		var rev int64
		var nodes, conns int
		if err := rows.Scan(&sid, &cid, &rev, &title, &payload, &reason, &nodes, &conns, &created); err != nil {
			return nil, err
		}
		snaps = append(snaps, map[string]any{
			"id": sid, "canvasId": cid, "revision": rev, "title": title,
			"nodeCount": nodes, "connectionCount": conns, "payloadBytes": len(payload),
			"reason": reason, "createdAt": created, "contentUpdatedAt": created,
		})
	}
	return map[string]any{"snapshots": snaps, "currentRevision": row.Revision}, nil
}

func (e *Engine) getCanvasSnapshot(canvasID, snapshotID string) (map[string]any, map[string]any, error) {
	var sid, cid, title, payload, reason, created string
	var rev int64
	var nodes, conns int
	err := e.DB.QueryRow(`SELECT id, canvas_id, revision, title, payload_json, reason, node_count, connection_count, created_at FROM canvas_snapshots WHERE id = ? AND canvas_id = ?`, snapshotID, canvasID).
		Scan(&sid, &cid, &rev, &title, &payload, &reason, &nodes, &conns, &created)
	if err != nil {
		return nil, nil, fmt.Errorf("snapshot not found")
	}
	snap := map[string]any{
		"id": sid, "canvasId": cid, "revision": rev, "title": title,
		"nodeCount": nodes, "connectionCount": conns, "payloadBytes": len(payload),
		"reason": reason, "createdAt": created, "contentUpdatedAt": created,
	}
	doc := map[string]any{}
	_ = json.Unmarshal([]byte(payload), &doc)
	doc["id"] = canvasID
	doc["title"] = title
	doc["revision"] = rev
	return snap, doc, nil
}

func (e *Engine) restoreCanvasSnapshot(canvasID, snapshotID string, revision int64) (map[string]any, canvasEnv) {
	existing, err := e.getCanvasProject(canvasID)
	if err != nil {
		return nil, canvasFail(404, err.Error(), "not_found")
	}
	if existing.Revision != revision {
		return nil, canvasConflict("请先刷新画布版本再恢复")
	}
	_, doc, err := e.getCanvasSnapshot(canvasID, snapshotID)
	if err != nil {
		return nil, canvasFail(404, err.Error(), "not_found")
	}
	_ = e.maybeSnapshot(existing, "before_restore")
	now := Now()
	newRev := existing.Revision + 1
	title := strings.TrimSpace(fmt.Sprint(doc["title"]))
	if title == "" {
		title = existing.Title
	}
	doc["revision"] = newRev
	doc["updatedAt"] = now
	payload, _ := json.Marshal(doc)
	_, err = e.DB.Exec(`UPDATE canvas_projects SET title=?, payload_json=?, revision=?, updated_at=? WHERE id=?`, title, string(payload), newRev, now, canvasID)
	if err != nil {
		return nil, canvasFail(500, err.Error(), "internal")
	}
	return map[string]any{"id": canvasID, "title": title, "revision": newRev, "createdAt": existing.CreatedAt, "updatedAt": now}, canvasOK(nil)
}

func (e *Engine) deleteCanvasProject(id string) error {
	_, err := e.DB.Exec(`UPDATE canvas_projects SET deleted_at = ? WHERE id = ? AND deleted_at = ''`, Now(), id)
	return err
}

func (e *Engine) createEmptyCanvas(title string) (map[string]any, error) {
	id := NewID()
	now := Now()
	if strings.TrimSpace(title) == "" {
		title = "Untitled canvas"
	}
	doc := map[string]any{
		"id": id, "title": title, "nodes": []any{}, "connections": []any{},
		"chatSessions": []any{}, "activeChatId": nil, "backgroundMode": "dots",
		"showImageInfo": false, "viewport": map[string]any{"x": 0, "y": 0, "k": 1},
		"directorScenes": []any{}, "createdAt": now, "updatedAt": now,
	}
	payload, _ := json.Marshal(doc)
	_, err := e.DB.Exec(`INSERT INTO canvas_projects(id, title, payload_json, timeline_json, revision, workspace_mode, theme, project_link, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		id, title, string(payload), "", 1, "simple", "", "", now, now)
	if err != nil {
		return nil, err
	}
	doc["revision"] = 1
	return doc, nil
}

func (e *Engine) BindCanvasSession(sessionID, canvasID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session required")
	}
	if _, err := e.getCanvasProject(canvasID); err != nil {
		return err
	}
	now := Now()
	_, err := e.DB.Exec(`INSERT INTO canvas_session_binds(session_id, canvas_id, updated_at) VALUES(?,?,?)
		ON CONFLICT(session_id) DO UPDATE SET canvas_id=excluded.canvas_id, updated_at=excluded.updated_at`, sessionID, canvasID, now)
	return err
}

func (e *Engine) CanvasSessionBind(sessionID string) (string, bool) {
	var id string
	err := e.DB.QueryRow(`SELECT canvas_id FROM canvas_session_binds WHERE session_id = ?`, sessionID).Scan(&id)
	if err == sql.ErrNoRows || id == "" {
		return "", false
	}
	return id, err == nil
}

func (e *Engine) saveCanvasPayload(id string, payload []byte, title string) error {
	now := Now()
	_, err := e.DB.Exec(`UPDATE canvas_projects SET payload_json=?, title=?, revision=revision+1, updated_at=? WHERE id=? AND deleted_at=''`, string(payload), title, now, id)
	return err
}
