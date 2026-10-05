package video

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// ImportHuobao copies a Huobao SQLite project (no API keys) into this file store.
// SQLite is only an external import format, not Yoyo's persist layer.
func (e *Engine) ImportHuobao(dbPath, staticDir string) (map[string]any, error) {
	src, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer src.Close()
	if staticDir == "" {
		staticDir = detectStaticDir(dbPath)
	}
	nDrama, nEp := 0, 0
	rows, err := src.Query(`SELECT id, title, IFNULL(description,''), IFNULL(genre,''), IFNULL(style,'3d'), IFNULL(aspect_ratio,'16:9') FROM dramas WHERE deleted_at IS NULL OR deleted_at = ''`)
	if err != nil {
		return nil, err
	}
	type drow struct {
		oldID int
		d     Drama
	}
	var dramas []drow
	for rows.Next() {
		var id int
		var d Drama
		if rows.Scan(&id, &d.Title, &d.Description, &d.Genre, &d.Style, &d.AspectRatio) != nil {
			continue
		}
		created, err := e.CreateDrama(d)
		if err != nil {
			continue
		}
		dramas = append(dramas, drow{oldID: id, d: created})
		nDrama++
	}
	rows.Close()

	for _, dr := range dramas {
		idMap := map[string]string{}
		e.importDramaAssets(src, staticDir, dr.oldID, dr.d.ID, idMap)
		eps, err := src.Query(`SELECT id, episode_number, title, IFNULL(content,''), IFNULL(script_content,''), IFNULL(resolution,'720p'), IFNULL(video_url,'') FROM episodes WHERE drama_id = ? AND (deleted_at IS NULL OR deleted_at = '')`, dr.oldID)
		if err != nil {
			continue
		}
		for eps.Next() {
			var oid, num int
			var title, content, script, res, videoURL string
			if eps.Scan(&oid, &num, &title, &content, &script, &res, &videoURL) != nil {
				continue
			}
			ep, err := e.CreateEpisode(dr.d.ID, title, content)
			if err != nil {
				continue
			}
			ep.ScriptContent = script
			ep.Resolution = res
			if h := e.importFile(staticDir, videoURL); h != "" {
				ep.VideoHash = h
			}
			_, _ = e.UpdateEpisode(ep)
			if h := ep.VideoHash; h != "" {
				if rec, err := getDoc[episodeRec](e, colEpisodes, ep.ID); err == nil {
					rec.VideoHash = h
					rec.UpdatedAt = Now()
					_ = e.putDoc(colEpisodes, rec.ID, rec)
				}
			}
			idMap["e:"+strconv.Itoa(oid)] = ep.ID
			nEp++
			e.linkImportedEpisode(src, oid, ep.ID, idMap)
			e.importStoryboards(src, staticDir, oid, ep.ID, idMap)
		}
		eps.Close()
	}
	return map[string]any{"ok": true, "dramas": nDrama, "episodes": nEp}, nil
}

func detectStaticDir(dbPath string) string {
	dir := filepath.Dir(dbPath)
	cands := []string{
		filepath.Join(dir, "static"),
		filepath.Join(filepath.Dir(dir), "static"),
		filepath.Join(dir, "data", "static"),
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

func pragmaCols(db *sql.DB, table string) map[string]bool {
	m := map[string]bool{}
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk) == nil {
			m[name] = true
		}
	}
	return m
}

func pickCol(cols map[string]bool, names ...string) string {
	for _, n := range names {
		if cols[n] {
			return n
		}
	}
	return names[0]
}

func (e *Engine) importDramaAssets(src *sql.DB, staticDir string, oldDrama int, dramaID string, idMap map[string]string) {
	now := Now()
	if rows, err := src.Query(`SELECT id, name, IFNULL(role,''), IFNULL(appearance,''), IFNULL(styling,''), IFNULL(final_prompt,''), IFNULL(image_url,'') FROM characters WHERE drama_id = ? AND (deleted_at IS NULL OR deleted_at = '')`, oldDrama); err == nil {
		for rows.Next() {
			var oid int
			var c Character
			var image string
			if rows.Scan(&oid, &c.Name, &c.Role, &c.Appearance, &c.Styling, &c.FinalPrompt, &image) != nil {
				continue
			}
			c.ID = NewID()
			c.DramaID = dramaID
			c.ImageHash = e.importFile(staticDir, image)
			_ = e.putDoc(colCharacters, c.ID, characterRec{Character: c, CreatedAt: now, UpdatedAt: now})
			idMap["c:"+strconv.Itoa(oid)] = c.ID
		}
		rows.Close()
	}
	sceneCols := pragmaCols(src, "scenes")
	timeCol := pickCol(sceneCols, "time", "time_of_day")
	if rows, err := src.Query(`SELECT id, location, IFNULL(`+timeCol+`,''), IFNULL(prompt,''), IFNULL(lighting,''), IFNULL(final_prompt,''), IFNULL(image_url,'') FROM scenes WHERE drama_id = ? AND (deleted_at IS NULL OR deleted_at = '')`, oldDrama); err == nil {
		for rows.Next() {
			var oid int
			var s Scene
			var image string
			if rows.Scan(&oid, &s.Location, &s.TimeOfDay, &s.Prompt, &s.Lighting, &s.FinalPrompt, &image) != nil {
				continue
			}
			s.ID = NewID()
			s.DramaID = dramaID
			s.ImageHash = e.importFile(staticDir, image)
			_ = e.putDoc(colScenes, s.ID, sceneRec{Scene: s, CreatedAt: now, UpdatedAt: now})
			idMap["s:"+strconv.Itoa(oid)] = s.ID
		}
		rows.Close()
	}
	if rows, err := src.Query(`SELECT id, name, IFNULL(type,''), IFNULL(description,''), IFNULL(final_prompt,''), IFNULL(image_url,'') FROM props WHERE drama_id = ? AND (deleted_at IS NULL OR deleted_at = '')`, oldDrama); err == nil {
		for rows.Next() {
			var oid int
			var p Prop
			var image string
			if rows.Scan(&oid, &p.Name, &p.Type, &p.Description, &p.FinalPrompt, &image) != nil {
				continue
			}
			p.ID = NewID()
			p.DramaID = dramaID
			p.ImageHash = e.importFile(staticDir, image)
			_ = e.putDoc(colProps, p.ID, propRec{Prop: p, CreatedAt: now, UpdatedAt: now})
			idMap["p:"+strconv.Itoa(oid)] = p.ID
		}
		rows.Close()
	}
}

func (e *Engine) linkImportedEpisode(src *sql.DB, oldEp int, episodeID string, idMap map[string]string) {
	linked := false
	if rows, err := src.Query(`SELECT character_id FROM episode_characters WHERE episode_id = ?`, oldEp); err == nil {
		for rows.Next() {
			var oid int
			if rows.Scan(&oid) != nil {
				continue
			}
			if id := idMap["c:"+strconv.Itoa(oid)]; id != "" {
				e.linkCharacter(episodeID, id)
				linked = true
			}
		}
		rows.Close()
	}
	if rows, err := src.Query(`SELECT scene_id FROM episode_scenes WHERE episode_id = ?`, oldEp); err == nil {
		for rows.Next() {
			var oid int
			if rows.Scan(&oid) != nil {
				continue
			}
			if id := idMap["s:"+strconv.Itoa(oid)]; id != "" {
				e.linkScene(episodeID, id)
				linked = true
			}
		}
		rows.Close()
	}
	if rows, err := src.Query(`SELECT prop_id FROM episode_props WHERE episode_id = ?`, oldEp); err == nil {
		for rows.Next() {
			var oid int
			if rows.Scan(&oid) != nil {
				continue
			}
			if id := idMap["p:"+strconv.Itoa(oid)]; id != "" {
				e.linkProp(episodeID, id)
				linked = true
			}
		}
		rows.Close()
	}
	if linked {
		return
	}
	for k, id := range idMap {
		switch {
		case strings.HasPrefix(k, "c:"):
			e.linkCharacter(episodeID, id)
		case strings.HasPrefix(k, "s:"):
			e.linkScene(episodeID, id)
		case strings.HasPrefix(k, "p:"):
			e.linkProp(episodeID, id)
		}
	}
}

func (e *Engine) importStoryboards(src *sql.DB, staticDir string, oldEp int, episodeID string, idMap map[string]string) {
	cols := pragmaCols(src, "storyboards")
	numCol := pickCol(cols, "storyboard_number", "shot_number")
	videoCol := pickCol(cols, "video_url")
	q := `SELECT id, ` + numCol + `, IFNULL(title,''), IFNULL(shot_type,''), IFNULL(description,''), IFNULL(video_prompt,''), IFNULL(duration,10), IFNULL(scene_id,0)`
	if videoCol != "" && cols[videoCol] {
		q += `, IFNULL(` + videoCol + `,'')`
	} else {
		q += `, ''`
	}
	q += ` FROM storyboards WHERE episode_id = ? AND (deleted_at IS NULL OR deleted_at = '') ORDER BY ` + numCol
	rows, err := src.Query(q, oldEp)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var oid, num, dur, sceneOID int
		var title, shotType, desc, prompt, videoURL string
		if rows.Scan(&oid, &num, &title, &shotType, &desc, &prompt, &dur, &sceneOID, &videoURL) != nil {
			continue
		}
		s := Shot{
			ID: NewID(), EpisodeID: episodeID, ShotNumber: num, Title: title, ShotType: shotType,
			Description: desc, VideoPrompt: prompt, Duration: dur, Status: "pending",
			SceneID: idMap["s:"+strconv.Itoa(sceneOID)],
		}
		if h := e.importFile(staticDir, videoURL); h != "" {
			s.VideoHash = h
			s.Status = "ready"
		}
		out, err := e.SaveShots(episodeID, []Shot{s}, false)
		if err != nil || len(out) == 0 {
			continue
		}
		shotID := out[0].ID
		idMap["b:"+strconv.Itoa(oid)] = shotID
		if dur > 0 {
			if rec, err := getDoc[shotRec](e, colShots, shotID); err == nil {
				rec.Duration = ClampProviderDuration(dur, "aliyun")
				_ = e.putDoc(colShots, shotID, rec)
			}
		}
		if s.VideoHash != "" {
			if rec, err := getDoc[shotRec](e, colShots, shotID); err == nil {
				rec.VideoHash = s.VideoHash
				rec.Status = "ready"
				rec.UpdatedAt = Now()
				_ = e.putDoc(colShots, shotID, rec)
			}
		}
		if chRows, err := src.Query(`SELECT character_id FROM storyboard_characters WHERE storyboard_id = ?`, oid); err == nil {
			var ids []string
			for chRows.Next() {
				var cid int
				if chRows.Scan(&cid) != nil {
					continue
				}
				if id := idMap["c:"+strconv.Itoa(cid)]; id != "" {
					ids = append(ids, id)
				}
			}
			chRows.Close()
			if rec, err := getDoc[shotRec](e, colShots, shotID); err == nil {
				rec.CharacterIDs = ids
				_ = e.putDoc(colShots, shotID, rec)
			}
		}
		if pRows, err := src.Query(`SELECT prop_id FROM storyboard_props WHERE storyboard_id = ?`, oid); err == nil {
			var ids []string
			for pRows.Next() {
				var pid int
				if pRows.Scan(&pid) != nil {
					continue
				}
				if id := idMap["p:"+strconv.Itoa(pid)]; id != "" {
					ids = append(ids, id)
				}
			}
			pRows.Close()
			if rec, err := getDoc[shotRec](e, colShots, shotID); err == nil {
				rec.PropIDs = ids
				_ = e.putDoc(colShots, shotID, rec)
			}
		}
	}
}

func (e *Engine) importFile(staticDir, rel string) string {
	rel = strings.TrimSpace(rel)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return ""
	}
	if strings.HasPrefix(rel, "http://") || strings.HasPrefix(rel, "https://") {
		return ""
	}
	p := rel
	if !filepath.IsAbs(rel) {
		if staticDir == "" {
			return ""
		}
		p = filepath.Join(staticDir, strings.TrimPrefix(rel, "static/"))
		if _, err := os.Stat(p); err != nil {
			p = filepath.Join(staticDir, rel)
		}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	h, err := e.PutBytes(b)
	if err != nil {
		return ""
	}
	return h
}
