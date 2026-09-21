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
		var id string
		if e.DB.QueryRow(`SELECT id FROM characters WHERE id = ? AND drama_id = ? AND deleted_at = ''`, raw, dramaID).Scan(&id) == nil {
			return id
		}
		key := NormalizeName(raw)
		rows, err := e.DB.Query(`SELECT id, name FROM characters WHERE drama_id = ? AND deleted_at = ''`, dramaID)
		if err != nil {
			return ""
		}
		defer rows.Close()
		for rows.Next() {
			var id, name string
			if rows.Scan(&id, &name) != nil {
				continue
			}
			if NormalizeName(name) == key {
				return id
			}
		}
	case "prop":
		var id string
		if e.DB.QueryRow(`SELECT id FROM props WHERE id = ? AND drama_id = ? AND deleted_at = ''`, raw, dramaID).Scan(&id) == nil {
			return id
		}
		key := NormalizeName(raw)
		rows, err := e.DB.Query(`SELECT id, name FROM props WHERE drama_id = ? AND deleted_at = ''`, dramaID)
		if err != nil {
			return ""
		}
		defer rows.Close()
		for rows.Next() {
			var id, name string
			if rows.Scan(&id, &name) != nil {
				continue
			}
			if NormalizeName(name) == key {
				return id
			}
		}
	case "scene":
		var id string
		if e.DB.QueryRow(`SELECT id FROM scenes WHERE id = ? AND drama_id = ? AND deleted_at = ''`, raw, dramaID).Scan(&id) == nil {
			return id
		}
		rows, err := e.DB.Query(`SELECT id, location, time_of_day FROM scenes WHERE drama_id = ? AND deleted_at = ''`, dramaID)
		if err != nil {
			return ""
		}
		defer rows.Close()
		for rows.Next() {
			var id, loc, tod string
			if rows.Scan(&id, &loc, &tod) != nil {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(loc), raw) || NormalizeSceneKey(loc, tod) == NormalizeName(raw) {
				return id
			}
		}
	}
	return ""
}

func (e *Engine) DeleteShot(id string) error {
	now := Now()
	_, err := e.DB.Exec(`UPDATE storyboards SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	return err
}

func (e *Engine) AttachImage(kind, id string, raw []byte) error {
	hash, _, err := e.StoreUpload(raw)
	if err != nil {
		return err
	}
	return e.UpdateAsset(kind, id, map[string]any{"image_hash": hash})
}

func (e *Engine) GenerateMissingAssets(episodeID string) ([]Job, error) {
	chars, _ := e.EpisodeCharacters(episodeID)
	scenes, _ := e.EpisodeScenes(episodeID)
	props, _ := e.EpisodeProps(episodeID)
	var out []Job
	for _, c := range chars {
		if !c.Linked || c.ImageHash != "" {
			continue
		}
		j, err := e.GenerateAsset("character", c.ID, episodeID)
		if err != nil {
			return out, err
		}
		out = append(out, j)
	}
	for _, s := range scenes {
		if !s.Linked || s.ImageHash != "" {
			continue
		}
		j, err := e.GenerateAsset("scene", s.ID, episodeID)
		if err != nil {
			return out, err
		}
		out = append(out, j)
	}
	for _, p := range props {
		if !p.Linked || p.ImageHash != "" {
			continue
		}
		j, err := e.GenerateAsset("prop", p.ID, episodeID)
		if err != nil {
			return out, err
		}
		out = append(out, j)
	}
	if len(out) > 0 {
		_ = e.patchPipeline(episodeID, "assets", "running", "")
	}
	return out, nil
}

func (e *Engine) GenerateMissingShots(episodeID string) ([]Job, error) {
	shots, err := e.ListShots(episodeID)
	if err != nil {
		return nil, err
	}
	var out []Job
	for _, s := range shots {
		if s.VideoHash != "" || s.Status == "generating" {
			continue
		}
		j, err := e.GenerateShot(s.ID)
		if err != nil {
			return out, err
		}
		out = append(out, j)
	}
	if len(out) > 0 {
		_ = e.patchPipeline(episodeID, "gen", "running", "")
	}
	return out, nil
}

func (e *Engine) SettingsMap() map[string]string {
	rows, err := e.DB.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return map[string]string{}
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			out[k] = v
		}
	}
	if _, ok := out["content_language"]; !ok {
		out["content_language"] = "zh"
	}
	return out
}
