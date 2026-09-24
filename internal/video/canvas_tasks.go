package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (e *Engine) createCanvasTask(req map[string]any) (map[string]any, error) {
	typ := strings.ToLower(firstNonEmpty(strAnyMap(req, "type"), "image"))
	prompt := strAnyMap(req, "prompt")
	projectID := strAnyMap(req, "projectId", "project_id")
	provider := strAnyMap(req, "provider")
	model := strAnyMap(req, "model")
	operation := strAnyMap(req, "operation")
	input := asMap(req["input"])
	now := Now()
	params := JobParams{
		CanvasProjectID:   projectID,
		Mode:              typ,
		ChannelID:         firstNonEmpty(provider, strAnyMap(input, "channelId")),
		Operation:         operation,
		ClientOperationID: strAnyMap(req, "clientOperationId"),
		RetryOf:           strAnyMap(req, "retryOf"),
		AttemptGroupID:    strAnyMap(req, "attemptGroupId"),
		InputJSON:         marshalJSON(input),
		MediaStage:        "",
		ResultState:       "NOT_AVAILABLE",
	}
	if ctx := asMap(req["clientContext"]); len(ctx) > 0 {
		params.NodeID = strAnyMap(ctx, "nodeId")
		b, _ := json.Marshal(ctx)
		params.ClientContext = b
	}
	if specRaw, ok := input["generationSpec"]; ok {
		b, _ := json.Marshal(specRaw)
		params.GenerationSpec = b
		var spec map[string]any
		_ = json.Unmarshal(b, &spec)
		if prompt == "" {
			prompt = strAnyMap(spec, "prompt")
		}
	}
	if prompt == "" {
		prompt = strAnyMap(input, "prompt")
	}
	jobType := "canvas"
	status := "queued"
	if typ == "text" || typ == "text_replay" {
		jobType = "canvas_text"
		status = "text_replay"
		params.Mode = "text"
	}
	j := Job{
		ID: NewID(), Type: jobType, Status: status,
		Provider: provider, Model: model, Prompt: prompt,
		Params: marshalJSON(params), CreatedAt: now, UpdatedAt: now,
	}
	if err := e.insertJob(j); err != nil {
		return nil, err
	}
	e.emit(j)
	e.appendTaskLog(j.ID, "info", "queued", "task created")
	return e.taskView(j), nil
}

func (e *Engine) listCanvasTasks(projectID string, activeOnly bool, limit int) []map[string]any {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id FROM jobs WHERE (type LIKE 'canvas%' OR (json_extract(params, '$.canvas_project_id') IS NOT NULL AND json_extract(params, '$.canvas_project_id') != ''))`
	var args []any
	if projectID != "" {
		q += ` AND json_extract(params, '$.canvas_project_id') = ?`
		args = append(args, projectID)
	}
	if activeOnly {
		q += ` AND status IN ('queued','running','polling','text_replay')`
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := e.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			continue
		}
		j, err := e.GetJob(id)
		if err != nil {
			continue
		}
		out = append(out, e.taskView(j))
	}
	return out
}

func (e *Engine) canvasTask(id string) (map[string]any, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return nil, err
	}
	return e.taskView(j), nil
}

func (e *Engine) taskView(j Job) map[string]any {
	p := parseParams(j.Params)
	status := j.Status
	if status == "polling" {
		status = "running"
	}
	preview := e.MediaURL(j.ResultHash)
	poster := e.MediaURL(j.PosterHash)
	if p.ResourceID != "" {
		if res, err := e.getCanvasResource(p.ResourceID); err == nil {
			preview = res.PublicURL
		}
	}
	progress := 0
	switch j.Status {
	case "queued":
		progress = 5
	case "running":
		progress = 35
	case "polling":
		progress = 70
	case "succeeded", "text_replay":
		if j.Status == "succeeded" {
			progress = 100
		} else {
			progress = 50
		}
	}
	out := map[string]any{
		"id": j.ID, "type": firstNonEmpty(p.Mode, j.Type), "status": status,
		"prompt": j.Prompt, "provider": firstNonEmpty(p.ChannelID, j.Provider), "model": j.Model,
		"projectId": p.CanvasProjectID, "operation": p.Operation,
		"providerRequestId": j.RemoteID, "error": j.Error, "attempts": p.RecoverAttempts,
		"previewUrl": preview, "previewPosterUrl": poster,
		"inputJson": p.InputJSON, "resultJson": p.ResultJSON, "resultState": p.ResultState,
		"textDraft": p.TextDraft, "mediaStage": p.MediaStage, "canRecoverMedia": j.Status == "failed" && p.RecoverAttempts < 3,
		"createdAt": j.CreatedAt, "updatedAt": j.UpdatedAt, "completedAt": j.CompletedAt,
		"created_at": j.CreatedAt, "updated_at": j.UpdatedAt,
		"clientOperationId": p.ClientOperationID, "retryOf": p.RetryOf, "attemptGroupId": p.AttemptGroupID,
		"progress": progress,
	}
	if len(p.ClientContext) > 0 {
		var ctx any
		_ = json.Unmarshal(p.ClientContext, &ctx)
		out["clientContext"] = ctx
	}
	if p.ResourceID != "" && j.Status == "succeeded" {
		out["outputs"] = []map[string]any{{
			"outputIndex": 0, "mediaType": firstNonEmpty(p.Mode, "image"),
			"materializedAssetId": "resource:" + p.ResourceID,
			"providerArtifactRef": j.RemoteID,
		}}
		out["resultState"] = "READY"
		kind := firstNonEmpty(p.Mode, "image")
		if kind == "image" || kind == "video" {
			out["previewKind"] = kind
		}
	}
	if j.Status == "text_replay" {
		out["status"] = "running"
	}
	return out
}

func (e *Engine) appendTaskLog(taskID, level, stage, message string) {
	_, _ = e.DB.Exec(`INSERT INTO canvas_task_logs(id, task_id, level, stage, message, created_at) VALUES(?,?,?,?,?,?)`,
		NewID(), taskID, level, stage, message, Now())
}

func (e *Engine) taskLogs(id string) []map[string]any {
	rows, err := e.DB.Query(`SELECT level, message, created_at FROM canvas_task_logs WHERE task_id = ? ORDER BY created_at ASC`, id)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var level, msg, created string
		if rows.Scan(&level, &msg, &created) != nil {
			continue
		}
		out = append(out, map[string]any{"level": level, "message": msg, "createdAt": created})
	}
	return out
}

func (e *Engine) appendTextDelta(id, content string) (map[string]any, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return nil, err
	}
	var seq int
	_ = e.DB.QueryRow(`SELECT COALESCE(MAX(seq), 0) FROM canvas_text_deltas WHERE task_id = ?`, id).Scan(&seq)
	seq++
	now := Now()
	_, err = e.DB.Exec(`INSERT INTO canvas_text_deltas(id, task_id, seq, content, created_at) VALUES(?,?,?,?,?)`, NewID(), id, seq, content, now)
	if err != nil {
		return nil, err
	}
	p := parseParams(j.Params)
	p.TextDraft += content
	j.Params = marshalJSON(p)
	_ = e.saveJob(j)
	e.hub("canvas_text_delta", map[string]any{"task_id": id, "seq": seq, "content": content})
	return map[string]any{"seq": seq, "content": content, "createdAt": now}, nil
}

func (e *Engine) textReplay(id string, after int) map[string]any {
	rows, err := e.DB.Query(`SELECT seq, content, created_at FROM canvas_text_deltas WHERE task_id = ? AND seq > ? ORDER BY seq ASC`, id, after)
	if err != nil {
		return map[string]any{"items": []any{}, "cursor": after}
	}
	defer rows.Close()
	items := []map[string]any{}
	cursor := after
	for rows.Next() {
		var seq int
		var content, created string
		if rows.Scan(&seq, &content, &created) != nil {
			continue
		}
		items = append(items, map[string]any{
			"id": fmt.Sprintf("%s-%d", id, seq), "taskId": id, "sequence": seq, "seq": seq,
			"content": content, "byteCount": len(content), "createdAt": created,
		})
		cursor = seq
	}
	draft := ""
	status := "running"
	complete := false
	progress := 0
	if t, err := e.canvasTask(id); err == nil {
		draft = strAnyMap(t, "textDraft")
		status = firstNonEmpty(strAnyMap(t, "status"), "running")
		complete = status == "succeeded" || status == "failed" || status == "cancelled"
		progress = intAny(t["progress"])
	}
	return map[string]any{
		"items": items, "deltas": items, "cursor": cursor,
		"textDraft": draft, "complete": complete, "status": status, "progress": progress,
	}
}

func (e *Engine) completeTextReplay(id, text string) (map[string]any, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return nil, err
	}
	p := parseParams(j.Params)
	if text != "" {
		p.TextDraft = text
	}
	p.ResultState = "READY"
	j.Params = marshalJSON(p)
	j.Status = "succeeded"
	j.CompletedAt = Now()
	if err := e.saveJob(j); err != nil {
		return nil, err
	}
	return e.taskView(j), nil
}

func (e *Engine) queryProviderTask(id string) (map[string]any, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"taskId": j.ID, "providerRequestId": j.RemoteID, "status": j.Status, "error": j.Error,
	}, nil
}

func (e *Engine) cancelCanvasTask(id string) (map[string]any, error) {
	if err := e.CancelJob(id); err != nil {
		return nil, err
	}
	return e.canvasTask(id)
}

func (e *Engine) retryCanvasTask(id string) (map[string]any, error) {
	j, err := e.RetryJob(id)
	if err != nil {
		return nil, err
	}
	return e.taskView(j), nil
}

func (e *Engine) recoverCanvasTask(id string) (map[string]any, error) {
	j, err := e.RecoverCanvasMedia(id)
	if err != nil {
		return nil, err
	}
	return e.taskView(j), nil
}

func (e *Engine) deleteCanvasTask(id string) error {
	_, err := e.DB.Exec(`DELETE FROM jobs WHERE id = ?`, id)
	return err
}
