// rpc.go 拥有 JSON-RPC 的读循环、按方法名分发，以及轨迹和状态通知。不定义 HTTP 服务器，那个在 server.go。

package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/im"
	"github.com/Shenchangxin/yoyo/internal/pages"
	"github.com/Shenchangxin/yoyo/internal/profile"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/schedule"
	"github.com/Shenchangxin/yoyo/internal/update"
)

// RPCRequest 是一条 JSON-RPC 2.0 请求。ID 原样回显，Params 保留原始 JSON。
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// RPCError 是 JSON-RPC 错误对象，含 code、message 和可选 data。
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// RPCResponse 是 JSON-RPC 2.0 应答。Result 和 Error 按编码时的有无出现。
type RPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// ServeRPC 从 r 读 JSON-RPC 请求，把 Dispatch 的应答写到 w。工作者进程里这对读写就是 stdin 和 stdout，stdout 是协议。
// Hub 订阅 * 后，每条轨迹写成无 id 通知 item.event。参数含 seq、type、item_kind、session、session_id、source。
// 流式增量只带 frame，正文不另放进 payload。其余仍带 payload，Notice 内联了 event 时一并写入。
// 类型为 approval 且 source 为 gate 时再写 approval.request。类型为 turn_end 时再写 turn.completed。状态订阅写成 state.changed，参数是 epoch、topic、object。
// 上下文取消时返回该错误。读到 EOF 返回 nil。
// ctx 在订阅循环里看 Done，在读循环里看 Err，非 nil 时返回该错误，并把 ctx 传给 Dispatch。
func ServeRPC(ctx context.Context, a *app.App, r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	var mu sync.Mutex
	enc := json.NewEncoder(w)
	ch, unsub := a.Hub.Subscribe("*")
	defer unsub()
	stateCh, unsubState := a.SubscribeState()
	defer unsubState()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case change, ok := <-stateCh:
				if !ok {
					return
				}
				mu.Lock()
				_ = enc.Encode(map[string]any{
					"jsonrpc": "2.0",
					"method":  "state.changed",
					"params": map[string]any{
						"epoch":  change.Epoch,
						"topic":  change.Topic,
						"object": change.Object,
					},
				})
				mu.Unlock()
			case ev, ok := <-ch:
				if !ok {
					return
				}
				mu.Lock()
				n := app.NoticeOf(ev)
				params := map[string]any{
					"seq":        n.Seq,
					"type":       n.Type,
					"item_kind":  ev.ItemKind,
					"session":    ev.SessionID,
					"session_id": ev.SessionID,
					"source":     ev.Source,
				}
				if n.Frame != nil && n.Event == nil {
					params["frame"] = n.Frame
				} else {
					params["payload"] = ev.Payload
					if n.Event != nil {
						params["event"] = n.Event
					}
					if n.Frame != nil {
						params["frame"] = n.Frame
					}
				}
				_ = enc.Encode(map[string]any{
					"jsonrpc": "2.0",
					"method":  "item.event",
					"params":  params,
				})
				if ev.Type == "approval" && ev.Source == "gate" {
					_ = enc.Encode(map[string]any{
						"jsonrpc": "2.0",
						"method":  "approval.request",
						"params": map[string]any{
							"id":      ev.Payload["id"],
							"session": ev.SessionID,
							"action":  ev.Payload["action"],
							"command": ev.Payload["command"],
							"path":    ev.Payload["path"],
							"level":   ev.Payload["level"],
						},
					})
				}
				if ev.Type == "turn_end" {
					_ = enc.Encode(map[string]any{
						"jsonrpc": "2.0",
						"method":  "turn.completed",
						"params": map[string]any{
							"session": ev.SessionID,
							"stop":    ev.Payload["stop"],
							"ok":      ev.Payload["ok"],
							"text":    ev.Payload["text"],
						},
					})
				}
				mu.Unlock()
			}
		}
	}()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var req RPCRequest
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		mu.Lock()
		err := enc.Encode(Dispatch(ctx, a, req))
		mu.Unlock()
		if err != nil {
			return err
		}
	}
}

// handleRPC 只接受 POST。请求体不是 JSON 时写 400，错误码 -32700。否则把 Dispatch 的结果写成 JSON。
func handleRPC(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST", 405)
			return
		}
		var req RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONStatus(w, 400, RPCResponse{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: err.Error()}})
			return
		}
		writeJSON(w, Dispatch(r.Context(), a, req))
	}
}

// Dispatch 要求 jsonrpc 为空或 2.0，否则返回 -32600。空 params 当成 {}。然后 callMethod。
// L3 错误的 code 是 403，data 是该错误。runtime.ErrBudget 的 code 是 429，data 是预算。其余错误的 code 是 -32000。成功后调用 a.NoteRPC。
// ctx 先交给 diaglog.With，再传给 DebugContext 和 callMethod。失败时也传给 ErrorContext。
// req 的 JSONRPC 不是空且不是 2.0 时直接返回 -32600。ID 写入应答。Params 长度为 0 时换成 {}。Method 传给 callMethod，成功且 a 非 nil 时再传给 NoteRPC。
func Dispatch(ctx context.Context, a *app.App, req RPCRequest) RPCResponse {
	res := RPCResponse{JSONRPC: "2.0", ID: req.ID}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		res.Error = &RPCError{Code: -32600, Message: "jsonrpc must be 2.0"}
		return res
	}
	params := req.Params
	if len(params) == 0 {
		params = []byte("{}")
	}
	rid := diaglog.NewRequestID()
	ctx = diaglog.With(ctx, diaglog.Fields{RequestID: rid, Component: "rpc", Cat: "rpc"})
	diaglog.DebugContext(ctx, "rpc", "component", "rpc", "cat", "rpc", "method", req.Method)
	result, err := callMethod(ctx, a, req.Method, params)
	if err != nil {
		diaglog.ErrorContext(ctx, "rpc failed", "component", "rpc", "cat", "rpc", "method", req.Method, "err", err)
		code := -32000
		var data any
		if e, ok := app.AsL3(err); ok {
			code = 403
			data = e
		}
		var b runtime.ErrBudget
		if errors.As(err, &b) {
			code = 429
			data = b
		}
		res.Error = &RPCError{Code: code, Message: err.Error(), Data: data}
		return res
	}
	res.Result = result
	if a != nil {
		a.NoteRPC(req.Method)
	}
	return res
}

// callMethod 按方法名调用 App。方法名为空时返回 ok。remote. 前缀先交给 remoteHandler，ok 为真或 err 非空就返回，否则继续。video.、drama.、media.、canvas.、connections. 前缀交给 VideoCall。
// turn.start 在 wait 为假时走 StartSendOpts，ErrQueued 返回 queued，否则返回 async；wait 为真时走 SendOpts 并返回 text。turn.interrupt 只返回 StopTurn 的 state。turn.stop 返回完整的 StopTurn。
// host.open 按 kind 分到 editor、terminal、url，其余走 OpenPath。about 在 Health 上加上 goos。logs.tail 带了 level、component 或 session 时改用日志过滤。provider.test 的 id 为空时走 TestProvider。fiber.unload 先 Dispose 同名 fiber，没有则 Unload WASM。skins.import 有 path 时走 ImportSkinPath，否则解码 bytes_b64。
// evolve.overlay 用 OverlayFlagConflict 拒绝 promote、sealed、behavior、index、baselines、k、rounds 这些混进来的进化参数。reuse_turn 缺省为真。超时按 overlay_k 夹到 1 到 4 后乘 15 分钟，再抬到 max_wall，上限 1 小时。传给 OverlaySession 的 K 仍是原来的 overlay_k。solver 为 heuristic 或 fixture 时用 HeuristicSolver，prompt-sensitive 时用 PromptSensitiveSolver。
// eval.run 有 snapshot 时走 RunEvalAt，超时 10 分钟。eval.best_of_n、eval.best_of_models、密封、迁移、索引和基线搜索超时 20 分钟。evolve.run 每轮 15 分钟，多轮相乘，上限 2 小时。update.apply 经 applyUpdateResult。config.set 把原始 params 交给 ApplyConfigPatch。key.set 把去掉空白的 value 写入 vault 的 default。未知方法返回 errMethod。
// ctx 传给 remoteHandler 和 SendOpts，也传给 context.WithTimeout。
// method 为空时返回 ok。带 remote. 前缀时传给 remoteHandler。带 video.、drama.、media.、canvas. 或 connections. 前缀时传给 VideoCall。其余进入 switch，未知时传给 errMethod。
func callMethod(ctx context.Context, a *app.App, method string, params json.RawMessage) (any, error) {
	if method == "" {
		return map[string]any{"ok": true}, nil
	}
	if remoteHandler != nil && strings.HasPrefix(method, "remote.") {
		out, ok, err := remoteHandler(ctx, a, method, params)
		if ok || err != nil {
			return out, err
		}
	}
	if strings.HasPrefix(method, "learn.") {
		return a.LearnCall(method, asMap(params))
	}
	if strings.HasPrefix(method, "video.") || strings.HasPrefix(method, "drama.") || strings.HasPrefix(method, "media.") || strings.HasPrefix(method, "canvas.") || strings.HasPrefix(method, "connections.") {
		return a.VideoCall(method, asMap(params))
	}
	switch method {
	case "health":
		return a.Health(), nil
	case "thread.start":
		var p struct {
			Workspace string `json:"workspace"`
			Channel   string `json:"channel"`
		}
		_ = json.Unmarshal(params, &p)
		return a.NewSessionOn(p.Workspace, p.Channel)
	case "thread.channel.set":
		var p struct {
			Session string `json:"session"`
			Channel string `json:"channel"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionChannel(p.Session, p.Channel)
	case "turn.start":
		var p struct {
			Session     string `json:"session"`
			Text        string `json:"text"`
			Plan        bool   `json:"plan"`
			Wait        bool   `json:"wait"`
			RequestID   string `json:"request_id"`
			Attachments []any  `json:"attachments"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if !p.Wait {
			var atts []app.Attachment
			for _, raw := range p.Attachments {
				b, _ := json.Marshal(raw)
				var att app.Attachment
				_ = json.Unmarshal(b, &att)
				atts = append(atts, att)
			}
			atts = app.WithRequestID(atts, p.RequestID)
			if err := a.StartSendOpts(p.Session, p.Text, p.Plan, atts); err != nil {
				if errors.Is(err, app.ErrQueued) {
					return map[string]any{"ok": true, "queued": true}, nil
				}
				return nil, err
			}
			return map[string]any{"ok": true, "async": true}, nil
		}
		out, err := a.SendOpts(ctx, p.Session, p.Text, nil, a.Hub.Publish, p.Plan)
		if err != nil {
			return nil, err
		}
		return map[string]any{"text": out}, nil
	case "turn.retry":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		if err := a.RetrySession(p.Session); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	case "turn.interrupt":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true, "state": a.StopTurn(p.Session).State}, nil
	case "turn.stop":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.StopTurn(p.Session), nil
	case "trajectory.page":
		var p struct {
			Session string `json:"session"`
			Before  int64  `json:"before"`
			Turns   int    `json:"turns"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.TrajectoryPage(id, p.Before, p.Turns)
	case "trajectory.outline":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.TrajectoryOutline(id)
	case "trajectory.around":
		var p struct {
			Session string `json:"session"`
			Seq     int64  `json:"seq"`
			Turns   int    `json:"turns"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.TrajectoryAround(id, p.Seq, p.Turns)
	case "live.since":
		var p struct {
			Session string `json:"session"`
			After   int64  `json:"after"`
		}
		_ = json.Unmarshal(params, &p)
		id := p.Session
		if strings.TrimSpace(id) != "" {
			resolved, err := a.ResolveSessionID(id)
			if err != nil {
				return nil, err
			}
			id = resolved
		}
		return a.LiveSince(id, p.After), nil
	case "live.sync":
		var p struct {
			Session string `json:"session"`
			After   int64  `json:"after"`
		}
		_ = json.Unmarshal(params, &p)
		id := p.Session
		if strings.TrimSpace(id) != "" {
			resolved, err := a.ResolveSessionID(id)
			if err != nil {
				return nil, err
			}
			id = resolved
		}
		return a.LiveSync(id, p.After), nil
	case "trajectory.get":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.Trajectory(id)
	case "trace.get":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.SessionTrace(id)
	case "spill.get":
		var p struct {
			Session string `json:"session"`
			ID      string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.SpillBlob(id, p.ID)
	case "session.dump":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.DumpSession(p.Session)
	case "session.list":
		return a.ListSessionIndex(), nil
	case "session.resolve":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": id}, nil
	case "approvals.list":
		return a.Gate.Pending(), nil
	case "approvals.resolve", "approval.respond":
		var p struct {
			ID       string `json:"id"`
			Decision string `json:"decision"`
			Answer   string `json:"answer"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ResolveApprovalAnswer(p.ID, p.Decision, p.Answer)
	case "harness.get":
		return a.HarnessState()
	case "harness.lineage":
		return a.HarnessLineage()
	case "harness.reveal":
		var p struct {
			Hash string `json:"hash"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.RevealHarness(p.Hash)
	case "harness.checkout":
		var p struct {
			Hash string `json:"hash"`
			L3   bool   `json:"l3"`
		}
		_ = json.Unmarshal(params, &p)
		if err := a.CheckoutOpts(p.Hash, app.CheckoutOpts{ConfirmL3: p.L3}); err != nil {
			return nil, err
		}
		return map[string]any{"active": a.ActiveHash()}, nil
	case "harness.rollback":
		if err := a.Rollback(); err != nil {
			return nil, err
		}
		return map[string]any{"active": a.ActiveHash()}, nil
	case "harness.diff":
		var p struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		_ = json.Unmarshal(params, &p)
		return a.DiffDetail(p.A, p.B)
	case "thread.list":
		return a.ListSessions()
	case "thread.resume":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ResumeSession(p.Session)
	case "thread.rename":
		var p struct {
			Session string `json:"session"`
			Title   string `json:"title"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.RenameSession(p.Session, p.Title)
	case "thread.fork":
		var p struct {
			Session string `json:"session"`
			From    string `json:"from"`
		}
		_ = json.Unmarshal(params, &p)
		if strings.TrimSpace(p.From) != "" {
			return a.ForkSessionFrom(p.Session, p.From)
		}
		return a.ForkSession(p.Session)
	case "thread.delete":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.DeleteSession(p.Session)
	case "thread.archive":
		var p struct {
			Session  string `json:"session"`
			Archived *bool  `json:"archived"`
		}
		_ = json.Unmarshal(params, &p)
		archived := true
		if p.Archived != nil {
			archived = *p.Archived
		}
		return a.ArchiveSession(p.Session, archived)
	case "thread.pin":
		var p struct {
			Session string `json:"session"`
			Pinned  bool   `json:"pinned"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PinSession(p.Session, p.Pinned)
	case "thread.search":
		var p struct {
			Query           string `json:"query"`
			IncludeArchived bool   `json:"include_archived"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SearchSessions(p.Query, p.IncludeArchived)
	case "thread.export":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		text, err := a.ExportSession(p.Session)
		return map[string]any{"markdown": text}, err
	case "thread.model.set":
		var p struct {
			Session string `json:"session"`
			Model   string `json:"model"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionModel(p.Session, p.Model)
	case "thread.auth.set":
		var p struct {
			Session string `json:"session"`
			Mode    string `json:"mode"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionAuthMode(p.Session, p.Mode)
	case "thread.workspace.set":
		var p struct {
			Session   string `json:"session"`
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionWorkspace(p.Session, p.Workspace)
	case "thread.isolate.set":
		var p struct {
			Session string `json:"session"`
			Isolate bool   `json:"isolate"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionIsolate(p.Session, p.Isolate)
	case "thread.skills.pin":
		var p struct {
			Session string   `json:"session"`
			Names   []string `json:"names"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionPinnedSkills(p.Session, p.Names)
	case "thread.compact":
		var p struct {
			Session string `json:"session"`
			Focus   string `json:"focus"`
		}
		_ = json.Unmarshal(params, &p)
		note, err := a.CompactSessionFocus(p.Session, p.Focus)
		return map[string]any{"note": note}, err
	case "thread.queue.list":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.QueueList(p.Session), nil
	case "thread.queue.cancel":
		var p struct {
			Session string `json:"session"`
			ID      string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": a.QueueCancel(p.Session, p.ID)}, nil
	case "thread.queue.reorder":
		var p struct {
			Session string `json:"session"`
			ID      string `json:"id"`
			Delta   int    `json:"delta"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": a.QueueReorder(p.Session, p.ID, p.Delta)}, nil
	case "thread.worktree.apply":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ApplySessionWorktree(p.Session)
	case "thread.connectors.set":
		var p struct {
			Session  string   `json:"session"`
			Accounts []string `json:"accounts"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionConnectors(p.Session, p.Accounts)
	case "turn.steer":
		var p struct {
			Session string `json:"session"`
			Text    string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.Steer(p.Session, p.Text)
	case "fs.search":
		var p struct {
			Workspace string `json:"workspace"`
			Query     string `json:"query"`
			Limit     int    `json:"limit"`
		}
		_ = json.Unmarshal(params, &p)
		ws := p.Workspace
		if ws == "" {
			ws = a.Workspace()
		}
		return runtime.FuzzySearch(ws, p.Query, p.Limit), nil
	case "fs.tree":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		ws := p.Workspace
		if ws == "" {
			ws = a.Workspace()
		}
		return runtime.ListTree(ws, 4000), nil
	case "skills.list":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ListSkills(p.Workspace), nil
	case "skills.get":
		var p struct {
			Workspace string `json:"workspace"`
			Name      string `json:"name"`
		}
		_ = json.Unmarshal(params, &p)
		sk := a.GetSkill(p.Workspace, p.Name)
		if sk == nil {
			return nil, fmt.Errorf("unknown skill")
		}
		return sk, nil
	case "skills.market":
		var p struct {
			Refresh bool   `json:"refresh"`
			Hub     string `json:"hub"`
			Query   string `json:"query"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SkillMarket(p.Refresh, p.Hub, p.Query)
	case "models.catalog":
		var p struct {
			Refresh bool `json:"refresh"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ModelCatalog(p.Refresh)
	case "skills.install":
		var p struct {
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(params, &p)
		return a.InstallMarketSkill(p.Slug)
	case "skills.uninstall":
		var p struct {
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.UninstallMarketSkill(p.Slug)
	case "packs.list":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ListPacks(p.Workspace), nil
	case "packs.install":
		var p struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		}
		_ = json.Unmarshal(params, &p)
		return a.InstallPack(p.ID, p.Path)
	case "packs.uninstall":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.UninstallPack(p.ID)
	case "packs.enable":
		var p struct {
			ID      string `json:"id"`
			Scope   string `json:"scope"`
			Enabled bool   `json:"enabled"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.EnablePack(p.ID, p.Scope, p.Enabled)
	case "packs.update":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.UpdatePack(p.ID)
	case "host.open":
		var p struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		}
		_ = json.Unmarshal(params, &p)
		switch p.Kind {
		case "editor":
			return map[string]any{"ok": true}, a.OpenInEditor(p.Path)
		case "terminal":
			return map[string]any{"ok": true}, a.OpenWorkspaceTerminal(p.Path)
		case "url":
			return map[string]any{"ok": true}, a.OpenURL(p.Path)
		default:
			return map[string]any{"ok": true}, a.OpenPath(p.Path)
		}
	case "key.status":
		return a.Vault.Status(), nil
	case "mcp.stop":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.StopMCP(p.Name)
	case "mcp.list":
		return map[string]any{"servers": a.MCP.Info(), "tools": a.MCP.Tools()}, nil
	case "logs.tail":
		var p struct {
			Limit     int    `json:"limit"`
			Level     string `json:"level"`
			Component string `json:"component"`
			Session   string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		out := a.Logs(p.Limit)
		if a.Log != nil && (p.Level != "" || p.Component != "" || p.Session != "") {
			recs, _ := a.Log.Tail(p.Limit, diaglog.Filter{Level: p.Level, Component: p.Component, SessionID: p.Session})
			out["lines"] = recs
			out["text"] = diaglog.FormatRecords(recs)
		}
		return out, nil
	case "journal.tail":
		var p struct {
			Limit int `json:"limit"`
		}
		_ = json.Unmarshal(params, &p)
		return a.JournalTail(p.Limit), nil
	case "logs.frontend":
		var p struct {
			Events []diaglog.FrontendEvent `json:"events"`
		}
		_ = json.Unmarshal(params, &p)
		if a.Log != nil {
			_ = a.Log.AppendFrontend(p.Events)
		}
		return map[string]any{"ok": true}, nil
	case "logs.export":
		var p struct {
			Dest string `json:"dest"`
		}
		_ = json.Unmarshal(params, &p)
		path, err := a.ExportDiagnostics(p.Dest)
		if err != nil {
			return nil, err
		}
		return map[string]any{"path": path}, nil
	case "logs.diagnose":
		var p struct {
			Session string `json:"session"`
			Limit   int    `json:"limit"`
		}
		_ = json.Unmarshal(params, &p)
		note, err := a.DiagnoseSession(p.Session, p.Limit)
		if err != nil {
			return nil, err
		}
		return map[string]any{"text": note}, nil
	case "doctor":
		return a.Doctor(), nil
	case "about":
		h := a.Health()
		h["goos"] = goruntime.GOOS
		return h, nil
	case "update.check":
		return a.CheckUpdate(), nil
	case "playbook.get":
		return a.Playbook()
	case "playbook.rate":
		var p struct {
			ID      string `json:"id"`
			Helpful bool   `json:"helpful"`
		}
		_ = json.Unmarshal(params, &p)
		return a.RatePlaybook(p.ID, p.Helpful)
	case "workspace.diff":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		text, err := a.WorkspaceDiff(p.Workspace)
		if err != nil {
			return nil, err
		}
		return map[string]any{"diff": text}, nil
	case "eval.run_safety":
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		return a.RunEvalOpts(tctx, nil, true)
	case "eval.run":
		var p struct {
			Snapshot string `json:"snapshot"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		if strings.TrimSpace(p.Snapshot) != "" {
			return a.RunEvalAt(tctx, nil, p.Snapshot)
		}
		return a.RunEval(tctx, nil)
	case "eval.best_of_n":
		var p struct {
			N int `json:"n"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.BestOfN(tctx, p.N, nil)
	case "eval.best_of_models":
		var p struct {
			Models []string `json:"models"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.BestOfModels(tctx, p.Models, nil)
	case "eval.run_tb":
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		return a.RunEvalTB(tctx, nil)
	case "eval.run_sealed":
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.RunEvalSealed(tctx, nil)
	case "eval.run_transfer":
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.RunEvalTransfer(tctx, nil)
	case "eval.run_index":
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.RunEvalIndex(tctx, nil)
	case "eval.run_behavior":
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		return a.RunEvalBehavior(tctx, nil)
	case "eval.last":
		return a.LastEval(), nil
	case "evolve.last":
		return a.LastEvolve(), nil
	case "eval.baselines":
		var p struct {
			N      int     `json:"n"`
			Sealed bool    `json:"sealed"`
			MaxUSD float64 `json:"max_usd"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.SearchBaselines(tctx, nil, p.N, app.EvolveRun{Sealed: p.Sealed, MaxUSD: p.MaxUSD, Baselines: p.N})
	case "evolve.run":
		var p struct {
			K         int     `json:"k"`
			Rounds    int     `json:"rounds"`
			Promote   bool    `json:"promote"`
			Sealed    bool    `json:"sealed"`
			Behavior  bool    `json:"behavior"`
			Index     bool    `json:"index"`
			Baselines int     `json:"baselines"`
			MaxUSD    float64 `json:"max_usd"`
		}
		_ = json.Unmarshal(params, &p)
		timeout := 15 * time.Minute
		if p.Rounds > 1 {
			timeout = time.Duration(p.Rounds) * 15 * time.Minute
			if timeout > 2*time.Hour {
				timeout = 2 * time.Hour
			}
		}
		tctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return a.EvolveWith(tctx, nil, nil, app.EvolveRun{
			K: p.K, Rounds: p.Rounds, PromoteActive: p.Promote, Sealed: p.Sealed,
			Behavior: p.Behavior, IndexTransfer: p.Index, Baselines: p.Baselines, MaxUSD: p.MaxUSD,
		})
	case "evolve.overlay":
		var p struct {
			Session     string          `json:"session"`
			Task        string          `json:"task"`
			MaxUSD      float64         `json:"max_usd"`
			MaxWall     json.RawMessage `json:"max_wall"`
			ReuseTurn   *bool           `json:"reuse_turn"`
			Solver      string          `json:"solver"`
			Promote     bool            `json:"promote"`
			Sealed      bool            `json:"sealed"`
			Behavior    bool            `json:"behavior"`
			Index       bool            `json:"index"`
			Baselines   int             `json:"baselines"`
			K           int             `json:"k"`
			OverlayK    int             `json:"overlay_k"`
			OverlayMode string          `json:"overlay_mode"`
			Rounds      int             `json:"rounds"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		var mix []string
		if p.Promote {
			mix = append(mix, "promote")
		}
		if p.Sealed {
			mix = append(mix, "sealed")
		}
		if p.Behavior {
			mix = append(mix, "behavior")
		}
		if p.Index {
			mix = append(mix, "index")
		}
		if p.Baselines > 0 {
			mix = append(mix, "baselines")
		}
		if p.K != 0 {
			mix = append(mix, "k")
		}
		if p.Rounds != 0 {
			mix = append(mix, "rounds")
		}
		if err := app.OverlayFlagConflict(mix...); err != nil {
			return nil, err
		}
		wall, err := parseOverlayWall(p.MaxWall)
		if err != nil {
			return nil, err
		}
		reuse := true
		if p.ReuseTurn != nil {
			reuse = *p.ReuseTurn
		}
		ok := p.OverlayK
		if ok < 1 {
			ok = 1
		}
		if ok > 4 {
			ok = 4
		}
		timeout := 15 * time.Minute * time.Duration(ok)
		if wall > timeout {
			timeout = wall
		}
		if timeout > time.Hour {
			timeout = time.Hour
		}
		tctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return a.OverlaySession(tctx, app.OverlayRun{
			SessionID: p.Session,
			Task:      p.Task,
			ReuseTurn: reuse,
			K:         p.OverlayK,
			Mode:      p.OverlayMode,
			MaxUSD:    p.MaxUSD,
			MaxWall:   wall,
			Client:    overlayFixtureSolver(p.Solver),
		})
	case "evolve.library":
		var p struct {
			Session string `json:"session"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.OverlayLibrary(p.Session)
	case "turn.running":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"running": a.Running(p.Session)}, nil
	case "turn.running_ids":
		return a.RunningIDs(), nil
	case "turn.running_status":
		return a.RunningStatus(), nil
	case "context.get":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ContextUsage(p.Session), nil
	case "context.inventory":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ContextInventory(p.Session), nil
	case "context.pin":
		var p struct {
			Session string `json:"session"`
			Kind    string `json:"kind"`
			Key     string `json:"key"`
			Pinned  *bool  `json:"pinned"`
		}
		_ = json.Unmarshal(params, &p)
		on := true
		if p.Pinned != nil {
			on = *p.Pinned
		}
		return a.ContextPin(p.Session, p.Kind, p.Key, on)
	case "thread.plan":
		var p struct {
			Session string `json:"session"`
			Text    string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionPlan(p.Session, p.Text)
	case "thread.restore_files":
		var p struct {
			Session string `json:"session"`
			From    string `json:"from"`
		}
		_ = json.Unmarshal(params, &p)
		note, err := a.RestoreSessionFiles(p.Session, p.From)
		return map[string]any{"note": note}, err
	case "config.get":
		return a.Config, nil
	case "config.set":
		return map[string]any{"ok": true}, a.ApplyConfigPatch(params)
	case "provider.list":
		return a.ListChatProviders(), nil
	case "provider.upsert":
		var p struct {
			Provider app.ChatProviderIn `json:"provider"`
			APIKey   string             `json:"api_key"`
		}
		_ = json.Unmarshal(params, &p)
		if p.Provider.Vendor == "" && p.Provider.Endpoint == "" && p.Provider.ID == "" {
			_ = json.Unmarshal(params, &p.Provider)
		}
		return a.UpsertChatProvider(p.Provider, p.APIKey)
	case "provider.delete":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.DeleteChatProvider(p.ID)
	case "provider.setDefault":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.SetDefaultChatProvider(p.ID)
	case "provider.test":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		if strings.TrimSpace(p.ID) != "" {
			return a.TestChatProvider(p.ID), nil
		}
		return a.TestProvider(), nil
	case "mcp.replace":
		var p struct {
			Servers []app.MCPServerConfig `json:"servers"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ReplaceMCP(p.Servers)
	case "key.set":
		var p struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(params, &p)
		a.Vault.Set("default", strings.TrimSpace(p.Value))
		return map[string]any{"ok": true}, nil
	case "mcp.start":
		var p struct {
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true, "tools": a.MCP.Tools()}, a.StartMCP(p.Name, p.Command, p.Args)
	case "archive.list":
		return a.Archive.List(), nil
	case "plugins.list":
		return map[string]any{"fibers": a.Kernel.Fibers(), "wasm": a.WASM.List(), "mcp": a.MCP.Info(), "tools": a.MCP.Tools()}, nil
	case "fiber.unload":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(params, &p)
		if f := a.Kernel.Fiber(p.Name); f != nil {
			return map[string]any{"ok": true}, f.Dispose()
		}
		return map[string]any{"ok": true}, a.WASM.Unload(p.Name)
	case "workspace.hunks":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.WorkspaceHunks(p.Workspace)
	case "workspace.preview":
		var p struct {
			Workspace string `json:"workspace"`
			Path      string `json:"path"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PreviewWorkspaceFile(p.Workspace, p.Path)
	case "workspace.blob":
		var p struct {
			Workspace string `json:"workspace"`
			Path      string `json:"path"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ReadWorkspaceBlob(p.Workspace, p.Path)
	case "workspace.apply_hunks":
		var p struct {
			Workspace string   `json:"workspace"`
			IDs       []string `json:"ids"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ApplyWorkspaceHunks(p.Workspace, p.IDs)
	case "workspace.reverse_hunks":
		var p struct {
			Workspace string   `json:"workspace"`
			IDs       []string `json:"ids"`
			Snapshot  string   `json:"snapshot"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ReverseWorkspaceHunks(p.Workspace, p.IDs, p.Snapshot)
	case "update.status":
		return a.UpdateStatus(), nil
	case "update.apply":
		var p struct {
			Exe string `json:"exe"`
		}
		_ = json.Unmarshal(params, &p)
		return applyUpdateResult(a.ApplyStagedUpdate(p.Exe))
	case "inbox.list":
		return a.InboxList(), nil
	case "inbox.dismiss":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": a.InboxDismiss(p.ID)}, nil
	case "inbox.read":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		a.InboxMarkRead(p.ID)
		return map[string]any{"ok": true}, nil
	case "projects.list":
		return a.ProjectsList(), nil
	case "projects.create":
		var p project.Project
		_ = json.Unmarshal(params, &p)
		return a.ProjectCreate(p), nil
	case "memory.list":
		var p struct {
			Q    string `json:"q"`
			Kind string `json:"kind"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryList(p.Q, p.Kind), nil
	case "memory.promote":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryPromote(p.ID), nil
	case "memory.forget":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryForget(p.ID), nil
	case "memory.write":
		var p struct {
			Kind    string `json:"kind"`
			Text    string `json:"text"`
			Project string `json:"project"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryWrite(p.Kind, p.Text, p.Project), nil
	case "schedule.list":
		return a.ScheduleList(), nil
	case "schedule.create":
		var p schedule.Job
		_ = json.Unmarshal(params, &p)
		return a.ScheduleCreate(p), nil
	case "schedule.cancel":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ScheduleCancel(p.ID), nil
	case "connectors.list":
		return a.ConnectorsList(), nil
	case "connectors.catalog":
		return a.ConnectorCatalog(), nil
	case "connectors.connect":
		var p connector.Account
		_ = json.Unmarshal(params, &p)
		return a.ConnectorConnect(p), nil
	case "connectors.disconnect":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": a.ConnectorDisconnect(p.ID)}, nil
	case "connectors.auth_url":
		var p struct {
			Provider string `json:"provider"`
			ClientID string `json:"client_id"`
			Redirect string `json:"redirect"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ConnectorAuthURL(p.Provider, p.ClientID, p.Redirect)
	case "connectors.token":
		var p struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ConnectorStoreToken(p.ID, p.Token)
	case "connectors.complete":
		var p struct {
			Provider string `json:"provider"`
			Code     string `json:"code"`
			ClientID string `json:"client_id"`
			Secret   string `json:"secret"`
			Redirect string `json:"redirect"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ConnectorComplete(p.Provider, p.Code, p.ClientID, p.Secret, p.Redirect)
	case "schedule.trigger":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.TriggerWebhook(p.ID)
	case "isolation.report":
		return a.IsolationReport(), nil
	case "runtime.pwsh.status":
		return a.PwshStatus(), nil
	case "runtime.pwsh.install":
		return a.PwshInstall(), nil
	case "review.queue":
		return a.ReviewQueue(), nil
	case "browser.view":
		return a.BrowserView(), nil
	case "browser.takeover":
		var p struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.BrowserTakeoverAt(p.Path)
	case "phone.status":
		return a.PhoneStatus(), nil
	case "phone.approve":
		var p struct {
			ID       string `json:"id"`
			Decision string `json:"decision"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.PhoneApprove(p.ID, p.Decision)
	case "phone.steer":
		var p struct {
			Session string `json:"session"`
			Text    string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.PhoneSteer(p.Session, p.Text)
	case "mcp.start_http":
		var p struct {
			Name     string `json:"name"`
			Endpoint string `json:"endpoint"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.StartMCPHTTP(p.Name, p.Endpoint)
	case "expert.admit":
		var p struct {
			Dir string `json:"dir"`
		}
		_ = json.Unmarshal(params, &p)
		return a.AdmitExpert(p.Dir)
	case "expert.distill":
		var p struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Body        string `json:"body"`
		}
		_ = json.Unmarshal(params, &p)
		path, err := a.DistillExpert(p.Name, p.Description, p.Body)
		return map[string]any{"path": path}, err
	case "computer.allow":
		var p struct {
			App string `json:"app"`
		}
		_ = json.Unmarshal(params, &p)
		a.ComputerAllow(p.App)
		return map[string]any{"ok": true}, nil
	case "skins.list":
		return a.ListSkins(), nil
	case "skins.get":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.GetSkin(p.ID)
	case "skins.import":
		var p struct {
			BytesB64 string `json:"bytes_b64"`
			Path     string `json:"path"`
		}
		_ = json.Unmarshal(params, &p)
		if strings.TrimSpace(p.Path) != "" {
			return a.ImportSkinPath(p.Path)
		}
		raw, err := decodeSkinB64(p.BytesB64)
		if err != nil {
			return nil, err
		}
		return a.ImportSkin(raw)
	case "skins.export":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		name, body, err := a.ExportSkin(p.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"filename": name, "bytes_b64": encodeSkinB64(body)}, nil
	case "skins.delete":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.DeleteSkin(p.ID)
	case "skins.activate":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ActivateSkin(p.ID)
	case "skins.resolve":
		var p struct {
			ID   string `json:"id"`
			Mode string `json:"mode"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ResolveSkin(p.ID, p.Mode)
	case "skins.save":
		var in app.SkinSaveIn
		if err := json.Unmarshal(params, &in); err != nil {
			return nil, err
		}
		return a.SaveSkin(in)
	case "personal.snapshot":
		return a.PersonalSnapshot(), nil
	case "personal.answer":
		var p struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalAnswer(p.ID, p.Text)
	case "personal.decide":
		var p struct {
			ID      string `json:"id"`
			Hash    string `json:"hash"`
			Approve bool   `json:"approve"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalDecide(p.ID, p.Hash, p.Approve)
	case "personal.idea":
		var p struct {
			ID     string `json:"id"`
			Action string `json:"action"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalIdea(p.ID, p.Action)
	case "personal.goal":
		var p struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Milestones  []string `json:"milestones"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalGoal(p.Title, p.Description, p.Milestones)
	case "personal.watch":
		var p struct {
			Title     string `json:"title"`
			URL       string `json:"url"`
			Condition string `json:"condition"`
			Value     string `json:"value"`
			Interval  int    `json:"interval_minutes"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalWatch(p.Title, p.URL, p.Condition, p.Value, p.Interval)
	case "personal.cancel":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalCancel(p.ID)
	case "personal.choice":
		var p struct {
			ID     string `json:"id"`
			Option string `json:"option"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalChoice(p.ID, p.Option)
	case "personal.ideas.refresh":
		return map[string]any{"ok": true}, a.PersonalRefreshIdeas()
	case "personal.pause":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalPause(p.ID)
	case "personal.resume":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalResume(p.ID)
	case "personal.retry":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalRetry(p.ID)
	case "personal.goal.status":
		var p struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalGoalStatus(p.ID, p.Status)
	case "personal.milestone":
		var p struct {
			GoalID      string `json:"goal_id"`
			MilestoneID string `json:"milestone_id"`
			Done        bool   `json:"done"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalMilestone(p.GoalID, p.MilestoneID, p.Done)
	case "personal.monitor.status":
		var p struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PersonalMonitorStatus(p.ID, p.Status)
	case "app.pause.set":
		var p struct {
			Paused bool `json:"paused"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true, "paused": p.Paused}, a.SetPaused(p.Paused)
	case "pages.list":
		var p struct {
			Q     string `json:"q"`
			Space string `json:"space_id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PagesList(p.Q, p.Space), nil
	case "pages.get":
		var p struct {
			ID    string `json:"id"`
			Space string `json:"space_id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PagesGet(p.Space, p.ID)
	case "pages.save":
		var p struct {
			pages.Page
			Expected int `json:"expected_revision"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PagesSave(p.Page, p.Expected)
	case "pages.delete":
		var p struct {
			ID    string `json:"id"`
			Space string `json:"space_id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.PagesDelete(p.Space, p.ID)
	case "pages.search":
		var p struct {
			Q     string `json:"q"`
			Space string `json:"space_id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PagesList(p.Q, p.Space), nil
	case "pages.reviews":
		return a.PagesReviews(), nil
	case "pages.propose":
		var p struct {
			Session  string `json:"session"`
			Title    string `json:"title"`
			Content  string `json:"content"`
			ID       string `json:"id"`
			Space    string `json:"space_id"`
			Parent   string `json:"parent_id"`
			Expected int    `json:"expected_revision"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PagesPropose(p.Session, p.Title, p.Content, p.ID, p.Space, p.Parent, p.Expected)
	case "pages.review.decide":
		var p struct {
			ID      string `json:"id"`
			Hash    string `json:"hash"`
			Approve bool   `json:"approve"`
		}
		_ = json.Unmarshal(params, &p)
		pg, rev, err := a.PagesDecide(p.ID, p.Hash, p.Approve)
		if err != nil {
			return nil, err
		}
		return map[string]any{"page": pg, "review": rev}, nil
	case "profiles.list":
		return a.ProfilesList(), nil
	case "profiles.get":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ProfilesGet(p.ID)
	case "profiles.save":
		var p profile.Profile
		_ = json.Unmarshal(params, &p)
		return a.ProfilesSave(p)
	case "thread.profile.set":
		var p struct {
			Session string `json:"session"`
			Profile string `json:"profile_id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionProfile(p.Session, p.Profile)
	case "thread.page.set":
		var p struct {
			Session string `json:"session"`
			Page    string `json:"page_id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionPage(p.Session, p.Page)
	case "schedule.retry":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ScheduleRetry(p.ID)
	case "connectors.mention":
		var p struct {
			Account string `json:"account"`
			Text    string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ConnectorMention(p.Account, p.Text)
	case "voice.transcribe":
		var p struct {
			Filename string `json:"filename"`
			Audio    string `json:"audio"`
		}
		_ = json.Unmarshal(params, &p)
		raw, err := decodeSkinB64(p.Audio)
		if err != nil {
			return nil, err
		}
		text, err := a.VoiceTranscribe(p.Filename, raw)
		return map[string]any{"text": text}, err
	case "voice.speak":
		var p struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		b, ctype, err := a.VoiceSpeak(p.Text)
		if err != nil {
			return nil, err
		}
		return map[string]any{"audio": encodeSkinB64(b), "content_type": ctype}, nil
	case "voice.receipt":
		var p struct {
			Session  string `json:"session"`
			Duration int    `json:"duration_sec"`
			Text     string `json:"transcript"`
		}
		_ = json.Unmarshal(params, &p)
		a.VoiceReceipt(p.Session, p.Duration, p.Text)
		return map[string]any{"ok": true}, nil
	case "idol.call":
		var p struct {
			Op      string `json:"op"`
			Payload string `json:"payload"`
		}
		_ = json.Unmarshal(params, &p)
		raw, err := a.IdolCall(p.Op, p.Payload)
		if err != nil {
			return nil, err
		}
		var decoded any
		if json.Unmarshal([]byte(raw), &decoded) != nil {
			return raw, nil
		}
		return decoded, nil
	case "orchestrator.view":
		if a == nil {
			return nil, fmt.Errorf("app is not running")
		}
		return a.OrchestratorView(), nil
	case "orchestrator.decide":
		var decide struct {
			TriggerID string `json:"triggerId"`
			Kind      string `json:"kind"`
			Reason    string `json:"reason"`
		}
		if err := json.Unmarshal(params, &decide); err != nil {
			return nil, err
		}
		if a == nil {
			return nil, fmt.Errorf("app is not running")
		}
		return map[string]any{"ok": true}, a.OrchestratorDecide(decide.TriggerID, decide.Kind, decide.Reason)
	case "orchestrator.mode":
		var mode struct {
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(params, &mode); err != nil {
			return nil, err
		}
		if a == nil {
			return nil, fmt.Errorf("app is not running")
		}
		return map[string]any{"ok": true}, a.OrchestratorMode(mode.Mode)
	case "orchestrator.paused":
		var paused struct {
			Paused bool `json:"paused"`
		}
		if err := json.Unmarshal(params, &paused); err != nil {
			return nil, err
		}
		if a == nil {
			return nil, fmt.Errorf("app is not running")
		}
		return map[string]any{"ok": true}, a.OrchestratorPaused(paused.Paused)
	case "orchestrator.admission":
		var cells struct {
			ListedUnaddressed   string `json:"listedUnaddressed"`
			UnlistedAddressed   string `json:"unlistedAddressed"`
			UnlistedUnaddressed string `json:"unlistedUnaddressed"`
		}
		if err := json.Unmarshal(params, &cells); err != nil {
			return nil, err
		}
		if a == nil {
			return nil, fmt.Errorf("app is not running")
		}
		return map[string]any{"ok": true}, a.OrchestratorAdmission(cells.ListedUnaddressed, cells.UnlistedAddressed, cells.UnlistedUnaddressed)
	case "im.snapshot":
		if a.IM == nil {
			return nil, fmt.Errorf("messaging is not running")
		}
		return a.IM.Snapshot(), nil
	case "im.settings":
		var p im.Settings
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if a.IM == nil {
			return nil, fmt.Errorf("messaging is not running")
		}
		return map[string]any{"ok": true}, a.IM.SaveSettings(p)
	case "im.account.save":
		var p struct {
			Account im.Account        `json:"account"`
			Secrets map[string]string `json:"secrets"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if a.IM == nil {
			return nil, fmt.Errorf("messaging is not running")
		}
		return a.IM.SaveAccount(p.Account, p.Secrets)
	case "im.account.delete":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if a.IM == nil {
			return nil, fmt.Errorf("messaging is not running")
		}
		return map[string]any{"ok": true}, a.IM.DeleteAccount(p.ID)
	case "im.send":
		var p struct {
			AccountID string `json:"account_id"`
			ChatID    string `json:"chat_id"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if a.IM == nil {
			return nil, fmt.Errorf("messaging is not running")
		}
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		return map[string]any{"ok": true}, a.IM.Push(ctx, p.AccountID, p.ChatID, p.Text)
	default:
		return nil, errMethod(method)
	}
}

// errMethod 是未知 JSON-RPC 方法的错误类型，字符串内容就是方法名。
type errMethod string

// Error 返回 method not found: 加上方法名。
func (e errMethod) Error() string { return "method not found: " + string(e) }

// decodeSkinB64 去掉空白。空字符串返回 empty skin payload。若逗号前带有 base64，则去掉 data URL 头，再用标准 Base64 解码。
func decodeSkinB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty skin payload")
	}
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(s[:i], "base64") {
		s = s[i+1:]
	}
	return base64.StdEncoding.DecodeString(s)
}

// encodeSkinB64 用标准 Base64 编码，不换行。
// b 传给 base64.StdEncoding.EncodeToString。
func encodeSkinB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// applyUpdateResult 在 err 为空时返回 ok。update.ErrQuitNeeded 返回 ok 和 quit。update.ErrOpenedPackage 返回 ok 和 opened。其他错误原样返回。
func applyUpdateResult(err error) (map[string]any, error) {
	if err == nil {
		return map[string]any{"ok": true}, nil
	}
	if errors.Is(err, update.ErrQuitNeeded) {
		return map[string]any{"ok": true, "quit": true}, nil
	}
	if errors.Is(err, update.ErrOpenedPackage) {
		return map[string]any{"ok": true, "opened": true}, nil
	}
	return nil, err
}

// asMap 把 JSON 解成 map。空参数、null 或解出来是 nil 时返回空 map。
// params 长度为 0 时返回空 map，否则交给 json.Unmarshal。结果是 nil 时也返回空 map。
func asMap(params json.RawMessage) map[string]any {
	m := map[string]any{}
	if len(params) == 0 {
		return m
	}
	_ = json.Unmarshal(params, &m)
	if m == nil {
		return map[string]any{}
	}
	return m
}

// overlayFixtureSolver 在名字是 heuristic 或 fixture 时返回 HeuristicSolver，是 prompt-sensitive 或 prompt_sensitive 时返回 PromptSensitiveSolver。其他名字返回 nil。
// name 去掉空白并转成小写后比较。等于 heuristic 或 fixture 时返回 HeuristicSolver，等于 prompt-sensitive 或 prompt_sensitive 时返回 PromptSensitiveSolver，其他返回 nil。
func overlayFixtureSolver(name string) runtime.Client {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "heuristic", "fixture":
		return runtime.HeuristicSolver{}
	case "prompt-sensitive", "prompt_sensitive":
		return runtime.PromptSensitiveSolver{}
	default:
		return nil
	}
}

// parseOverlayWall 把 max_wall 收成 time.Duration。空或 null 返回 0。字符串按 time.ParseDuration 解析，空字符串返回 0。数字按秒换算，小于等于 0 返回 0。两种都不是则返回 overlay: max_wall 错误。
// raw 长度为 0 或内容是 null 时返回 0。能解成字符串时去掉空白，空字符串返回 0，否则交给 time.ParseDuration。能解成数字时，小于等于 0 返回 0，否则按秒换算。
func parseOverlayWall(raw json.RawMessage) (time.Duration, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, nil
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("overlay: max_wall: %w", err)
		}
		return d, nil
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, fmt.Errorf("overlay: max_wall")
	}
	if n <= 0 {
		return 0, nil
	}
	return time.Duration(n * float64(time.Second)), nil
}
