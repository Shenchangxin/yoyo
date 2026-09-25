package video

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func (e *Engine) handleCanvasSkills(method, path string, query, body map[string]any) canvasEnv {
	if path == "skills" && method == "GET" {
		return canvasOK(e.listCanvasSkills(query))
	}
	if path == "skills" && method == "POST" {
		return canvasOK(map[string]any{"skill": e.upsertUserSkill("", body)})
	}
	if path == "skills/presets" && method == "GET" {
		return canvasOK(map[string]any{"presets": builtinSkillPresets()})
	}
	if path == "skills/added" && method == "GET" {
		query["scope"] = "mine"
		query["page"] = 1
		query["pageSize"] = 200
		page := e.listCanvasSkills(query)
		return canvasOK(map[string]any{"skills": page["skills"]})
	}
	if path == "skills/install" && method == "POST" {
		return canvasOK(map[string]any{"skill": e.upsertUserSkill("", body)})
	}
	if path == "skills/install/github" && method == "POST" {
		return canvasFail(400, "桌面端不从 GitHub 安装技能，请粘贴 Markdown 或在本机创建", "unsupported")
	}
	if p, ok := matchSimple(path, "skills/:id/files"); ok && method == "GET" {
		skill, err := e.getCanvasSkill(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(map[string]any{"files": skillFiles(skill)})
	}
	if p, ok := matchSimple(path, "skills/:id/file"); ok && method == "GET" {
		skill, err := e.getCanvasSkill(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		filePath := strAnyMap(query, "path")
		if filePath == "" {
			filePath = "SKILL.md"
		}
		content, err := skillFileContent(skill, filePath)
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		meta := skillFileMeta(skill, filePath)
		return canvasOK(map[string]any{"file": map[string]any{"file": meta, "content": content, "binary": false}})
	}
	if p, ok := matchSimple(path, "skills/:id/file/raw"); ok && method == "GET" {
		skill, err := e.getCanvasSkill(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		content, err := skillFileContent(skill, strAnyMap(query, "path"))
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(map[string]any{"content": content})
	}
	if p, ok := matchSimple(path, "skills/:id/add"); ok && method == "POST" {
		added := true
		return canvasOK(map[string]any{"skill": e.setSkillFlag(p["id"], &added, nil)})
	}
	if p, ok := matchSimple(path, "skills/:id/add"); ok && method == "DELETE" {
		added := false
		return canvasOK(map[string]any{"skill": e.setSkillFlag(p["id"], &added, nil)})
	}
	if p, ok := matchSimple(path, "skills/:id/like"); ok && method == "POST" {
		liked := true
		return canvasOK(map[string]any{"skill": e.setSkillFlag(p["id"], nil, &liked)})
	}
	if p, ok := matchSimple(path, "skills/:id/like"); ok && method == "DELETE" {
		liked := false
		return canvasOK(map[string]any{"skill": e.setSkillFlag(p["id"], nil, &liked)})
	}
	if p, ok := matchSimple(path, "skills/:id/sync"); ok && method == "POST" {
		skill, err := e.getCanvasSkill(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		return canvasOK(map[string]any{"skill": skill})
	}
	if p, ok := matchSimple(path, "skills/:id/bundle"); ok && method == "GET" {
		skill, err := e.getCanvasSkill(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		instruction, _ := skill["instruction"].(string)
		return canvasOK(map[string]any{"bundle": map[string]any{
			"skillId": skill["skillId"], "name": skill["skillName"], "description": skill["description"],
			"versionId": skill["versionId"], "version": skill["version"], "contentHash": skill["contentHash"],
			"files": []map[string]any{{"path": "SKILL.md", "mimeType": "text/markdown", "contentBase64": ""}},
			"instruction": instruction,
		}})
	}
	if p, ok := matchSimple(path, "skills/:id/search"); ok && method == "GET" {
		skill, err := e.getCanvasSkill(p["id"])
		if err != nil {
			return canvasFail(404, err.Error(), "not_found")
		}
		q := strings.TrimSpace(strAnyMap(query, "q"))
		instruction, _ := skill["instruction"].(string)
		results := []map[string]any{}
		if q != "" && strings.Contains(strings.ToLower(instruction), strings.ToLower(q)) {
			results = append(results, map[string]any{"path": "SKILL.md", "line": 1, "snippet": q})
		}
		return canvasOK(map[string]any{"results": results})
	}
	if p, ok := matchSimple(path, "skills/:id"); ok {
		id := p["id"]
		switch method {
		case "GET":
			skill, err := e.getCanvasSkill(id)
			if err != nil {
				return canvasFail(404, err.Error(), "not_found")
			}
			return canvasOK(map[string]any{"skill": skill})
		case "PUT":
			return canvasOK(map[string]any{"skill": e.upsertUserSkill(id, body)})
		case "DELETE":
			if err := e.deleteUserSkill(id); err != nil {
				return canvasFail(400, err.Error(), "bad_request")
			}
			return canvasOK(map[string]any{"deleted": true})
		}
	}
	return canvasOK(map[string]any{"skill": map[string]any{}, "files": []any{}, "results": []any{}, "deleted": true})
}

func matchSimple(path, pat string) (map[string]string, bool) {
	return matchPath(splitPath(path), pat)
}

func (e *Engine) listCanvasSkills(query map[string]any) map[string]any {
	page := intAny(query["page"])
	if page < 1 {
		page = 1
	}
	pageSize := intAny(query["pageSize"])
	if pageSize <= 0 {
		pageSize = 20
	}
	scope := strAnyMap(query, "scope")
	if scope == "" {
		scope = "public"
	}
	sort := strAnyMap(query, "sort")
	search := strings.ToLower(strings.TrimSpace(strAnyMap(query, "search")))
	tag := strAnyMap(query, "tag")
	all := e.allCanvasSkills()
	filtered := []map[string]any{}
	for _, skill := range all {
		if tag != "" && tag != "all" && fmt.Sprint(skill["tag"]) != tag {
			continue
		}
		if search != "" {
			blob := strings.ToLower(fmt.Sprint(skill["skillName"]) + " " + fmt.Sprint(skill["description"]) + " " + fmt.Sprint(skill["instruction"]))
			if !strings.Contains(blob, search) {
				continue
			}
		}
		switch scope {
		case "mine":
			if !boolAny(skill["isAdded"]) {
				continue
			}
		case "created":
			if !boolAny(skill["isOwner"]) {
				continue
			}
		case "favorites":
			if !boolAny(skill["isLike"]) {
				continue
			}
		}
		filtered = append(filtered, skill)
	}
	switch sort {
	case "new":
		sortSkillMaps(filtered, "createdAt")
	case "updated":
		sortSkillMaps(filtered, "updatedAt")
	default:
		sortSkillMaps(filtered, "addedCount")
	}
	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return map[string]any{
		"skills":     filtered[start:end],
		"total":      total,
		"totalCount": total,
		"hasMore":    end < total,
		"nextOffset": end,
		"page":       page,
		"pageSize":   pageSize,
		"categories": skillCategories(),
	}
}

func sortSkillMaps(items []map[string]any, key string) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if skillSortValue(items[j], key) > skillSortValue(items[i], key) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func skillSortValue(skill map[string]any, key string) string {
	if key == "addedCount" || key == "likeCount" {
		return fmt.Sprintf("%010d", intAny(skill[key]))
	}
	return fmt.Sprint(skill[key])
}

func (e *Engine) allCanvasSkills() []map[string]any {
	out := []map[string]any{}
	seen := map[string]bool{}
	for _, def := range builtinCanvasSkills() {
		skill := e.hydrateSkill(def.toMap())
		out = append(out, skill)
		seen[fmt.Sprint(skill["skillId"])] = true
	}
	rows, err := e.DB.Query(`SELECT id, payload_json FROM canvas_user_skills WHERE deleted_at = ''`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, payload string
			if rows.Scan(&id, &payload) != nil {
				continue
			}
			item := map[string]any{}
			_ = json.Unmarshal([]byte(payload), &item)
			if fmt.Sprint(item["skillId"]) == "" {
				item["skillId"] = id
			}
			if seen[fmt.Sprint(item["skillId"])] {
				continue
			}
			out = append(out, e.hydrateSkill(item))
		}
	}
	return out
}

func (e *Engine) getCanvasSkill(id string) (map[string]any, error) {
	id = strings.TrimSpace(id)
	for _, skill := range e.allCanvasSkills() {
		if fmt.Sprint(skill["skillId"]) == id {
			return skill, nil
		}
	}
	return nil, fmt.Errorf("skill not found")
}

func (def canvasSkillDef) toMap() map[string]any {
	now := "2026-03-01T08:00:00Z"
	sum := sha256.Sum256([]byte(def.Body))
	return map[string]any{
		"skillId": def.ID, "skillName": def.Name, "description": def.Desc, "instruction": def.Body,
		"versionId": def.ID + "-v1", "version": "1.0.0", "contentHash": hex.EncodeToString(sum[:8]),
		"fileCount": 1, "totalBytes": utf8.RuneCountInString(def.Body),
		"sourceType": "builtin", "sourceUrl": "", "sourceRef": "", "sourceSubdir": "", "sourceCommit": "",
		"syncStatus": "synced", "autoUpdate": false, "status": 1, "markdownUrl": "",
		"createdAt": now, "updatedAt": now, "source": 0, "tag": def.Tag, "sortWeight": def.Added,
		"isPrivate": false, "likeCount": def.Likes, "isLike": false, "ownerUid": "yoyo",
		"effectiveUser": map[string]any{"name": "Yoyo 工作室", "avatarUrl": "", "uid": "yoyo"},
		"originalSkillId": nil, "showcaseMedia": skillShowcases(def.Cover),
		"addedCount": def.Added, "isTest": false, "extraInfo": "", "isAdded": true, "isOwner": false,
		"baseLikeCount": def.Likes,
	}
}

func skillShowcases(cover string) []map[string]any {
	if cover == "" {
		return []map[string]any{}
	}
	return []map[string]any{{"type": "image", "showcaseUri": cover, "showcaseUrl": cover}}
}

func (e *Engine) hydrateSkill(skill map[string]any) map[string]any {
	id := fmt.Sprint(skill["skillId"])
	added := boolAny(skill["isAdded"])
	liked := boolAny(skill["isLike"])
	var addedFlag, likedFlag int
	err := e.DB.QueryRow(`SELECT added, liked FROM canvas_skill_flags WHERE skill_id = ?`, id).Scan(&addedFlag, &likedFlag)
	if err == nil {
		added = addedFlag != 0
		liked = likedFlag != 0
	}
	skill["isAdded"] = added
	skill["isLike"] = liked
	base := intAny(skill["baseLikeCount"])
	if skill["baseLikeCount"] == nil {
		base = intAny(skill["likeCount"])
		skill["baseLikeCount"] = base
	}
	likes := base
	if liked {
		likes++
	}
	skill["likeCount"] = likes
	return skill
}

func (e *Engine) setSkillFlag(id string, added *bool, liked *bool) map[string]any {
	skill, err := e.getCanvasSkill(id)
	if err != nil {
		return map[string]any{}
	}
	curAdded := boolAny(skill["isAdded"])
	curLiked := boolAny(skill["isLike"])
	if added != nil {
		curAdded = *added
	}
	if liked != nil {
		curLiked = *liked
	}
	ai, li := 0, 0
	if curAdded {
		ai = 1
	}
	if curLiked {
		li = 1
	}
	_, _ = e.DB.Exec(`INSERT INTO canvas_skill_flags(skill_id, added, liked, updated_at) VALUES(?,?,?,?)
		ON CONFLICT(skill_id) DO UPDATE SET added=excluded.added, liked=excluded.liked, updated_at=excluded.updated_at`,
		id, ai, li, Now())
	skill["isAdded"] = curAdded
	skill["isLike"] = curLiked
	return e.hydrateSkill(skill)
}

func (e *Engine) upsertUserSkill(id string, in map[string]any) map[string]any {
	if id == "" {
		id = strAnyMap(in, "skillId", "id")
	}
	if id == "" {
		id = "user-" + NewID()
	}
	now := Now()
	name := firstNonEmpty(strAnyMap(in, "skillName", "name"), "未命名技能")
	desc := strAnyMap(in, "description")
	instruction := firstNonEmpty(strAnyMap(in, "instruction"), strAnyMap(in, "markdown"), strAnyMap(in, "content"))
	tag := firstNonEmpty(strAnyMap(in, "tag"), "others")
	skill := map[string]any{
		"skillId": id, "skillName": name, "description": desc, "instruction": instruction,
		"versionId": id + "-v" + strconv.FormatInt(time.Now().Unix(), 10), "version": "1.0.0",
		"contentHash": id, "fileCount": 1, "totalBytes": utf8.RuneCountInString(instruction),
		"sourceType": "markdown", "sourceUrl": strAnyMap(in, "markdownUrl"), "sourceRef": "", "sourceSubdir": "", "sourceCommit": "",
		"syncStatus": "synced", "autoUpdate": false, "status": 1, "markdownUrl": strAnyMap(in, "markdownUrl"),
		"createdAt": now, "updatedAt": now, "source": 1, "tag": tag, "sortWeight": 0,
		"isPrivate": boolAny(in["isPrivate"]), "likeCount": 0, "isLike": false, "ownerUid": "local",
		"effectiveUser": map[string]any{"name": "我", "avatarUrl": "", "uid": "local"},
		"originalSkillId": nil, "showcaseMedia": in["showcaseMedia"],
		"addedCount": 1, "isTest": false, "extraInfo": strAnyMap(in, "extraInfo"), "isAdded": true, "isOwner": true,
	}
	if skill["showcaseMedia"] == nil {
		skill["showcaseMedia"] = []any{}
	}
	raw, _ := json.Marshal(skill)
	_, _ = e.DB.Exec(`INSERT INTO canvas_user_skills(id, payload_json, created_at, updated_at) VALUES(?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET payload_json=excluded.payload_json, updated_at=excluded.updated_at, deleted_at=''`,
		id, string(raw), now, now)
	_, _ = e.DB.Exec(`INSERT INTO canvas_skill_flags(skill_id, added, liked, updated_at) VALUES(?,?,?,?)
		ON CONFLICT(skill_id) DO UPDATE SET added=1, updated_at=excluded.updated_at`, id, 1, 0, now)
	return e.hydrateSkill(skill)
}

func (e *Engine) deleteUserSkill(id string) error {
	res, err := e.DB.Exec(`UPDATE canvas_user_skills SET deleted_at = ? WHERE id = ? AND deleted_at = ''`, Now(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("只能删除本机创建的技能")
	}
	return nil
}

func skillFiles(skill map[string]any) []map[string]any {
	return []map[string]any{skillFileMeta(skill, "SKILL.md")}
}

func skillFileMeta(skill map[string]any, path string) map[string]any {
	instruction, _ := skill["instruction"].(string)
	sum := sha256.Sum256([]byte(instruction))
	return map[string]any{
		"path": path, "kind": "markdown", "mimeType": "text/markdown",
		"size": utf8.RuneCountInString(instruction), "sha256": hex.EncodeToString(sum[:]),
	}
}

func skillFileContent(skill map[string]any, path string) (string, error) {
	if path == "" || path == "SKILL.md" {
		instruction, _ := skill["instruction"].(string)
		return instruction, nil
	}
	return "", fmt.Errorf("file not found")
}

func builtinSkillPresets() []map[string]any {
	return []map[string]any{
		{ "presetId": "preset-drama-start", "name": "开一部短剧", "scene": "drama",
			"skillIds": []string{"drama-extractor", "drama-episode-outline", "drama-storyboard", "drama-prompt-video"},
			"rationale": "从正文抽出资产，压成一集，再拆镜生成。", "source": "yoyo-studio", "evidence": "短剧制作顺序", "upgrade": "" },
		{ "presetId": "preset-drama-look", "name": "定妆与场景", "scene": "drama",
			"skillIds": []string{"drama-prompt-character", "drama-prompt-scene", "creative-turnaround", "creative-style-bible"},
			"rationale": "先锁角色和场景，再生成镜头。", "source": "yoyo-studio", "evidence": "一致性优先", "upgrade": "" },
		{ "presetId": "preset-ecom", "name": "一条投放广告", "scene": "ecommerce",
			"skillIds": []string{"ecom-15s-ad", "ecom-ugc-brief", "social-douyin-hook", "ecom-sku-hero"},
			"rationale": "主张、钩子、口播与英雄镜头一次齐。", "source": "yoyo-studio", "evidence": "信息流广告结构", "upgrade": "" },
		{ "presetId": "preset-social", "name": "账号连载", "scene": "social",
			"skillIds": []string{"social-series-bible", "social-douyin-hook", "social-caption-pack", "social-broll"},
			"rationale": "栏目化比单条爆款更稳。", "source": "yoyo-studio", "evidence": "系列内容", "upgrade": "" },
		{ "presetId": "preset-creative", "name": "画风立项", "scene": "creative",
			"skillIds": []string{"creative-style-bible", "creative-world-bible", "creative-lighting", "creative-shot-grammar"},
			"rationale": "立项先锁媒介和光线，减少漂移。", "source": "yoyo-studio", "evidence": "美术基线", "upgrade": "" },
		{ "presetId": "preset-qc", "name": "成片质检", "scene": "others",
			"skillIds": []string{"drama-continuity", "others-qc-slate", "others-legal-mark", "others-privacy-faces"},
			"rationale": "导出前查连贯、乱码、商标和可识别真人。", "source": "yoyo-studio", "evidence": "质检顺序", "upgrade": "" },
		{ "presetId": "preset-live", "name": "直播切片", "scene": "ecommerce",
			"skillIds": []string{"ecom-live-script", "ecom-live-3min", "social-douyin-hook", "ecom-objection"},
			"rationale": "先钩子再卖点，切片能独立成立。", "source": "yoyo-studio", "evidence": "直播切片", "upgrade": "" },
		{ "presetId": "preset-fight", "name": "一场可剪动作", "scene": "drama",
			"skillIds": []string{"drama-fight-coverage", "drama-180-axis", "drama-insert-cutaway", "creative-camera-moves"},
			"rationale": "先地理和轴线，再覆盖和插入。", "source": "yoyo-studio", "evidence": "动作覆盖", "upgrade": "" },
	}
}
