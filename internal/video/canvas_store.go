package video

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const canvasHistoryLimit = 20
const canvasHistoryInterval = 5 * time.Minute

type canvasProjectRow struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	PayloadJSON   string `json:"payload_json"`
	TimelineJSON  string `json:"timeline_json"`
	Revision      int64  `json:"revision"`
	WorkspaceMode string `json:"workspace_mode"`
	Theme         string `json:"theme"`
	ProjectLink   string `json:"project_link"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	DeletedAt     string `json:"deleted_at,omitempty"`
}

func (e *Engine) getCanvasProject(id string) (canvasProjectRow, error) {
	r, err := getDoc[canvasProjectRow](e, colProjects, id)
	if err != nil || r.DeletedAt != "" {
		return canvasProjectRow{}, fmt.Errorf("canvas not found")
	}
	return r, nil
}

func (e *Engine) listCanvasProjectRows(search, sortKey string, page, pageSize int) ([]canvasProjectRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 40
	}
	q := strings.ToLower(strings.TrimSpace(search))
	var all []canvasProjectRow
	for _, r := range loadCol[canvasProjectRow](e, colProjects) {
		if r.DeletedAt != "" {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(r.Title), q) && !strings.Contains(strings.ToLower(r.ID), q) {
			continue
		}
		all = append(all, r)
	}
	switch sortKey {
	case "name":
		sort.Slice(all, func(i, j int) bool { return strings.ToLower(all[i].Title) < strings.ToLower(all[j].Title) })
	case "nodes":
		sort.Slice(all, func(i, j int) bool {
			ni, nj := canvasNodeCount(all[i].PayloadJSON), canvasNodeCount(all[j].PayloadJSON)
			if ni != nj {
				return ni > nj
			}
			return all[i].UpdatedAt > all[j].UpdatedAt
		})
	default:
		sort.Slice(all, func(i, j int) bool { return all[i].UpdatedAt > all[j].UpdatedAt })
	}
	total := len(all)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	out := all[start:end]
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
	row := canvasProjectRow{
		ID: id, Title: title, PayloadJSON: string(payload), Revision: newRev,
		WorkspaceMode: "simple", ProjectLink: projectLink, CreatedAt: created, UpdatedAt: now,
	}
	if found {
		_ = e.maybeSnapshot(existing, reason)
		row.TimelineJSON = existing.TimelineJSON
		row.WorkspaceMode = existing.WorkspaceMode
		row.Theme = existing.Theme
	}
	if err := e.putDoc(colProjects, id, row); err != nil {
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
		if rec, err := getDoc[canvasResourceRec](e, colResources, id); err != nil || rec.DeletedAt != "" {
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
		var latest string
		for _, s := range loadCol[canvasSnapshotRec](e, colSnapshots) {
			if s.CanvasID == existing.ID && s.CreatedAt > latest {
				latest = s.CreatedAt
			}
		}
		if latest != "" {
			if t, err := time.Parse(time.RFC3339Nano, latest); err == nil && time.Since(t) < canvasHistoryInterval {
				return nil
			}
		}
	}
	now := Now()
	snap := canvasSnapshotRec{
		ID: NewID(), CanvasID: existing.ID, Revision: existing.Revision, Title: existing.Title,
		PayloadJSON: existing.PayloadJSON, Reason: reason,
		NodeCount: canvasNodeCount(existing.PayloadJSON), ConnectionCount: canvasConnectionCount(existing.PayloadJSON),
		CreatedAt: now,
	}
	if err := e.putDoc(colSnapshots, snap.ID, snap); err != nil {
		return err
	}
	var ids []canvasSnapshotRec
	for _, s := range loadCol[canvasSnapshotRec](e, colSnapshots) {
		if s.CanvasID == existing.ID {
			ids = append(ids, s)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].CreatedAt > ids[j].CreatedAt })
	if len(ids) > canvasHistoryLimit {
		for _, s := range ids[canvasHistoryLimit:] {
			_ = e.delDoc(colSnapshots, s.ID)
		}
	}
	return nil
}

func (e *Engine) listCanvasHistory(id string) (map[string]any, error) {
	row, err := e.getCanvasProject(id)
	if err != nil {
		return nil, err
	}
	var snaps []map[string]any
	var list []canvasSnapshotRec
	for _, s := range loadCol[canvasSnapshotRec](e, colSnapshots) {
		if s.CanvasID == id {
			list = append(list, s)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt > list[j].CreatedAt })
	if len(list) > canvasHistoryLimit {
		list = list[:canvasHistoryLimit]
	}
	for _, s := range list {
		snaps = append(snaps, map[string]any{
			"id": s.ID, "canvasId": s.CanvasID, "revision": s.Revision, "title": s.Title,
			"nodeCount": s.NodeCount, "connectionCount": s.ConnectionCount, "payloadBytes": len(s.PayloadJSON),
			"reason": s.Reason, "createdAt": s.CreatedAt, "contentUpdatedAt": s.CreatedAt,
		})
	}
	if snaps == nil {
		snaps = []map[string]any{}
	}
	return map[string]any{"snapshots": snaps, "currentRevision": row.Revision}, nil
}

func (e *Engine) getCanvasSnapshot(canvasID, snapshotID string) (map[string]any, map[string]any, error) {
	s, err := getDoc[canvasSnapshotRec](e, colSnapshots, snapshotID)
	if err != nil || s.CanvasID != canvasID {
		return nil, nil, fmt.Errorf("snapshot not found")
	}
	snap := map[string]any{
		"id": s.ID, "canvasId": s.CanvasID, "revision": s.Revision, "title": s.Title,
		"nodeCount": s.NodeCount, "connectionCount": s.ConnectionCount, "payloadBytes": len(s.PayloadJSON),
		"reason": s.Reason, "createdAt": s.CreatedAt, "contentUpdatedAt": s.CreatedAt,
	}
	doc := map[string]any{}
	_ = json.Unmarshal([]byte(s.PayloadJSON), &doc)
	doc["id"] = canvasID
	doc["title"] = s.Title
	doc["revision"] = s.Revision
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
	existing.Title = title
	existing.PayloadJSON = string(payload)
	existing.Revision = newRev
	existing.UpdatedAt = now
	if err := e.putDoc(colProjects, canvasID, existing); err != nil {
		return nil, canvasFail(500, err.Error(), "internal")
	}
	return map[string]any{"id": canvasID, "title": title, "revision": newRev, "createdAt": existing.CreatedAt, "updatedAt": now}, canvasOK(nil)
}

func (e *Engine) deleteCanvasProject(id string) error {
	r, err := e.getCanvasProject(id)
	if err != nil {
		return err
	}
	r.DeletedAt = Now()
	return e.putDoc(colProjects, id, r)
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
	row := canvasProjectRow{
		ID: id, Title: title, PayloadJSON: string(payload), Revision: 1,
		WorkspaceMode: "simple", CreatedAt: now, UpdatedAt: now,
	}
	if err := e.putDoc(colProjects, id, row); err != nil {
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
	m := e.loadCanvasBinds()
	m[sessionID] = canvasID
	return e.saveCanvasBinds(m)
}

func (e *Engine) CanvasSessionBind(sessionID string) (string, bool) {
	m := e.loadCanvasBinds()
	id := m[sessionID]
	if id == "" {
		return "", false
	}
	return id, true
}

func (e *Engine) saveCanvasPayload(id string, payload []byte, title string) error {
	r, err := e.getCanvasProject(id)
	if err != nil {
		return err
	}
	r.PayloadJSON = string(payload)
	r.Title = title
	r.Revision++
	r.UpdatedAt = Now()
	return e.putDoc(colProjects, id, r)
}
