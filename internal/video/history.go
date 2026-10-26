// history.go 列出画布、短剧和短片工程，并按会话找回当前工程。

package video

// ProjectHistory 拼出画布、短剧和短片工程。画布和短剧最多各取现有列表；短剧没有绑到集时用该剧的第一集。
func (e *Engine) ProjectHistory() ([]map[string]any, error) {
	canvasSess := e.latestCanvasSessions()
	dramaSess := e.latestDramaSessions()
	rows, _, err := e.listCanvasProjectRows("", "", 1, 200)
	if err != nil {
		return nil, err
	}
	dramas, err := e.ListDramas()
	if err != nil {
		return nil, err
	}
	studios := e.ListStudioProjects()
	out := make([]map[string]any, 0, len(rows)+len(dramas)+len(studios))
	for _, r := range rows {
		item := map[string]any{
			"kind":       "canvas",
			"id":         r.ID,
			"title":      r.Title,
			"updated_at": r.UpdatedAt,
			"session_id": canvasSess[r.ID],
		}
		out = append(out, item)
	}
	for _, d := range dramas {
		hit := dramaSess[d.ID]
		ep := hit.EpisodeID
		if ep == "" {
			ep = e.firstEpisodeID(d.ID)
		}
		out = append(out, map[string]any{
			"kind":       "drama",
			"id":         d.ID,
			"title":      d.Title,
			"updated_at": d.UpdatedAt,
			"session_id": hit.SessionID,
			"episode_id": ep,
		})
	}
	studioSess := e.latestStudioSessions()
	for _, p := range studios {
		out = append(out, map[string]any{
			"kind":       "studio",
			"id":         p.ID,
			"title":      p.Title,
			"updated_at": p.UpdatedAt,
			"session_id": studioSess[p.ID],
			"status":     p.Status,
			"gate":       p.Gate,
		})
	}
	return out, nil
}

// SessionProject 按会话绑定填回画布、短片和短剧。已有 kind 时不再用后面的绑定覆盖种类和标题。
func (e *Engine) SessionProject(sessionID string) map[string]any {
	out := map[string]any{"session_id": sessionID}
	if cid, ok := e.CanvasSessionBind(sessionID); ok {
		if row, err := e.getCanvasProject(cid); err == nil {
			out["kind"] = "canvas"
			out["id"] = row.ID
			out["canvas_id"] = row.ID
			out["title"] = row.Title
		}
	}
	if pid, ok := e.StudioSessionBind(sessionID); ok {
		if p, err := e.GetStudioProject(pid); err == nil {
			out["studio_id"] = p.ID
			out["gate"] = p.Gate
			out["status"] = p.Status
			if out["kind"] == nil {
				out["kind"] = "studio"
				out["id"] = p.ID
				out["title"] = p.Title
			}
		}
	}
	if b, ok := e.SessionBind(sessionID); ok {
		out["drama_id"] = b.DramaID
		out["episode_id"] = b.EpisodeID
		if d, err := e.GetDrama(b.DramaID); err == nil {
			out["drama_title"] = d.Title
			if out["kind"] == nil {
				out["kind"] = "drama"
				out["id"] = d.ID
				out["title"] = d.Title
			}
		}
	}
	return out
}

// dramaSessionHit 记下一部短剧最近一次绑定的会话和集。
type dramaSessionHit struct {
	SessionID string
	EpisodeID string
}

// latestCanvasSessions 为每个画布保留绑定表里第一次出现的会话。引擎为空时返回空 map。
func (e *Engine) latestCanvasSessions() map[string]string {
	out := map[string]string{}
	if e == nil {
		return out
	}
	for sessionID, canvasID := range e.loadCanvasBinds() {
		if canvasID == "" {
			continue
		}
		if _, ok := out[canvasID]; !ok {
			out[canvasID] = sessionID
		}
	}
	return out
}

// latestDramaSessions 为每部短剧保留绑定表里第一次出现的会话和集。
func (e *Engine) latestDramaSessions() map[string]dramaSessionHit {
	out := map[string]dramaSessionHit{}
	if e == nil {
		return out
	}
	for _, b := range e.loadBinds() {
		if b.DramaID == "" {
			continue
		}
		if _, ok := out[b.DramaID]; !ok {
			out[b.DramaID] = dramaSessionHit{SessionID: b.SessionID, EpisodeID: b.EpisodeID}
		}
	}
	return out
}

// LatestStudioSession 返回短片工程第一次出现的会话。引擎或工程标识为空时返回空串。
func (e *Engine) LatestStudioSession(projectID string) string {
	if e == nil || projectID == "" {
		return ""
	}
	return e.latestStudioSessions()[projectID]
}

// latestStudioSessions 为每个短片工程保留绑定表里第一次出现的会话。
func (e *Engine) latestStudioSessions() map[string]string {
	out := map[string]string{}
	if e == nil {
		return out
	}
	for sessionID, projectID := range e.loadStudioBinds() {
		if projectID == "" {
			continue
		}
		if _, ok := out[projectID]; !ok {
			out[projectID] = sessionID
		}
	}
	return out
}

// firstEpisodeID 返回该剧列表里的第一集标识。没有集时返回空串。
func (e *Engine) firstEpisodeID(dramaID string) string {
	if e == nil || dramaID == "" {
		return ""
	}
	eps, err := e.ListEpisodes(dramaID)
	if err != nil || len(eps) == 0 {
		return ""
	}
	return eps[0].ID
}
