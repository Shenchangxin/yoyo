package video

import "strings"

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

func (e *Engine) SettingsMap() map[string]string {
	out := e.kvSettings()
	if _, ok := out["content_language"]; !ok {
		out["content_language"] = "zh"
	}
	return out
}
