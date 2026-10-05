package video

// migrateSQLiteIfPresent is a one-shot reader of a legacy video.sqlite.
// Product data lives in video/docs JSON files. Do not add new SQLite writes.


import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const migratedMarker = "_migrated_from_sqlite.json"

func migrateSQLiteIfPresent(e *Engine) error {
	if e == nil || e.docs == nil {
		return nil
	}
	src := filepath.Join(e.Dir, "video.sqlite")
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if e.docs.Exists(migratedMarker) {
		return nil
	}
	db, err := sql.Open("sqlite", src+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()

	migrateKV := func() {
		rows, err := db.Query(`SELECT key, value FROM settings`)
		if err != nil {
			return
		}
		defer rows.Close()
		m := e.kvSettings()
		for rows.Next() {
			var k, v string
			if rows.Scan(&k, &v) == nil && k != "" {
				m[k] = v
			}
		}
		_ = e.putJSON(fileSettings, m)
	}

	if e.Conn != nil {
		rows, err := db.Query(`SELECT id, service_type, provider, name, base_url, vault_key, model, models, priority, is_default, is_active, settings, created_at, updated_at FROM providers`)
		if err == nil {
			for rows.Next() {
				var id, st, vendor, name, base, vaultKey, model, models, settings, created, updated string
				var priority, isDef, isActive int
				if rows.Scan(&id, &st, &vendor, &name, &base, &vaultKey, &model, &models, &priority, &isDef, &isActive, &settings, &created, &updated) != nil {
					continue
				}
				_, _ = e.Conn.ImportProvider(st, vendor, name, base, vaultKey, model, models, priority, isActive != 0, isDef != 0, settings, created, updated, id)
			}
			rows.Close()
		}
		if rows, err := db.Query(`SELECT id, plugin_id, name, capability, base_url, vault_key, model, models, enabled, created_at, updated_at FROM canvas_channels`); err == nil {
			for rows.Next() {
				var id, plugin, name, cap, base, vaultKey, model, models, created, updated string
				var enabled int
				if rows.Scan(&id, &plugin, &name, &cap, &base, &vaultKey, &model, &models, &enabled, &created, &updated) != nil {
					continue
				}
				id = strings.TrimPrefix(id, "ch-")
				_, _ = e.Conn.ImportProvider(cap, plugin, name, base, vaultKey, model, models, 0, enabled != 0, false, "{}", created, updated, id)
			}
			rows.Close()
		}
		e.Conn.MergeSameAccount()
	}

	copyRows := func(q string, scan func(*sql.Rows) error) {
		rows, err := db.Query(q)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			_ = scan(rows)
		}
	}

	copyRows(`SELECT id, name, value, prompt, description, sort_order, is_active, seeded, created_at, updated_at FROM style_presets`, func(rows *sql.Rows) error {
		var r styleRec
		var active, seeded int
		if err := rows.Scan(&r.ID, &r.Name, &r.Value, &r.Prompt, &r.Description, &r.SortOrder, &active, &seeded, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return err
		}
		r.IsActive = active != 0
		r.Seeded = seeded != 0
		return e.putDoc(colStyles, r.ID, r)
	})
	migrateKV()

	copyRows(`SELECT id, type, status, provider, model, vault_key, remote_id, prompt, params, result_hash, poster_hash, error, drama_id, episode_id, storyboard_id, character_id, scene_id, prop_id, created_at, updated_at, completed_at FROM jobs`, func(rows *sql.Rows) error {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.Status, &j.Provider, &j.Model, &j.VaultKey, &j.RemoteID, &j.Prompt, &j.Params, &j.ResultHash, &j.PosterHash, &j.Error, &j.DramaID, &j.EpisodeID, &j.StoryboardID, &j.CharacterID, &j.SceneID, &j.PropID, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt); err != nil {
			return err
		}
		return e.putDoc(colJobs, j.ID, j)
	})
	copyRows(`SELECT id, title, description, genre, style, aspect_ratio, status, thumbnail_hash, created_at, updated_at, deleted_at FROM dramas`, func(rows *sql.Rows) error {
		var r dramaRec
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Genre, &r.Style, &r.AspectRatio, &r.Status, &r.ThumbnailHash, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colDramas, r.ID, r)
	})

	epLinks := map[string]*episodeRec{}
	copyRows(`SELECT id, drama_id, episode_number, title, content, script_content, status, video_hash, poster_hash, image_provider_id, video_provider_id, image_model, video_model, tts_provider_id, tts_model, resolution, pipeline, created_at, updated_at, deleted_at FROM episodes`, func(rows *sql.Rows) error {
		var r episodeRec
		if err := rows.Scan(&r.ID, &r.DramaID, &r.EpisodeNumber, &r.Title, &r.Content, &r.ScriptContent, &r.Status, &r.VideoHash, &r.PosterHash, &r.ImageProviderID, &r.VideoProviderID, &r.ImageModel, &r.VideoModel, &r.TTSProviderID, &r.TTSModel, &r.Resolution, &r.Pipeline, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		epLinks[r.ID] = &r
		return nil
	})
	link := func(q, kind string) {
		rows, err := db.Query(q)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var epID, assetID string
			if rows.Scan(&epID, &assetID) != nil {
				continue
			}
			rec := epLinks[epID]
			if rec == nil {
				continue
			}
			switch kind {
			case "c":
				rec.CharacterIDs = addID(rec.CharacterIDs, assetID)
			case "s":
				rec.SceneIDs = addID(rec.SceneIDs, assetID)
			case "p":
				rec.PropIDs = addID(rec.PropIDs, assetID)
			}
		}
	}
	link(`SELECT episode_id, character_id FROM episode_characters`, "c")
	link(`SELECT episode_id, scene_id FROM episode_scenes`, "s")
	link(`SELECT episode_id, prop_id FROM episode_props`, "p")
	for _, rec := range epLinks {
		_ = e.putDoc(colEpisodes, rec.ID, rec)
	}

	copyRows(`SELECT id, drama_id, name, role, appearance, styling, final_prompt, image_hash, sort_order, created_at, updated_at, deleted_at FROM characters`, func(rows *sql.Rows) error {
		var r characterRec
		if err := rows.Scan(&r.ID, &r.DramaID, &r.Name, &r.Role, &r.Appearance, &r.Styling, &r.FinalPrompt, &r.ImageHash, &r.SortOrder, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colCharacters, r.ID, r)
	})
	copyRows(`SELECT id, drama_id, location, time_of_day, prompt, lighting, final_prompt, image_hash, created_at, updated_at, deleted_at FROM scenes`, func(rows *sql.Rows) error {
		var r sceneRec
		if err := rows.Scan(&r.ID, &r.DramaID, &r.Location, &r.TimeOfDay, &r.Prompt, &r.Lighting, &r.FinalPrompt, &r.ImageHash, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colScenes, r.ID, r)
	})
	copyRows(`SELECT id, drama_id, name, type, description, final_prompt, image_hash, created_at, updated_at, deleted_at FROM props`, func(rows *sql.Rows) error {
		var r propRec
		if err := rows.Scan(&r.ID, &r.DramaID, &r.Name, &r.Type, &r.Description, &r.FinalPrompt, &r.ImageHash, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colProps, r.ID, r)
	})

	shots := map[string]*shotRec{}
	copyRows(`SELECT id, episode_id, scene_id, shot_number, title, shot_type, angle, movement, atmosphere, description, video_prompt, duration, video_hash, poster_hash, status, created_at, updated_at, deleted_at FROM storyboards`, func(rows *sql.Rows) error {
		var r shotRec
		if err := rows.Scan(&r.ID, &r.EpisodeID, &r.SceneID, &r.ShotNumber, &r.Title, &r.ShotType, &r.Angle, &r.Movement, &r.Atmosphere, &r.Description, &r.VideoPrompt, &r.Duration, &r.VideoHash, &r.PosterHash, &r.Status, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		shots[r.ID] = &r
		return nil
	})
	if rows, err := db.Query(`SELECT storyboard_id, character_id FROM storyboard_characters`); err == nil {
		for rows.Next() {
			var sid, cid string
			if rows.Scan(&sid, &cid) == nil {
				if s := shots[sid]; s != nil {
					s.CharacterIDs = addID(s.CharacterIDs, cid)
				}
			}
		}
		rows.Close()
	}
	if rows, err := db.Query(`SELECT storyboard_id, prop_id FROM storyboard_props`); err == nil {
		for rows.Next() {
			var sid, pid string
			if rows.Scan(&sid, &pid) == nil {
				if s := shots[sid]; s != nil {
					s.PropIDs = addID(s.PropIDs, pid)
				}
			}
		}
		rows.Close()
	}
	for _, s := range shots {
		_ = e.putDoc(colShots, s.ID, s)
	}

	binds := e.loadBinds()
	if rows, err := db.Query(`SELECT session_id, episode_id, drama_id FROM session_binds`); err == nil {
		for rows.Next() {
			var b Bind
			if rows.Scan(&b.SessionID, &b.EpisodeID, &b.DramaID) == nil {
				binds[b.SessionID] = b
			}
		}
		rows.Close()
		_ = e.saveBinds(binds)
	}
	cb := e.loadCanvasBinds()
	if rows, err := db.Query(`SELECT session_id, canvas_id FROM canvas_session_binds`); err == nil {
		for rows.Next() {
			var sid, cid string
			if rows.Scan(&sid, &cid) == nil {
				cb[sid] = cid
			}
		}
		rows.Close()
		_ = e.saveCanvasBinds(cb)
	}

	copyRows(`SELECT id, title, payload_json, timeline_json, revision, workspace_mode, theme, project_link, created_at, updated_at, deleted_at FROM canvas_projects`, func(rows *sql.Rows) error {
		var r canvasProjectRow
		if err := rows.Scan(&r.ID, &r.Title, &r.PayloadJSON, &r.TimelineJSON, &r.Revision, &r.WorkspaceMode, &r.Theme, &r.ProjectLink, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colProjects, r.ID, r)
	})
	copyRows(`SELECT id, canvas_id, revision, title, payload_json, reason, node_count, connection_count, created_at FROM canvas_snapshots`, func(rows *sql.Rows) error {
		var r canvasSnapshotRec
		if err := rows.Scan(&r.ID, &r.CanvasID, &r.Revision, &r.Title, &r.PayloadJSON, &r.Reason, &r.NodeCount, &r.ConnectionCount, &r.CreatedAt); err != nil {
			return err
		}
		return e.putDoc(colSnapshots, r.ID, r)
	})
	copyRows(`SELECT id, cas_hash, kind, mime, bytes, width, height, duration_ms, poster_hash, playback_status, file_name, created_at, updated_at, deleted_at FROM canvas_resources`, func(rows *sql.Rows) error {
		var r canvasResourceRec
		if err := rows.Scan(&r.ID, &r.CASHash, &r.Kind, &r.Mime, &r.Bytes, &r.Width, &r.Height, &r.DurationMs, &r.PosterHash, &r.PlaybackStatus, &r.FileName, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colResources, r.ID, r)
	})
	copyRows(`SELECT id, name, position, created_at, updated_at, deleted_at FROM canvas_asset_folders`, func(rows *sql.Rows) error {
		var r canvasFolderRec
		if err := rows.Scan(&r.ID, &r.Name, &r.Position, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colFolders, r.ID, r)
	})
	copyRows(`SELECT id, folder_id, kind, category, title, resource_id, payload_json, status, created_at, updated_at, deleted_at FROM canvas_assets`, func(rows *sql.Rows) error {
		var r canvasAssetRec
		if err := rows.Scan(&r.ID, &r.FolderID, &r.Kind, &r.Category, &r.Title, &r.ResourceID, &r.PayloadJSON, &r.Status, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colAssets, r.ID, r)
	})
	copyRows(`SELECT id, project_id, kind, title, payload_json, sort_order, created_at, updated_at, deleted_at FROM canvas_project_units`, func(rows *sql.Rows) error {
		var r canvasUnitRec
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Kind, &r.Title, &r.PayloadJSON, &r.SortOrder, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colUnits, r.ID, r)
	})
	copyRows(`SELECT id, topic, category, situation, lesson, steps_json, status, created_at, updated_at FROM canvas_lessons`, func(rows *sql.Rows) error {
		var r canvasLessonRec
		if err := rows.Scan(&r.ID, &r.Topic, &r.Category, &r.Situation, &r.Lesson, &r.StepsJSON, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return err
		}
		return e.putDoc(colLessons, r.ID, r)
	})
	copyRows(`SELECT id, payload_json, created_at, updated_at, deleted_at FROM canvas_user_skills`, func(rows *sql.Rows) error {
		var r canvasUserSkillRec
		if err := rows.Scan(&r.ID, &r.PayloadJSON, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt); err != nil {
			return err
		}
		return e.putDoc(colUserSkills, r.ID, r)
	})
	flags := e.loadSkillFlags()
	if rows, err := db.Query(`SELECT skill_id, added, liked, updated_at FROM canvas_skill_flags`); err == nil {
		for rows.Next() {
			var id, updated string
			var added, liked int
			if rows.Scan(&id, &added, &liked, &updated) == nil {
				flags[id] = skillFlagRec{Added: added != 0, Liked: liked != 0, UpdatedAt: updated}
			}
		}
		rows.Close()
		_ = e.saveSkillFlags(flags)
	}

	_ = e.putJSON(migratedMarker, map[string]any{"from": src, "at": Now()})
	_ = os.Rename(src, src+".bak")
	return nil
}
