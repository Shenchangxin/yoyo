package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/video"
)

func (a *App) startCanvasAgent(turn video.CanvasAgentTurn) error {
	if a == nil || a.Video == nil {
		return fmt.Errorf("视频模块还没有就绪")
	}
	sid, err := a.canvasAgentSession(turn.CanvasID)
	if err != nil {
		return err
	}
	if err := a.Video.BindCanvasSession(sid, turn.CanvasID); err != nil {
		return err
	}
	a.Video.RememberCanvasAgentSession(turn.CanvasID, sid, turn.RunID)
	if strings.TrimSpace(turn.Model) != "" {
		_, _ = a.SetSessionModel(sid, turn.Model)
	}
	mode := capability.AuthAsk
	if turn.Permission == "auto" {
		mode = capability.AuthFull
	}
	if _, err := a.SetSessionAuthMode(sid, mode); err != nil {
		return err
	}
	if len(turn.SkillIDs) > 0 {
		if _, err := a.SetSessionPinnedSkills(sid, turn.SkillIDs); err != nil {
			return err
		}
	}
	if err := a.ensureCanvasAgentWatch(sid); err != nil {
		return err
	}
	err = a.StartSend(sid, canvasAgentPrompt(turn), false)
	if errors.Is(err, ErrQueued) {
		return nil
	}
	return err
}

func (a *App) stopCanvasAgent(sessionID string) {
	if a == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	_ = a.Interrupt(sessionID)
}

func (a *App) decideCanvasAgent(approvalID, decision string) error {
	if a == nil || a.Gate == nil {
		return fmt.Errorf("没有审批网关")
	}
	d := capability.Deny
	if decision == "approve" {
		d = capability.Once
	}
	return a.Gate.Resolve(approvalID, d)
}

func (a *App) canvasAgentSession(canvasID string) (string, error) {
	if v, ok := a.agentSess.Load(canvasID); ok {
		if id, _ := v.(string); id != "" {
			if _, err := a.GetSession(id); err == nil {
				return id, nil
			}
		}
	}
	meta, err := a.NewSessionOn(a.VideoWorkspace(), session.ChannelVideo)
	if err != nil {
		return "", err
	}
	meta.Title = "画布助手"
	_ = a.writeSession(meta)
	a.agentSess.Store(canvasID, meta.ID)
	return meta.ID, nil
}

func (a *App) ensureCanvasAgentWatch(sessionID string) error {
	if a == nil || a.Hub == nil {
		return fmt.Errorf("对话事件通道还没有就绪")
	}
	if _, loaded := a.agentWatch.LoadOrStore(sessionID, true); loaded {
		return nil
	}
	ch, unsub := a.Hub.Subscribe(sessionID)
	go a.canvasAgentLoop(sessionID, ch, unsub)
	return nil
}

func (a *App) canvasAgentLoop(sessionID string, ch <-chan trace.Event, unsub func()) {
	defer unsub()
	defer a.agentWatch.Delete(sessionID)
	var buf strings.Builder
	var epoch int
	for ev := range ch {
		if a == nil || a.Video == nil {
			return
		}
		switch ev.Type {
		case trace.TypeTurnStart:
			epoch = a.Video.CanvasAgentEpoch(sessionID)
			buf.Reset()
		case trace.TypeAssistantStream:
			if epoch == 0 || a.Video.CanvasAgentEpoch(sessionID) != epoch {
				continue
			}
			delta := assistantDelta(ev)
			if delta == "" {
				continue
			}
			buf.WriteString(delta)
			a.Video.CanvasAgentStream(sessionID, epoch, buf.String())
		case trace.TypeTurnEnd:
			if epoch == 0 || a.Video.CanvasAgentEpoch(sessionID) != epoch {
				continue
			}
			text := buf.String()
			if ev.Payload != nil {
				if t, _ := ev.Payload["text"].(string); strings.TrimSpace(t) != "" {
					text = t
				}
			}
			okTurn := true
			if ev.Payload != nil {
				if v, exists := ev.Payload["ok"]; exists {
					okTurn, _ = v.(bool)
				}
			}
			a.Video.CanvasAgentFinish(sessionID, epoch, text, okTurn)
			buf.Reset()
		case trace.TypeError:
			if epoch == 0 || a.Video.CanvasAgentEpoch(sessionID) != epoch {
				continue
			}
			a.Video.CanvasAgentFail(sessionID, epoch, agentEventMessage(ev))
		case trace.TypeToolResult:
			if epoch == 0 || a.Video.CanvasAgentEpoch(sessionID) != epoch || ev.Payload == nil {
				continue
			}
			name, _ := ev.Payload["name"].(string)
			id, _ := ev.Payload["id"].(string)
			a.Video.CanvasAgentTool(sessionID, epoch, name, id)
		case trace.TypeApproval:
			if epoch == 0 || a.Video.CanvasAgentEpoch(sessionID) != epoch {
				continue
			}
			a.Video.CanvasAgentApproval(sessionID, epoch, ev.Payload)
		}
	}
}

func canvasAgentPrompt(turn video.CanvasAgentTurn) string {
	var b strings.Builder
	fmt.Fprintf(&b, "你正在 Yoyo 的视频画布里工作。当前画布 id 是 %s。请用已挂上的画布工具查看和修改这块画布，不要把工作区里的普通文件当成画布。\n", turn.CanvasID)
	switch turn.Permission {
	case "read_only":
		b.WriteString("当前是只读权限：不要调用会修改画布、整理节点、生成媒体或写入文件的工具，只回答和查看。\n")
	case "auto":
		b.WriteString("当前允许直接修改画布。生成媒体如果触发确认，等待用户批准后再继续。\n")
	default:
		b.WriteString("修改画布和生成媒体前按系统的确认流程执行。\n")
	}
	if len(turn.SkillIDs) > 0 {
		fmt.Fprintf(&b, "用户点名的技能：%s。需要时读取这些技能再动手。\n", strings.Join(turn.SkillIDs, "、"))
	}
	b.WriteString("\n用户要求：\n")
	b.WriteString(turn.Prompt)
	return b.String()
}

func assistantDelta(ev trace.Event) string {
	if ev.Payload == nil {
		return ""
	}
	chunk, _ := ev.Payload["chunk"].(map[string]any)
	if chunk == nil {
		return ""
	}
	if typ, _ := chunk["type"].(string); typ != "" && typ != "text-delta" {
		return ""
	}
	text, _ := chunk["text"].(string)
	return text
}

func agentEventMessage(ev trace.Event) string {
	if ev.Payload == nil {
		return "画布助手执行失败"
	}
	for _, key := range []string{"error", "title", "detail", "message", "text"} {
		if s, _ := ev.Payload[key].(string); strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return "画布助手执行失败"
}
