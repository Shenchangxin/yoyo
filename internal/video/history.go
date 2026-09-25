package video

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
	out := make([]map[string]any, 0, len(rows)+len(dramas))
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
	return out, nil
}

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

type dramaSessionHit struct {
	SessionID string
	EpisodeID string
}

func (e *Engine) latestCanvasSessions() map[string]string {
	out := map[string]string{}
	if e == nil || e.DB == nil {
		return out
	}
	rows, err := e.DB.Query(`SELECT canvas_id, session_id FROM canvas_session_binds ORDER BY updated_at DESC`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var canvasID, sessionID string
		if err := rows.Scan(&canvasID, &sessionID); err != nil {
			continue
		}
		if _, ok := out[canvasID]; !ok {
			out[canvasID] = sessionID
		}
	}
	return out
}

func (e *Engine) latestDramaSessions() map[string]dramaSessionHit {
	out := map[string]dramaSessionHit{}
	if e == nil || e.DB == nil {
		return out
	}
	rows, err := e.DB.Query(`SELECT drama_id, episode_id, session_id FROM session_binds ORDER BY updated_at DESC`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var dramaID, episodeID, sessionID string
		if err := rows.Scan(&dramaID, &episodeID, &sessionID); err != nil {
			continue
		}
		if _, ok := out[dramaID]; !ok {
			out[dramaID] = dramaSessionHit{SessionID: sessionID, EpisodeID: episodeID}
		}
	}
	return out
}

func (e *Engine) firstEpisodeID(dramaID string) string {
	var id string
	if e == nil || e.DB == nil || dramaID == "" {
		return ""
	}
	_ = e.DB.QueryRow(`SELECT id FROM episodes WHERE drama_id = ? AND deleted_at = '' ORDER BY episode_number LIMIT 1`, dramaID).Scan(&id)
	return id
}
