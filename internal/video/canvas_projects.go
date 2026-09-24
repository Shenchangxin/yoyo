package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (e *Engine) listDramaProjects() []map[string]any {
	rows, err := e.DB.Query(`SELECT id, title, payload_json, created_at, updated_at FROM canvas_project_units WHERE kind = 'project' AND deleted_at = '' ORDER BY updated_at DESC`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, title, payload, created, updated string
		if rows.Scan(&id, &title, &payload, &created, &updated) != nil {
			continue
		}
		item := map[string]any{"id": id, "name": title, "title": title, "createdAt": created, "updatedAt": updated, "status": "active"}
		var extra map[string]any
		_ = json.Unmarshal([]byte(payload), &extra)
		for k, v := range extra {
			if _, exists := item[k]; !exists {
				item[k] = v
			}
		}
		out = append(out, item)
	}
	return out
}

func (e *Engine) upsertDramaProject(in map[string]any) map[string]any {
	id := strAnyMap(in, "id")
	if id == "" {
		id = NewID()
	}
	now := Now()
	title := firstNonEmpty(strAnyMap(in, "name"), strAnyMap(in, "title"), "Untitled project")
	payload, _ := json.Marshal(in)
	var n int
	_ = e.DB.QueryRow(`SELECT COUNT(*) FROM canvas_project_units WHERE id = ?`, id).Scan(&n)
	if n == 0 {
		_, _ = e.DB.Exec(`INSERT INTO canvas_project_units(id, project_id, kind, title, payload_json, sort_order, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)`,
			id, id, "project", title, string(payload), 0, now, now)
	} else {
		_, _ = e.DB.Exec(`UPDATE canvas_project_units SET title=?, payload_json=?, updated_at=? WHERE id=?`, title, string(payload), now, id)
	}
	in["id"] = id
	in["name"] = title
	in["title"] = title
	in["updatedAt"] = now
	if in["createdAt"] == nil {
		in["createdAt"] = now
	}
	return in
}

func (e *Engine) getDramaProject(id string) (map[string]any, error) {
	var title, payload, created, updated string
	err := e.DB.QueryRow(`SELECT title, payload_json, created_at, updated_at FROM canvas_project_units WHERE id = ? AND kind = 'project' AND deleted_at = ''`, id).Scan(&title, &payload, &created, &updated)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	item := map[string]any{"id": id, "name": title, "title": title, "createdAt": created, "updatedAt": updated}
	var extra map[string]any
	_ = json.Unmarshal([]byte(payload), &extra)
	for k, v := range extra {
		item[k] = v
	}
	item["units"] = e.listProjectUnits(id)
	return item, nil
}

func (e *Engine) deleteDramaProject(id string) error {
	now := Now()
	_, err := e.DB.Exec(`UPDATE canvas_project_units SET deleted_at = ? WHERE id = ? OR project_id = ?`, now, id, id)
	return err
}

func (e *Engine) listProjectUnits(projectID string) []map[string]any {
	rows, err := e.DB.Query(`SELECT id, kind, title, payload_json, sort_order, created_at, updated_at FROM canvas_project_units WHERE project_id = ? AND kind != 'project' AND deleted_at = '' ORDER BY sort_order ASC, created_at ASC`, projectID)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, kind, title, payload, created, updated string
		var sort int
		if rows.Scan(&id, &kind, &title, &payload, &sort, &created, &updated) != nil {
			continue
		}
		item := map[string]any{"id": id, "projectId": projectID, "kind": kind, "title": title, "position": sort, "createdAt": created, "updatedAt": updated}
		var extra map[string]any
		_ = json.Unmarshal([]byte(payload), &extra)
		for k, v := range extra {
			item[k] = v
		}
		out = append(out, item)
	}
	return out
}

func (e *Engine) upsertProjectUnit(projectID string, in map[string]any) map[string]any {
	id := strAnyMap(in, "id")
	if id == "" {
		id = NewID()
	}
	now := Now()
	title := firstNonEmpty(strAnyMap(in, "title"), strAnyMap(in, "name"), "Chapter")
	kind := firstNonEmpty(strAnyMap(in, "kind"), "chapter")
	payload, _ := json.Marshal(in)
	sort := intAny(in["position"])
	var n int
	_ = e.DB.QueryRow(`SELECT COUNT(*) FROM canvas_project_units WHERE id = ?`, id).Scan(&n)
	if n == 0 {
		_, _ = e.DB.Exec(`INSERT INTO canvas_project_units(id, project_id, kind, title, payload_json, sort_order, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)`,
			id, projectID, kind, title, string(payload), sort, now, now)
	} else {
		_, _ = e.DB.Exec(`UPDATE canvas_project_units SET title=?, payload_json=?, sort_order=?, updated_at=? WHERE id=?`, title, string(payload), sort, now, id)
	}
	in["id"] = id
	in["projectId"] = projectID
	in["title"] = title
	in["kind"] = kind
	return in
}

func (e *Engine) deleteProjectUnit(id string) error {
	_, err := e.DB.Exec(`UPDATE canvas_project_units SET deleted_at = ? WHERE id = ?`, Now(), id)
	return err
}

func (e *Engine) listAssetsPage(kind, category, folderID, status, q string, page, pageSize int, uncategorized bool) map[string]any {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 40
	}
	where := `deleted_at = ''`
	var args []any
	if kind != "" {
		where += ` AND kind = ?`
		args = append(args, kind)
	}
	if category != "" {
		where += ` AND category = ?`
		args = append(args, category)
	}
	if folderID != "" {
		where += ` AND folder_id = ?`
		args = append(args, folderID)
	}
	if uncategorized {
		where += ` AND folder_id = ''`
	}
	if status != "" {
		where += ` AND status = ?`
		args = append(args, status)
	}
	if q != "" {
		where += ` AND title LIKE ?`
		args = append(args, "%"+q+"%")
	}
	var total int
	_ = e.DB.QueryRow(`SELECT COUNT(*) FROM canvas_assets WHERE `+where, args...).Scan(&total)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := e.DB.Query(`SELECT id, folder_id, kind, category, title, resource_id, payload_json, status, created_at, updated_at FROM canvas_assets WHERE `+where+` ORDER BY updated_at DESC LIMIT ? OFFSET ?`, args...)
	assets := []map[string]any{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, folder, knd, cat, title, resID, payload, st, created, updated string
			if rows.Scan(&id, &folder, &knd, &cat, &title, &resID, &payload, &st, &created, &updated) != nil {
				continue
			}
			item := map[string]any{"id": id, "folderId": folder, "kind": knd, "category": cat, "title": title, "resourceId": resID, "status": st, "createdAt": created, "updatedAt": updated}
			var extra map[string]any
			_ = json.Unmarshal([]byte(payload), &extra)
			for k, v := range extra {
				item[k] = v
			}
			assets = append(assets, item)
		}
	}
	return map[string]any{
		"assets": assets, "page": page, "pageSize": pageSize, "total": total, "hasMore": page*pageSize < total,
		"kindCounts": map[string]int{}, "categoryCounts": map[string]int{}, "folderCounts": map[string]int{},
	}
}

func (e *Engine) upsertAsset(in map[string]any) map[string]any {
	id := strAnyMap(in, "id")
	if id == "" {
		id = NewID()
	}
	now := Now()
	payload, _ := json.Marshal(in)
	_, _ = e.DB.Exec(`INSERT INTO canvas_assets(id, folder_id, kind, category, title, resource_id, payload_json, status, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET folder_id=excluded.folder_id, kind=excluded.kind, category=excluded.category, title=excluded.title, resource_id=excluded.resource_id, payload_json=excluded.payload_json, status=excluded.status, updated_at=excluded.updated_at, deleted_at=''`,
		id, strAnyMap(in, "folderId"), firstNonEmpty(strAnyMap(in, "kind"), "image"), strAnyMap(in, "category"), firstNonEmpty(strAnyMap(in, "title"), "Asset"), strAnyMap(in, "resourceId"), string(payload), firstNonEmpty(strAnyMap(in, "status"), "ready"), now, now)
	in["id"] = id
	in["updatedAt"] = now
	return in
}

func (e *Engine) getAsset(id string) (map[string]any, error) {
	page := e.listAssetsPage("", "", "", "", "", 1, 500, false)
	for _, a := range page["assets"].([]map[string]any) {
		if fmt.Sprint(a["id"]) == id {
			return a, nil
		}
	}
	return nil, fmt.Errorf("asset not found")
}

func (e *Engine) deleteAsset(id string) error {
	_, err := e.DB.Exec(`UPDATE canvas_assets SET deleted_at = ? WHERE id = ?`, Now(), id)
	return err
}

func (e *Engine) listAssetFolders() []map[string]any {
	rows, err := e.DB.Query(`SELECT id, name, position, created_at, updated_at FROM canvas_asset_folders WHERE deleted_at = '' ORDER BY position ASC`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, created, updated string
		var pos int
		if rows.Scan(&id, &name, &pos, &created, &updated) != nil {
			continue
		}
		out = append(out, map[string]any{"id": id, "name": name, "position": pos, "createdAt": created, "updatedAt": updated})
	}
	return out
}

func (e *Engine) upsertAssetFolder(id, name string) map[string]any {
	if id == "" {
		id = NewID()
	}
	now := Now()
	if strings.TrimSpace(name) == "" {
		name = "Folder"
	}
	_, _ = e.DB.Exec(`INSERT INTO canvas_asset_folders(id, name, position, created_at, updated_at) VALUES(?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, updated_at=excluded.updated_at`, id, name, 0, now, now)
	return map[string]any{"id": id, "name": name, "position": 0, "createdAt": now, "updatedAt": now}
}

func (e *Engine) deleteAssetFolder(id string) error {
	_, err := e.DB.Exec(`UPDATE canvas_asset_folders SET deleted_at = ? WHERE id = ?`, Now(), id)
	return err
}

func (e *Engine) moveAssetsFolder(ids []any, folderID string) {
	for _, id := range ids {
		_, _ = e.DB.Exec(`UPDATE canvas_assets SET folder_id = ?, updated_at = ? WHERE id = ?`, folderID, Now(), fmt.Sprint(id))
	}
}

func (e *Engine) localUser() map[string]any {
	now := Now()
	return map[string]any{
		"id": "local", "username": "yoyo", "displayName": "Yoyo", "role": "admin", "status": "active",
		"createdAt": now, "updatedAt": now,
	}
}

func (e *Engine) authSession() map[string]any {
	return map[string]any{
		"user": e.localUser(),
		"runtimeLimits": map[string]any{"activeTaskLimit": 8, "resourceUploadMB": 512, "recycleBinRetentionDays": 30},
		"drawingEngine": map[string]any{"defaultEngine": "tldraw"},
		"features": map[string]any{
			"welcomeEnabled": false, "shortDramaEnabled": true, "taskCenterEnabled": true,
			"creditsEnabled": false, "customChannelsEnabled": true, "frontendModelsEnabled": true,
			"pluginCenterEnabled": true, "systemPluginsVisibleToUsers": true,
		},
	}
}

func (e *Engine) disabledOSS() map[string]any {
	return map[string]any{
		"enabled": false, "provider": "none", "s3Preset": "", "region": "", "endpoint": "", "cdnBaseUrl": "",
		"cdnAuthMode": "", "requireCDN": false, "allowPrivateProxy": false, "bucket": "", "accessKeyId": "",
		"hasAccessKeySecret": false, "hasSessionToken": false, "pathStyle": false, "allowUserS3": false,
		"publicBaseUrl": "", "pathPrefix": "",
	}
}
