package video

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ImportHuobao copies a Huobao SQLite project (no API keys) into this store.
func (e *Engine) ImportHuobao(dbPath, staticDir string) (map[string]any, error) {
	src, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer src.Close()
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
	idMap := map[string]string{}
	for _, dr := range dramas {
		eps, err := src.Query(`SELECT id, episode_number, title, IFNULL(content,''), IFNULL(script_content,''), IFNULL(resolution,'720p') FROM episodes WHERE drama_id = ? AND (deleted_at IS NULL OR deleted_at = '')`, dr.oldID)
		if err != nil {
			continue
		}
		for eps.Next() {
			var oid, num int
			var title, content, script, res string
			if eps.Scan(&oid, &num, &title, &content, &script, &res) != nil {
				continue
			}
			ep, err := e.CreateEpisode(dr.d.ID, title, content)
			if err != nil {
				continue
			}
			ep.ScriptContent = script
			ep.Resolution = res
			_, _ = e.UpdateEpisode(ep)
			idMap["e:"+strconv.Itoa(oid)] = ep.ID
			nEp++
			e.importLinked(src, staticDir, dr.oldID, dr.d.ID, ep.ID, oid, idMap)
		}
		eps.Close()
	}
	return map[string]any{"ok": true, "dramas": nDrama, "episodes": nEp}, nil
}

func (e *Engine) importLinked(src *sql.DB, staticDir string, oldDrama int, dramaID, episodeID string, oldEp int, idMap map[string]string) {
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
			_, _ = e.DB.Exec(`INSERT INTO characters(id, drama_id, name, role, appearance, styling, final_prompt, image_hash, sort_order, created_at, updated_at, deleted_at) VALUES(?,?,?,?,?,?,?,?,0,?,?, '')`,
				c.ID, dramaID, c.Name, c.Role, c.Appearance, c.Styling, c.FinalPrompt, c.ImageHash, now, now)
			e.linkCharacter(episodeID, c.ID)
			idMap["c:"+strconv.Itoa(oid)] = c.ID
		}
		rows.Close()
	}
	if rows, err := src.Query(`SELECT id, location, IFNULL(time_of_day,''), IFNULL(prompt,''), IFNULL(lighting,''), IFNULL(final_prompt,''), IFNULL(image_url,'') FROM scenes WHERE drama_id = ? AND (deleted_at IS NULL OR deleted_at = '')`, oldDrama); err == nil {
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
			_, _ = e.DB.Exec(`INSERT INTO scenes(id, drama_id, location, time_of_day, prompt, lighting, final_prompt, image_hash, created_at, updated_at, deleted_at) VALUES(?,?,?,?,?,?,?,?,?,?, '')`,
				s.ID, dramaID, s.Location, s.TimeOfDay, s.Prompt, s.Lighting, s.FinalPrompt, s.ImageHash, now, now)
			e.linkScene(episodeID, s.ID)
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
			_, _ = e.DB.Exec(`INSERT INTO props(id, drama_id, name, type, description, final_prompt, image_hash, created_at, updated_at, deleted_at) VALUES(?,?,?,?,?,?,?,?,?, '')`,
				p.ID, dramaID, p.Name, p.Type, p.Description, p.FinalPrompt, p.ImageHash, now, now)
			e.linkProp(episodeID, p.ID)
			idMap["p:"+strconv.Itoa(oid)] = p.ID
		}
		rows.Close()
	}
	if rows, err := src.Query(`SELECT id, shot_number, IFNULL(title,''), IFNULL(shot_type,''), IFNULL(description,''), IFNULL(video_prompt,''), IFNULL(duration,10), IFNULL(scene_id,0) FROM storyboards WHERE episode_id = ? AND (deleted_at IS NULL OR deleted_at = '') ORDER BY shot_number`, oldEp); err == nil {
		for rows.Next() {
			var oid, num, dur, sceneOID int
			var title, shotType, desc, prompt string
			if rows.Scan(&oid, &num, &title, &shotType, &desc, &prompt, &dur, &sceneOID) != nil {
				continue
			}
			s := Shot{
				ID: NewID(), EpisodeID: episodeID, ShotNumber: num, Title: title, ShotType: shotType,
				Description: desc, VideoPrompt: prompt, Duration: dur, Status: "pending",
				SceneID: idMap["s:"+strconv.Itoa(sceneOID)],
			}
			_, _ = e.SaveShots(episodeID, []Shot{s}, false)
			idMap["b:"+strconv.Itoa(oid)] = s.ID
		}
		rows.Close()
	}
}

func (e *Engine) importFile(staticDir, rel string) string {
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || staticDir == "" {
		return ""
	}
	p := rel
	if !filepath.IsAbs(rel) {
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
