// desktop 是绑到 Wails 窗口的服务。RPC 非空时界面经 JSON-RPC 跟工作者说话，否则直接用同进程的 App。
// YOYO_ISOLATE=1 时 OpenIsolated 再启动同一可执行文件并带 YOYO_WORKER=1。health 超过 8 秒或调用失败则退回同进程。这不是安全沙箱，工作者仍握有 vault 和工具。
package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/update"
)

// Service 是绑给 Wails 的门面。RPC 非空时窗口经 JSON-RPC 调用工作者，该进程由 YOYO_WORKER=1 启动，卡住的回合不会冻住窗口。否则使用同进程的 App。
type Service struct {
	App                *app.App
	RPC                *api.LineClient
	cmd                *exec.Cmd
	gui                *application.App
	win                application.Window
	companion          application.Window
	companionDismissed atomic.Bool
	companionPlaced    atomic.Bool
	companionClosing   atomic.Bool
	companionClamping  atomic.Bool
	companionCaption   atomic.Bool
	companionCall      atomic.Bool
	companionFocus     atomic.Bool
	companionHomeX     atomic.Int32
	companionHomeY     atomic.Int32
	companionHomeOK    atomic.Bool
	companionShapeGen  atomic.Uint64
	agentBusy          atomic.Bool
	lastPulseSession   atomic.Value
	cursorWatch        atomic.Bool
	tray               *application.SystemTray
	menuLocale         string
	chromeMu           sync.Mutex
	quitting           atomic.Bool
	toastsSinceClear   atomic.Bool
	workerLog          *os.File
}

// NewService 包住同进程的 App。RPC 留空，表示不走工作者。
// a 写入 App。
func NewService(a *app.App) *Service { return &Service{App: a} }

// decode 先把 v 编成 JSON，再解进 T。传入的 err 非空时直接返回 T 的零值和该错误。
func decode[T any](v any, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return zero, err
	}
	var out T
	return out, json.Unmarshal(b, &out)
}

// call 在 RPC 为空时返回 no rpc。否则把方法和参数交给 LineClient.Call。
// method 在 RPC 非 nil 时传给 LineClient.Call。
// params 在 RPC 非 nil 时传给 LineClient.Call。
func (s *Service) call(method string, params any) (any, error) {
	if s.RPC == nil {
		return nil, fmt.Errorf("no rpc")
	}
	return s.RPC.Call(method, params)
}

// Health 在 RPC 非空时调用 health，并把 isolation.sandbox_badge_ok 抄到 isolated；isolation_kind 还空时，再把 isolation.os.kind 抄过来。否则返回 App.Health。
func (s *Service) Health() map[string]any {
	if s.RPC != nil {
		v, err := s.call("health", nil)
		m, _ := decode[map[string]any](v, err)
		if m == nil {
			m = map[string]any{}
		}
		if iso, ok := m["isolation"].(map[string]any); ok {
			if badge, ok := iso["sandbox_badge_ok"].(bool); ok {
				m["isolated"] = badge
			}
			if m["isolation_kind"] == nil {
				if os, ok := iso["os"].(map[string]any); ok {
					if kind, ok := os["kind"].(string); ok {
						m["isolation_kind"] = kind
					}
				}
			}
		}
		return m
	}
	return s.App.Health()
}

// GetConfig 在 RPC 非空时调用 config.get。解码失败时返回零值 Config。否则返回 App.Config。
func (s *Service) GetConfig() app.Config {
	if s.RPC != nil {
		v, err := s.call("config.get", nil)
		c, _ := decode[app.Config](v, err)
		return c
	}
	return s.App.Config
}

// SetConfig 先把补丁编成 JSON。RPC 非空时调用 config.set，成功后再 applyDesktop。同进程则 ApplyConfigPatch，成功后同样 applyDesktop。
// cfg 编成 JSON。RPC 非空时作为 config.set 的参数。同进程把编码结果传给 ApplyConfigPatch。
func (s *Service) SetConfig(cfg map[string]any) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if s.RPC != nil {
		_, err := s.call("config.set", cfg)
		if err == nil {
			s.applyDesktop(s.GetConfig())
		}
		return err
	}
	if err := s.App.ApplyConfigPatch(b); err != nil {
		return err
	}
	s.applyDesktop(s.App.Config)
	return nil
}

// SetLocale 通过 SetConfig 写入 locale。
func (s *Service) SetLocale(locale string) error {
	return s.SetConfig(map[string]any{"locale": locale})
}

// BeginQuit 把 quitting 标成真，关窗口时就不再收进托盘。
func (s *Service) BeginQuit() {
	s.quitting.Store(true)
}

// Quitting 返回 BeginQuit 是否已经调用。
func (s *Service) Quitting() bool {
	return s.quitting.Load()
}

// applyDesktop 记下菜单语言，按配置设置窗口置顶，同步桌面宠物，写入开机启动，并重建菜单。
// cfg 的 Locale 写入 menuLocale。窗口非 nil 时 AlwaysOnTop 传给 SetAlwaysOnTop。整份传给 syncCompanion，StartAtLogin 传给 SetStartAtLogin。
func (s *Service) applyDesktop(cfg app.Config) {
	s.menuLocale = cfg.Locale
	if s.win != nil {
		s.win.SetAlwaysOnTop(cfg.AlwaysOnTop)
	}
	s.syncCompanion(cfg)
	_ = SetStartAtLogin(cfg.StartAtLogin)
	s.RebuildMenus()
}

// SetAPIKey 去掉 value 两端的空白。RPC 非空时调用 key.set。否则写入 vault 的 default。
func (s *Service) SetAPIKey(value string) {
	value = strings.TrimSpace(value)
	if s.RPC != nil {
		_, _ = s.call("key.set", map[string]any{"value": value})
		return
	}
	s.App.Vault.Set("default", value)
}

// CreateSession 调用 CreateSessionOn，频道固定为 agent。
// workspace 和 agent 传给 CreateSessionOn。
func (s *Service) CreateSession(workspace string) (app.SessionMeta, error) {
	return s.CreateSessionOn(workspace, "agent")
}

// CreateSessionOn 在 RPC 非空时调用 thread.start，否则 App.NewSessionOn。成功后发出 yoyo:sessions。
// workspace 在 RPC 非空时放进 thread.start，否则传给 App.NewSessionOn。
// channel 在 RPC 非空时放进 thread.start，否则传给 App.NewSessionOn。
func (s *Service) CreateSessionOn(workspace, channel string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.start", map[string]any{"workspace": workspace, "channel": channel}))
	} else {
		m, err = s.App.NewSessionOn(workspace, channel)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

// SetSessionChannel 在 RPC 非空时调用 thread.channel.set，否则 App.SetSessionChannel。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.channel.set 的 session。否则传给 App.SetSessionChannel。
func (s *Service) SetSessionChannel(id, channel string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.channel.set", map[string]any{"session": id, "channel": channel}))
	} else {
		m, err = s.App.SetSessionChannel(id, channel)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

// ListSessions 在 RPC 非空时调用 thread.list，否则 App.ListSessions。
func (s *Service) ListSessions() ([]app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[[]app.SessionMeta](s.call("thread.list", nil))
	}
	return s.App.ListSessions()
}

// ForkSession 调用 ForkSessionFrom，起点为空。
// id 和空字符串传给 ForkSessionFrom。
func (s *Service) ForkSession(id string) (app.SessionMeta, error) {
	return s.ForkSessionFrom(id, "")
}

// ForkSessionFrom 在 RPC 非空时调用 thread.fork。同进程在 from 非空时走 ForkSessionFrom，否则走 ForkSession。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.fork 的 session。from 非空时和 from 一起传给 App.ForkSessionFrom，否则传给 App.ForkSession。
func (s *Service) ForkSessionFrom(id, from string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.fork", map[string]any{"session": id, "from": from}))
	} else if strings.TrimSpace(from) != "" {
		m, err = s.App.ForkSessionFrom(id, from)
	} else {
		m, err = s.App.ForkSession(id)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

// RenameSession 在 RPC 非空时调用 thread.rename，再 thread.list 找回该会话并发出 yoyo:sessions。列表失败或找不到时，返回只含 id 和标题的元数据；找不到时发出的载荷是 id 字符串。同进程改名后取 GetSession，成功则发出 yoyo:sessions。
// title 在 RPC 非空时放进 thread.rename。列表失败或找不到时写入返回元数据的 Title。同进程传给 App.RenameSession。
func (s *Service) RenameSession(id, title string) (app.SessionMeta, error) {
	var err error
	if s.RPC != nil {
		_, err = s.call("thread.rename", map[string]any{"session": id, "title": title})
		if err != nil {
			return app.SessionMeta{}, err
		}
		list, e2 := decode[[]app.SessionMeta](s.call("thread.list", nil))
		if e2 != nil {
			return app.SessionMeta{ID: id, Title: title}, nil
		}
		for _, m := range list {
			if m.ID == id {
				s.emitSessions(m)
				return m, nil
			}
		}
		s.emitSessions(id)
		return app.SessionMeta{ID: id, Title: title}, nil
	}
	err = s.App.RenameSession(id, title)
	if err != nil {
		return app.SessionMeta{}, err
	}
	m, err := s.App.GetSession(id)
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

// Playbook 在 RPC 非空时调用 playbook.get，否则走 App.Playbook。
func (s *Service) Playbook() (any, error) {
	if s.RPC != nil {
		return s.call("playbook.get", nil)
	}
	return s.App.Playbook()
}

// RatePlaybook 在 RPC 非空时调用 playbook.rate，否则走 App.RatePlaybook。
// id 在 RPC 非空时放进 playbook.rate，否则传给 App.RatePlaybook。
// helpful 在 RPC 非空时放进 playbook.rate，否则传给 App.RatePlaybook。
func (s *Service) RatePlaybook(id string, helpful bool) (any, error) {
	if s.RPC != nil {
		return s.call("playbook.rate", map[string]any{"id": id, "helpful": helpful})
	}
	return s.App.RatePlaybook(id, helpful)
}

// WorkspaceDiff 在 RPC 非空时调用 workspace.diff，并读取结果里的 diff 字符串。否则走 App.WorkspaceDiff。
func (s *Service) WorkspaceDiff(workspace string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("workspace.diff", map[string]any{"workspace": workspace})
		if err != nil {
			return "", err
		}
		m, _ := v.(map[string]any)
		if m != nil {
			if d, ok := m["diff"].(string); ok {
				return d, nil
			}
		}
		return fmt.Sprint(v), nil
	}
	return s.App.WorkspaceDiff(workspace)
}

// WorkspaceHunks 在 RPC 非空时调用 workspace.hunks，否则走 App.WorkspaceHunks。
func (s *Service) WorkspaceHunks(workspace string) (any, error) {
	if s.RPC != nil {
		return s.call("workspace.hunks", map[string]any{"workspace": workspace})
	}
	return s.App.WorkspaceHunks(workspace)
}

// PreviewWorkspaceFile 在 RPC 非空时调用 workspace.preview，否则走 App.PreviewWorkspaceFile。
// path 在 RPC 非空时放进 workspace.preview，否则传给 App.PreviewWorkspaceFile。
func (s *Service) PreviewWorkspaceFile(workspace, path string) (any, error) {
	if s.RPC != nil {
		return s.call("workspace.preview", map[string]any{"workspace": workspace, "path": path})
	}
	return s.App.PreviewWorkspaceFile(workspace, path)
}

// ReadWorkspaceBlob 在 RPC 非空时调用 workspace.blob，否则走 App.ReadWorkspaceBlob。
// path 在 RPC 非空时放进 workspace.blob，否则传给 App.ReadWorkspaceBlob。
func (s *Service) ReadWorkspaceBlob(workspace, path string) (any, error) {
	if s.RPC != nil {
		return s.call("workspace.blob", map[string]any{"workspace": workspace, "path": path})
	}
	return s.App.ReadWorkspaceBlob(workspace, path)
}

// ApplyHunks 在 RPC 非空时调用 workspace.apply_hunks，否则走 App.ApplyWorkspaceHunks。
// ids 在 RPC 非空时放进 workspace.apply_hunks，否则传给 App.ApplyWorkspaceHunks。
func (s *Service) ApplyHunks(workspace string, ids []string) error {
	if s.RPC != nil {
		_, err := s.call("workspace.apply_hunks", map[string]any{"workspace": workspace, "ids": ids})
		return err
	}
	return s.App.ApplyWorkspaceHunks(workspace, ids)
}

// ReverseHunks 在 RPC 非空时调用 workspace.reverse_hunks，否则走 App.ReverseWorkspaceHunks。
// ids 在 RPC 非空时放进 workspace.reverse_hunks，否则传给 App.ReverseWorkspaceHunks。
// snapshot 在 RPC 非空时放进 workspace.reverse_hunks，否则传给 App.ReverseWorkspaceHunks。
func (s *Service) ReverseHunks(workspace string, ids []string, snapshot string) error {
	if s.RPC != nil {
		_, err := s.call("workspace.reverse_hunks", map[string]any{"workspace": workspace, "ids": ids, "snapshot": snapshot})
		return err
	}
	return s.App.ReverseWorkspaceHunks(workspace, ids, snapshot)
}

// RunEvalSafety 在 RPC 非空时调用 eval.run_safety。同进程用 10 分钟超时调用 RunEvalOpts，安全档为真。
func (s *Service) RunEvalSafety() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_safety", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEvalOpts(ctx, nil, true)
}

// RunEvalTB 在 RPC 非空时调用 eval.run_tb。同进程用 10 分钟超时调用 RunEvalTB。
func (s *Service) RunEvalTB() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_tb", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEvalTB(ctx, nil)
}

// RunEvalSealed 在 RPC 非空时调用 eval.run_sealed。同进程用 20 分钟超时调用 RunEvalSealed。
func (s *Service) RunEvalSealed() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_sealed", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	return s.App.RunEvalSealed(ctx, nil)
}

// RunEvalTransfer 在 RPC 非空时调用 eval.run_transfer。同进程用 20 分钟超时调用 RunEvalTransfer。
func (s *Service) RunEvalTransfer() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_transfer", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	return s.App.RunEvalTransfer(ctx, nil)
}

// RunEvalIndex 在 RPC 非空时调用 eval.run_index。同进程用 20 分钟超时调用 RunEvalIndex。
func (s *Service) RunEvalIndex() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_index", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	return s.App.RunEvalIndex(ctx, nil)
}

// RunEvalBehavior 在 RPC 非空时调用 eval.run_behavior。同进程用 10 分钟超时调用 RunEvalBehavior。
func (s *Service) RunEvalBehavior() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_behavior", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEvalBehavior(ctx, nil)
}

// LastEval 在 RPC 非空时调用 eval.last。同进程返回 App.LastEval，错误为 nil。
func (s *Service) LastEval() (any, error) {
	if s.RPC != nil {
		return s.call("eval.last", nil)
	}
	return s.App.LastEval(), nil
}

// LastEvolve 在 RPC 非空时调用 evolve.last。同进程返回 App.LastEvolve，错误为 nil。
func (s *Service) LastEvolve() (any, error) {
	if s.RPC != nil {
		return s.call("evolve.last", nil)
	}
	return s.App.LastEvolve(), nil
}

// Trajectory 在 RPC 非空时调用 trajectory.get。同进程先 ResolveSessionID，再取轨迹。
// id 在 RPC 非空时作为 trajectory.get 的 session。否则传给 ResolveSessionID，结果再传给 App.Trajectory。
func (s *Service) Trajectory(id string) ([]trace.Event, error) {
	if s.RPC != nil {
		return decode[[]trace.Event](s.call("trajectory.get", map[string]any{"session": id}))
	}
	resolved, err := s.App.ResolveSessionID(id)
	if err != nil {
		return nil, err
	}
	return s.App.Trajectory(resolved)
}

// TrajectoryPage 在 RPC 非空时调用 trajectory.page，参数是 before 和 turns。同进程先解析会话 id。
// beforeSeq 在 RPC 非空时作为 trajectory.page 的 before。否则传给 App.TrajectoryPage。
func (s *Service) TrajectoryPage(id string, beforeSeq int64, turns int) (runtime.TrajectoryPage, error) {
	if s.RPC != nil {
		return decode[runtime.TrajectoryPage](s.call("trajectory.page", map[string]any{
			"session": id, "before": beforeSeq, "turns": turns,
		}))
	}
	resolved, err := s.App.ResolveSessionID(id)
	if err != nil {
		return runtime.TrajectoryPage{}, err
	}
	return s.App.TrajectoryPage(resolved, beforeSeq, turns)
}

// TrajectoryOutline 在 RPC 非空时调用 trajectory.outline。结果为 nil 时换成空切片。同进程先解析会话 id。
func (s *Service) TrajectoryOutline(id string) ([]trace.OutlineTurn, error) {
	if s.RPC != nil {
		out, err := decode[[]trace.OutlineTurn](s.call("trajectory.outline", map[string]any{"session": id}))
		if out == nil {
			out = []trace.OutlineTurn{}
		}
		return out, err
	}
	resolved, err := s.App.ResolveSessionID(id)
	if err != nil {
		return nil, err
	}
	return s.App.TrajectoryOutline(resolved)
}

// TrajectoryAround 在 RPC 非空时调用 trajectory.around，参数是 seq 和 turns。同进程先解析会话 id。
// userSeq 在 RPC 非空时作为 trajectory.around 的 seq。否则传给 App.TrajectoryAround。
func (s *Service) TrajectoryAround(id string, userSeq int64, turns int) (runtime.TrajectoryPage, error) {
	if s.RPC != nil {
		return decode[runtime.TrajectoryPage](s.call("trajectory.around", map[string]any{
			"session": id, "seq": userSeq, "turns": turns,
		}))
	}
	resolved, err := s.App.ResolveSessionID(id)
	if err != nil {
		return runtime.TrajectoryPage{}, err
	}
	return s.App.TrajectoryAround(resolved, userSeq, turns)
}

// LiveSince 在 RPC 非空时调用 live.since。同进程会先解析非空的会话 id，再返回 App.LiveSince，错误为 nil。
// sessionID 在 RPC 非空时作为 live.since 的 session。同进程去掉空白后非空时传给 ResolveSessionID，再和 after 传给 App.LiveSince。
// after 在 RPC 非空时放进 live.since。否则传给 App.LiveSince。
func (s *Service) LiveSince(sessionID string, after int64) ([]trace.Event, error) {
	if s.RPC != nil {
		return decode[[]trace.Event](s.call("live.since", map[string]any{
			"session": sessionID, "after": after,
		}))
	}
	resolved := strings.TrimSpace(sessionID)
	if resolved != "" {
		id, err := s.App.ResolveSessionID(resolved)
		if err != nil {
			return nil, err
		}
		resolved = id
	}
	return s.App.LiveSince(resolved, after), nil
}

// LiveSync 在 RPC 非空时调用 live.sync，否则解析会话后走 App.LiveSync。结果在事件之外带环形缓冲头 Head、Gap 和 More。Gap 为真表示 after 落在环外。More 为真表示这一页被截到 1024 条，用完还要再拉。
// sessionID 在 RPC 非空时作为 live.sync 的 session。同进程去掉空白后非空时传给 ResolveSessionID，再传给 App.LiveSync。
func (s *Service) LiveSync(sessionID string, after int64) (app.LiveBatch, error) {
	if s.RPC != nil {
		return decode[app.LiveBatch](s.call("live.sync", map[string]any{
			"session": sessionID, "after": after,
		}))
	}
	resolved := strings.TrimSpace(sessionID)
	if resolved != "" {
		id, err := s.App.ResolveSessionID(resolved)
		if err != nil {
			return app.LiveBatch{}, err
		}
		resolved = id
	}
	return s.App.LiveSync(resolved, after), nil
}

// StopTurn 在 RPC 非空时调用 turn.stop。调用失败时返回该会话 id 且 state 为 idle 的 InterruptResult。否则返回 App.StopTurn。
// sessionID 在 RPC 非空时作为 turn.stop 的 session。调用失败时写入 InterruptResult 的 SessionID。否则传给 App.StopTurn。
func (s *Service) StopTurn(sessionID string) app.InterruptResult {
	if s.RPC != nil {
		res, err := decode[app.InterruptResult](s.call("turn.stop", map[string]any{"session": sessionID}))
		if err != nil {
			return app.InterruptResult{SessionID: sessionID, State: "idle"}
		}
		return res
	}
	return s.App.StopTurn(sessionID)
}

// SetRendererMemory 把档位交给 setRendererMemory。当前实现不改 WebView，直接返回 nil。
// level 和窗口一起传给 setRendererMemory。
func (s *Service) SetRendererMemory(level string) error {
	return setRendererMemory(s.win, level)
}

// SessionTrace 在 RPC 非空时调用 trace.get。同进程先解析会话 id。
func (s *Service) SessionTrace(id string) (any, error) {
	if s.RPC != nil {
		return s.call("trace.get", map[string]any{"session": id})
	}
	resolved, err := s.App.ResolveSessionID(id)
	if err != nil {
		return nil, err
	}
	return s.App.SessionTrace(resolved)
}

// SpillBlob 在 RPC 非空时调用 spill.get。同进程先解析会话 id。
// sessionID 在 RPC 非空时作为 spill.get 的 session。否则传给 ResolveSessionID。
// blobID 在 RPC 非空时作为 spill.get 的 id。否则传给 App.SpillBlob。
func (s *Service) SpillBlob(sessionID, blobID string) (any, error) {
	if s.RPC != nil {
		return s.call("spill.get", map[string]any{"session": sessionID, "id": blobID})
	}
	resolved, err := s.App.ResolveSessionID(sessionID)
	if err != nil {
		return nil, err
	}
	return s.App.SpillBlob(resolved, blobID)
}

// DumpSession 在 RPC 非空时调用 session.dump，否则走 App.DumpSession。这里不先把短 id 解析成长 id。
func (s *Service) DumpSession(id string) (any, error) {
	if s.RPC != nil {
		return s.call("session.dump", map[string]any{"session": id})
	}
	return s.App.DumpSession(id)
}

// ListSessionIndex 在 RPC 非空时调用 session.list。同进程返回 App.ListSessionIndex，错误为 nil。
func (s *Service) ListSessionIndex() (any, error) {
	if s.RPC != nil {
		return s.call("session.list", nil)
	}
	return s.App.ListSessionIndex(), nil
}

// ResolveSessionID 在 RPC 非空时调用 session.resolve，并读取结果里的 id。id 为空时返回 session resolve: empty id。否则走 App.ResolveSessionID。
func (s *Service) ResolveSessionID(id string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("session.resolve", map[string]any{"session": id})
		if err != nil {
			return "", err
		}
		m, _ := v.(map[string]any)
		if m != nil {
			if got, ok := m["id"].(string); ok {
				return got, nil
			}
		}
		return "", fmt.Errorf("session resolve: empty id")
	}
	return s.App.ResolveSessionID(id)
}

// Send 在 RPC 非空时调用 turn.start，且 wait 为真，返回结果里的 text。同进程走 App.Send，发布函数是 Hub.Publish。
// sessionID 在 RPC 非空时作为 turn.start 的 session。否则传给 App.Send。
func (s *Service) Send(sessionID, text string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("turn.start", map[string]any{"session": sessionID, "text": text, "wait": true})
		if err != nil {
			return "", err
		}
		m, _ := v.(map[string]any)
		if m != nil {
			if t, ok := m["text"].(string); ok {
				return t, nil
			}
		}
		return "", nil
	}
	return s.App.Send(context.Background(), sessionID, text, nil, s.App.Hub.Publish)
}

// StartSend 调用 StartSendOpts，附件为 nil。
// sessionID 传给 StartSendOpts。
// text 传给 StartSendOpts。
// plan 传给 StartSendOpts。
func (s *Service) StartSend(sessionID, text string, plan bool) error {
	return s.StartSendOpts(sessionID, text, plan, nil)
}

// StartSendOpts 在 RPC 非空时调用 turn.start，wait 为假，并带上附件。同进程走 App.StartSendOpts，遇到 ErrQueued 时当成成功返回。
// sessionID 在 RPC 非空时作为 turn.start 的 session。否则传给 App.StartSendOpts。
// text 在 RPC 非空时放进 turn.start。否则传给 App.StartSendOpts。
// plan 在 RPC 非空时放进 turn.start。否则传给 App.StartSendOpts。
// atts 在 RPC 非空时作为 turn.start 的 attachments。否则传给 App.StartSendOpts。
func (s *Service) StartSendOpts(sessionID, text string, plan bool, atts []app.Attachment) error {
	if s.RPC != nil {
		_, err := s.call("turn.start", map[string]any{"session": sessionID, "text": text, "plan": plan, "wait": false, "attachments": atts})
		return err
	}
	err := s.App.StartSendOpts(sessionID, text, plan, atts)
	if errors.Is(err, app.ErrQueued) {
		return nil
	}
	return err
}

// RetrySession 在 RPC 非空时调用 turn.retry，否则走 App.RetrySession。
// sessionID 在 RPC 非空时作为 turn.retry 的 session。否则传给 App.RetrySession。
func (s *Service) RetrySession(sessionID string) error {
	if s.RPC != nil {
		_, err := s.call("turn.retry", map[string]any{"session": sessionID})
		return err
	}
	return s.App.RetrySession(sessionID)
}

// Interrupt 在 sessionID 为空时对 RunningIDs 里的每个会话再调一次自己。RPC 非空时调用 turn.interrupt，否则走 App.Interrupt。
func (s *Service) Interrupt(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		for _, id := range s.RunningIDs() {
			_ = s.Interrupt(id)
		}
		return nil
	}
	if s.RPC != nil {
		_, err := s.call("turn.interrupt", map[string]any{"session": sessionID})
		return err
	}
	return s.App.Interrupt(sessionID)
}

// Running 在 RPC 非空时调用 turn.running，并读取 running 字段。否则走 App.Running。
// sessionID 在 RPC 非空时作为 turn.running 的 session。否则传给 App.Running。
func (s *Service) Running(sessionID string) bool {
	if s.RPC != nil {
		v, err := s.call("turn.running", map[string]any{"session": sessionID})
		m, _ := decode[map[string]any](v, err)
		b, _ := m["running"].(bool)
		return b
	}
	return s.App.Running(sessionID)
}

// RunningIDs 在 RPC 非空时调用 turn.running_ids。结果为 nil 时返回空切片。否则走 App.RunningIDs。
func (s *Service) RunningIDs() []string {
	if s.RPC != nil {
		v, err := s.call("turn.running_ids", nil)
		ids, _ := decode[[]string](v, err)
		if ids == nil {
			return []string{}
		}
		return ids
	}
	return s.App.RunningIDs()
}

// RunningStatus 在 RPC 非空时调用 turn.running_status。结果为 nil 时返回空切片。App 为空时返回 nil。否则走 App.RunningStatus。
func (s *Service) RunningStatus() []app.RunStatus {
	if s.RPC != nil {
		v, err := s.call("turn.running_status", nil)
		out, _ := decode[[]app.RunStatus](v, err)
		if out == nil {
			return []app.RunStatus{}
		}
		return out
	}
	if s.App == nil {
		return nil
	}
	return s.App.RunningStatus()
}

// ContextUsage 在 RPC 非空时调用 context.get，忽略错误。否则走 App.ContextUsage。
// sessionID 在 RPC 非空时作为 context.get 的 session。否则传给 App.ContextUsage。
func (s *Service) ContextUsage(sessionID string) any {
	if s.RPC != nil {
		v, _ := s.call("context.get", map[string]any{"session": sessionID})
		return v
	}
	return s.App.ContextUsage(sessionID)
}

// ContextInventory 在 RPC 非空时调用 context.inventory，忽略错误。否则走 App.ContextInventory。
// sessionID 在 RPC 非空时作为 context.inventory 的 session。否则传给 App.ContextInventory。
func (s *Service) ContextInventory(sessionID string) any {
	if s.RPC != nil {
		v, _ := s.call("context.inventory", map[string]any{"session": sessionID})
		return v
	}
	return s.App.ContextInventory(sessionID)
}

// SetSessionPlan 在 RPC 非空时调用 thread.plan，否则走 App.SetSessionPlan。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.plan 的 session。否则传给 App.SetSessionPlan。
// text 在 RPC 非空时放进 thread.plan。否则传给 App.SetSessionPlan。
func (s *Service) SetSessionPlan(id, text string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.plan", map[string]any{"session": id, "text": text}))
	} else {
		m, err = s.App.SetSessionPlan(id, text)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

// RestoreSessionFiles 在 RPC 非空时调用 thread.restore_files，并返回 note。结果不是带 note 的对象时，把值格式化成字符串。否则走 App.RestoreSessionFiles。
// id 在 RPC 非空时作为 thread.restore_files 的 session。否则传给 App.RestoreSessionFiles。
// from 在 RPC 非空时放进 thread.restore_files。否则传给 App.RestoreSessionFiles。
func (s *Service) RestoreSessionFiles(id, from string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("thread.restore_files", map[string]any{"session": id, "from": from})
		if err != nil {
			return "", err
		}
		if m, ok := v.(map[string]any); ok {
			note, _ := m["note"].(string)
			return note, nil
		}
		return fmt.Sprint(v), nil
	}
	return s.App.RestoreSessionFiles(id, from)
}

// ContextPin 在 RPC 非空时调用 context.pin，否则走 App.ContextPin。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 context.pin 的 session。否则传给 App.ContextPin。
// kind 在 RPC 非空时放进 context.pin。否则传给 App.ContextPin。
// key 在 RPC 非空时放进 context.pin。否则传给 App.ContextPin。
// pinned 在 RPC 非空时放进 context.pin。否则传给 App.ContextPin。
func (s *Service) ContextPin(id, kind, key string, pinned bool) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("context.pin", map[string]any{"session": id, "kind": kind, "key": key, "pinned": pinned}))
	} else {
		m, err = s.App.ContextPin(id, kind, key, pinned)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

// PendingApprovals 在 RPC 非空时调用 approvals.list，忽略错误。否则返回 Gate.Pending。
func (s *Service) PendingApprovals() any {
	if s.RPC != nil {
		v, _ := s.call("approvals.list", nil)
		return v
	}
	return s.App.Gate.Pending()
}

// ResolveApproval 调用 ResolveApprovalAnswer，回答文本为空。
// id 传给 ResolveApprovalAnswer。
// decision 传给 ResolveApprovalAnswer。
func (s *Service) ResolveApproval(id, decision string) error {
	return s.ResolveApprovalAnswer(id, decision, "")
}

// ResolveApprovalAnswer 在 RPC 非空时调用 approvals.resolve，并带上 answer。否则走 App.ResolveApprovalAnswer。
// id 在 RPC 非空时放进 approvals.resolve。否则传给 App.ResolveApprovalAnswer。
// decision 在 RPC 非空时放进 approvals.resolve。否则传给 App.ResolveApprovalAnswer。
func (s *Service) ResolveApprovalAnswer(id, decision, answer string) error {
	if s.RPC != nil {
		_, err := s.call("approvals.resolve", map[string]any{"id": id, "decision": decision, "answer": answer})
		return err
	}
	return s.App.ResolveApprovalAnswer(id, decision, answer)
}

// StartMCP 在 RPC 非空时调用 mcp.start，否则走 App.StartMCP。
// name 在 RPC 非空时放进 mcp.start。否则传给 App.StartMCP。
// command 在 RPC 非空时放进 mcp.start。否则传给 App.StartMCP。
// args 在 RPC 非空时放进 mcp.start。否则传给 App.StartMCP。
func (s *Service) StartMCP(name, command string, args []string) error {
	if s.RPC != nil {
		_, err := s.call("mcp.start", map[string]any{"name": name, "command": command, "args": args})
		return err
	}
	return s.App.StartMCP(name, command, args)
}

// Harness 在 RPC 非空时调用 harness.get，否则走 App.HarnessState。
func (s *Service) Harness() (map[string]any, error) {
	if s.RPC != nil {
		return decode[map[string]any](s.call("harness.get", nil))
	}
	return s.App.HarnessState()
}

// HarnessLineage 在 RPC 非空时调用 harness.lineage，否则走 App.HarnessLineage。
func (s *Service) HarnessLineage() (app.HarnessLineage, error) {
	if s.RPC != nil {
		return decode[app.HarnessLineage](s.call("harness.lineage", nil))
	}
	return s.App.HarnessLineage()
}

// RevealHarness 在 RPC 非空时调用 harness.reveal，否则走 App.RevealHarness。
// hash 在 RPC 非空时放进 harness.reveal。否则传给 App.RevealHarness。
func (s *Service) RevealHarness(hash string) error {
	if s.RPC != nil {
		_, err := s.call("harness.reveal", map[string]any{"hash": hash})
		return err
	}
	return s.App.RevealHarness(hash)
}

// Checkout 在 RPC 非空时调用 harness.checkout，不带 l3。否则走 App.Checkout。
// hash 在 RPC 非空时放进 harness.checkout。否则传给 App.Checkout。
func (s *Service) Checkout(hash string) error {
	if s.RPC != nil {
		_, err := s.call("harness.checkout", map[string]any{"hash": hash})
		return err
	}
	return s.App.Checkout(hash)
}

// CheckoutL3 在 RPC 非空时调用 harness.checkout，且 l3 为真。否则走 CheckoutOpts，ConfirmL3 为真。
// hash 在 RPC 非空时放进 harness.checkout。否则传给 App.CheckoutOpts。
func (s *Service) CheckoutL3(hash string) error {
	if s.RPC != nil {
		_, err := s.call("harness.checkout", map[string]any{"hash": hash, "l3": true})
		return err
	}
	return s.App.CheckoutOpts(hash, app.CheckoutOpts{ConfirmL3: true})
}

// Rollback 在 RPC 非空时调用 harness.rollback，否则走 App.Rollback。
func (s *Service) Rollback() error {
	if s.RPC != nil {
		_, err := s.call("harness.rollback", nil)
		return err
	}
	return s.App.Rollback()
}

// Diff 在 RPC 非空时调用 harness.diff，并返回其中的 text。否则走 App.Diff。
// b 在 RPC 非空时放进 harness.diff。否则传给 App.Diff。
func (s *Service) Diff(a, b string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("harness.diff", map[string]any{"a": a, "b": b})
		m, err2 := decode[map[string]any](v, err)
		if err2 != nil {
			return "", err2
		}
		t, _ := m["text"].(string)
		return t, nil
	}
	return s.App.Diff(a, b)
}

// DiffDetail 在 RPC 非空时调用 harness.diff，返回整份对象。否则走 App.DiffDetail。
// b 在 RPC 非空时放进 harness.diff。否则传给 App.DiffDetail。
func (s *Service) DiffDetail(a, b string) (map[string]any, error) {
	if s.RPC != nil {
		return decode[map[string]any](s.call("harness.diff", map[string]any{"a": a, "b": b}))
	}
	return s.App.DiffDetail(a, b)
}

// RunEval 在 RPC 非空时调用 eval.run。同进程用 10 分钟超时调用 RunEval。
func (s *Service) RunEval() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEval(ctx, nil)
}

// RunEvalAt 在 RPC 非空时调用 eval.run 并带上 snapshot。同进程用 10 分钟超时；snapshot 为空时走 RunEval，否则走 RunEvalAt。
func (s *Service) RunEvalAt(snapshot string) (any, error) {
	snapshot = strings.TrimSpace(snapshot)
	if s.RPC != nil {
		return s.call("eval.run", map[string]any{"snapshot": snapshot})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if snapshot == "" {
		return s.App.RunEval(ctx, nil)
	}
	return s.App.RunEvalAt(ctx, nil, snapshot)
}

// BestOfN 在 RPC 非空时调用 eval.best_of_n。同进程用 20 分钟超时调用 BestOfN。
func (s *Service) BestOfN(n int) (app.BestOfNReport, error) {
	if s.RPC != nil {
		return decode[app.BestOfNReport](s.call("eval.best_of_n", map[string]any{"n": n}))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	return s.App.BestOfN(ctx, n, nil)
}

// BestOfModels 在 RPC 非空时调用 eval.best_of_models。同进程用 20 分钟超时调用 BestOfModels。
func (s *Service) BestOfModels(models []string) (app.BestOfNReport, error) {
	if s.RPC != nil {
		return decode[app.BestOfNReport](s.call("eval.best_of_models", map[string]any{"models": models}))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	return s.App.BestOfModels(ctx, models, nil)
}

// Evolve 调用 EvolveK，k 为 3。
func (s *Service) Evolve() (evolve.CycleResult, error) {
	return s.EvolveK(3)
}

// EvolveK 调用 EvolveRun，轮数为 1，sealed 和 promote 都为假。
// k 和 1、false、false 传给 EvolveRun。
func (s *Service) EvolveK(k int) (evolve.CycleResult, error) {
	return s.EvolveRun(k, 1, false, false)
}

// EvolveRun 调用 EvolveLab。behavior 和 index 为假，baselines 为 0，maxUSD 为 0。
// k 传给 EvolveLab。
// rounds 传给 EvolveLab。
// sealed 传给 EvolveLab。
// promote 传给 EvolveLab。
func (s *Service) EvolveRun(k, rounds int, sealed, promote bool) (evolve.CycleResult, error) {
	return s.EvolveLab(k, rounds, sealed, promote, false, false, 0, 0)
}

// EvolveLab 在 RPC 非空时调用 evolve.run。同进程超时为每轮 15 分钟，轮数大于 1 时相乘，上限 2 小时，然后 EvolveWith。
// k 在 RPC 非空时放进 evolve.run。否则写入 EvolveRun 的 K。
// rounds 在 RPC 非空时放进 evolve.run。大于 1 时用来乘 15 分钟，上限 2 小时。否则写入 EvolveRun 的 Rounds。
// sealed 在 RPC 非空时放进 evolve.run。否则写入 EvolveRun 的 Sealed。
// promote 在 RPC 非空时放进 evolve.run。否则写入 EvolveRun 的 PromoteActive。
// behavior 在 RPC 非空时放进 evolve.run。否则写入 EvolveRun 的 Behavior。
// index 在 RPC 非空时放进 evolve.run。否则写入 EvolveRun 的 IndexTransfer。
// baselines 在 RPC 非空时放进 evolve.run。否则写入 EvolveRun 的 Baselines。
// maxUSD 在 RPC 非空时作为 evolve.run 的 max_usd。否则写入 EvolveRun 的 MaxUSD。
func (s *Service) EvolveLab(k, rounds int, sealed, promote, behavior, index bool, baselines int, maxUSD float64) (evolve.CycleResult, error) {
	if s.RPC != nil {
		return decode[evolve.CycleResult](s.call("evolve.run", map[string]any{
			"k": k, "rounds": rounds, "sealed": sealed, "promote": promote,
			"behavior": behavior, "index": index, "baselines": baselines, "max_usd": maxUSD,
		}))
	}
	timeout := 15 * time.Minute
	if rounds > 1 {
		timeout = time.Duration(rounds) * 15 * time.Minute
		if timeout > 2*time.Hour {
			timeout = 2 * time.Hour
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.App.EvolveWith(ctx, nil, nil, app.EvolveRun{
		K: k, Rounds: rounds, Sealed: sealed, PromoteActive: promote,
		Behavior: behavior, IndexTransfer: index, Baselines: baselines, MaxUSD: maxUSD,
	})
}

// OverlaySession 在 RPC 非空时调用 evolve.overlay。同进程把小于 1 的 overlayK 按 1 计算超时，每份 15 分钟，上限 1 小时，然后 App.OverlaySession。
// sessionID 在 RPC 非空时作为 evolve.overlay 的 session。否则写入 OverlayRun 的 SessionID。
// task 在 RPC 非空时放进 evolve.overlay。否则写入 OverlayRun 的 Task。
// reuseTurn 在 RPC 非空时作为 evolve.overlay 的 reuse_turn。否则写入 OverlayRun 的 ReuseTurn。
// maxUSD 在 RPC 非空时作为 evolve.overlay 的 max_usd。否则写入 OverlayRun 的 MaxUSD。
// overlayMode 在 RPC 非空时作为 evolve.overlay 的 overlay_mode。否则写入 OverlayRun 的 Mode。
func (s *Service) OverlaySession(sessionID, task string, reuseTurn bool, maxUSD float64, overlayK int, overlayMode string) (evolve.OverlayResult, error) {
	if s.RPC != nil {
		return decode[evolve.OverlayResult](s.call("evolve.overlay", map[string]any{
			"session": sessionID, "task": task, "reuse_turn": reuseTurn, "max_usd": maxUSD, "overlay_k": overlayK, "overlay_mode": overlayMode,
		}))
	}
	ok := overlayK
	if ok < 1 {
		ok = 1
	}
	timeout := 15 * time.Minute * time.Duration(ok)
	if timeout > time.Hour {
		timeout = time.Hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.App.OverlaySession(ctx, app.OverlayRun{
		SessionID: sessionID, Task: task, ReuseTurn: reuseTurn, MaxUSD: maxUSD, K: overlayK, Mode: overlayMode,
	})
}

// OverlayLibrary 在 RPC 非空时调用 evolve.library，否则走 App.OverlayLibrary。
// sessionID 在 RPC 非空时作为 evolve.library 的 session。否则传给 App.OverlayLibrary。
func (s *Service) OverlayLibrary(sessionID string) (evolve.LibraryView, error) {
	if s.RPC != nil {
		return decode[evolve.LibraryView](s.call("evolve.library", map[string]any{"session": sessionID}))
	}
	return s.App.OverlayLibrary(sessionID)
}

// Archive 在 RPC 非空时调用 archive.list。解码失败时返回 nil 切片。否则返回 App.Archive.List。
func (s *Service) Archive() []evolve.Node {
	if s.RPC != nil {
		v, err := s.call("archive.list", nil)
		n, _ := decode[[]evolve.Node](v, err)
		return n
	}
	return s.App.Archive.List()
}

// Plugins 在 RPC 非空时调用 plugins.list。否则返回 fibers、wasm、mcp 和 tools。
func (s *Service) Plugins() map[string]any {
	if s.RPC != nil {
		v, err := s.call("plugins.list", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return map[string]any{"fibers": s.App.Kernel.Fibers(), "wasm": s.App.WASM.List(), "mcp": s.App.MCP.Info(), "tools": s.App.MCP.Tools()}
}

// UnloadFiber 在 RPC 非空时调用 fiber.unload。同进程先 Dispose 同名 fiber，没有则 WASM.Unload。
// name 在 RPC 非空时放进 fiber.unload。否则传给 Kernel.Fiber，找到时 Dispose，否则传给 WASM.Unload。
func (s *Service) UnloadFiber(name string) error {
	if s.RPC != nil {
		_, err := s.call("fiber.unload", map[string]any{"name": name})
		return err
	}
	if f := s.App.Kernel.Fiber(name); f != nil {
		return f.Dispose()
	}
	return s.App.WASM.Unload(name)
}

// ApplyUpdate 在 RPC 非空时调用 update.apply，exe 为空。同进程走 ApplyStagedUpdate。ErrQuitNeeded 变成 quit，ErrOpenedPackage 变成 opened。quit 为真且 gui 非空时发出 yoyo:do-quit。
func (s *Service) ApplyUpdate() error {
	var (
		err error
		out map[string]any
	)
	if s.RPC != nil {
		v, callErr := s.call("update.apply", map[string]any{"exe": ""})
		out, err = decode[map[string]any](v, callErr)
	} else if s.App != nil {
		err = s.App.ApplyStagedUpdate("")
		if errors.Is(err, update.ErrQuitNeeded) {
			out, err = map[string]any{"ok": true, "quit": true}, nil
		} else if errors.Is(err, update.ErrOpenedPackage) {
			out, err = map[string]any{"ok": true, "opened": true}, nil
		}
	}
	if err != nil {
		return err
	}
	if flagTrue(out["quit"]) && s.gui != nil {
		s.gui.Event.Emit("yoyo:do-quit", nil)
	}
	return nil
}

// flagTrue 在值为布尔真，或字符串 true、1 时返回真。其他类型返回假。
// v 是 bool 时返回该值。是字符串时等于 true 或 1 则返回真。其他类型返回假。
func flagTrue(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

// UpdateStatus 在 RPC 非空时调用 update.status。App 非空时走 App.UpdateStatus。两者都空时返回空 map。
func (s *Service) UpdateStatus() map[string]any {
	if s.RPC != nil {
		v, err := s.call("update.status", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	if s.App != nil {
		return s.App.UpdateStatus()
	}
	return map[string]any{}
}

// Close 在工作者进程还在时杀掉它，并关掉工作者日志文件。
func (s *Service) Close() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	if s.workerLog != nil {
		_ = s.workerLog.Close()
		s.workerLog = nil
	}
}
