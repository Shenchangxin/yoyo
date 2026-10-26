// 行为：画布生成任务的登记、列表、文本回放、取消、重试和媒体恢复。本文件不调用模型。

package video

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// createCanvasTask 把画布生成请求登记成任务。文本任务进入 text_replay，其余进入 queued。
// req 读取 type、prompt、projectId、project_id、provider、model、operation、input、clientOperationId、retryOf、attemptGroupId、clientContext；type 为空时按 image，为 text 或 text_replay 时把任务类型写成 canvas_text、状态写成 text_replay。
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

// listCanvasTasks 列出画布任务。只要活动任务时，只留排队、运行、轮询和文本回放。
// projectID 非空时必须等于任务参数里的 CanvasProjectID。
// activeOnly 为真时只保留状态为 queued、running、polling 或 text_replay 的任务。
// limit 小于等于 0 时改成 50，结果多过 limit 时截断。
func (e *Engine) listCanvasTasks(projectID string, activeOnly bool, limit int) []map[string]any {
	if limit <= 0 {
		limit = 50
	}
	active := map[string]bool{"queued": true, "running": true, "polling": true, "text_replay": true}
	var jobs []Job
	for _, j := range loadCol[Job](e, colJobs) {
		p := parseParams(j.Params)
		isCanvas := strings.HasPrefix(j.Type, "canvas") || p.CanvasProjectID != ""
		if !isCanvas {
			continue
		}
		if projectID != "" && p.CanvasProjectID != projectID {
			continue
		}
		if activeOnly && !active[j.Status] {
			continue
		}
		jobs = append(jobs, j)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt > jobs[j].CreatedAt })
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	out := []map[string]any{}
	for _, j := range jobs {
		out = append(out, e.taskView(j))
	}
	return out
}

// canvasTask 按 id 返回任务视图。
func (e *Engine) canvasTask(id string) (map[string]any, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return nil, err
	}
	return e.taskView(j), nil
}

// taskView 把任务收成前端字段。轮询状态对外显示为 running。
// j 的 ID、Type、Status、Prompt、Provider、Model、ResultHash、PosterHash、RemoteID、Error、CreatedAt、UpdatedAt、CompletedAt 和 Params 写入返回字段；Status 为 polling 时对外写成 running，progress 按 queued 5、running 35、polling 70、succeeded 100、text_replay 50，失败且 RecoverAttempts 小于 3 时 canRecoverMedia 为真。
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

// appendTaskLog 追加一条任务日志。
// taskID 写入 TaskID。
// level 写入 Level。
// stage 写入 Stage。
// message 写入 Message。
func (e *Engine) appendTaskLog(taskID, level, stage, message string) {
	rec := canvasTaskLogRec{ID: NewID(), TaskID: taskID, Level: level, Stage: stage, Message: message, CreatedAt: Now()}
	_ = e.putDoc(colTaskLogs, rec.ID, rec)
}

// taskLogs 按时间顺序列出任务日志。
// id 与日志的 TaskID 比较，相等的才留下。
func (e *Engine) taskLogs(id string) []map[string]any {
	var recs []canvasTaskLogRec
	for _, r := range loadCol[canvasTaskLogRec](e, colTaskLogs) {
		if r.TaskID == id {
			recs = append(recs, r)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].CreatedAt < recs[j].CreatedAt })
	out := []map[string]any{}
	for _, r := range recs {
		out = append(out, map[string]any{"level": r.Level, "message": r.Message, "createdAt": r.CreatedAt})
	}
	return out
}

// appendTextDelta 给文本任务追加一段增量，并更新草稿。
// id 传给 GetJob，并作为增量的 TaskID。
// content 写入增量的 Content，并追加到 TextDraft。
func (e *Engine) appendTextDelta(id, content string) (map[string]any, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return nil, err
	}
	seq := 0
	for _, d := range loadCol[canvasDeltaRec](e, colTextDeltas) {
		if d.TaskID == id && d.Seq > seq {
			seq = d.Seq
		}
	}
	seq++
	now := Now()
	rec := canvasDeltaRec{ID: NewID(), TaskID: id, Seq: seq, Content: content, CreatedAt: now}
	if err := e.putDoc(colTextDeltas, rec.ID, rec); err != nil {
		return nil, err
	}
	p := parseParams(j.Params)
	p.TextDraft += content
	j.Params = marshalJSON(p)
	_ = e.saveJob(j)
	e.hub("canvas_text_delta", map[string]any{"task_id": id, "seq": seq, "content": content})
	return map[string]any{"seq": seq, "content": content, "createdAt": now}, nil
}

// textReplay 返回序号大于 after 的文本增量，以及草稿和是否已经结束。
// id 用来匹配 TaskID，并传给 canvasTask。
func (e *Engine) textReplay(id string, after int) map[string]any {
	var recs []canvasDeltaRec
	for _, d := range loadCol[canvasDeltaRec](e, colTextDeltas) {
		if d.TaskID == id && d.Seq > after {
			recs = append(recs, d)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Seq < recs[j].Seq })
	items := []map[string]any{}
	cursor := after
	for _, d := range recs {
		items = append(items, map[string]any{
			"id": fmt.Sprintf("%s-%d", id, d.Seq), "taskId": id, "sequence": d.Seq, "seq": d.Seq,
			"content": d.Content, "byteCount": len(d.Content), "createdAt": d.CreatedAt,
		})
		cursor = d.Seq
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

// completeTextReplay 用最终文本结束文本回放，并把任务标成成功。
// id 传给 GetJob。
// text 非空时写入 TextDraft。
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

// queryProviderTask 返回任务上已保存的供应商请求号和状态，不向供应商再发请求。
func (e *Engine) queryProviderTask(id string) (map[string]any, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"taskId": j.ID, "providerRequestId": j.RemoteID, "status": j.Status, "error": j.Error,
	}, nil
}

// cancelCanvasTask 取消任务并返回取消后的视图。
// id 传给 CancelJob 和 canvasTask。
func (e *Engine) cancelCanvasTask(id string) (map[string]any, error) {
	if err := e.CancelJob(id); err != nil {
		return nil, err
	}
	return e.canvasTask(id)
}

// retryCanvasTask 重试任务并返回新的任务视图。
// id 传给 RetryJob。
func (e *Engine) retryCanvasTask(id string) (map[string]any, error) {
	j, err := e.RetryJob(id)
	if err != nil {
		return nil, err
	}
	return e.taskView(j), nil
}

// recoverCanvasTask 调用 RecoverCanvasMedia，再返回任务视图。
// id 传给 RecoverCanvasMedia。
func (e *Engine) recoverCanvasTask(id string) (map[string]any, error) {
	j, err := e.RecoverCanvasMedia(id)
	if err != nil {
		return nil, err
	}
	return e.taskView(j), nil
}

// deleteCanvasTask 删掉这条任务文档。
// id 传给 delDoc，从 colJobs 删除。
func (e *Engine) deleteCanvasTask(id string) error {
	return e.delDoc(colJobs, id)
}
