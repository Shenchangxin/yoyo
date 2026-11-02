// effects_run.go 执行已经落盘的六类副作用。
// 不决定接不接。那一步仍在引擎里。

package superagent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type relayBody struct {
	Ticket   string `json:"ticket"`
	State    string `json:"state"`
	Summary  string `json:"summary"`
	Question string `json:"question"`
}

type noticeBody struct {
	Session string `json:"session"`
	Text    string `json:"text"`
}

// runLaunch 启动一张工单。暂时的网络失败会再试。其他失败结束工单，不再自动启动。
func (rt *Runtime) runLaunch(payload any) error {
	id, _ := payload.(string)
	if strings.TrimSpace(id) == "" || rt == nil {
		return nil
	}
	if rt.ready != nil {
		if err := rt.ready(); err != nil {
			return fmt.Errorf("%w", ErrLater)
		}
	}
	if rt.exec == nil {
		return fmt.Errorf("%w", ErrLater)
	}
	err := rt.exec.Launch(id)
	if err == nil || errors.Is(err, ErrLater) {
		return err
	}
	switch err.Error() {
	case "already", "not_pending", "unknownTicket":
		return nil
	}
	if passingError(err.Error()) {
		return err
	}
	if rt.eng != nil {
		_ = rt.eng.FailStart(id, err.Error())
	}
	return nil
}

// runDelivery 把还没送到工人的输入送进去。工人还没起来时先等。
func (rt *Runtime) runDelivery(payload any) error {
	id, _ := payload.(string)
	if rt == nil || rt.exec == nil {
		return fmt.Errorf("%w", ErrLater)
	}
	return rt.exec.DeliverPending(id)
}

// runAssessment 把一轮评估标成已经交给人。正文不写进左边的对话。
func (rt *Runtime) runAssessment(payload any) error {
	id, _ := payload.(string)
	if rt != nil && rt.eng != nil {
		rt.eng.DeliverEval(id)
	}
	return nil
}

// runReply 发送发件箱里的一条。同一工单前面还有没发出的，就先等。
func (rt *Runtime) runReply(payload any) error {
	id, _ := payload.(string)
	if rt == nil {
		return nil
	}
	return rt.sendOutbound(id)
}

// runRelay 按顺序把需要人看的报告写回原会话。还没接上写回时先等。
func (rt *Runtime) runRelay(payload any) error {
	raw, _ := payload.(string)
	var body relayBody
	if json.Unmarshal([]byte(raw), &body) != nil || body.Ticket == "" {
		return nil
	}
	if rt == nil {
		return nil
	}
	rt.mu.Lock()
	fn := rt.report
	rt.mu.Unlock()
	if fn == nil {
		return fmt.Errorf("%w", ErrLater)
	}
	fn(body.Ticket, body.State, body.Summary, body.Question)
	return nil
}

// runNotice 按顺序把一条通知写进桌面会话，不因此再开一轮路由。
func (rt *Runtime) runNotice(payload any) error {
	raw, _ := payload.(string)
	var body noticeBody
	if json.Unmarshal([]byte(raw), &body) != nil || body.Session == "" || strings.TrimSpace(body.Text) == "" {
		return nil
	}
	if rt == nil {
		return nil
	}
	rt.mu.Lock()
	fn := rt.notice
	rt.mu.Unlock()
	if fn == nil {
		return nil
	}
	fn(body.Session, body.Text)
	return nil
}

// queueReport 把一份需要人看的报告放进有序队列。
func (rt *Runtime) queueReport(ticketID, state, summary, question string) {
	if rt == nil || rt.effects == nil {
		return
	}
	raw, _ := json.Marshal(relayBody{Ticket: ticketID, State: state, Summary: summary, Question: question})
	key := ticketID + "\n" + state + "\n" + clip(summary+question, 80)
	rt.effects.Enqueue("relay", key, string(raw))
	rt.effects.Pump(time.Now())
}

// noteClose 给跟随 thread 的桌面会话排一条关闭通知。
func (rt *Runtime) noteClose(thread, outcome string) {
	if rt == nil || rt.eng == nil || rt.effects == nil {
		return
	}
	text := "这条拉取请求已关闭"
	if outcome == "merged" {
		text = "这条拉取请求已合并"
	}
	for _, sid := range rt.eng.DeskSessionsFollowing(thread) {
		raw, _ := json.Marshal(noticeBody{Session: sid, Text: text})
		rt.effects.Enqueue("notice", thread+"\n"+sid, string(raw))
	}
}

// loadWorkerLog 读出工单上次的会话日志。没有数据库时用内存里的一份。
func (rt *Runtime) loadWorkerLog(id string) []byte {
	if rt == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.db != nil {
		var body string
		if err := rt.db.QueryRow(`SELECT body FROM worker_log WHERE ticket = ?`, id).Scan(&body); err == nil && body != "" {
			return []byte(body)
		}
	}
	if rt.logs == nil {
		return nil
	}
	return append([]byte(nil), rt.logs[id]...)
}

// saveWorkerLog 记下工单的会话日志。有数据库时同时写入。
func (rt *Runtime) saveWorkerLog(id string, body []byte) {
	if rt == nil || strings.TrimSpace(id) == "" || len(body) == 0 {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.logs == nil {
		rt.logs = map[string][]byte{}
	}
	rt.logs[id] = append([]byte(nil), body...)
	if rt.db == nil {
		return
	}
	_, _ = rt.db.Exec(`INSERT INTO worker_log(ticket, body) VALUES(?, ?) ON CONFLICT(ticket) DO UPDATE SET body = excluded.body`, id, string(body))
}
