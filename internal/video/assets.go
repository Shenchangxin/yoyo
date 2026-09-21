package video

import (
	"fmt"
	"strings"
)

func (e *Engine) EpisodeCharacters(episodeID string) ([]Character, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	rows, err := e.DB.Query(`SELECT c.id, c.drama_id, c.name, c.role, c.appearance, c.styling, c.final_prompt, c.image_hash, c.sort_order,
		EXISTS(SELECT 1 FROM episode_characters x WHERE x.episode_id = ? AND x.character_id = c.id)
		FROM characters c WHERE c.drama_id = ? AND c.deleted_at = '' ORDER BY c.sort_order, c.name`, episodeID, ep.DramaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Character
	for rows.Next() {
		var c Character
		var linked int
		if err := rows.Scan(&c.ID, &c.DramaID, &c.Name, &c.Role, &c.Appearance, &c.Styling, &c.FinalPrompt, &c.ImageHash, &c.SortOrder, &linked); err != nil {
			return nil, err
		}
		c.Linked = linked != 0
		out = append(out, c)
	}
	if out == nil {
		out = []Character{}
	}
	return out, nil
}

func (e *Engine) EpisodeScenes(episodeID string) ([]Scene, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	rows, err := e.DB.Query(`SELECT s.id, s.drama_id, s.location, s.time_of_day, s.prompt, s.lighting, s.final_prompt, s.image_hash,
		EXISTS(SELECT 1 FROM episode_scenes x WHERE x.episode_id = ? AND x.scene_id = s.id)
		FROM scenes s WHERE s.drama_id = ? AND s.deleted_at = '' ORDER BY s.location`, episodeID, ep.DramaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Scene
	for rows.Next() {
		var s Scene
		var linked int
		if err := rows.Scan(&s.ID, &s.DramaID, &s.Location, &s.TimeOfDay, &s.Prompt, &s.Lighting, &s.FinalPrompt, &s.ImageHash, &linked); err != nil {
			return nil, err
		}
		s.Linked = linked != 0
		out = append(out, s)
	}
	if out == nil {
		out = []Scene{}
	}
	return out, nil
}

func (e *Engine) EpisodeProps(episodeID string) ([]Prop, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	rows, err := e.DB.Query(`SELECT p.id, p.drama_id, p.name, p.type, p.description, p.final_prompt, p.image_hash,
		EXISTS(SELECT 1 FROM episode_props x WHERE x.episode_id = ? AND x.prop_id = p.id)
		FROM props p WHERE p.drama_id = ? AND p.deleted_at = '' ORDER BY p.name`, episodeID, ep.DramaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Prop
	for rows.Next() {
		var p Prop
		var linked int
		if err := rows.Scan(&p.ID, &p.DramaID, &p.Name, &p.Type, &p.Description, &p.FinalPrompt, &p.ImageHash, &linked); err != nil {
			return nil, err
		}
		p.Linked = linked != 0
		out = append(out, p)
	}
	if out == nil {
		out = []Prop{}
	}
	return out, nil
}

func (e *Engine) linkCharacter(episodeID, characterID string) {
	_, _ = e.DB.Exec(`INSERT OR IGNORE INTO episode_characters(episode_id, character_id) VALUES(?,?)`, episodeID, characterID)
}
func (e *Engine) linkScene(episodeID, sceneID string) {
	_, _ = e.DB.Exec(`INSERT OR IGNORE INTO episode_scenes(episode_id, scene_id) VALUES(?,?)`, episodeID, sceneID)
}
func (e *Engine) linkProp(episodeID, propID string) {
	_, _ = e.DB.Exec(`INSERT OR IGNORE INTO episode_props(episode_id, prop_id) VALUES(?,?)`, episodeID, propID)
}

func (e *Engine) SaveCharacters(episodeID string, items []Character) ([]Character, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	existing, _ := e.EpisodeCharacters(episodeID)
	byKey := map[string]Character{}
	for _, c := range existing {
		byKey[NormalizeName(c.Name)] = c
	}
	now := Now()
	var out []Character
	for i, c := range items {
		key := NormalizeName(c.Name)
		if key == "" {
			continue
		}
		if old, ok := byKey[key]; ok {
			if c.Appearance != "" {
				old.Appearance = c.Appearance
			}
			if c.Styling != "" {
				old.Styling = c.Styling
			}
			if c.Role != "" {
				old.Role = c.Role
			}
			_, _ = e.DB.Exec(`UPDATE characters SET role=?, appearance=?, styling=?, updated_at=? WHERE id=?`, old.Role, old.Appearance, old.Styling, now, old.ID)
			e.linkCharacter(episodeID, old.ID)
			out = append(out, old)
			continue
		}
		c.ID = NewID()
		c.DramaID = ep.DramaID
		c.SortOrder = i
		_, err := e.DB.Exec(`INSERT INTO characters(id, drama_id, name, role, appearance, styling, final_prompt, image_hash, sort_order, created_at, updated_at, deleted_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?, '')`, c.ID, c.DramaID, c.Name, c.Role, c.Appearance, c.Styling, c.FinalPrompt, c.ImageHash, c.SortOrder, now, now)
		if err != nil {
			return nil, err
		}
		e.linkCharacter(episodeID, c.ID)
		out = append(out, c)
	}
	_ = e.patchPipeline(episodeID, "extract", "done", "")
	return out, nil
}

func (e *Engine) SaveScenes(episodeID string, items []Scene) ([]Scene, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	existing, _ := e.EpisodeScenes(episodeID)
	byKey := map[string]Scene{}
	for _, s := range existing {
		byKey[NormalizeSceneKey(s.Location, s.TimeOfDay)] = s
	}
	now := Now()
	var out []Scene
	for _, s := range items {
		if strings.TrimSpace(s.Location) == "" {
			continue
		}
		key := NormalizeSceneKey(s.Location, s.TimeOfDay)
		if old, ok := byKey[key]; ok {
			e.linkScene(episodeID, old.ID)
			out = append(out, old)
			continue
		}
		s.ID = NewID()
		s.DramaID = ep.DramaID
		_, err := e.DB.Exec(`INSERT INTO scenes(id, drama_id, location, time_of_day, prompt, lighting, final_prompt, image_hash, created_at, updated_at, deleted_at)
			VALUES(?,?,?,?,?,?,?,?,?,?, '')`, s.ID, s.DramaID, s.Location, s.TimeOfDay, s.Prompt, s.Lighting, s.FinalPrompt, s.ImageHash, now, now)
		if err != nil {
			return nil, err
		}
		e.linkScene(episodeID, s.ID)
		out = append(out, s)
	}
	return out, nil
}

func (e *Engine) SaveProps(episodeID string, items []Prop) ([]Prop, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	if len(items) > 3 {
		items = items[:3]
	}
	existing, _ := e.EpisodeProps(episodeID)
	byKey := map[string]Prop{}
	for _, p := range existing {
		byKey[NormalizeName(p.Name)] = p
	}
	now := Now()
	var out []Prop
	for _, p := range items {
		key := NormalizeName(p.Name)
		if key == "" {
			continue
		}
		if old, ok := byKey[key]; ok {
			e.linkProp(episodeID, old.ID)
			out = append(out, old)
			continue
		}
		p.ID = NewID()
		p.DramaID = ep.DramaID
		_, err := e.DB.Exec(`INSERT INTO props(id, drama_id, name, type, description, final_prompt, image_hash, created_at, updated_at, deleted_at)
			VALUES(?,?,?,?,?,?,?,?,?, '')`, p.ID, p.DramaID, p.Name, p.Type, p.Description, p.FinalPrompt, p.ImageHash, now, now)
		if err != nil {
			return nil, err
		}
		e.linkProp(episodeID, p.ID)
		out = append(out, p)
	}
	return out, nil
}

func (e *Engine) UpdateAsset(kind, id string, fields map[string]any) error {
	now := Now()
	switch kind {
	case "character":
		_, err := e.DB.Exec(`UPDATE characters SET appearance=COALESCE(NULLIF(?,''), appearance), styling=COALESCE(NULLIF(?,''), styling), final_prompt=COALESCE(NULLIF(?,''), final_prompt), image_hash=COALESCE(NULLIF(?,''), image_hash), updated_at=? WHERE id=?`,
			strMap(fields, "appearance"), strMap(fields, "styling"), strMap(fields, "final_prompt"), strMap(fields, "image_hash"), now, id)
		return err
	case "scene":
		_, err := e.DB.Exec(`UPDATE scenes SET prompt=COALESCE(NULLIF(?,''), prompt), lighting=COALESCE(NULLIF(?,''), lighting), final_prompt=COALESCE(NULLIF(?,''), final_prompt), image_hash=COALESCE(NULLIF(?,''), image_hash), updated_at=? WHERE id=?`,
			strMap(fields, "prompt"), strMap(fields, "lighting"), strMap(fields, "final_prompt"), strMap(fields, "image_hash"), now, id)
		return err
	case "prop":
		_, err := e.DB.Exec(`UPDATE props SET description=COALESCE(NULLIF(?,''), description), final_prompt=COALESCE(NULLIF(?,''), final_prompt), image_hash=COALESCE(NULLIF(?,''), image_hash), updated_at=? WHERE id=?`,
			strMap(fields, "description"), strMap(fields, "final_prompt"), strMap(fields, "image_hash"), now, id)
		return err
	}
	return fmt.Errorf("unknown asset")
}

func (e *Engine) SaveFinalPrompt(kind, id, prompt, style string) error {
	prompt = e.PrefixStyle(style, prompt)
	return e.UpdateAsset(kind, id, map[string]any{"final_prompt": prompt})
}

func (e *Engine) ListShots(episodeID string) ([]Shot, error) {
	rows, err := e.DB.Query(`SELECT id, episode_id, scene_id, shot_number, title, shot_type, angle, movement, atmosphere, description, video_prompt, duration, video_hash, poster_hash, status FROM storyboards WHERE episode_id = ? AND deleted_at = '' ORDER BY shot_number`, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Shot
	for rows.Next() {
		var s Shot
		if err := rows.Scan(&s.ID, &s.EpisodeID, &s.SceneID, &s.ShotNumber, &s.Title, &s.ShotType, &s.Angle, &s.Movement, &s.Atmosphere, &s.Description, &s.VideoPrompt, &s.Duration, &s.VideoHash, &s.PosterHash, &s.Status); err != nil {
			return nil, err
		}
		s.CharacterIDs = e.shotChars(s.ID)
		s.PropIDs = e.shotProps(s.ID)
		out = append(out, s)
	}
	if out == nil {
		out = []Shot{}
	}
	return out, nil
}

func (e *Engine) shotChars(id string) []string {
	rows, err := e.DB.Query(`SELECT character_id FROM storyboard_characters WHERE storyboard_id = ?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var x string
		if rows.Scan(&x) == nil {
			out = append(out, x)
		}
	}
	return out
}

func (e *Engine) shotProps(id string) []string {
	rows, err := e.DB.Query(`SELECT prop_id FROM storyboard_props WHERE storyboard_id = ?`, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var x string
		if rows.Scan(&x) == nil {
			out = append(out, x)
		}
	}
	return out
}

func (e *Engine) SaveShots(episodeID string, shots []Shot, replace bool) ([]Shot, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	if replace {
		existing, _ := e.ListShots(episodeID)
		for _, old := range existing {
			_, _ = e.DB.Exec(`DELETE FROM storyboard_characters WHERE storyboard_id = ?`, old.ID)
			_, _ = e.DB.Exec(`DELETE FROM storyboard_props WHERE storyboard_id = ?`, old.ID)
		}
		_, _ = e.DB.Exec(`DELETE FROM storyboards WHERE episode_id = ?`, episodeID)
	}
	now := Now()
	var out []Shot
	for i, s := range shots {
		if s.ShotNumber == 0 {
			s.ShotNumber = i + 1
		}
		s.Duration = ClampSegment(s.Duration, first(s.ShotType, "narrative"))
		s.EpisodeID = episodeID
		s.CharacterIDs = e.resolveIDs("character", ep.DramaID, s.CharacterIDs)
		s.PropIDs = e.resolveIDs("prop", ep.DramaID, s.PropIDs)
		if s.SceneID != "" {
			s.SceneID = first(e.resolveOne("scene", ep.DramaID, s.SceneID), s.SceneID)
			e.linkScene(episodeID, s.SceneID)
		}
		var existingID string
		_ = e.DB.QueryRow(`SELECT id FROM storyboards WHERE episode_id = ? AND shot_number = ?`, episodeID, s.ShotNumber).Scan(&existingID)
		if existingID != "" {
			s.ID = existingID
			_, err := e.DB.Exec(`UPDATE storyboards SET scene_id=?, title=?, shot_type=?, angle=?, movement=?, atmosphere=?, description=?, video_prompt=?, duration=?, updated_at=?, deleted_at='' WHERE id=?`,
				s.SceneID, s.Title, s.ShotType, s.Angle, s.Movement, s.Atmosphere, s.Description, s.VideoPrompt, s.Duration, now, s.ID)
			if err != nil {
				return nil, err
			}
		} else {
			if s.ID == "" {
				s.ID = NewID()
			}
			_, err := e.DB.Exec(`INSERT INTO storyboards(id, episode_id, scene_id, shot_number, title, shot_type, angle, movement, atmosphere, description, video_prompt, duration, video_hash, poster_hash, status, created_at, updated_at, deleted_at)
				VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?, '')`,
				s.ID, episodeID, s.SceneID, s.ShotNumber, s.Title, s.ShotType, s.Angle, s.Movement, s.Atmosphere, s.Description, s.VideoPrompt, s.Duration, s.VideoHash, s.PosterHash, first(s.Status, "pending"), now, now)
			if err != nil {
				return nil, err
			}
		}
		_, _ = e.DB.Exec(`DELETE FROM storyboard_characters WHERE storyboard_id = ?`, s.ID)
		for _, id := range s.CharacterIDs {
			_, _ = e.DB.Exec(`INSERT OR IGNORE INTO storyboard_characters(storyboard_id, character_id) VALUES(?,?)`, s.ID, id)
			e.ensureCharLinked(ep.DramaID, episodeID, id)
		}
		_, _ = e.DB.Exec(`DELETE FROM storyboard_props WHERE storyboard_id = ?`, s.ID)
		for _, id := range s.PropIDs {
			_, _ = e.DB.Exec(`INSERT OR IGNORE INTO storyboard_props(storyboard_id, prop_id) VALUES(?,?)`, s.ID, id)
			e.ensurePropLinked(ep.DramaID, episodeID, id)
		}
		out = append(out, s)
	}
	_ = e.patchPipeline(episodeID, "storyboard", "done", "")
	return out, nil
}

func (e *Engine) UpdateShot(s Shot) (Shot, error) {
	now := Now()
	if s.Duration > 0 {
		s.Duration = ClampSegment(s.Duration, "narrative")
	}
	_, err := e.DB.Exec(`UPDATE storyboards SET title=COALESCE(NULLIF(?,''), title), atmosphere=COALESCE(NULLIF(?,''), atmosphere), description=COALESCE(NULLIF(?,''), description), video_prompt=COALESCE(NULLIF(?,''), video_prompt), duration=CASE WHEN ? > 0 THEN ? ELSE duration END, scene_id=COALESCE(NULLIF(?,''), scene_id), updated_at=? WHERE id=?`,
		s.Title, s.Atmosphere, s.Description, s.VideoPrompt, s.Duration, s.Duration, s.SceneID, now, s.ID)
	if err != nil {
		return s, err
	}
	if s.CharacterIDs != nil {
		_, _ = e.DB.Exec(`DELETE FROM storyboard_characters WHERE storyboard_id = ?`, s.ID)
		for _, id := range s.CharacterIDs {
			_, _ = e.DB.Exec(`INSERT OR IGNORE INTO storyboard_characters(storyboard_id, character_id) VALUES(?,?)`, s.ID, id)
		}
	}
	if s.PropIDs != nil {
		_, _ = e.DB.Exec(`DELETE FROM storyboard_props WHERE storyboard_id = ?`, s.ID)
		for _, id := range s.PropIDs {
			_, _ = e.DB.Exec(`INSERT OR IGNORE INTO storyboard_props(storyboard_id, prop_id) VALUES(?,?)`, s.ID, id)
		}
	}
	return s, nil
}

func (e *Engine) GetShot(id string) (Shot, error) {
	var s Shot
	err := e.DB.QueryRow(`SELECT id, episode_id, scene_id, shot_number, title, shot_type, angle, movement, atmosphere, description, video_prompt, duration, video_hash, poster_hash, status FROM storyboards WHERE id = ?`, id).
		Scan(&s.ID, &s.EpisodeID, &s.SceneID, &s.ShotNumber, &s.Title, &s.ShotType, &s.Angle, &s.Movement, &s.Atmosphere, &s.Description, &s.VideoPrompt, &s.Duration, &s.VideoHash, &s.PosterHash, &s.Status)
	if err != nil {
		return s, fmt.Errorf("shot not found")
	}
	s.CharacterIDs = e.shotChars(s.ID)
	s.PropIDs = e.shotProps(s.ID)
	return s, nil
}

func (e *Engine) ensureCharLinked(dramaID, episodeID, characterID string) {
	var did string
	if e.DB.QueryRow(`SELECT drama_id FROM characters WHERE id = ?`, characterID).Scan(&did) != nil || did != dramaID {
		return
	}
	e.linkCharacter(episodeID, characterID)
}

func (e *Engine) ensurePropLinked(dramaID, episodeID, propID string) {
	var did string
	if e.DB.QueryRow(`SELECT drama_id FROM props WHERE id = ?`, propID).Scan(&did) != nil || did != dramaID {
		return
	}
	e.linkProp(episodeID, propID)
}

func (e *Engine) ShotRefs(s Shot) ([]AssetRef, error) {
	var refs []AssetRef
	if s.SceneID != "" {
		var name, hash string
		if e.DB.QueryRow(`SELECT location, image_hash FROM scenes WHERE id = ?`, s.SceneID).Scan(&name, &hash) == nil && hash != "" {
			if u, err := e.RefDataURL(hash); err == nil {
				refs = append(refs, AssetRef{Name: name, URL: u})
			}
		}
	}
	for _, id := range s.CharacterIDs {
		var name, hash string
		if e.DB.QueryRow(`SELECT name, image_hash FROM characters WHERE id = ?`, id).Scan(&name, &hash) == nil && hash != "" {
			if u, err := e.RefDataURL(hash); err == nil {
				refs = append(refs, AssetRef{Name: name, URL: u})
			}
		}
	}
	for _, id := range s.PropIDs {
		var name, hash string
		if e.DB.QueryRow(`SELECT name, image_hash FROM props WHERE id = ?`, id).Scan(&name, &hash) == nil && hash != "" {
			if u, err := e.RefDataURL(hash); err == nil {
				refs = append(refs, AssetRef{Name: name, URL: u})
			}
		}
	}
	return refs, nil
}

func (e *Engine) GenerateShot(shotID string) (Job, error) {
	s, err := e.GetShot(shotID)
	if err != nil {
		return Job{}, err
	}
	ep, err := e.GetEpisode(s.EpisodeID)
	if err != nil {
		return Job{}, err
	}
	d, err := e.GetDrama(ep.DramaID)
	if err != nil {
		return Job{}, err
	}
	refs, _ := e.ShotRefs(s)
	p, _ := e.GetProvider(ep.VideoProviderID)
	prompt, urls := ResolvePromptRefs(s.VideoPrompt, refs, strings.EqualFold(p.Provider, "aliyun"))
	if prompt == "" && len(urls) == 0 {
		return Job{}, fmt.Errorf("shot needs a prompt or reference stills")
	}
	lim := 9
	if strings.EqualFold(p.Provider, "aliyun") {
		lim = 10
	}
	if len(urls) > lim {
		urls = urls[:lim]
	}
	j, err := e.EnqueueVideo(EnqueueVideo{
		Prompt: prompt, ProviderID: ep.VideoProviderID, Duration: s.Duration,
		AspectRatio: d.AspectRatio, Resolution: ep.Resolution, Audio: true, Refs: urls,
		DramaID: d.ID, EpisodeID: ep.ID, ShotID: s.ID,
	})
	if err == nil {
		_, _ = e.DB.Exec(`UPDATE storyboards SET status = 'generating', updated_at = ? WHERE id = ?`, Now(), s.ID)
	}
	return j, err
}

func (e *Engine) GenerateAsset(kind, id, episodeID string) (Job, error) {
	var prompt, dramaID string
	in := EnqueueImage{Size: "1920x1080"}
	switch kind {
	case "character":
		var c Character
		err := e.DB.QueryRow(`SELECT id, drama_id, final_prompt, appearance, styling FROM characters WHERE id = ?`, id).
			Scan(&c.ID, &c.DramaID, &c.FinalPrompt, &c.Appearance, &c.Styling)
		if err != nil {
			return Job{}, err
		}
		d, _ := e.GetDrama(c.DramaID)
		prompt = c.FinalPrompt
		if prompt == "" {
			prompt = e.PrefixStyle(d.Style, "character design sheet, left a tight front portrait, right a three-view turnaround, "+c.Appearance+". Costume: "+c.Styling+". Empty background.")
		}
		in.CharacterID = id
		dramaID = c.DramaID
	case "scene":
		var s Scene
		err := e.DB.QueryRow(`SELECT id, drama_id, final_prompt, prompt, lighting, location FROM scenes WHERE id = ?`, id).
			Scan(&s.ID, &s.DramaID, &s.FinalPrompt, &s.Prompt, &s.Lighting, &s.Location)
		if err != nil {
			return Job{}, err
		}
		d, _ := e.GetDrama(s.DramaID)
		prompt = s.FinalPrompt
		if prompt == "" {
			prompt = e.PrefixStyle(d.Style, "establishing wide shot, no people, "+s.Location+". "+s.Prompt+". Light: "+s.Lighting+".")
		}
		in.SceneID = id
		dramaID = s.DramaID
	case "prop":
		var p Prop
		err := e.DB.QueryRow(`SELECT id, drama_id, final_prompt, description, name FROM props WHERE id = ?`, id).
			Scan(&p.ID, &p.DramaID, &p.FinalPrompt, &p.Description, &p.Name)
		if err != nil {
			return Job{}, err
		}
		d, _ := e.GetDrama(p.DramaID)
		prompt = p.FinalPrompt
		if prompt == "" {
			prompt = e.PrefixStyle(d.Style, "product still of "+p.Name+" on a white seamless background, "+p.Description+", no hands, no scene.")
		}
		in.PropID = id
		dramaID = p.DramaID
	default:
		return Job{}, fmt.Errorf("unknown asset")
	}
	in.Prompt = prompt
	in.DramaID = dramaID
	if episodeID == "" {
		_ = e.DB.QueryRow(`SELECT id FROM episodes WHERE drama_id = ? AND deleted_at = '' ORDER BY episode_number LIMIT 1`, dramaID).Scan(&episodeID)
	}
	in.EpisodeID = episodeID
	if episodeID != "" {
		if ep, err := e.GetEpisode(episodeID); err == nil {
			in.ProviderID = ep.ImageProviderID
		}
	}
	return e.EnqueueImage(in)
}

func (e *Engine) MergeEpisode(episodeID string, shotIDs []string) (Job, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return Job{}, err
	}
	shots, err := e.ListShots(episodeID)
	if err != nil {
		return Job{}, err
	}
	allow := map[string]bool{}
	for _, id := range shotIDs {
		allow[id] = true
	}
	var hashes []string
	var used []string
	for _, s := range shots {
		if len(allow) > 0 && !allow[s.ID] {
			continue
		}
		if s.VideoHash == "" {
			continue
		}
		hashes = append(hashes, s.VideoHash)
		used = append(used, s.ID)
	}
	if len(hashes) == 0 {
		return Job{}, fmt.Errorf("no generated clips to stitch")
	}
	return e.EnqueueMerge(episodeID, ep.DramaID, hashes, used)
}

func strMap(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k]; ok {
		return fmt.Sprint(v)
	}
	return ""
}
