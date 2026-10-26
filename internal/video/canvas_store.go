// 行为：画布项目的读取、按修订号保存、历史快照和会话绑定。

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

// getCanvasProject 读取未删除的画布项目行。
// id 传给 getDoc，从 colProjects 取值。
func (e *Engine) getCanvasProject(id string) (canvasProjectRow, error) {
	r, err := getDoc[canvasProjectRow](e, colProjects, id)
	if err != nil || r.DeletedAt != "" {
		return canvasProjectRow{}, fmt.Errorf("canvas not found")
	}
	return r, nil
}

// listCanvasProjectRows 按标题或 id 搜索画布，支持按名称、节点数或更新时间排序并分页。
// search 去掉空白并转成小写，非空时要出现在标题或 ID 的小写文本里。
// sortKey 为 name 时按标题排序，为 nodes 时按节点数再按更新时间，其他值按更新时间。
// page 小于 1 时改成 1，再用来计算切片起点。
// pageSize 小于等于 0 时改成 40，再用来计算切片长度。
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

// decodeProjectDoc 把画布 JSON 解成对象。空文档、null 或坏 JSON 返回格式错误。
// raw 去掉空白后为空或等于 null 时返回格式错误，否则按 JSON 对象解析，结果为 nil 也返回格式错误。
func decodeProjectDoc(raw []byte) (map[string]any, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, fmt.Errorf("画布数据格式错误")
	}
	m := map[string]any{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("画布数据格式错误")
	}
	if m == nil {
		return nil, fmt.Errorf("画布数据格式错误")
	}
	return m, nil
}

// projectDocFromRow 把项目行补成前端文档，缺的节点、连线、会话和视口用空值补上。
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

// canvasSummary 返回画布摘要，预览节点最多 4 个。
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

// canvasNodeCount 数出文档里的节点数量。
// payload 按 JSON 解析后返回 nodes 的长度。
func canvasNodeCount(payload string) int {
	var document struct {
		Nodes []json.RawMessage `json:"nodes"`
	}
	_ = json.Unmarshal([]byte(payload), &document)
	return len(document.Nodes)
}

// canvasConnectionCount 数出文档里的连线数量。
// payload 按 JSON 解析后返回 connections 的长度。
func canvasConnectionCount(payload string) int {
	var document struct {
		Connections []json.RawMessage `json:"connections"`
	}
	_ = json.Unmarshal([]byte(payload), &document)
	return len(document.Connections)
}

// upsertCanvasProject 按客户端修订号保存画布。修订号不一致则冲突。非修复保存会先检查引用的资源都在。
// raw 传给 decodeProjectDoc。
// repair 为 false 时把序列化后的文档传给 validateCanvasResources；为 true 时把快照原因写成 before_resource_repair。
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
		title = "无限画布"
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

// validateCanvasResources 拒绝引用了已缺失资源的画布。
// payload 传给 collectResourceIDs。
func (e *Engine) validateCanvasResources(payload []byte) error {
	ids := collectResourceIDs(payload)
	for _, id := range ids {
		if rec, err := getDoc[canvasResourceRec](e, colResources, id); err != nil || rec.DeletedAt != "" {
			return fmt.Errorf("画布引用了缺失资源 %s", id)
		}
	}
	return nil
}

// collectResourceIDs 从文档文本里收集 resource: 后面的资源 id。
// payload 转成字符串后查找 resource:，并收集后面的连续标识字符。
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

// maybeSnapshot 在保存前写入快照。普通保存间隔短于 5 分钟则跳过，恢复和修复强制记一版，并只留最近 20 条。
// existing 的 ID、Revision、Title、PayloadJSON 写入快照，并用 ID 找出同一画布的旧快照。
// reason 为 before_restore 或 before_resource_repair 时不看 5 分钟间隔，并写入快照的 Reason。
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

// listCanvasHistory 列出画布最近的快照和当前修订号。
// id 传给 getCanvasProject，并筛选同一画布的快照，超过 20 条时只留前 20 条。
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

// getCanvasSnapshot 读取一条快照及其当时的画布文档。
// canvasID 必须与快照的 CanvasID 相等，并写入返回文档的 id。
// snapshotID 传给 getDoc，从 colSnapshots 取值。
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

// restoreCanvasSnapshot 在修订号仍匹配时把快照写回画布，并先记一份恢复前快照。
// canvasID 传给 getCanvasProject、getCanvasSnapshot 和 putDoc。
// snapshotID 传给 getCanvasSnapshot。
// revision 与当前 Revision 不等时返回冲突。
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

// deleteCanvasProject 把画布标成已删除。
// id 传给 getCanvasProject 和 putDoc。
func (e *Engine) deleteCanvasProject(id string) error {
	r, err := e.getCanvasProject(id)
	if err != nil {
		return err
	}
	r.DeletedAt = Now()
	return e.putDoc(colProjects, id, r)
}

// createEmptyCanvas 创建空画布，标题默认是无限画布，修订号从 1 开始。
// title 去掉空白后为空时写成无限画布，并写入文档和项目行。
func (e *Engine) createEmptyCanvas(title string) (map[string]any, error) {
	id := NewID()
	now := Now()
	if strings.TrimSpace(title) == "" {
		title = "无限画布"
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

// BindCanvasSession 把会话绑到已经存在的画布。
// sessionID 去掉空白后为空则返回 session required，否则作为绑定表的键。
// canvasID 传给 getCanvasProject，并作为绑定值。
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

// CanvasSessionBind 返回会话当前绑定的画布 id。
// sessionID 作为绑定表的键取出画布 id。
func (e *Engine) CanvasSessionBind(sessionID string) (string, bool) {
	m := e.loadCanvasBinds()
	id := m[sessionID]
	if id == "" {
		return "", false
	}
	return id, true
}

// saveCanvasPayload 在期望修订号仍匹配时写回画布正文并把修订号加一。
// id 传给 getCanvasProject 和 putDoc。
// payload 写入 PayloadJSON。
// title 写入 Title。
// expectedRev 与当前 Revision 不等时返回「画布已被更新，请重新加载」。
func (e *Engine) saveCanvasPayload(id string, payload []byte, title string, expectedRev int64) error {
	r, err := e.getCanvasProject(id)
	if err != nil {
		return err
	}
	if r.Revision != expectedRev {
		return fmt.Errorf("画布已被更新，请重新加载")
	}
	r.PayloadJSON = string(payload)
	r.Title = title
	r.Revision++
	r.UpdatedAt = Now()
	return e.putDoc(colProjects, id, r)
}
