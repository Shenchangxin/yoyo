package video

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

func dramaProjectRecord(p map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range p {
		if k == "units" {
			continue
		}
		out[k] = v
	}
	if out["status"] == nil {
		out["status"] = "active"
	}
	if strAnyMap(out, "name") == "" {
		out["name"] = firstNonEmpty(strAnyMap(out, "title"), "Untitled project")
	}
	return out
}

func dramaProjectSummaries(list []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		out = append(out, map[string]any{
			"project":            dramaProjectRecord(p),
			"canvasCount":        0,
			"assetCount":         0,
			"unitCount":          0,
			"completedUnitCount": 0,
		})
	}
	return out
}

func unitToMap(u canvasUnitRec) map[string]any {
	item := map[string]any{
		"id": u.ID, "projectId": u.ProjectID, "kind": u.Kind, "title": u.Title,
		"position": u.SortOrder, "createdAt": u.CreatedAt, "updatedAt": u.UpdatedAt,
	}
	var extra map[string]any
	_ = json.Unmarshal([]byte(u.PayloadJSON), &extra)
	for k, v := range extra {
		if _, exists := item[k]; !exists {
			item[k] = v
		}
	}
	return item
}

func (e *Engine) listDramaProjects() []map[string]any {
	var recs []canvasUnitRec
	for _, u := range loadCol[canvasUnitRec](e, colUnits) {
		if u.Kind == "project" && u.DeletedAt == "" {
			recs = append(recs, u)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].UpdatedAt > recs[j].UpdatedAt })
	out := []map[string]any{}
	for _, u := range recs {
		item := map[string]any{"id": u.ID, "name": u.Title, "title": u.Title, "createdAt": u.CreatedAt, "updatedAt": u.UpdatedAt, "status": "active"}
		var extra map[string]any
		_ = json.Unmarshal([]byte(u.PayloadJSON), &extra)
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
	rec, err := getDoc[canvasUnitRec](e, colUnits, id)
	if err != nil {
		rec = canvasUnitRec{ID: id, ProjectID: id, Kind: "project", CreatedAt: now}
	}
	rec.Title = title
	rec.PayloadJSON = string(payload)
	rec.UpdatedAt = now
	rec.DeletedAt = ""
	_ = e.putDoc(colUnits, id, rec)
	in["id"] = id
	in["name"] = title
	in["title"] = title
	in["updatedAt"] = now
	if in["createdAt"] == nil {
		in["createdAt"] = rec.CreatedAt
	}
	return in
}

func (e *Engine) getDramaProject(id string) (map[string]any, error) {
	u, err := getDoc[canvasUnitRec](e, colUnits, id)
	if err != nil || u.Kind != "project" || u.DeletedAt != "" {
		return nil, fmt.Errorf("project not found")
	}
	item := map[string]any{"id": id, "name": u.Title, "title": u.Title, "createdAt": u.CreatedAt, "updatedAt": u.UpdatedAt}
	var extra map[string]any
	_ = json.Unmarshal([]byte(u.PayloadJSON), &extra)
	for k, v := range extra {
		item[k] = v
	}
	item["units"] = e.listProjectUnits(id)
	return item, nil
}

func (e *Engine) deleteDramaProject(id string) error {
	now := Now()
	for _, u := range loadCol[canvasUnitRec](e, colUnits) {
		if u.ID == id || u.ProjectID == id {
			u.DeletedAt = now
			_ = e.putDoc(colUnits, u.ID, u)
		}
	}
	return nil
}

func (e *Engine) listProjectUnits(projectID string) []map[string]any {
	var recs []canvasUnitRec
	for _, u := range loadCol[canvasUnitRec](e, colUnits) {
		if u.ProjectID == projectID && u.Kind != "project" && u.DeletedAt == "" {
			recs = append(recs, u)
		}
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].SortOrder != recs[j].SortOrder {
			return recs[i].SortOrder < recs[j].SortOrder
		}
		return recs[i].CreatedAt < recs[j].CreatedAt
	})
	out := []map[string]any{}
	for _, u := range recs {
		item := unitToMap(u)
		item["projectId"] = projectID
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
	sortOrder := intAny(in["position"])
	rec, err := getDoc[canvasUnitRec](e, colUnits, id)
	if err != nil {
		rec = canvasUnitRec{ID: id, CreatedAt: now}
	}
	rec.ProjectID = projectID
	rec.Kind = kind
	rec.Title = title
	rec.PayloadJSON = string(payload)
	rec.SortOrder = sortOrder
	rec.UpdatedAt = now
	rec.DeletedAt = ""
	_ = e.putDoc(colUnits, id, rec)
	in["id"] = id
	in["projectId"] = projectID
	in["title"] = title
	in["kind"] = kind
	return in
}

func (e *Engine) deleteProjectUnit(id string) error {
	rec, err := getDoc[canvasUnitRec](e, colUnits, id)
	if err != nil {
		return err
	}
	rec.DeletedAt = Now()
	return e.putDoc(colUnits, id, rec)
}

func matchAsset(a canvasAssetRec, kind, category, folderID, status, q string, uncategorized bool, excludeKind string) bool {
	if a.DeletedAt != "" {
		return false
	}
	if excludeKind != "" && a.Kind == excludeKind {
		return false
	}
	if kind != "" && a.Kind != kind {
		return false
	}
	if category != "" && a.Category != category {
		return false
	}
	if folderID != "" && a.FolderID != folderID {
		return false
	}
	if uncategorized && a.FolderID != "" {
		return false
	}
	if status == "active" {
		if a.Status == "archived" {
			return false
		}
	} else if status != "" && a.Status != status {
		return false
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q != "" {
		hay := strings.ToLower(a.Title + " " + a.PayloadJSON)
		if !strings.Contains(hay, q) {
			return false
		}
	}
	return true
}

func assetToMap(a canvasAssetRec) map[string]any {
	item := map[string]any{
		"id": a.ID, "folderId": a.FolderID, "kind": a.Kind, "category": a.Category, "title": a.Title,
		"resourceId": a.ResourceID, "status": a.Status, "createdAt": a.CreatedAt, "updatedAt": a.UpdatedAt,
	}
	var extra map[string]any
	_ = json.Unmarshal([]byte(a.PayloadJSON), &extra)
	for k, v := range extra {
		item[k] = v
	}
	return item
}

func (e *Engine) listAssetsPage(kind, category, folderID, status, q string, page, pageSize int, uncategorized bool, excludeKind string) map[string]any {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 40
	}
	var matched []canvasAssetRec
	for _, a := range loadCol[canvasAssetRec](e, colAssets) {
		if matchAsset(a, kind, category, folderID, status, q, uncategorized, excludeKind) {
			matched = append(matched, a)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].UpdatedAt > matched[j].UpdatedAt })
	total := len(matched)
	kindCounts, categoryCounts, folderCounts := e.scanAssetFacets(status, q, excludeKind)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	assets := []map[string]any{}
	for _, a := range matched[start:end] {
		assets = append(assets, assetToMap(a))
	}
	return map[string]any{
		"assets": assets, "page": page, "pageSize": pageSize, "total": total, "hasMore": page*pageSize < total,
		"kindCounts": kindCounts, "categoryCounts": categoryCounts, "folderCounts": folderCounts,
	}
}

func (e *Engine) scanAssetFacets(status, q, excludeKind string) (map[string]int, map[string]int, map[string]int) {
	kinds := map[string]int{}
	cats := map[string]int{}
	folders := map[string]int{}
	for _, a := range loadCol[canvasAssetRec](e, colAssets) {
		if !matchAsset(a, "", "", "", status, q, false, excludeKind) {
			continue
		}
		kinds[a.Kind]++
		cat := a.Category
		if cat == "" {
			cat = "other"
		}
		cats[cat]++
		folders[a.FolderID]++
	}
	return kinds, cats, folders
}

func (e *Engine) upsertAsset(in map[string]any) map[string]any {
	id := strAnyMap(in, "id")
	if id == "" {
		id = NewID()
	}
	now := Now()
	payload, _ := json.Marshal(in)
	rec, err := getDoc[canvasAssetRec](e, colAssets, id)
	if err != nil {
		rec = canvasAssetRec{ID: id, CreatedAt: now}
	}
	rec.FolderID = strAnyMap(in, "folderId")
	rec.Kind = firstNonEmpty(strAnyMap(in, "kind"), "image")
	rec.Category = strAnyMap(in, "category")
	rec.Title = firstNonEmpty(strAnyMap(in, "title"), "Asset")
	rec.ResourceID = strAnyMap(in, "resourceId")
	rec.PayloadJSON = string(payload)
	rec.Status = firstNonEmpty(strAnyMap(in, "status"), "confirmed")
	rec.UpdatedAt = now
	rec.DeletedAt = ""
	_ = e.putDoc(colAssets, id, rec)
	in["id"] = id
	in["updatedAt"] = now
	return in
}

func (e *Engine) getAsset(id string) (map[string]any, error) {
	a, err := getDoc[canvasAssetRec](e, colAssets, id)
	if err != nil || a.DeletedAt != "" {
		return nil, fmt.Errorf("asset not found")
	}
	return assetToMap(a), nil
}

func (e *Engine) deleteAsset(id string) error {
	a, err := getDoc[canvasAssetRec](e, colAssets, id)
	if err != nil {
		return err
	}
	a.DeletedAt = Now()
	return e.putDoc(colAssets, id, a)
}

func (e *Engine) listAssetFolders() []map[string]any {
	var recs []canvasFolderRec
	for _, f := range loadCol[canvasFolderRec](e, colFolders) {
		if f.DeletedAt == "" {
			recs = append(recs, f)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Position < recs[j].Position })
	out := []map[string]any{}
	for _, f := range recs {
		out = append(out, map[string]any{"id": f.ID, "name": f.Name, "position": f.Position, "createdAt": f.CreatedAt, "updatedAt": f.UpdatedAt})
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
	rec, err := getDoc[canvasFolderRec](e, colFolders, id)
	if err != nil {
		rec = canvasFolderRec{ID: id, CreatedAt: now}
	}
	rec.Name = name
	rec.UpdatedAt = now
	rec.DeletedAt = ""
	_ = e.putDoc(colFolders, id, rec)
	return map[string]any{"id": id, "name": name, "position": rec.Position, "createdAt": rec.CreatedAt, "updatedAt": now}
}

func (e *Engine) deleteAssetFolder(id string) error {
	f, err := getDoc[canvasFolderRec](e, colFolders, id)
	if err != nil {
		return err
	}
	f.DeletedAt = Now()
	return e.putDoc(colFolders, id, f)
}

func (e *Engine) moveAssetsFolder(ids []any, folderID string) {
	now := Now()
	for _, id := range ids {
		a, err := getDoc[canvasAssetRec](e, colAssets, fmt.Sprint(id))
		if err != nil {
			continue
		}
		a.FolderID = folderID
		a.UpdatedAt = now
		_ = e.putDoc(colAssets, a.ID, a)
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
		"user":          e.localUser(),
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
	if e.Conn != nil {
		if c, err := e.Conn.Active("storage", ""); err == nil && c.Protocol == "s3" {
			st := c.Settings
			if st == nil {
				st = map[string]any{}
			}
			return map[string]any{
				"enabled": true, "provider": "s3", "s3Preset": strAnyMap(st, "preset"),
				"region": strAnyMap(st, "region"), "endpoint": c.Endpoint,
				"cdnBaseUrl": strAnyMap(st, "cdn_base_url"), "cdnAuthMode": "", "requireCDN": false,
				"allowPrivateProxy": true, "bucket": strAnyMap(st, "bucket"), "accessKeyId": "",
				"hasAccessKeySecret": c.HasKey, "hasSessionToken": false,
				"pathStyle": boolAny(st["path_style"]), "allowUserS3": true,
				"publicBaseUrl": strAnyMap(st, "public_base_url"), "pathPrefix": strAnyMap(st, "path_prefix"),
			}
		}
	}
	return map[string]any{
		"enabled": false, "provider": "cas", "s3Preset": "", "region": "", "endpoint": "", "cdnBaseUrl": "",
		"cdnAuthMode": "", "requireCDN": false, "allowPrivateProxy": false, "bucket": "", "accessKeyId": "",
		"hasAccessKeySecret": false, "hasSessionToken": false, "pathStyle": false, "allowUserS3": true,
		"publicBaseUrl": "", "pathPrefix": "",
	}
}

func (e *Engine) applyOSS(in map[string]any) map[string]any {
	if e.Conn == nil {
		return e.disabledOSS()
	}
	provider := strings.ToLower(strAnyMap(in, "provider"))
	enabled := boolAny(in["enabled"])
	if !enabled || provider == "" || provider == "cas" {
		_ = e.Conn.SetDefault(connection.CapStorage, "cas-local")
		return e.disabledOSS()
	}
	id := ""
	if c, err := e.Conn.Active(connection.CapStorage, ""); err == nil && c.Protocol == "s3" {
		id = c.ID
	}
	st := map[string]any{
		"preset":          strAnyMap(in, "s3Preset", "preset"),
		"region":          strAnyMap(in, "region"),
		"bucket":          strAnyMap(in, "bucket"),
		"path_style":      boolAny(in["pathStyle"]),
		"cdn_base_url":    strAnyMap(in, "cdnBaseUrl", "cdn_base_url"),
		"public_base_url": strAnyMap(in, "publicBaseUrl", "public_base_url"),
		"path_prefix":     strAnyMap(in, "pathPrefix", "path_prefix"),
	}
	if ak := strAnyMap(in, "accessKeyId", "access_key_id"); ak != "" {
		st["access_key_id"] = ak
	}
	c := connection.Connection{
		ID: id, Name: firstNonEmpty(strAnyMap(in, "name"), "S3"), Vendor: "s3", Protocol: "s3",
		Endpoint: strAnyMap(in, "endpoint"), Capabilities: []string{connection.CapStorage},
		Active: true, Settings: st,
	}
	secret := strAnyMap(in, "accessKeySecret", "secretAccessKey", "secret")
	out, err := e.Conn.Upsert(c, secret)
	if err != nil {
		return e.disabledOSS()
	}
	_ = e.Conn.SetDefault(connection.CapStorage, out.ID)
	return e.disabledOSS()
}

func (e *Engine) testOSS() map[string]any {
	if e.Conn == nil {
		return map[string]any{"ok": false, "message": "no connections"}
	}
	c, err := e.Conn.Active(connection.CapStorage, "")
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}
	}
	r := connection.TestConnection(c, e.Conn)
	msg := strAnyMap(r, "error")
	if boolAny(r["ok"]) {
		msg = "ok"
	}
	return map[string]any{"ok": boolAny(r["ok"]), "message": msg, "reachable": r["reachable"]}
}
