package video

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (e *Engine) CanvasCall(method string, params map[string]any) (any, error) {
	if params == nil {
		params = map[string]any{}
	}
	switch method {
	case "canvas.http":
		return e.canvasHTTP(params), nil
	case "canvas.bind":
		sid := strAnyMap(params, "session_id")
		cid := strAnyMap(params, "project_id", "canvas_id")
		return map[string]any{"ok": true}, e.BindCanvasSession(sid, cid)
	case "canvas.projects.list":
		rows, _, err := e.listCanvasProjectRows("", "", 1, 200)
		if err != nil {
			return nil, err
		}
		out := []map[string]any{}
		for _, r := range rows {
			out = append(out, canvasSummary(r))
		}
		return out, nil
	case "canvas.projects.create":
		return e.createEmptyCanvas(strAnyMap(params, "title"))
	case "canvas.projects.get":
		row, err := e.getCanvasProject(strAnyMap(params, "id"))
		if err != nil {
			return nil, err
		}
		return e.projectDocFromRow(row), nil
	case "canvas.models.catalog":
		return e.modelCatalog(), nil
	default:
		return nil, fmt.Errorf("unknown canvas method %s", method)
	}
}

func (e *Engine) canvasHTTP(params map[string]any) canvasEnv {
	method := strings.ToUpper(strAnyMap(params, "method"))
	path := strAnyMap(params, "path")
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/api")
	path = strings.TrimPrefix(path, "/")
	query := asMap(params["query"])
	body := params["body"]
	bodyMap := asMap(body)
	parts := splitPath(path)
	match := func(pat string) (map[string]string, bool) {
		return matchPath(parts, pat)
	}

	switch {
	case method == "GET" && path == "auth/session":
		return canvasOK(e.authSession())
	case method == "GET" && path == "public/appearance":
		return canvasOK(map[string]any{"appearance": publicAppearance()})
	case method == "GET" && path == "admin/settings/appearance":
		return canvasOK(map[string]any{"setting": adminAppearance()})
	case method == "PATCH" && path == "admin/settings/appearance":
		return canvasOK(map[string]any{"setting": adminAppearance()})
	case method == "DELETE" && path == "admin/settings/appearance":
		return canvasOK(map[string]any{"setting": adminAppearance()})
	case method == "GET" && path == "features":
		return canvasOK(map[string]any{"features": e.authSession()["features"]})
	case method == "GET" && path == "channels/system":
		return canvasOK(map[string]any{"channels": e.modelCatalog()["channels"]})
	case method == "GET" && path == "model-catalog":
		return canvasOK(e.modelCatalog())
	case method == "POST" && path == "model-catalog/quote":
		return canvasOK(map[string]any{"quote": map[string]any{"amountMicrocredits": 0, "estimated": true, "billingMode": "fixed_request", "quantity": 1}})
	case method == "GET" && path == "settings/oss":
		return canvasOK(map[string]any{"setting": e.disabledOSS()})
	case method == "PATCH" && path == "settings/oss":
		return canvasOK(map[string]any{"setting": e.disabledOSS()})
	case method == "POST" && path == "settings/oss/test":
		return canvasOK(map[string]any{"ok": false, "message": "object storage is replaced by local CAS"})
	case method == "GET" && path == "settings/prompt-templates":
		return canvasOK(map[string]any{"templates": []any{}})
	case method == "GET" && path == "resources/storage-usage":
		return canvasOK(map[string]any{"usage": e.storageUsage()})
	case method == "POST" && path == "resources/access":
		items := []map[string]any{}
		reqs := asSlice(body)
		if len(reqs) == 0 {
			reqs = asSlice(bodyMap["items"])
		}
		for _, raw := range reqs {
			m := asMap(raw)
			id := strAnyMap(m, "resourceId", "id")
			acc, err := e.resourceAccess(id, strAnyMap(m, "purpose"), firstNonEmpty(strAnyMap(m, "variant"), "original"))
			if err != nil {
				items = append(items, map[string]any{"resourceId": id, "error": map[string]any{"msg": err.Error()}})
				continue
			}
			items = append(items, map[string]any{"resourceId": id, "access": acc})
		}
		return canvasOK(map[string]any{"items": items})
	case method == "POST" && path == "resources":
		raw, err := decodeB64Any(strAnyMap(params, "file_b64"))
		if err != nil {
			return canvasFail(400, "file required", "bad_request")
		}
		res, err := e.putCanvasBytes(strAnyMap(bodyMap, "kind"), strAnyMap(params, "file_type"), firstNonEmpty(strAnyMap(params, "file_name"), strAnyMap(bodyMap, "fileName")), raw, intAny(bodyMap["width"]), intAny(bodyMap["height"]), intAny(bodyMap["durationMs"]), strAnyMap(bodyMap, "id"))
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"resource": res})
	case method == "POST" && path == "resources/uploads":
		u, err := e.startChunkUpload(strAnyMap(bodyMap, "kind"), strAnyMap(bodyMap, "fileName"), intAny(bodyMap["size"]), intAny(bodyMap["width"]), intAny(bodyMap["height"]), intAny(bodyMap["durationMs"]))
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"uploadId": u.ID, "chunkSize": u.ChunkSize, "chunkCount": u.ChunkCount})
	case method == "POST" && path == "resources/import":
		u := strAnyMap(bodyMap, "url")
		raw, err := e.DownloadURL(u)
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		res, err := e.putCanvasBytes(strAnyMap(bodyMap, "kind"), "", strAnyMap(bodyMap, "fileName"), raw, 0, 0, 0, "")
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"resource": res})
	}

	if p, ok := match("resources/uploads/:id/chunks/:index"); ok && method == "PUT" {
		raw, err := decodeB64Any(strAnyMap(params, "chunk_b64"))
		if err != nil {
			raw, err = decodeB64Any(strAnyMap(params, "file_b64"))
		}
		if err != nil {
			return canvasFail(400, "chunk required", "bad_request")
		}
		if err := e.putChunk(p["id"], intAny(p["index"]), raw); err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"index": intAny(p["index"])})
	}
	if p, ok := match("resources/uploads/:id/complete"); ok && method == "POST" {
		res, err := e.completeChunkUpload(p["id"])
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"resource": res})
	}
	if p, ok := match("resources/:id"); ok && method == "GET" {
		res, err := e.getCanvasResource(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(map[string]any{"resource": res})
	}
	if p, ok := match("resources/:id/ark-private-asset"); ok && method == "POST" {
		return canvasOK(map[string]any{"sync": map[string]any{"resourceId": p["id"], "status": "skipped"}})
	}

	if path == "canvas-projects" && method == "GET" {
		if strAnyMap(query, "page") != "" {
			page, pageSize := intAny(query["page"]), intAny(query["pageSize"])
			rows, total, err := e.listCanvasProjectRows(strAnyMap(query, "q"), strAnyMap(query, "sort"), page, pageSize)
			if err != nil {
				return canvasFail(500, err.Error(), "internal")
			}
			projects := []map[string]any{}
			for _, r := range rows {
				projects = append(projects, canvasSummary(r))
			}
			return canvasOK(map[string]any{"projects": projects, "page": page, "pageSize": pageSize, "total": total, "hasMore": page*pageSize < total})
		}
		rows, _, err := e.listCanvasProjectRows("", "", 1, 200)
		if err != nil {
			return canvasFail(500, err.Error(), "internal")
		}
		projects := []map[string]any{}
		for _, r := range rows {
			projects = append(projects, canvasSummary(r))
		}
		return canvasOK(map[string]any{"projects": projects})
	}
	if path == "canvas-projects" && method == "POST" {
		doc, err := e.createEmptyCanvas(strAnyMap(bodyMap, "title"))
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"project": doc})
	}
	if p, ok := match("canvas-projects/:id"); ok {
		switch method {
		case "GET":
			row, err := e.getCanvasProject(p["id"])
			if err != nil {
				return canvasFail(404, err.Error(), "not_found")
			}
			return canvasOK(map[string]any{"project": e.projectDocFromRow(row)})
		case "PUT":
			raw, _ := json.Marshal(bodyMap["project"])
			if repair := boolAny(bodyMap["repairMissingResources"]); repair {
				sum, env := e.upsertCanvasProject(raw, true)
				if env.Code != 0 {
					return env
				}
				return canvasOK(map[string]any{"project": sum})
			}
			sum, env := e.upsertCanvasProject(raw, false)
			if env.Code != 0 {
				return env
			}
			return canvasOK(map[string]any{"project": sum})
		case "DELETE":
			if err := e.deleteCanvasProject(p["id"]); err != nil {
				return canvasFail(400, err.Error(), "bad_request")
			}
			return canvasOK(map[string]any{"id": p["id"]})
		}
	}
	if p, ok := match("canvas-projects/:id/history"); ok && method == "GET" {
		out, err := e.listCanvasHistory(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(out)
	}
	if p, ok := match("canvas-projects/:id/history/:snapshotId"); ok && method == "GET" {
		snap, proj, err := e.getCanvasSnapshot(p["id"], p["snapshotId"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(map[string]any{"snapshot": snap, "project": proj})
	}
	if p, ok := match("canvas-projects/:id/history/:snapshotId/restore"); ok && method == "POST" {
		sum, env := e.restoreCanvasSnapshot(p["id"], p["snapshotId"], int64(intAny(bodyMap["revision"])))
		if env.Code != 0 {
			return env
		}
		return canvasOK(map[string]any{"project": sum})
	}
	if p, ok := match("canvas-projects/:id/import/tapnow"); ok && method == "POST" {
		_ = p
		return canvasFail(400, "TapNow import is not available on this desktop build", "unsupported")
	}

	if path == "user-data/snapshot" && method == "GET" {
		rows, _, _ := e.listCanvasProjectRows("", "", 1, 200)
		projects := []any{}
		for _, r := range rows {
			projects = append(projects, e.projectDocFromRow(r))
		}
		page := e.listAssetsPage("", "", "", "", "", 1, 500, false)
		return canvasOK(map[string]any{"assets": page["assets"], "projects": projects})
	}

	if path == "assets" && method == "GET" {
		page := e.listAssetsPage(strAnyMap(query, "kind"), strAnyMap(query, "category"), strAnyMap(query, "folderId"), strAnyMap(query, "status"), strAnyMap(query, "q"), intAny(query["page"]), intAny(query["pageSize"]), boolAny(query["uncategorized"]))
		if strAnyMap(query, "page") == "" && strAnyMap(query, "pageSize") == "" {
			return canvasOK(map[string]any{"assets": page["assets"]})
		}
		return canvasOK(page)
	}
	if path == "assets/batch" && method == "POST" {
		ids := asSlice(bodyMap["ids"])
		assets := []map[string]any{}
		for _, id := range ids {
			if a, err := e.getAsset(fmt.Sprint(id)); err == nil {
				assets = append(assets, a)
			}
		}
		return canvasOK(map[string]any{"assets": assets})
	}
	if path == "assets/batch-delete" && method == "POST" {
		ids := asSlice(body)
		if len(ids) == 0 {
			ids = asSlice(bodyMap["ids"])
		}
		out := []string{}
		for _, id := range ids {
			_ = e.deleteAsset(fmt.Sprint(id))
			out = append(out, fmt.Sprint(id))
		}
		return canvasOK(map[string]any{"ids": out})
	}
	if path == "assets/folder" && method == "PATCH" {
		e.moveAssetsFolder(asSlice(bodyMap["assetIds"]), strAnyMap(bodyMap, "folderId"))
		return canvasOK(map[string]any{"assetIds": bodyMap["assetIds"], "folderId": strAnyMap(bodyMap, "folderId")})
	}
	if p, ok := match("assets/:id"); ok {
		switch method {
		case "GET":
			a, err := e.getAsset(p["id"])
			if err != nil {
				return canvasFail(404, err.Error(), "not_found")
			}
			return canvasOK(map[string]any{"asset": a})
		case "PUT":
			asset := asMap(bodyMap["asset"])
			if strAnyMap(asset, "id") == "" {
				asset["id"] = p["id"]
			}
			return canvasOK(map[string]any{"asset": e.upsertAsset(asset)})
		case "DELETE":
			_ = e.deleteAsset(p["id"])
			return canvasOK(map[string]any{"id": p["id"]})
		}
	}
	if path == "asset-folders" && method == "GET" {
		return canvasOK(map[string]any{"folders": e.listAssetFolders()})
	}
	if path == "asset-folders" && method == "POST" {
		return canvasOK(map[string]any{"folder": e.upsertAssetFolder("", strAnyMap(bodyMap, "name"))})
	}
	if p, ok := match("asset-folders/:id"); ok && method == "PATCH" {
		return canvasOK(map[string]any{"folder": e.upsertAssetFolder(p["id"], strAnyMap(bodyMap, "name"))})
	}
	if p, ok := match("asset-folders/:id"); ok && method == "DELETE" {
		_ = e.deleteAssetFolder(p["id"])
		return canvasOK(map[string]any{"id": p["id"]})
	}

	if path == "tasks" && method == "POST" {
		t, err := e.createCanvasTask(bodyMap)
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if path == "tasks" && method == "GET" {
		return canvasOK(e.listCanvasTasks(strAnyMap(query, "projectId"), strAnyMap(query, "activeOnly") == "true", intAny(query["pageSize"])))
	}
	if path == "timeline/renders" && method == "POST" {
		t, err := e.createTimelineRender(bodyMap)
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if path == "timeline/transcriptions" && method == "POST" {
		t, err := e.createTimelineTranscribe(bodyMap)
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if p, ok := match("tasks/:id"); ok && method == "GET" {
		t, err := e.canvasTask(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(t)
	}
	if p, ok := match("tasks/:id"); ok && method == "DELETE" {
		_ = e.deleteCanvasTask(p["id"])
		return canvasOK(map[string]any{"ok": true})
	}
	if p, ok := match("tasks/:id/text-deltas"); ok && method == "POST" {
		item, err := e.appendTextDelta(p["id"], strAnyMap(bodyMap, "content"))
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(item)
	}
	if p, ok := match("tasks/:id/text-deltas"); ok && method == "GET" {
		return canvasOK(e.textReplay(p["id"], intAny(query["after"])))
	}
	if p, ok := match("tasks/:id/text-replay-complete"); ok && method == "POST" {
		t, err := e.completeTextReplay(p["id"], strAnyMap(bodyMap, "text"))
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if p, ok := match("tasks/:id/retry"); ok && method == "POST" {
		t, err := e.retryCanvasTask(p["id"])
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if p, ok := match("tasks/:id/recover-media"); ok && method == "POST" {
		t, err := e.recoverCanvasTask(p["id"])
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if p, ok := match("tasks/:id/cancel"); ok && method == "POST" {
		t, err := e.cancelCanvasTask(p["id"])
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if p, ok := match("tasks/:id/query-provider"); ok && method == "POST" {
		t, err := e.queryProviderTask(p["id"])
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(t)
	}
	if p, ok := match("tasks/:id/logs"); ok && method == "GET" {
		return canvasOK(e.taskLogs(p["id"]))
	}

	if path == "channels" && method == "GET" {
		list := []map[string]any{}
		for _, c := range e.listCanvasChannels() {
			b, _ := json.Marshal(c)
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			list = append(list, m)
		}
		return canvasOK(map[string]any{"channels": list, "total": len(list)})
	}
	if path == "channels" && method == "POST" {
		ch, err := e.upsertCanvasChannel(bodyMap, strAnyMap(params, "api_key", "apiKey"))
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"channel": ch})
	}
	if p, ok := match("channels/:id"); ok {
		switch method {
		case "GET":
			ch, err := e.getCanvasChannel(p["id"])
			if err != nil {
				return canvasFail(404, err.Error(), "not_found")
			}
			return canvasOK(map[string]any{"channel": ch})
		case "PATCH", "PUT":
			bodyMap["id"] = p["id"]
			ch, err := e.upsertCanvasChannel(bodyMap, strAnyMap(params, "api_key", "apiKey"))
			if err != nil {
				return canvasFail(400, err.Error(), "bad_request")
			}
			return canvasOK(map[string]any{"channel": ch})
		case "DELETE":
			_, _ = e.DB.Exec(`DELETE FROM canvas_channels WHERE id = ?`, p["id"])
			return canvasOK(map[string]any{"id": p["id"]})
		}
	}
	if p, ok := match("channels/:id/test"); ok && method == "POST" {
		ch, err := e.getCanvasChannel(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(map[string]any{"ok": ch.BaseURL != "", "channelId": ch.ID, "pluginId": ch.PluginID})
	}
	if p, ok := match("channels/:id/models"); ok && method == "GET" {
		ch, err := e.getCanvasChannel(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		models := []any{}
		_ = json.Unmarshal([]byte(ch.Models), &models)
		return canvasOK(map[string]any{"models": models})
	}

	if path == "plugins" && method == "GET" {
		return canvasOK(e.listPluginPayload())
	}
	if path == "plugins/status" && method == "GET" {
		pl := e.listPluginPayload()
		return canvasOK(map[string]any{"statuses": pl["statuses"], "states": pl["states"]})
	}
	if path == "plugins/catalog" && method == "GET" {
		return canvasOK(e.pluginCatalog(strAnyMap(query, "scope"), strAnyMap(query, "capability")))
	}
	if path == "plugins" && method == "POST" {
		raw, err := decodeB64Any(strAnyMap(params, "file_b64"))
		if err != nil {
			return canvasFail(400, "plugin package required", "bad_request")
		}
		rec, err := e.installUserPlugin(raw, strAnyMap(params, "file_name"))
		if err != nil {
			return canvasFail(400, err.Error(), "bad_request")
		}
		return canvasOK(map[string]any{"plugin": rec})
	}
	if p, ok := match("plugins/:id/activation"); ok && method == "PUT" {
		return canvasOK(map[string]any{"state": map[string]any{
			"pluginId": p["id"], "platformAvailable": true, "userEnabled": boolAny(bodyMap["enabled"]),
			"userConfigured": true, "effectiveEnabled": boolAny(bodyMap["enabled"]), "canToggle": true, "canConfigure": true,
		}})
	}

	if strings.HasPrefix(path, "plugins/eagle") {
		return e.eagleHTTP(method, path, query, bodyMap)
	}
	if path == "runninghub/workflow-info" || path == "runninghub/app-info" {
		return canvasOK(map[string]any{"nodes": []any{}, "note": "configure RunningHub in a canvas channel"})
	}

	if path == "projects" && method == "GET" {
		list := e.listDramaProjects()
		if strAnyMap(query, "page") != "" {
			return canvasOK(map[string]any{"projects": list, "page": intAny(query["page"]), "pageSize": intAny(query["pageSize"]), "total": len(list), "hasMore": false})
		}
		return canvasOK(map[string]any{"projects": list})
	}
	if path == "projects" && method == "POST" {
		return canvasOK(map[string]any{"project": e.upsertDramaProject(bodyMap)})
	}
	if p, ok := match("projects/:id"); ok && method == "GET" {
		proj, err := e.getDramaProject(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(proj)
	}
	if p, ok := match("projects/:id"); ok && method == "PATCH" {
		bodyMap["id"] = p["id"]
		return canvasOK(map[string]any{"project": e.upsertDramaProject(bodyMap)})
	}
	if p, ok := match("projects/:id"); ok && method == "DELETE" {
		_ = e.deleteDramaProject(p["id"])
		return canvasOK(map[string]any{"id": p["id"]})
	}
	if p, ok := match("projects/:id/core"); ok && method == "GET" {
		proj, err := e.getDramaProject(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(proj)
	}
	if p, ok := match("projects/:id/overview"); ok && method == "GET" {
		proj, err := e.getDramaProject(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(proj)
	}
	if p, ok := match("projects/:id/units"); ok && method == "GET" {
		return canvasOK(map[string]any{"units": e.listProjectUnits(p["id"]), "canvasCounts": map[string]int{}})
	}
	if p, ok := match("projects/:id/units"); ok && method == "POST" {
		return canvasOK(map[string]any{"unit": e.upsertProjectUnit(p["id"], bodyMap)})
	}
	if p, ok := match("projects/:id/units/import"); ok && method == "POST" {
		out := []map[string]any{}
		for _, u := range asSlice(bodyMap["units"]) {
			out = append(out, e.upsertProjectUnit(p["id"], asMap(u)))
		}
		return canvasOK(map[string]any{"units": out})
	}
	if p, ok := match("projects/:id/units/reorder"); ok && method == "PATCH" {
		return canvasOK(map[string]any{"unitIds": bodyMap["unitIds"], "projectId": p["id"]})
	}
	if p, ok := match("projects/:projectId/units/:unitId"); ok {
		switch method {
		case "GET":
			for _, u := range e.listProjectUnits(p["projectId"]) {
				if fmt.Sprint(u["id"]) == p["unitId"] {
					return canvasOK(map[string]any{"unit": u})
				}
			}
			return canvasFail(404, "unit not found", "not_found")
		case "PATCH":
			bodyMap["id"] = p["unitId"]
			return canvasOK(map[string]any{"unit": e.upsertProjectUnit(p["projectId"], bodyMap)})
		case "DELETE":
			_ = e.deleteProjectUnit(p["unitId"])
			return canvasOK(map[string]any{"id": p["unitId"]})
		}
	}
	if p, ok := match("projects/:projectId/units/:unitId/workspace"); ok && method == "GET" {
		return canvasOK(map[string]any{"unitId": p["unitId"], "projectId": p["projectId"], "steps": []any{}})
	}
	if p, ok := match("projects/:id/canvases"); ok && method == "GET" {
		rows, _, _ := e.listCanvasProjectRows("", "", 1, 40)
		list := []map[string]any{}
		for _, r := range rows {
			if r.ProjectLink == p["id"] {
				list = append(list, canvasSummary(r))
			}
		}
		return canvasOK(map[string]any{"canvases": list, "page": 1, "total": len(list), "hasMore": false})
	}
	if p, ok := match("projects/:id/assets"); ok && (method == "GET" || method == "POST") {
		if method == "POST" {
			bodyMap["category"] = "project"
			bodyMap["projectId"] = p["id"]
			return canvasOK(map[string]any{"asset": e.upsertAsset(bodyMap)})
		}
		page := e.listAssetsPage("", "", "", "", "", 1, 100, false)
		page["projectId"] = p["id"]
		return canvasOK(page)
	}
	if p, ok := match("projects/:id/canvas-links"); ok && method == "POST" {
		return canvasOK(map[string]any{"link": map[string]any{"id": NewID(), "projectId": p["id"], "canvasId": strAnyMap(bodyMap, "canvasId"), "unitId": strAnyMap(bodyMap, "unitId"), "role": strAnyMap(bodyMap, "role")}})
	}

	if path == "style-profiles" && method == "GET" {
		return canvasOK(map[string]any{"profiles": []any{}})
	}
	if path == "style-profiles" && method == "POST" {
		return canvasOK(map[string]any{"profile": map[string]any{"id": NewID(), "profileJson": bodyMap["profileJson"]}})
	}
	if path == "skills" || strings.HasPrefix(path, "skills/") {
		if path == "skills" && method == "GET" {
			return canvasOK(map[string]any{"skills": []any{}, "total": 0})
		}
		if path == "skills/presets" {
			return canvasOK(map[string]any{"presets": []any{}})
		}
		if path == "skills/added" {
			return canvasOK(map[string]any{"skills": []any{}})
		}
		return canvasOK(map[string]any{"skill": map[string]any{}, "files": []any{}, "results": []any{}, "deleted": true})
	}
	if path == "tools" || strings.HasPrefix(path, "tools/") {
		return canvasOK(map[string]any{"tools": []any{}, "total": 0})
	}
	if path == "wallet" || strings.HasPrefix(path, "wallet/") || strings.HasPrefix(path, "admin/") {
		return canvasOK(map[string]any{})
	}
	if path == "diagnostics/preview" {
		return canvasOK(map[string]any{"redacted": true})
	}
	if strings.HasPrefix(path, "prompts/") || path == "prompt-operations" {
		op := strings.TrimPrefix(path, "prompts/")
		text := e.promptCompile(op, bodyMap)
		return canvasOK(map[string]any{"prompt": text, "operation": op})
	}

	// Unmapped endpoints stay empty-success so the canvas island does not crash.
	if method == "GET" {
		return canvasOK(map[string]any{})
	}
	return canvasOK(map[string]any{"ok": true})
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func matchPath(parts []string, pattern string) (map[string]string, bool) {
	pat := splitPath(pattern)
	if len(pat) != len(parts) {
		return nil, false
	}
	out := map[string]string{}
	for i, p := range pat {
		if strings.HasPrefix(p, ":") {
			out[strings.TrimPrefix(p, ":")] = parts[i]
			continue
		}
		if p != parts[i] {
			return nil, false
		}
	}
	return out, true
}

func (e *Engine) eagleHTTP(method, path string, query, body map[string]any) canvasEnv {
	base := strings.TrimRight(strAnyMap(query, "baseUrl"), "/")
	if base == "" {
		base = "http://127.0.0.1:41595"
	}
	if !strings.HasPrefix(base, "http://127.0.0.1") && !strings.HasPrefix(base, "http://localhost") {
		return canvasFail(400, "Eagle bridge only allows localhost", "forbidden")
	}
	client := &http.Client{Timeout: 8 * time.Second}
	switch {
	case strings.HasSuffix(path, "/library") && method == "GET":
		res, err := client.Get(base + "/api/library/info")
		if err != nil {
			return canvasOK(map[string]any{"library": map[string]any{"folders": []any{}, "name": "Eagle"}})
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
		var payload any
		_ = json.Unmarshal(raw, &payload)
		return canvasOK(map[string]any{"library": payload})
	case strings.HasSuffix(path, "/items") && method == "GET":
		return canvasOK(map[string]any{"items": []any{}})
	case strings.HasSuffix(path, "/items") && method == "POST":
		return canvasOK(map[string]any{"item": map[string]any{"id": NewID()}})
	case strings.HasSuffix(path, "/folders") && method == "POST":
		return canvasOK(map[string]any{"created": true})
	default:
		return canvasOK(map[string]any{})
	}
}
