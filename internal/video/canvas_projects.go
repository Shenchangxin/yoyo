// 行为：短剧项目、章节单元、素材和文件夹的读写，以及本机会话和对象存储设置。

package video

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

// dramaProjectRecord 复制短剧项目字段并去掉 units。没有状态时标成 active。
// p 里除 units 以外的键复制到结果；status 为 nil 时写成 active，name 为空时用 title，title 也空则写成 Untitled project。
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

// dramaProjectSummaries 把短剧项目收成列表摘要，计数字段先置 0。
// list 逐条传给 dramaProjectRecord，并配上为 0 的 canvasCount、assetCount、unitCount 和 completedUnitCount。
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

// unitToMap 把单元记录和额外 JSON 收成前端用的 map。
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

// listDramaProjects 列出未删除的短剧项目，按更新时间倒序。
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

// upsertDramaProject 创建或更新一条短剧项目记录。
// in 整份编码进 PayloadJSON，并读取 id、name、title；id 为空时换成 NewID，name 和 title 都空时写成 Untitled project，createdAt 为 nil 时补上记录里的创建时间，再把 id、name、title、updatedAt 写回 in。
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

// getDramaProject 读取短剧项目，并附上它的单元。
// id 传给 getDoc 和 listProjectUnits，并写入返回记录的 id。
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

// deleteDramaProject 把项目及其单元标成已删除。
// id 与单元的 ID 或 ProjectID 相等时写上删除时间。
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

// listProjectUnits 列出项目下未删除的非项目单元，按排序和创建时间排列。
// projectID 与单元的 ProjectID 比较，相等、种类不是 project 且未删除的才留下，并写入每条的 projectId。
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

// upsertProjectUnit 创建或更新项目下的一个单元。
// projectID 写入记录的 ProjectID，并写回 in 的 projectId。
// in 读取 id、title、name、kind、position；id 为空时换成 NewID，title 和 name 都空时写成 Chapter，kind 为空时写成 chapter，整份编码进 PayloadJSON，再把 id、projectId、title、kind 写回 in。
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

// deleteProjectUnit 把单元标成已删除。
// id 传给 getDoc 和 putDoc。
func (e *Engine) deleteProjectUnit(id string) error {
	rec, err := getDoc[canvasUnitRec](e, colUnits, id)
	if err != nil {
		return err
	}
	rec.DeletedAt = Now()
	return e.putDoc(colUnits, id, rec)
}

// matchAsset 按种类、分类、文件夹、状态和搜索词判断素材是否入选。
// kind 非空且与素材 Kind 不等时返回 false。
// category 非空时与素材分类比较，分类为空则按 other。
// folderID 非空且与素材 FolderID 不等时返回 false。
// status 为 active 时排除 archived，其他非空值必须与素材 Status 相等。
// q 去掉空白并转成小写后，非空时要出现在标题和 PayloadJSON 的小写文本里。
// uncategorized 为真且素材 FolderID 非空时返回 false。
// excludeKind 非空且与素材 Kind 相等时返回 false。
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
	if category != "" {
		got := a.Category
		if got == "" {
			got = "other"
		}
		if got != category {
			return false
		}
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

// assetToMap 把素材记录和额外 JSON 收成前端用的 map。
func assetToMap(a canvasAssetRec) map[string]any {
	item := map[string]any{}
	var extra map[string]any
	_ = json.Unmarshal([]byte(a.PayloadJSON), &extra)
	for k, v := range extra {
		item[k] = v
	}
	item["id"] = a.ID
	item["folderId"] = a.FolderID
	item["kind"] = a.Kind
	item["category"] = a.Category
	item["title"] = a.Title
	item["resourceId"] = a.ResourceID
	item["status"] = a.Status
	item["createdAt"] = a.CreatedAt
	item["updatedAt"] = a.UpdatedAt
	return item
}

// listAssetsPage 分页列出匹配的素材，并附上种类、分类和文件夹计数。
// kind 传给 matchAsset 和 scanAssetFacets。
// category 传给 matchAsset。
// folderID 传给 matchAsset。
// status 传给 matchAsset 和 scanAssetFacets。
// q 传给 matchAsset 和 scanAssetFacets。
// page 小于 1 时改成 1，再用来计算切片起点。
// pageSize 小于等于 0 时改成 40，再用来计算切片长度。
// uncategorized 传给 matchAsset。
// excludeKind 传给 matchAsset 和 scanAssetFacets。
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
	kindCounts, categoryCounts, folderCounts := e.scanAssetFacets(kind, status, q, excludeKind)
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

// scanAssetFacets 在当前筛选下统计种类、分类和文件夹数量。
// kind 传给 matchAsset。
// status 传给 matchAsset。
// q 传给 matchAsset。
// excludeKind 传给 matchAsset。
func (e *Engine) scanAssetFacets(kind, status, q, excludeKind string) (map[string]int, map[string]int, map[string]int) {
	kinds := map[string]int{}
	cats := map[string]int{}
	folders := map[string]int{}
	for _, a := range loadCol[canvasAssetRec](e, colAssets) {
		if !matchAsset(a, kind, "", "", status, q, false, excludeKind) {
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

// upsertAsset 创建或更新一条素材。
// in 读取 id、folderId、kind、category、title、resourceId、status；id 为空时换成 NewID，kind 为空时写成 image，title 为空时写成 Asset，status 为空时写成 confirmed，整份编码进 PayloadJSON，再把 id 和 updatedAt 写回 in。
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

// getAsset 读取未删除的素材。
// id 传给 getDoc，从 colAssets 取值。
func (e *Engine) getAsset(id string) (map[string]any, error) {
	a, err := getDoc[canvasAssetRec](e, colAssets, id)
	if err != nil || a.DeletedAt != "" {
		return nil, fmt.Errorf("asset not found")
	}
	return assetToMap(a), nil
}

// deleteAsset 把素材标成已删除。
// id 传给 getDoc 和 putDoc，从 colAssets 取值。
func (e *Engine) deleteAsset(id string) error {
	a, err := getDoc[canvasAssetRec](e, colAssets, id)
	if err != nil {
		return err
	}
	a.DeletedAt = Now()
	return e.putDoc(colAssets, id, a)
}

// listAssetFolders 列出未删除的素材文件夹，按位置排序。
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

// upsertAssetFolder 创建或改名一个素材文件夹。
// id 为空时换成 NewID，并作为文件夹文档的键。
// name 去掉空白后为空时写成 Folder，再写入文件夹的 Name。
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

// deleteAssetFolder 把文件夹标成已删除。
// id 传给 getDoc 和 putDoc，从 colFolders 取值。
func (e *Engine) deleteAssetFolder(id string) error {
	f, err := getDoc[canvasFolderRec](e, colFolders, id)
	if err != nil {
		return err
	}
	f.DeletedAt = Now()
	return e.putDoc(colFolders, id, f)
}

// moveAssetsFolder 把多条素材改到同一个文件夹。
// ids 逐个转成字符串后传给 getDoc。
// folderID 传给 setAssetFolder。
func (e *Engine) moveAssetsFolder(ids []any, folderID string) {
	now := Now()
	for _, id := range ids {
		a, err := getDoc[canvasAssetRec](e, colAssets, fmt.Sprint(id))
		if err != nil {
			continue
		}
		_ = e.setAssetFolder(a, folderID, now)
	}
}

// setAssetFolder 同时改素材记录和额外 JSON 里的文件夹。
// a 写入 FolderID、UpdatedAt 和 PayloadJSON 后，用 a.ID 调用 putDoc。
// folderID 写入 a.FolderID 和 extra 的 folderId。
// now 写入 a.UpdatedAt 和 extra 的 updatedAt。
func (e *Engine) setAssetFolder(a canvasAssetRec, folderID, now string) error {
	a.FolderID = folderID
	a.UpdatedAt = now
	var extra map[string]any
	if a.PayloadJSON != "" {
		_ = json.Unmarshal([]byte(a.PayloadJSON), &extra)
	}
	if extra == nil {
		extra = map[string]any{}
	}
	extra["folderId"] = folderID
	extra["updatedAt"] = now
	raw, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	a.PayloadJSON = string(raw)
	return e.putDoc(colAssets, a.ID, a)
}

// localUser 返回本机固定用户 yoyo。
func (e *Engine) localUser() map[string]any {
	now := Now()
	return map[string]any{
		"id": "local", "username": "yoyo", "displayName": "Yoyo", "role": "admin", "status": "active",
		"createdAt": now, "updatedAt": now,
	}
}

// authSession 返回本机登录会话、运行限制和功能开关。
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

// disabledOSS 有活动的 S3 连接时返回其公开设置，否则返回未启用的本地 CAS。
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

// applyOSS 把对象存储设置写成连接。未启用或选 CAS 时切回本地默认。
// in 用 strAnyMap 读取 provider、s3Preset、preset、region、bucket、cdnBaseUrl、cdn_base_url、publicBaseUrl、public_base_url、pathPrefix、path_prefix、accessKeyId、access_key_id、name、endpoint、accessKeySecret、secretAccessKey、secret，用 boolAny 读取 enabled 和 pathStyle；未启用、provider 为空或为 cas 时把存储默认设为 cas-local，否则用这些字段组成连接传给 Conn.Upsert。
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

// testOSS 探测当前活动的存储连接是否可达。
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
