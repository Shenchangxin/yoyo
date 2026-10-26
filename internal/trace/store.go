// Package trace 把会话事件追加到 JSONL。Append 只在类型是 user 时 fsync；AppendDurable 每次都 fsync。
// 写出前把 Seq 置 0，Seq 不进文件。分页光标在旁边的 .idx，文件头版本是 2。
// DefaultPageTurns 是 8。DefaultPageBytes 是 256_000，本包的分页函数不按它截断字节。
package trace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EventType 是 JSONL 里 type 字段的取值。
type EventType string

const (
	// TypeSystem 是系统事件，值是 system。
	TypeSystem EventType = "system"
	// TypeUser 是用户事件，值是 user。Append 对它 fsync。
	TypeUser EventType = "user"
	// TypeAssistant 是助手正文，值是 assistant。
	TypeAssistant EventType = "assistant"
	// TypeReasoning 是推理，值是 reasoning。
	TypeReasoning EventType = "reasoning"
	// TypeToolCall 是工具调用，值是 tool_call。
	TypeToolCall EventType = "tool_call"
	// TypeToolResult 是工具结果，值是 tool_result。
	TypeToolResult EventType = "tool_result"
	// TypeInject 是上下文注入，值是 context_injection。
	TypeInject EventType = "context_injection"
	// TypeSubagent 是子代理，值是 subagent。
	TypeSubagent EventType = "subagent"
	// TypeCompact 是压缩，值是 compaction。Kind 为 checkpoint 时 ReadFromCheckpoint 从这一行开始。
	TypeCompact EventType = "compaction"
	// TypeError 是错误，值是 error。
	TypeError EventType = "error"
	// TypeEval 是评测，值是 eval。
	TypeEval EventType = "eval"
	// TypeEvolve 是演化，值是 evolve。
	TypeEvolve EventType = "evolve"
	// TypeTurnEnd 是回合结束，值是 turn_end。
	TypeTurnEnd EventType = "turn_end"
	// TypeUsage 是模型用量，值是 model_usage。
	TypeUsage EventType = "model_usage"
	// TypeApproval 是审批，值是 approval。
	TypeApproval EventType = "approval"
	// TypePlan 是计划，值是 plan。
	TypePlan EventType = "plan"
	// TypeFileChange 是文件变更，值是 file_change。
	TypeFileChange EventType = "file_change"
	// TypeAsk 是向用户提问，值是 ask_user。
	TypeAsk EventType = "ask_user"
	// TypePersonal 是个人事件，值是 personal。
	TypePersonal EventType = "personal"
	// TypeVoiceCall 是语音通话，值是 voice_call。
	TypeVoiceCall EventType = "voice_call"
	// TypePause 是暂停，值是 pause。
	TypePause EventType = "pause"
	// TypeTurnStart 是回合开始，值是 turn_start。
	TypeTurnStart EventType = "turn_start"
	// TypeStepStart 是步骤开始，值是 step_start。
	TypeStepStart EventType = "step_start"
	// TypeStepEnd 是步骤结束，值是 step_end。
	TypeStepEnd EventType = "step_end"
	// TypeAssistantStream 是助手流式片段，值是 assistant_stream。
	TypeAssistantStream EventType = "assistant_stream"
	// TypeInbox 是收件箱，值是 inbox。
	TypeInbox EventType = "inbox"
)

// ItemKindOf 把事件类型映射成 item_kind。user、assistant、reasoning、tool_call、tool_result、inject、subagent、compact、turn_end、approval、plan、file_change、ask_user、turn_start、step_start、step_end、assistant_stream、inbox 有专用短名。其余返回类型字符串本身。
func ItemKindOf(t EventType) string {
	switch t {
	case TypeUser:
		return "user"
	case TypeAssistant:
		return "assistant"
	case TypeReasoning:
		return "reasoning"
	case TypeToolCall:
		return "tool_call"
	case TypeToolResult:
		return "tool_result"
	case TypeInject:
		return "inject"
	case TypeSubagent:
		return "subagent"
	case TypeCompact:
		return "compact"
	case TypeTurnEnd:
		return "turn_end"
	case TypeApproval:
		return "approval"
	case TypePlan:
		return "plan"
	case TypeFileChange:
		return "file_change"
	case TypeAsk:
		return "ask_user"
	case TypeTurnStart:
		return "turn_start"
	case TypeStepStart:
		return "step_start"
	case TypeStepEnd:
		return "step_end"
	case TypeAssistantStream:
		return "assistant_stream"
	case TypeInbox:
		return "inbox"
	default:
		return string(t)
	}
}

// Event 是 JSONL 里的一行。
type Event struct {
	// TS 是时间。为零时 append 和 Replace 写成现在的 UTC。
	TS time.Time `json:"ts"`
	// Type 是事件类型。
	Type EventType `json:"type"`
	// Source 是来源。值为 steer 时分页不把它算成操作者回合。
	Source string `json:"source"`
	// SessionID 决定写到哪个 jsonl 文件。
	SessionID string `json:"session_id"`
	// HarnessSnapshot 是马具快照，可空。
	HarnessSnapshot string `json:"harness_snapshot,omitempty"`
	// ModelFingerprint 是模型指纹，可空。
	ModelFingerprint string `json:"model_fingerprint,omitempty"`
	// TaskID 是任务 id，可空。
	TaskID string `json:"task_id,omitempty"`
	// TurnID 是回合 id，可空。
	TurnID string `json:"turn_id,omitempty"`
	// ItemKind 为空时按 ItemKindOf(Type) 填上。
	ItemKind string `json:"item_kind,omitempty"`
	// Payload 是其余字段。分页会看其中的 kind、name、text、id。
	Payload map[string]any `json:"payload,omitempty"`
	// Seq 是实时总线或分页用的光标。写出前被置 0，不进入 JSONL。
	Seq int64 `json:"seq,omitempty"`
}

// Store 把每个会话写成 Dir 下的 jsonl，并在内存里保留 .idx。
type Store struct {
	// Dir 是会话文件所在目录。
	Dir string
	// mu 保护追加、读取和索引。
	mu sync.Mutex
	// mem 是按会话 id 缓存的索引。
	mem map[string]*sessionIndex
}

// NewStore 用 dir 建存储，内存索引开始是空的。这时不创建目录。
func NewStore(dir string) *Store {
	return &Store{Dir: dir, mem: map[string]*sessionIndex{}}
}

// path 返回该会话的 jsonl 路径。sessionID 含路径分隔符时，先以 0o755 创建父目录，再返回 Dir 下对应的 .jsonl。否则是 Dir/<sessionID>.jsonl。
func (s *Store) path(sessionID string) string {
	clean := filepath.FromSlash(sessionID)
	if strings.ContainsRune(clean, os.PathSeparator) {
		dir := filepath.Join(s.Dir, filepath.Dir(clean))
		_ = os.MkdirAll(dir, 0o755)
		return filepath.Join(s.Dir, clean+".jsonl")
	}
	return filepath.Join(s.Dir, sessionID+".jsonl")
}

// Append 追加一行。只有类型是 user 时才 fsync。
// ev 交给 append；类型是 user 时第二个实参为 true。
func (s *Store) Append(ev Event) error {
	return s.append(ev, ev.Type == TypeUser)
}

// AppendDurable 追加一行并 fsync，不看类型。
// ev 和 true 一起交给 append。
func (s *Store) AppendDurable(ev Event) error {
	return s.append(ev, true)
}

// append 补上 ItemKind 和 TS，把 Seq 置 0，再以 O_APPEND 写入，权限 0o644。durable 为真时 Sync。成功后更新内存索引。
// ev 补上 ItemKind 和 TS，Seq 置 0，再按 SessionID 追加写入。
func (s *Store) append(ev Event, durable bool) error {
	if ev.ItemKind == "" {
		ev.ItemKind = ItemKindOf(ev.Type)
	}
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	ev.Seq = 0
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	p := s.path(ev.SessionID)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	off := st.Size()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(ev); err != nil {
		return err
	}
	n, err := f.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if durable {
		if err := f.Sync(); err != nil {
			return err
		}
	}
	s.recordAppendLocked(ev.SessionID, ev, off, n)
	return nil
}

// Read 读出该会话的全部事件。文件不存在时返回 nil, nil。
// sessionID 以 0、0 交给 readRangeLocked。
func (s *Store) Read(sessionID string) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readRangeLocked(sessionID, 0, 0)
}

// Remove 删掉内存索引、.idx 文件和 jsonl。jsonl 本来就不存在时返回 nil。
// sessionID 交给 dropIndexLocked 和 path。
func (s *Store) Remove(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropIndexLocked(sessionID)
	err := os.Remove(s.path(sessionID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Replace 用 evs 整文件重写该会话。每行 Seq 置 0，空 ItemKind 和零 TS 会补上。写完后丢掉旧索引并重建。
// sessionID 交给 path，写完后交给 dropIndexLocked 和 rebuildIndexLocked。
func (s *Store) Replace(sessionID string, evs []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.path(sessionID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i := range evs {
		ev := evs[i]
		ev.Seq = 0
		if ev.ItemKind == "" {
			ev.ItemKind = ItemKindOf(ev.Type)
		}
		if ev.TS.IsZero() {
			ev.TS = time.Now().UTC()
		}
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
	s.dropIndexLocked(sessionID)
	s.mem[sessionID] = s.rebuildIndexLocked(sessionID)
	return nil
}

// ListSessions 返回 Dir 这一层里扩展名是 .jsonl 的文件名，去掉后缀。不递归。目录不存在时返回 nil, nil。
func (s *Store) ListSessions() ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".jsonl" {
			ids = append(ids, name[:len(name)-len(".jsonl")])
		}
	}
	return ids, nil
}
