package video

import (
	"fmt"
	"sort"
	"strings"
)

func (e *Engine) EpisodeCharacters(episodeID string) ([]Character, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	rec, _ := getDoc[episodeRec](e, colEpisodes, episodeID)
	linked := map[string]bool{}
	for _, id := range rec.CharacterIDs {
		linked[id] = true
	}
	var out []Character
	for _, c := range loadCol[characterRec](e, colCharacters) {
		if c.DramaID != ep.DramaID || c.DeletedAt != "" {
			continue
		}
		item := c.Character
		item.Linked = linked[c.ID]
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Name < out[j].Name
	})
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
	rec, _ := getDoc[episodeRec](e, colEpisodes, episodeID)
	linked := map[string]bool{}
	for _, id := range rec.SceneIDs {
		linked[id] = true
	}
	var out []Scene
	for _, s := range loadCol[sceneRec](e, colScenes) {
		if s.DramaID != ep.DramaID || s.DeletedAt != "" {
			continue
		}
		item := s.Scene
		item.Linked = linked[s.ID]
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Location < out[j].Location })
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
	rec, _ := getDoc[episodeRec](e, colEpisodes, episodeID)
	linked := map[string]bool{}
	for _, id := range rec.PropIDs {
		linked[id] = true
	}
	var out []Prop
	for _, p := range loadCol[propRec](e, colProps) {
		if p.DramaID != ep.DramaID || p.DeletedAt != "" {
			continue
		}
		item := p.Prop
		item.Linked = linked[p.ID]
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if out == nil {
		out = []Prop{}
	}
	return out, nil
}

func (e *Engine) linkCharacter(episodeID, characterID string) {
	rec, err := getDoc[episodeRec](e, colEpisodes, episodeID)
	if err != nil {
		return
	}
	rec.CharacterIDs = addID(rec.CharacterIDs, characterID)
	_ = e.putDoc(colEpisodes, rec.ID, rec)
}
func (e *Engine) linkScene(episodeID, sceneID string) {
	rec, err := getDoc[episodeRec](e, colEpisodes, episodeID)
	if err != nil {
		return
	}
	rec.SceneIDs = addID(rec.SceneIDs, sceneID)
	_ = e.putDoc(colEpisodes, rec.ID, rec)
}
func (e *Engine) linkProp(episodeID, propID string) {
	rec, err := getDoc[episodeRec](e, colEpisodes, episodeID)
	if err != nil {
		return
	}
	rec.PropIDs = addID(rec.PropIDs, propID)
	_ = e.putDoc(colEpisodes, rec.ID, rec)
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
			rec, _ := getDoc[characterRec](e, colCharacters, old.ID)
			rec.Character = old
			rec.UpdatedAt = now
			_ = e.putDoc(colCharacters, old.ID, rec)
			e.linkCharacter(episodeID, old.ID)
			out = append(out, old)
			continue
		}
		c.ID = NewID()
		c.DramaID = ep.DramaID
		c.SortOrder = i
		rec := characterRec{Character: c, CreatedAt: now, UpdatedAt: now}
		if err := e.putDoc(colCharacters, c.ID, rec); err != nil {
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
		rec := sceneRec{Scene: s, CreatedAt: now, UpdatedAt: now}
		if err := e.putDoc(colScenes, s.ID, rec); err != nil {
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
		rec := propRec{Prop: p, CreatedAt: now, UpdatedAt: now}
		if err := e.putDoc(colProps, p.ID, rec); err != nil {
			return nil, err
		}
		e.linkProp(episodeID, p.ID)
		out = append(out, p)
	}
	return out, nil
}

func (e *Engine) UpdateAsset(kind, id string, fields map[string]any) error {
	now := Now()
	has := func(k string) bool { _, ok := fields[k]; return ok }
	set := func(k string) string { return strMap(fields, k) }
	switch kind {
	case "character":
		rec, err := getDoc[characterRec](e, colCharacters, id)
		if err != nil {
			return err
		}
		if has("name") {
			rec.Name = set("name")
		}
		if has("role") {
			rec.Role = set("role")
		}
		if has("appearance") {
			rec.Appearance = set("appearance")
		}
		if has("styling") {
			rec.Styling = set("styling")
		}
		if has("final_prompt") {
			rec.FinalPrompt = set("final_prompt")
		}
		if has("image_hash") {
			rec.ImageHash = set("image_hash")
		}
		rec.UpdatedAt = now
		return e.putDoc(colCharacters, id, rec)
	case "scene":
		rec, err := getDoc[sceneRec](e, colScenes, id)
		if err != nil {
			return err
		}
		if has("location") {
			rec.Location = set("location")
		}
		if has("time_of_day") {
			rec.TimeOfDay = set("time_of_day")
		}
		if has("prompt") {
			rec.Prompt = set("prompt")
		}
		if has("lighting") {
			rec.Lighting = set("lighting")
		}
		if has("final_prompt") {
			rec.FinalPrompt = set("final_prompt")
		}
		if has("image_hash") {
			rec.ImageHash = set("image_hash")
		}
		rec.UpdatedAt = now
		return e.putDoc(colScenes, id, rec)
	case "prop":
		rec, err := getDoc[propRec](e, colProps, id)
		if err != nil {
			return err
		}
		if has("name") {
			rec.Name = set("name")
		}
		if has("type") {
			rec.Type = set("type")
		}
		if has("description") {
			rec.Description = set("description")
		}
		if has("final_prompt") {
			rec.FinalPrompt = set("final_prompt")
		}
		if has("image_hash") {
			rec.ImageHash = set("image_hash")
		}
		rec.UpdatedAt = now
		return e.putDoc(colProps, id, rec)
	}
	return fmt.Errorf("unknown asset")
}

func (e *Engine) SaveFinalPrompt(kind, id, prompt, style string) error {
	prompt = e.PrefixStyle(style, prompt)
	return e.UpdateAsset(kind, id, map[string]any{"final_prompt": prompt})
}

func (e *Engine) ListShots(episodeID string) ([]Shot, error) {
	var out []Shot
	for _, rec := range loadCol[shotRec](e, colShots) {
		if rec.EpisodeID != episodeID || rec.DeletedAt != "" {
			continue
		}
		out = append(out, rec.Shot)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ShotNumber < out[j].ShotNumber })
	if out == nil {
		out = []Shot{}
	}
	return out, nil
}

func (e *Engine) SaveShots(episodeID string, shots []Shot, replace bool) ([]Shot, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	if replace {
		for _, old := range loadCol[shotRec](e, colShots) {
			if old.EpisodeID == episodeID {
				_ = e.delDoc(colShots, old.ID)
			}
		}
	}
	now := Now()
	var out []Shot
	existingByNum := map[int]shotRec{}
	if !replace {
		for _, rec := range loadCol[shotRec](e, colShots) {
			if rec.EpisodeID == episodeID && rec.DeletedAt == "" {
				existingByNum[rec.ShotNumber] = rec
			}
		}
	}
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
		if old, ok := existingByNum[s.ShotNumber]; ok {
			s.ID = old.ID
			if s.VideoHash == "" {
				s.VideoHash = old.VideoHash
			}
			if s.PosterHash == "" {
				s.PosterHash = old.PosterHash
			}
			if s.Status == "" {
				s.Status = old.Status
			}
			rec := old
			rec.Shot = s
			rec.UpdatedAt = now
			rec.DeletedAt = ""
			if err := e.putDoc(colShots, s.ID, rec); err != nil {
				return nil, err
			}
		} else {
			if s.ID == "" {
				s.ID = NewID()
			}
			if s.Status == "" {
				s.Status = "pending"
			}
			rec := shotRec{Shot: s, CreatedAt: now, UpdatedAt: now}
			if err := e.putDoc(colShots, s.ID, rec); err != nil {
				return nil, err
			}
		}
		for _, id := range s.CharacterIDs {
			e.ensureCharLinked(ep.DramaID, episodeID, id)
		}
		for _, id := range s.PropIDs {
			e.ensurePropLinked(ep.DramaID, episodeID, id)
		}
		out = append(out, s)
	}
	_ = e.patchPipeline(episodeID, "storyboard", "done", "")
	return out, nil
}

func (e *Engine) UpdateShot(s Shot) (Shot, error) {
	fields := map[string]any{
		"title": s.Title, "atmosphere": s.Atmosphere, "description": s.Description,
		"video_prompt": s.VideoPrompt, "duration": s.Duration, "scene_id": s.SceneID,
	}
	if s.CharacterIDs != nil {
		fields["character_ids"] = s.CharacterIDs
	}
	if s.PropIDs != nil {
		fields["prop_ids"] = s.PropIDs
	}
	return e.PatchShot(s.ID, fields)
}

func (e *Engine) PatchShot(id string, fields map[string]any) (Shot, error) {
	cur, err := e.GetShot(id)
	if err != nil {
		return cur, err
	}
	has := func(k string) bool { _, ok := fields[k]; return ok }
	if has("title") {
		cur.Title = strMap(fields, "title")
	}
	if has("atmosphere") {
		cur.Atmosphere = strMap(fields, "atmosphere")
	}
	if has("description") {
		cur.Description = strMap(fields, "description")
	}
	if has("video_prompt") {
		cur.VideoPrompt = strMap(fields, "video_prompt")
	}
	if has("scene_id") {
		cur.SceneID = strMap(fields, "scene_id")
	}
	if has("duration") {
		n := intArg(fields["duration"])
		if n > 0 {
			provider := "volcengine"
			if ep, err := e.GetEpisode(cur.EpisodeID); err == nil {
				if p, err := e.GetProvider(ep.VideoProviderID); err == nil {
					provider = p.Provider
				}
			}
			cur.Duration = ClampProviderDuration(n, provider)
		}
	}
	if has("character_ids") {
		cur.CharacterIDs = decodeStringSlice(fields["character_ids"])
	}
	if has("prop_ids") {
		cur.PropIDs = decodeStringSlice(fields["prop_ids"])
	}
	rec, err := getDoc[shotRec](e, colShots, cur.ID)
	if err != nil {
		return cur, err
	}
	rec.Shot = cur
	rec.UpdatedAt = Now()
	if err := e.putDoc(colShots, cur.ID, rec); err != nil {
		return cur, err
	}
	return cur, nil
}

func (e *Engine) GetShot(id string) (Shot, error) {
	rec, err := getDoc[shotRec](e, colShots, id)
	if err != nil {
		return Shot{}, fmt.Errorf("shot not found")
	}
	return rec.Shot, nil
}

func (e *Engine) ensureCharLinked(dramaID, episodeID, characterID string) {
	rec, err := getDoc[characterRec](e, colCharacters, characterID)
	if err != nil || rec.DramaID != dramaID {
		return
	}
	e.linkCharacter(episodeID, characterID)
}

func (e *Engine) ensurePropLinked(dramaID, episodeID, propID string) {
	rec, err := getDoc[propRec](e, colProps, propID)
	if err != nil || rec.DramaID != dramaID {
		return
	}
	e.linkProp(episodeID, propID)
}

func (e *Engine) ShotRefs(s Shot) ([]AssetRef, error) {
	var refs []AssetRef
	if s.SceneID != "" {
		if rec, err := getDoc[sceneRec](e, colScenes, s.SceneID); err == nil && rec.ImageHash != "" {
			if u, err := e.RefDataURL(rec.ImageHash); err == nil {
				refs = append(refs, AssetRef{Name: rec.Location, URL: u})
			}
		}
	}
	for _, id := range s.CharacterIDs {
		if rec, err := getDoc[characterRec](e, colCharacters, id); err == nil && rec.ImageHash != "" {
			if u, err := e.RefDataURL(rec.ImageHash); err == nil {
				refs = append(refs, AssetRef{Name: rec.Name, URL: u})
			}
		}
	}
	for _, id := range s.PropIDs {
		if rec, err := getDoc[propRec](e, colProps, id); err == nil && rec.ImageHash != "" {
			if u, err := e.RefDataURL(rec.ImageHash); err == nil {
				refs = append(refs, AssetRef{Name: rec.Name, URL: u})
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
		Prompt: prompt, ProviderID: ep.VideoProviderID, Model: ep.VideoModel, Duration: s.Duration,
		AspectRatio: d.AspectRatio, Resolution: ep.Resolution, Audio: true, Refs: urls,
		DramaID: d.ID, EpisodeID: ep.ID, ShotID: s.ID,
	})
	if err == nil {
		if rec, getErr := getDoc[shotRec](e, colShots, s.ID); getErr == nil {
			rec.Status = "generating"
			rec.UpdatedAt = Now()
			_ = e.putDoc(colShots, s.ID, rec)
		}
		_ = e.patchPipeline(ep.ID, "gen", "running", "")
	}
	return j, err
}

func (e *Engine) GenerateAsset(kind, id, episodeID string) (Job, error) {
	var prompt, dramaID string
	in := EnqueueImage{}
	switch kind {
	case "character":
		c, err := getDoc[characterRec](e, colCharacters, id)
		if err != nil {
			return Job{}, err
		}
		if IsNarrator(c.Name, c.Role) {
			return Job{}, fmt.Errorf("narrator has no still")
		}
		d, _ := e.GetDrama(c.DramaID)
		prompt = strings.TrimSpace(c.FinalPrompt)
		if prompt == "" {
			prompt = e.stillFallback("character", d, c.Appearance, c.Styling, "", "", c.Name, "")
			_ = e.UpdateAsset("character", id, map[string]any{"final_prompt": prompt})
		}
		in.CharacterID = id
		in.Size = ImageSizeFor("character", d.AspectRatio)
		in.AspectRatio = d.AspectRatio
		dramaID = c.DramaID
	case "scene":
		s, err := getDoc[sceneRec](e, colScenes, id)
		if err != nil {
			return Job{}, err
		}
		d, _ := e.GetDrama(s.DramaID)
		prompt = strings.TrimSpace(s.FinalPrompt)
		if prompt == "" {
			prompt = e.stillFallback("scene", d, "", "", s.Location, s.Lighting, "", s.Prompt)
			_ = e.UpdateAsset("scene", id, map[string]any{"final_prompt": prompt})
		}
		in.SceneID = id
		in.Size = ImageSizeFor("scene", d.AspectRatio)
		in.AspectRatio = d.AspectRatio
		dramaID = s.DramaID
	case "prop":
		p, err := getDoc[propRec](e, colProps, id)
		if err != nil {
			return Job{}, err
		}
		d, _ := e.GetDrama(p.DramaID)
		prompt = strings.TrimSpace(p.FinalPrompt)
		if prompt == "" {
			prompt = e.stillFallback("prop", d, "", "", "", "", p.Name, p.Description)
			_ = e.UpdateAsset("prop", id, map[string]any{"final_prompt": prompt})
		}
		in.PropID = id
		in.Size = ImageSizeFor("prop", d.AspectRatio)
		in.AspectRatio = d.AspectRatio
		dramaID = p.DramaID
	default:
		return Job{}, fmt.Errorf("unknown asset")
	}
	in.Prompt = prompt
	in.DramaID = dramaID
	if episodeID == "" {
		episodeID = e.firstEpisodeID(dramaID)
	}
	in.EpisodeID = episodeID
	if episodeID != "" {
		if ep, err := e.GetEpisode(episodeID); err == nil {
			in.ProviderID = ep.ImageProviderID
			in.Model = ep.ImageModel
		}
	}
	j, err := e.EnqueueImage(in)
	if err == nil && episodeID != "" {
		_ = e.patchPipeline(episodeID, "assets", "running", "")
	}
	return j, err
}

func (e *Engine) stillFallback(kind string, d Drama, appearance, styling, location, lighting, name, extra string) string {
	zh := chineseContent(e.ContentLanguage())
	var body string
	switch kind {
	case "character":
		if zh {
			body = "角色设定图，左为正脸特写，右为三视图，空背景。" + appearance + "。服装：" + styling + "。"
		} else {
			body = "character design sheet, left a tight front portrait, right a three-view turnaround, " + appearance + ". Costume: " + styling + ". Empty background."
		}
	case "scene":
		if zh {
			body = "空镜建立镜头，画面中不能出现任何人。" + location + "。" + extra + "。光线：" + lighting + "。"
		} else {
			body = "establishing wide shot, no people, " + location + ". " + extra + ". Light: " + lighting + "."
		}
	default:
		if zh {
			body = "白底单品质感静物，" + name + "，" + extra + "，无手，无场景。"
		} else {
			body = "product still of " + name + " on a white seamless background, " + extra + ", no hands, no scene."
		}
	}
	return e.PrefixStyle(d.Style, body)
}

func (e *Engine) CreateAsset(kind, episodeID string, fields map[string]any) (any, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return nil, err
	}
	now := Now()
	switch kind {
	case "character":
		c := Character{
			ID: NewID(), DramaID: ep.DramaID, Name: strMap(fields, "name"), Role: strMap(fields, "role"),
			Appearance: strMap(fields, "appearance"), Styling: strMap(fields, "styling"),
			FinalPrompt: strMap(fields, "final_prompt"),
		}
		if strings.TrimSpace(c.Name) == "" {
			return nil, fmt.Errorf("name required")
		}
		if err := e.putDoc(colCharacters, c.ID, characterRec{Character: c, CreatedAt: now, UpdatedAt: now}); err != nil {
			return nil, err
		}
		e.linkCharacter(episodeID, c.ID)
		c.Linked = true
		return c, nil
	case "scene":
		s := Scene{
			ID: NewID(), DramaID: ep.DramaID, Location: strMap(fields, "location"), TimeOfDay: strMap(fields, "time_of_day"),
			Prompt: strMap(fields, "prompt"), Lighting: strMap(fields, "lighting"), FinalPrompt: strMap(fields, "final_prompt"),
		}
		if strings.TrimSpace(s.Location) == "" {
			return nil, fmt.Errorf("location required")
		}
		if err := e.putDoc(colScenes, s.ID, sceneRec{Scene: s, CreatedAt: now, UpdatedAt: now}); err != nil {
			return nil, err
		}
		e.linkScene(episodeID, s.ID)
		s.Linked = true
		return s, nil
	case "prop":
		p := Prop{
			ID: NewID(), DramaID: ep.DramaID, Name: strMap(fields, "name"), Type: strMap(fields, "type"),
			Description: strMap(fields, "description"), FinalPrompt: strMap(fields, "final_prompt"),
		}
		if strings.TrimSpace(p.Name) == "" {
			return nil, fmt.Errorf("name required")
		}
		if err := e.putDoc(colProps, p.ID, propRec{Prop: p, CreatedAt: now, UpdatedAt: now}); err != nil {
			return nil, err
		}
		e.linkProp(episodeID, p.ID)
		p.Linked = true
		return p, nil
	}
	return nil, fmt.Errorf("unknown asset")
}

func (e *Engine) DeleteAsset(kind, id string) error {
	now := Now()
	switch kind {
	case "character":
		rec, err := getDoc[characterRec](e, colCharacters, id)
		if err != nil {
			return err
		}
		rec.DeletedAt = now
		rec.UpdatedAt = now
		if err := e.putDoc(colCharacters, id, rec); err != nil {
			return err
		}
		e.unlinkAssetFromEpisodes(id, "character")
		for _, s := range loadCol[shotRec](e, colShots) {
			if containsID(s.CharacterIDs, id) {
				s.CharacterIDs = removeID(s.CharacterIDs, id)
				_ = e.putDoc(colShots, s.ID, s)
			}
		}
		return nil
	case "scene":
		rec, err := getDoc[sceneRec](e, colScenes, id)
		if err != nil {
			return err
		}
		rec.DeletedAt = now
		rec.UpdatedAt = now
		if err := e.putDoc(colScenes, id, rec); err != nil {
			return err
		}
		e.unlinkAssetFromEpisodes(id, "scene")
		for _, s := range loadCol[shotRec](e, colShots) {
			if s.SceneID == id {
				s.SceneID = ""
				_ = e.putDoc(colShots, s.ID, s)
			}
		}
		return nil
	case "prop":
		rec, err := getDoc[propRec](e, colProps, id)
		if err != nil {
			return err
		}
		rec.DeletedAt = now
		rec.UpdatedAt = now
		if err := e.putDoc(colProps, id, rec); err != nil {
			return err
		}
		e.unlinkAssetFromEpisodes(id, "prop")
		for _, s := range loadCol[shotRec](e, colShots) {
			if containsID(s.PropIDs, id) {
				s.PropIDs = removeID(s.PropIDs, id)
				_ = e.putDoc(colShots, s.ID, s)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown asset")
}

func (e *Engine) unlinkAssetFromEpisodes(id, kind string) {
	for _, ep := range loadCol[episodeRec](e, colEpisodes) {
		changed := false
		switch kind {
		case "character":
			n := removeID(ep.CharacterIDs, id)
			if len(n) != len(ep.CharacterIDs) {
				ep.CharacterIDs = n
				changed = true
			}
		case "scene":
			n := removeID(ep.SceneIDs, id)
			if len(n) != len(ep.SceneIDs) {
				ep.SceneIDs = n
				changed = true
			}
		case "prop":
			n := removeID(ep.PropIDs, id)
			if len(n) != len(ep.PropIDs) {
				ep.PropIDs = n
				changed = true
			}
		}
		if changed {
			_ = e.putDoc(colEpisodes, ep.ID, ep)
		}
	}
}

func (e *Engine) ApplyJob(id string) (Job, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return j, err
	}
	if j.Status != "succeeded" || j.ResultHash == "" {
		return j, fmt.Errorf("job has no media")
	}
	return j, e.writeBack(j)
}

func decodeStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		var out []string
		for _, x := range t {
			s := strings.TrimSpace(fmt.Sprint(x))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
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
