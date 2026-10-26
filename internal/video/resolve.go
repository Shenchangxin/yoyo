// resolve.go 把名字收成短剧里的资产标识，并补齐缺图和缺片。

package video

import "strings"

// resolveIDs 把一批名字或标识收成不重复的文档标识。收不到的丢掉。
func (e *Engine) resolveIDs(kind, dramaID string, ids []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range ids {
		id := e.resolveOne(kind, dramaID, raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// resolveOne 先按标识读取未删除且属于该剧的文档，再按归一化名字匹配。场景还比较地点原文，以及地点加时段的键是否等于归一化后的输入。角色、道具、场景以外返回空串。
func (e *Engine) resolveOne(kind, dramaID, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch kind {
	case "character":
		if rec, err := getDoc[characterRec](e, colCharacters, raw); err == nil && rec.DeletedAt == "" && rec.DramaID == dramaID {
			return rec.ID
		}
		key := NormalizeName(raw)
		for _, rec := range loadCol[characterRec](e, colCharacters) {
			if rec.DramaID == dramaID && rec.DeletedAt == "" && NormalizeName(rec.Name) == key {
				return rec.ID
			}
		}
	case "prop":
		if rec, err := getDoc[propRec](e, colProps, raw); err == nil && rec.DeletedAt == "" && rec.DramaID == dramaID {
			return rec.ID
		}
		key := NormalizeName(raw)
		for _, rec := range loadCol[propRec](e, colProps) {
			if rec.DramaID == dramaID && rec.DeletedAt == "" && NormalizeName(rec.Name) == key {
				return rec.ID
			}
		}
	case "scene":
		if rec, err := getDoc[sceneRec](e, colScenes, raw); err == nil && rec.DeletedAt == "" && rec.DramaID == dramaID {
			return rec.ID
		}
		for _, rec := range loadCol[sceneRec](e, colScenes) {
			if rec.DramaID != dramaID || rec.DeletedAt != "" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(rec.Location), raw) || NormalizeSceneKey(rec.Location, rec.TimeOfDay) == NormalizeName(raw) {
				return rec.ID
			}
		}
	}
	return ""
}

// DeleteShot 给镜头写上删除时间和更新时间，不从文档库抹掉。
func (e *Engine) DeleteShot(id string) error {
	rec, err := getDoc[shotRec](e, colShots, id)
	if err != nil {
		return err
	}
	now := Now()
	rec.DeletedAt = now
	rec.UpdatedAt = now
	return e.putDoc(colShots, id, rec)
}

// AttachImage 把字节存成上传并写入 image_hash。该资产挂着的第一集如果不再缺静帧，就把 assets 标成 done。
func (e *Engine) AttachImage(kind, id string, raw []byte) error {
	hash, _, err := e.StoreUpload(raw)
	if err != nil {
		return err
	}
	if err := e.UpdateAsset(kind, id, map[string]any{"image_hash": hash}); err != nil {
		return err
	}
	episodeID := e.firstLinkedEpisode(kind, id)
	if episodeID != "" && e.missingStills(episodeID) == 0 {
		_ = e.patchPipeline(episodeID, "assets", "done", "")
	}
	return nil
}

// firstLinkedEpisode 返回第一集未删除、且挂着该角色、场景或道具的集标识。
func (e *Engine) firstLinkedEpisode(kind, id string) string {
	for _, ep := range loadCol[episodeRec](e, colEpisodes) {
		if ep.DeletedAt != "" {
			continue
		}
		switch kind {
		case "character":
			if containsID(ep.CharacterIDs, id) {
				return ep.ID
			}
		case "scene":
			if containsID(ep.SceneIDs, id) {
				return ep.ID
			}
		case "prop":
			if containsID(ep.PropIDs, id) {
				return ep.ID
			}
		}
	}
	return ""
}

// missingStills 数已挂上但没有静帧的场景和道具，以及已挂上、没有静帧、且不是旁白的角色。
func (e *Engine) missingStills(episodeID string) int {
	n := 0
	chars, _ := e.EpisodeCharacters(episodeID)
	for _, c := range chars {
		if c.Linked && c.ImageHash == "" && !IsNarrator(c.Name, c.Role) {
			n++
		}
	}
	scenes, _ := e.EpisodeScenes(episodeID)
	for _, s := range scenes {
		if s.Linked && s.ImageHash == "" {
			n++
		}
	}
	props, _ := e.EpisodeProps(episodeID)
	for _, p := range props {
		if p.Linked && p.ImageHash == "" {
			n++
		}
	}
	return n
}

// missingClips 数没有视频哈希的镜头。读镜头失败时返回 0。
func (e *Engine) missingClips(episodeID string) int {
	shots, err := e.ListShots(episodeID)
	if err != nil {
		return 0
	}
	n := 0
	for _, s := range shots {
		if s.VideoHash == "" {
			n++
		}
	}
	return n
}

// GenerateMissingAssets 为缺静帧的已挂角色、场景和道具排队。旁白和已有静帧的跳过。排出了任务就把 assets 标成 running。一条都没排出且有错误时返回第一个错误。不再缺静帧时把 assets 标成 done。
func (e *Engine) GenerateMissingAssets(episodeID string) ([]Job, error) {
	chars, _ := e.EpisodeCharacters(episodeID)
	scenes, _ := e.EpisodeScenes(episodeID)
	props, _ := e.EpisodeProps(episodeID)
	var out []Job
	var first error
	enqueue := func(kind, id string) {
		j, err := e.GenerateAsset(kind, id, episodeID)
		if err != nil {
			if first == nil {
				first = err
			}
			return
		}
		out = append(out, j)
	}
	for _, c := range chars {
		if !c.Linked || c.ImageHash != "" || IsNarrator(c.Name, c.Role) {
			continue
		}
		enqueue("character", c.ID)
	}
	for _, s := range scenes {
		if !s.Linked || s.ImageHash != "" {
			continue
		}
		enqueue("scene", s.ID)
	}
	for _, p := range props {
		if !p.Linked || p.ImageHash != "" {
			continue
		}
		enqueue("prop", p.ID)
	}
	if len(out) > 0 {
		_ = e.patchPipeline(episodeID, "assets", "running", "")
		return out, nil
	}
	if first != nil {
		return out, first
	}
	if e.missingStills(episodeID) == 0 {
		_ = e.patchPipeline(episodeID, "assets", "done", "")
	}
	return out, nil
}

// GenerateMissingShots 为没有视频且状态不是 generating 的镜头排队。排出了任务就把 gen 标成 running。一条都没排出且有错误时返回第一个错误。不再缺片时把 gen 标成 done。
func (e *Engine) GenerateMissingShots(episodeID string) ([]Job, error) {
	shots, err := e.ListShots(episodeID)
	if err != nil {
		return nil, err
	}
	var out []Job
	var first error
	for _, s := range shots {
		if s.VideoHash != "" || s.Status == "generating" {
			continue
		}
		j, err := e.GenerateShot(s.ID)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		out = append(out, j)
	}
	if len(out) > 0 {
		_ = e.patchPipeline(episodeID, "gen", "running", "")
		return out, nil
	}
	if first != nil {
		return out, first
	}
	if e.missingClips(episodeID) == 0 {
		_ = e.patchPipeline(episodeID, "gen", "done", "")
	}
	return out, nil
}

// SettingsMap 读出键值设置。缺 content_language 时补 zh，缺 studio_aspect 时补 9:16，缺 studio_publish_platform 时补 douyin。
func (e *Engine) SettingsMap() map[string]string {
	out := e.kvSettings()
	if _, ok := out["content_language"]; !ok {
		out["content_language"] = "zh"
	}
	if _, ok := out["studio_aspect"]; !ok {
		out["studio_aspect"] = "9:16"
	}
	if _, ok := out["studio_publish_platform"]; !ok {
		out["studio_publish_platform"] = "douyin"
	}
	return out
}
