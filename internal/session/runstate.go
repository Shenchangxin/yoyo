// 本文件把运行状态写到 <id>.run.json。载入时丢掉仍标成运行中的状态，以及 high_risk、send_as_you、computer_use。

package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// QueuedTurn 是忙着的会话里一条可持久的待办回合。
type QueuedTurn struct {
	// ID 是队列项标识。Enqueue 和 PrependQueue 在它为空时各自生成。
	ID string `json:"id,omitempty"`
	// Text 是待发送文本。
	Text string `json:"text"`
	// Plan 表示这一条要按计划跑。
	Plan bool `json:"plan"`
	// Resume 表示这一条是恢复用的。
	Resume bool `json:"resume,omitempty"`
	// Attachments 是附带的文件。
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment 是队列项上的一个附件。
type Attachment struct {
	// Path 是本地路径，可空。
	Path string `json:"path,omitempty"`
	// Name 是显示名，可空。
	Name string `json:"name,omitempty"`
	// MIME 是媒体类型，可空。
	MIME string `json:"mime,omitempty"`
	// DataB64 是内容的 base64，可空。
	DataB64 string `json:"data_b64,omitempty"`
}

// PendingApproval 是一次还没决定的审批。
type PendingApproval struct {
	// ID 是审批标识。
	ID string `json:"id"`
	// Action 是待审批动作，可空。
	Action string `json:"action,omitempty"`
	// Level 是级别，可空。
	Level string `json:"level,omitempty"`
	// Path 是相关路径，可空。
	Path string `json:"path,omitempty"`
	// Command 是相关命令，可空。
	Command string `json:"command,omitempty"`
}

// RunState 是写在磁盘上的运行状态。LoadRun 不会把 running 原样交还，也不会恢复 high_risk、send_as_you、computer_use。
type RunState struct {
	// Status 是 idle、running、awaiting_approval、compacting、cancelled 之一，或调用方写入的其他字符串。
	Status string `json:"status"`
	// Queue 是还没开始的回合。
	Queue []QueuedTurn `json:"queue,omitempty"`
	// Steers 是运行中插入的转向文本。
	Steers []string `json:"steers,omitempty"`
	// SessionCaps 是本会话能力。写入前会经 dropHighRisk。
	SessionCaps []string `json:"session_caps,omitempty"`
	// PendingApprovals 是未完成的审批。LoadRun 在把运行中状态收成 idle 时清空它。
	PendingApprovals []PendingApproval `json:"pending_approvals,omitempty"`
	// ResumeText 在 MarkTurnInFlight 里总是写成空。用户原文在 JSONL。
	ResumeText string `json:"resume_text,omitempty"`
	// ResumePlan 表示未完成回合是否带计划。
	ResumePlan bool `json:"resume_plan,omitempty"`
	// ResumeInFlight 表示有未完成回合。
	ResumeInFlight bool `json:"resume_in_flight,omitempty"`
	// AskQuestion 是向用户提的问题。LoadRun 收成 idle 时清空。
	AskQuestion string `json:"ask_question,omitempty"`
	// AuthMode 是授权模式。
	AuthMode string `json:"auth_mode,omitempty"`
	// ConnectorAllow 是允许的连接器账号。
	ConnectorAllow []string `json:"connector_allow,omitempty"`
}

// runPath 是 dir/<id>.run.json。
func runPath(dir, id string) string {
	return filepath.Join(dir, id+".run.json")
}

// LoadRun 读取运行状态。文件不存在时返回 StatusIdle 和 nil。
// 状态是 running、compacting 或 awaiting_approval 时改成 idle，并清空 PendingApprovals 和 AskQuestion。
// SessionCaps 再经 dropSticky，丢掉 high_risk、send_as_you、computer_use。
func LoadRun(dir, id string) (RunState, error) {
	b, err := os.ReadFile(runPath(dir, id))
	if err != nil {
		if os.IsNotExist(err) {
			return RunState{Status: StatusIdle}, nil
		}
		return RunState{}, err
	}
	var st RunState
	if err := json.Unmarshal(b, &st); err != nil {
		return RunState{}, err
	}
	if st.Status == StatusRunning || st.Status == StatusCompacting || st.Status == StatusAwaitingApproval {
		st.Status = StatusIdle
		st.PendingApprovals = nil
		st.AskQuestion = ""
	}
	st.SessionCaps = dropSticky(st.SessionCaps)
	return st, nil
}

// SaveRun 以 0o755 创建 dir，再把状态缩进写成 run 文件，权限 0o644。
func SaveRun(dir, id string, st RunState) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(runPath(dir, id), b, 0o644)
}

// RemoveRun 删除 run 文件。文件本来就不存在时返回 nil。
func RemoveRun(dir, id string) error {
	err := os.Remove(runPath(dir, id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// dropSticky 丢掉去掉空白并小写后等于 high_risk、send_as_you 或 computer_use 的项。其余原样保留。
func dropSticky(caps []string) []string {
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		n := strings.ToLower(strings.TrimSpace(c))
		if n == "high_risk" || n == "send_as_you" || n == "computer_use" {
			continue
		}
		out = append(out, c)
	}
	return out
}

// dropHighRisk 就是 dropSticky，同样丢掉 high_risk、send_as_you、computer_use。
func dropHighRisk(caps []string) []string { return dropSticky(caps) }
