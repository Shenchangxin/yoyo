// 行为：保存画布助手的运行状态，供画布面板轮询。模型调用不在本文件。Agent.Start 由 app 接到会话的 StartSend，再进入 runtime.Run。这里不是第二条循环。

package video

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const colAgentProfile = "canvas_agent_profile"

// CanvasAgentTurn 是画布面板交给助手的一轮用户消息，包含画布、提示、权限、模型和点名技能。
type CanvasAgentTurn struct {
	RunID      string
	CanvasID   string
	Prompt     string
	Permission string
	Model      string
	SkillIDs   []string
	Epoch      int
}

// CanvasAgentBridge 由 app 填上。Start、Stop、Decide 都回到 app：Start 进入已绑定会话的 StartSend，模型调用在 runtime.Run。视频引擎只保存画布面板轮询的运行状态，这不是第二条循环。
type CanvasAgentBridge struct {
	Start  func(CanvasAgentTurn) error
	Stop   func(sessionID string)
	Decide func(approvalID, decision string) error
}

type canvasAgentBook struct {
	mu             sync.Mutex
	runs           map[string]*canvasAgentRun
	byKey          map[string]string
	bySession      map[string]string
	canvasSessions map[string]string
}

type canvasAgentRun struct {
	ID         string
	CanvasID   string
	SessionID  string
	Status     string
	Permission string
	Model      string
	CreatedAt  string
	UpdatedAt  string
	Failure    string
	MessageID  string
	Text       string
	Epoch      int
	Revision   int
	Seq        int
	LastKey    string
	Approval   map[string]any
	Events     []map[string]any
}

type agentProfileRec struct {
	User     string                   `json:"user"`
	UserRev  int                      `json:"user_rev"`
	UserHash string                   `json:"user_hash"`
	Canvases map[string]agentLayerRec `json:"canvases,omitempty"`
}

type agentLayerRec struct {
	Content string `json:"content"`
	Rev     int    `json:"rev"`
	Hash    string `json:"hash"`
}

// lockAgents 锁住助手运行簿，并在第一次使用时建好运行、幂等键和会话索引。
func (e *Engine) lockAgents() *canvasAgentBook {
	b := &e.agents
	b.mu.Lock()
	if b.runs == nil {
		b.runs = map[string]*canvasAgentRun{}
		b.byKey = map[string]string{}
		b.bySession = map[string]string{}
		b.canvasSessions = map[string]string{}
	}
	return b
}

// canvasAgentHTTP 分发画布助手的能力、人设、运行、消息、插话、取消和审批路由。
func (e *Engine) canvasAgentHTTP(method, path string, query, body map[string]any) canvasEnv {
	parts := splitPath(path)
	match := func(pat string) (map[string]string, bool) {
		return matchPath(parts, pat)
	}
	if path == "agent/capabilities" && method == "GET" {
		return canvasOK(canvasAgentCapabilities())
	}
	if path == "agent/profile" && method == "GET" {
		return canvasOK(e.agentProfileView(strAnyMap(query, "canvasId")))
	}
	if path == "agent/profile" && method == "PATCH" {
		return e.patchAgentProfile(body)
	}
	if path == "agent/runs" && method == "POST" {
		return e.createCanvasAgentRun(body)
	}
	if p, ok := match("agent/runs/:id"); ok && method == "GET" {
		view, ok := e.canvasAgentView(p["id"])
		if !ok {
			return canvasFail(404, "run not found", "not_found")
		}
		return canvasOK(map[string]any{"run": view})
	}
	if p, ok := match("agent/runs/:id/messages"); ok && method == "POST" {
		return e.messageCanvasAgent(p["id"], body)
	}
	if p, ok := match("agent/runs/:id/interjections"); ok && method == "POST" {
		env := e.messageCanvasAgent(p["id"], map[string]any{
			"prompt":         strAnyMap(body, "text"),
			"idempotencyKey": strAnyMap(body, "messageId"),
			"permissionMode": strAnyMap(body, "permissionMode"),
			"canvasId":       strAnyMap(body, "canvasId"),
		})
		if env.Code != 0 {
			return env
		}
		return canvasOK(map[string]any{"accepted": true, "pending": 1})
	}
	if p, ok := match("agent/runs/:id/cancel"); ok && method == "POST" {
		e.cancelCanvasAgent(p["id"])
		return canvasOK(map[string]any{"accepted": true})
	}
	if p, ok := match("agent/runs/:runId/approvals/:approvalId/decision"); ok && method == "POST" {
		return e.decideCanvasAgent(p["runId"], p["approvalId"], strAnyMap(body, "decision"))
	}
	return canvasFail(404, "agent route not found", "not_found")
}

// canvasAgentCapabilities 返回画布助手声明的权限模式、上下文范围和能力集版本。
func canvasAgentCapabilities() map[string]any {
	return map[string]any{
		"version":              1,
		"permissionModes":      []string{"read_only", "auto", "request_approval"},
		"contextScopes":        []string{"canvas", "project", "selection"},
		"skills":               true,
		"writeTools":           true,
		"capabilitySetVersion": "yoyo-canvas-1",
	}
}

// createCanvasAgentRun 按幂等键创建或复用一次运行，然后把这一轮交给 launchCanvasAgent。
func (e *Engine) createCanvasAgentRun(body map[string]any) canvasEnv {
	canvasID := strAnyMap(body, "canvasId", "canvas_id")
	prompt := bodyText(body, "prompt")
	key := strAnyMap(body, "idempotencyKey", "idempotency_key")
	if canvasID == "" || prompt == "" || key == "" {
		return canvasFail(400, "canvas, prompt, and idempotency key are required", "bad_request")
	}
	perm := strAnyMap(body, "permissionMode")
	if perm == "" {
		perm = "request_approval"
	}
	if perm != "read_only" && perm != "auto" && perm != "request_approval" {
		return canvasFail(400, "unsupported permission", "bad_request")
	}
	b := e.lockAgents()
	if id := b.byKey[key]; id != "" {
		view := b.runs[id].view()
		b.mu.Unlock()
		return canvasOK(map[string]any{"run": view})
	}
	now := Now()
	run := &canvasAgentRun{
		ID: NewID(), CanvasID: canvasID, Status: "running", Permission: perm,
		Model: agentModel(body), CreatedAt: now, UpdatedAt: now, LastKey: key,
		Epoch: 1, Revision: 1,
	}
	run.MessageID = fmt.Sprintf("assistant-%s-%d", run.ID, run.Epoch)
	if sid := b.canvasSessions[canvasID]; sid != "" {
		run.SessionID = sid
		b.bySession[sid] = run.ID
	}
	b.runs[run.ID] = run
	b.byKey[key] = run.ID
	turn := CanvasAgentTurn{
		RunID: run.ID, CanvasID: canvasID, Prompt: prompt, Permission: perm,
		Model: run.Model, SkillIDs: strList(body["skillIds"]), Epoch: run.Epoch,
	}
	b.mu.Unlock()
	e.launchCanvasAgent(turn)
	view, _ := e.canvasAgentView(turn.RunID)
	return canvasOK(map[string]any{"run": view})
}

// messageCanvasAgent 给已有运行追加一轮用户消息。相同幂等键直接返回当前视图。
func (e *Engine) messageCanvasAgent(id string, body map[string]any) canvasEnv {
	prompt := bodyText(body, "prompt")
	if prompt == "" {
		prompt = bodyText(body, "text")
	}
	key := strAnyMap(body, "idempotencyKey", "idempotency_key", "messageId")
	if strings.TrimSpace(id) == "" || prompt == "" {
		return canvasFail(400, "run and prompt are required", "bad_request")
	}
	b := e.lockAgents()
	run := b.runs[id]
	if run == nil {
		b.mu.Unlock()
		return canvasFail(404, "run not found", "not_found")
	}
	if key != "" && run.LastKey == key {
		view := run.view()
		b.mu.Unlock()
		return canvasOK(map[string]any{"run": view})
	}
	run.Epoch++
	run.Revision++
	run.Status = "running"
	run.Failure = ""
	run.Text = ""
	run.Approval = nil
	run.UpdatedAt = Now()
	run.LastKey = key
	run.MessageID = fmt.Sprintf("assistant-%s-%d", run.ID, run.Epoch)
	if perm := strAnyMap(body, "permissionMode"); perm == "read_only" || perm == "auto" || perm == "request_approval" {
		run.Permission = perm
	}
	if model := agentModel(body); model != "" {
		run.Model = model
	}
	if sid := b.canvasSessions[run.CanvasID]; sid != "" {
		run.SessionID = sid
		b.bySession[sid] = run.ID
	}
	turn := CanvasAgentTurn{
		RunID: run.ID, CanvasID: run.CanvasID, Prompt: prompt, Permission: run.Permission,
		Model: run.Model, SkillIDs: strList(body["skillIds"]), Epoch: run.Epoch,
	}
	b.mu.Unlock()
	e.launchCanvasAgent(turn)
	view, _ := e.canvasAgentView(turn.RunID)
	return canvasOK(map[string]any{"run": view})
}

// launchCanvasAgent 在桥的 Start 已接上时异步调用它。Start 为空就把这次运行标成失败。本函数不调用模型。
func (e *Engine) launchCanvasAgent(turn CanvasAgentTurn) {
	if e.Agent == nil || e.Agent.Start == nil {
		e.CanvasAgentFailRun(turn.RunID, turn.Epoch, "画布助手还没有接到对话运行时")
		return
	}
	go func() {
		if err := e.Agent.Start(turn); err != nil {
			e.CanvasAgentFailRun(turn.RunID, turn.Epoch, err.Error())
		}
	}()
}

// cancelCanvasAgent 把尚未结束的运行标成取消，并在已有会话时调用 Agent.Stop。
func (e *Engine) cancelCanvasAgent(id string) {
	b := e.lockAgents()
	run := b.runs[id]
	sessionID := ""
	if run != nil && run.Status != "completed" && run.Status != "failed" && run.Status != "cancelled" && run.Status != "rejected" {
		run.Status = "cancelled"
		run.Approval = nil
		run.UpdatedAt = Now()
		run.Revision++
		sessionID = run.SessionID
	}
	b.mu.Unlock()
	if sessionID != "" && e.Agent != nil && e.Agent.Stop != nil {
		e.Agent.Stop(sessionID)
	}
}

// decideCanvasAgent 把批准或拒绝交给 Agent.Decide，并清掉本地的等待审批状态。
func (e *Engine) decideCanvasAgent(runID, approvalID, decision string) canvasEnv {
	decision = strings.TrimSpace(decision)
	if decision != "approve" && decision != "reject" {
		return canvasFail(400, "decision must be approve or reject", "bad_request")
	}
	if e.Agent == nil || e.Agent.Decide == nil {
		return canvasFail(404, "approval gateway is not attached", "not_found")
	}
	if err := e.Agent.Decide(approvalID, decision); err != nil {
		return canvasFail(404, err.Error(), "not_found")
	}
	b := e.lockAgents()
	defer b.mu.Unlock()
	run := b.runs[runID]
	if run == nil {
		return canvasFail(404, "run not found", "not_found")
	}
	run.Approval = nil
	if run.Status == "waiting_approval" {
		run.Status = "running"
	}
	run.UpdatedAt = Now()
	run.Revision++
	run.push("approval_decided", map[string]any{"approvalId": approvalID, "decision": decision, "text": decision})
	return canvasOK(map[string]any{"accepted": true})
}

// canvasAgentView 在锁内返回一次运行的轮询视图。找不到时 ok 为 false。
func (e *Engine) canvasAgentView(id string) (map[string]any, bool) {
	b := e.lockAgents()
	defer b.mu.Unlock()
	run := b.runs[id]
	if run == nil {
		return nil, false
	}
	return run.view(), true
}

// RememberCanvasAgentSession 记住画布、会话和这次运行的对应关系，供之后的轮询和事件回写使用。
func (e *Engine) RememberCanvasAgentSession(canvasID, sessionID, runID string) {
	if e == nil {
		return
	}
	b := e.lockAgents()
	defer b.mu.Unlock()
	if canvasID != "" && sessionID != "" {
		b.canvasSessions[canvasID] = sessionID
	}
	run := b.runs[runID]
	if run != nil && sessionID != "" {
		run.SessionID = sessionID
		b.bySession[sessionID] = runID
	}
}

// CanvasAgentEpoch 返回该会话当前应接受的轮次。没有对应运行时返回 0。
func (e *Engine) CanvasAgentEpoch(sessionID string) int {
	if e == nil {
		return 0
	}
	b := e.lockAgents()
	defer b.mu.Unlock()
	run := b.runBySession(sessionID)
	if run == nil {
		return 0
	}
	return run.Epoch
}

// CanvasAgentStream 用这一轮的实时助手文本替换运行里的正文，并把状态保持为 running。
func (e *Engine) CanvasAgentStream(sessionID string, epoch int, text string) {
	e.touchAgent(sessionID, epoch, func(run *canvasAgentRun) {
		if run.Status == "waiting_approval" {
			return
		}
		run.Status = "running"
		run.Text = text
		run.UpdatedAt = Now()
	})
}

// CanvasAgentFinish 结束当前轮次。ok 为真时标成完成，正文为空则写入默认完成句；否则标成失败。
func (e *Engine) CanvasAgentFinish(sessionID string, epoch int, text string, ok bool) {
	e.touchAgent(sessionID, epoch, func(run *canvasAgentRun) {
		if strings.TrimSpace(text) != "" {
			run.Text = strings.TrimSpace(text)
		}
		run.Approval = nil
		run.UpdatedAt = Now()
		run.Revision++
		if ok {
			if strings.TrimSpace(run.Text) == "" {
				run.Text = "已完成，这一轮没有新的文字回复。"
			}
			run.Status = "completed"
			run.Failure = ""
			return
		}
		run.Status = "failed"
		if run.Failure == "" {
			run.Failure = "画布助手这一轮没有完成"
		}
	})
}

// CanvasAgentFail 把当前轮次标成失败并写入原因。轮次不一致时不改更新的运行。
func (e *Engine) CanvasAgentFail(sessionID string, epoch int, msg string) {
	e.touchAgent(sessionID, epoch, func(run *canvasAgentRun) {
		run.Status = "failed"
		run.Failure = strings.TrimSpace(msg)
		if run.Failure == "" {
			run.Failure = "画布助手执行失败"
		}
		run.Approval = nil
		run.UpdatedAt = Now()
		run.Revision++
	})
}

// CanvasAgentFailRun 在会话还不存在时，按运行 id 和轮次把启动失败写回去。
func (e *Engine) CanvasAgentFailRun(runID string, epoch int, msg string) {
	if e == nil {
		return
	}
	b := e.lockAgents()
	defer b.mu.Unlock()
	run := b.runs[runID]
	if run == nil || run.Epoch != epoch {
		return
	}
	if run.Status == "cancelled" || run.Status == "rejected" {
		return
	}
	run.Status = "failed"
	run.Failure = strings.TrimSpace(msg)
	if run.Failure == "" {
		run.Failure = "画布助手执行失败"
	}
	run.UpdatedAt = Now()
	run.Revision++
}

// CanvasAgentTool 在会改画布的工具完成后推一条 canvas_updated，让打开的编辑器重新加载。
func (e *Engine) CanvasAgentTool(sessionID string, epoch int, name, callID string) {
	e.touchAgent(sessionID, epoch, func(run *canvasAgentRun) {
		if !canvasAgentMutates(name) {
			return
		}
		run.Approval = nil
		if run.Status == "waiting_approval" {
			run.Status = "running"
		}
		run.UpdatedAt = Now()
		run.push("canvas_updated", map[string]any{
			"canvasId": run.CanvasID,
			"toolName": name,
			"callId":   callID,
		})
	})
}

// CanvasAgentApproval 把审批请求放进运行视图，并把状态改为 waiting_approval。
func (e *Engine) CanvasAgentApproval(sessionID string, epoch int, payload map[string]any) {
	view := approvalView(payload)
	if view == nil {
		return
	}
	e.touchAgent(sessionID, epoch, func(run *canvasAgentRun) {
		run.Status = "waiting_approval"
		run.Approval = view
		run.UpdatedAt = Now()
		run.Revision++
	})
}

// touchAgent 在锁内按会话和轮次找到仍在进行的运行，再执行 fn。已结束或轮次不同则返回。
func (e *Engine) touchAgent(sessionID string, epoch int, fn func(*canvasAgentRun)) {
	if e == nil || epoch == 0 {
		return
	}
	b := e.lockAgents()
	defer b.mu.Unlock()
	run := b.runBySession(sessionID)
	if run == nil || run.Epoch != epoch {
		return
	}
	switch run.Status {
	case "cancelled", "rejected", "completed", "failed":
		return
	}
	fn(run)
}

// runBySession 用会话 id 查出对应的运行。
func (b *canvasAgentBook) runBySession(sessionID string) *canvasAgentRun {
	id := b.bySession[sessionID]
	if id == "" {
		return nil
	}
	return b.runs[id]
}

// push 给运行追加一条事件，只保留最近 40 条。
func (run *canvasAgentRun) push(typ string, payload map[string]any) {
	run.Seq++
	ev := map[string]any{
		"seq":       run.Seq,
		"type":      typ,
		"runId":     run.ID,
		"eventId":   fmt.Sprintf("%s-%d", run.ID, run.Seq),
		"createdAt": Now(),
		"payload":   payload,
	}
	run.Events = append(run.Events, ev)
	if len(run.Events) > 40 {
		run.Events = append([]map[string]any(nil), run.Events[len(run.Events)-40:]...)
	}
}

// view 复制运行的状态、事件、正文和审批，供面板轮询。
func (run *canvasAgentRun) view() map[string]any {
	events := make([]any, 0, len(run.Events))
	for _, ev := range run.Events {
		cp := make(map[string]any, len(ev))
		for k, v := range ev {
			cp[k] = v
		}
		events = append(events, cp)
	}
	out := map[string]any{
		"id":             run.ID,
		"canvasId":       run.CanvasID,
		"status":         run.Status,
		"permissionMode": run.Permission,
		"revision":       run.Revision,
		"cleanupPending": false,
		"createdAt":      run.CreatedAt,
		"updatedAt":      run.UpdatedAt,
		"events":         events,
	}
	if run.Model != "" {
		out["model"] = run.Model
	}
	if run.Failure != "" {
		out["failureMessage"] = run.Failure
	}
	if strings.TrimSpace(run.Text) != "" && run.MessageID != "" {
		out["activeMessage"] = map[string]any{"messageId": run.MessageID, "text": run.Text}
	}
	if run.Approval != nil {
		cp := make(map[string]any, len(run.Approval))
		for k, v := range run.Approval {
			cp[k] = v
		}
		out["approval"] = cp
	}
	return out
}

// agentProfileView 读出用户层和可选的画布层人设，并带上修订号和哈希。
func (e *Engine) agentProfileView(canvasID string) map[string]any {
	rec, err := getDoc[agentProfileRec](e, colAgentProfile, "local")
	if err != nil {
		rec = agentProfileRec{}
	}
	if rec.UserRev < 1 {
		rec.UserRev = 1
	}
	if rec.UserHash == "" {
		rec.UserHash = hashText(rec.User)
	}
	layers := []any{map[string]any{
		"scope": "user", "content": rec.User, "revision": rec.UserRev, "hash": rec.UserHash,
	}}
	rev := rec.UserRev
	if canvasID != "" {
		var layer agentLayerRec
		if rec.Canvases != nil {
			layer = rec.Canvases[canvasID]
		}
		if layer.Rev < 1 {
			layer.Rev = 1
		}
		if layer.Hash == "" {
			layer.Hash = hashText(layer.Content)
		}
		layers = append(layers, map[string]any{
			"scope": "canvas", "canvasId": canvasID, "content": layer.Content, "revision": layer.Rev, "hash": layer.Hash,
		})
		if layer.Rev > rev {
			rev = layer.Rev
		}
	}
	return map[string]any{
		"revision": strconv.Itoa(rev),
		"hash":     hashText(rec.User + "|" + canvasID),
		"layers":   layers,
	}
}

// patchAgentProfile 按 scope 更新用户层或指定画布层的人设正文。
func (e *Engine) patchAgentProfile(body map[string]any) canvasEnv {
	scope := strAnyMap(body, "scope")
	canvasID := strAnyMap(body, "canvasId")
	content := ""
	if v, ok := body["content"]; ok && v != nil {
		content = fmt.Sprint(v)
		if content == "<nil>" {
			content = ""
		}
	}
	rec, err := getDoc[agentProfileRec](e, colAgentProfile, "local")
	if err != nil {
		rec = agentProfileRec{}
	}
	if rec.Canvases == nil {
		rec.Canvases = map[string]agentLayerRec{}
	}
	if scope == "canvas" {
		layer := rec.Canvases[canvasID]
		layer.Content = content
		layer.Rev++
		if layer.Rev < 1 {
			layer.Rev = 1
		}
		layer.Hash = hashText(content)
		rec.Canvases[canvasID] = layer
	} else {
		rec.User = content
		rec.UserRev++
		if rec.UserRev < 1 {
			rec.UserRev = 1
		}
		rec.UserHash = hashText(content)
	}
	if err := e.putDoc(colAgentProfile, "local", rec); err != nil {
		return canvasFail(400, err.Error(), "bad_request")
	}
	return canvasOK(e.agentProfileView(canvasID))
}

// agentModel 从请求里取模型名，或拼成通道 id 加模型键。
func agentModel(body map[string]any) string {
	if m := strAnyMap(body, "model"); m != "" {
		return m
	}
	ch := strAnyMap(body, "channelId")
	key := strAnyMap(body, "channelModelKey")
	if key == "" {
		key = strAnyMap(body, "logicalModelId")
	}
	if ch != "" && key != "" {
		return ch + "::" + key
	}
	return ""
}

// bodyText 取出请求字段里的非空文本。
func bodyText(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return ""
	}
	return s
}

// strList 把字符串切片或任意切片收成字符串列表。
func strList(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s == "" || s == "<nil>" {
				continue
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

// hashText 取文本 SHA-256 的前 6 字节，编成十六进制。
func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

// canvasAgentMutates 判断工具名是否会改画布，从而需要通知编辑器重载。
func canvasAgentMutates(name string) bool {
	switch name {
	case "canvas_apply_ops", "canvas_arrange_nodes", "canvas_create_storyboard", "canvas_edit_storyboard", "canvas_edit_batch_table", "generate_media", "image_layer_split", "image_annotation_render":
		return true
	default:
		return false
	}
}

// approvalView 把审批载荷收成画布面板用的确认视图。载荷没有 id 时返回 nil。
func approvalView(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	id, _ := payload["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	action, _ := payload["action"].(string)
	action = strings.TrimSpace(action)
	if action == "" {
		action = "tool"
	}
	command, _ := payload["command"].(string)
	path, _ := payload["path"].(string)
	summary := strings.TrimSpace(strings.TrimSpace(command) + " " + strings.TrimSpace(path))
	if summary == "" {
		summary = action
	}
	title := "确认执行操作"
	if action != "tool" {
		title = "确认 " + action
	}
	return map[string]any{
		"approvalId": id,
		"toolName":   action,
		"reason":     summary,
		"call": map[string]any{
			"id":       id,
			"function": map[string]any{"name": action, "arguments": "{}"},
		},
		"preview": map[string]any{
			"kind":        "agent_approval",
			"title":       title,
			"description": summary,
			"items": []any{map[string]any{
				"operation": "plan_step",
				"nodeTitle": action,
				"summary":   summary,
			}},
		},
	}
}
