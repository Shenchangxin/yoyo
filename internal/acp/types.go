// types.go 拥有 ACP v1 客户端用到的协议类型：握手、能力、认证方法、会话、内容块、会话更新、工具调用、权限和文件读写。
// 解码保持宽松：能力字段是布尔或对象都算声明了；认不出的更新种类只留原文。
// client.go 和 acptest 使用这些类型。

package acp

import (
	"encoding/json"
	"strconv"
	"strings"
)

// ProtocolVersion 是这个客户端实现的 ACP 版本。
const ProtocolVersion = 1

// Flag 是一项能力。对方写 true 或一个对象都算声明了，false、null 或没写都算没有。
type Flag bool

// UnmarshalJSON 按上面的规则解码。
func (f *Flag) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	*f = Flag(s != "" && s != "false" && s != "null")
	return nil
}

// Implementation 是一端的名字和版本，用在 clientInfo 和 agentInfo。
type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// FileSystemCapability 声明客户端能替 agent 读写文本文件。
type FileSystemCapability struct {
	ReadTextFile  bool `json:"readTextFile"`
	WriteTextFile bool `json:"writeTextFile"`
}

// ClientAuthCapability 声明客户端能替 agent 开终端登录。
type ClientAuthCapability struct {
	Terminal bool `json:"terminal"`
}

// ClientCapabilities 是客户端在 initialize 里声明的能力。
type ClientCapabilities struct {
	FS       FileSystemCapability  `json:"fs"`
	Terminal bool                  `json:"terminal"`
	Auth     *ClientAuthCapability `json:"auth,omitempty"`
}

// InitializeRequest 是 initialize 的参数。
type InitializeRequest struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
	ClientInfo         *Implementation    `json:"clientInfo,omitempty"`
}

// PromptCapabilities 是 agent 能收的提示内容种类。
type PromptCapabilities struct {
	Image           Flag `json:"image"`
	Audio           Flag `json:"audio"`
	EmbeddedContext Flag `json:"embeddedContext"`
}

// McpCapabilities 是 agent 能连的 MCP 传输。都没有时只能用 stdio。
type McpCapabilities struct {
	HTTP Flag `json:"http"`
	SSE  Flag `json:"sse"`
}

// AgentAuthCapabilities 是 agent 的认证能力。
type AgentAuthCapabilities struct {
	Logout Flag `json:"logout"`
}

// SessionCapabilities 是 agent 在会话上额外支持的操作。
type SessionCapabilities struct {
	Resume                Flag `json:"resume"`
	Close                 Flag `json:"close"`
	Delete                Flag `json:"delete"`
	List                  Flag `json:"list"`
	AdditionalDirectories Flag `json:"additionalDirectories"`
	SystemPrompt          Flag `json:"systemPrompt"`
}

// AgentCapabilities 是 agent 在 initialize 里声明的能力。
type AgentCapabilities struct {
	LoadSession         Flag                   `json:"loadSession"`
	PromptCapabilities  PromptCapabilities     `json:"promptCapabilities"`
	McpCapabilities     McpCapabilities        `json:"mcpCapabilities"`
	Auth                *AgentAuthCapabilities `json:"auth,omitempty"`
	SessionCapabilities SessionCapabilities    `json:"sessionCapabilities"`
}

// AuthMethod 是一种登录方式。Type 为空时视为 agent 型（由 agent 自己完成）；terminal 型要客户端带 Args 和 Env 开终端跑 agent。
type AuthMethod struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Type        string          `json:"type,omitempty"`
	Args        []string        `json:"args,omitempty"`
	Env         json.RawMessage `json:"env,omitempty"`
}

// Kind 返回 agent 或 terminal。没写 type 时是 agent。
func (m AuthMethod) Kind() string {
	if strings.EqualFold(m.Type, "terminal") {
		return "terminal"
	}
	return "agent"
}

// EnvPairs 把 Env 解成 KEY=VALUE。对象和 [{name,value}] 两种写法都认。
func (m AuthMethod) EnvPairs() []string {
	if len(m.Env) == 0 {
		return nil
	}
	var list []EnvVariable
	if json.Unmarshal(m.Env, &list) == nil {
		out := make([]string, 0, len(list))
		for _, kv := range list {
			if kv.Name != "" {
				out = append(out, kv.Name+"="+kv.Value)
			}
		}
		return out
	}
	var obj map[string]string
	if json.Unmarshal(m.Env, &obj) == nil {
		out := make([]string, 0, len(obj))
		for k, v := range obj {
			out = append(out, k+"="+v)
		}
		return out
	}
	return nil
}

// InitializeResponse 是 initialize 的结果。ProtocolVersion 按原文留着，Version 再解释。
type InitializeResponse struct {
	ProtocolVersion   json.RawMessage   `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities"`
	AuthMethods       []AuthMethod      `json:"authMethods,omitempty"`
	AgentInfo         *Implementation   `json:"agentInfo,omitempty"`
}

// Version 返回对方的协议版本。写成数字或数字字符串都认，认不出时为 0。
func (r InitializeResponse) Version() int {
	s := strings.TrimSpace(string(r.ProtocolVersion))
	if unq, err := strconv.Unquote(s); err == nil {
		s = unq
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// EnvVariable 是一个环境变量。
type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HTTPHeader 是一个 HTTP 头。
type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// McpServer 是交给 agent 的一个 MCP 服务器。Type 为 http 或 sse 时用 URL 和 Headers，为空时是 stdio。
type McpServer struct {
	Type    string
	Name    string
	Command string
	Args    []string
	Env     []EnvVariable
	URL     string
	Headers []HTTPHeader
}

// MarshalJSON 按种类只写协议要的字段。stdio 的 args 和 env 即使为空也要写成数组。
func (s McpServer) MarshalJSON() ([]byte, error) {
	switch s.Type {
	case "http", "sse":
		headers := s.Headers
		if headers == nil {
			headers = []HTTPHeader{}
		}
		return json.Marshal(struct {
			Type    string       `json:"type"`
			Name    string       `json:"name"`
			URL     string       `json:"url"`
			Headers []HTTPHeader `json:"headers"`
		}{s.Type, s.Name, s.URL, headers})
	default:
		args := s.Args
		if args == nil {
			args = []string{}
		}
		env := s.Env
		if env == nil {
			env = []EnvVariable{}
		}
		return json.Marshal(struct {
			Name    string        `json:"name"`
			Command string        `json:"command"`
			Args    []string      `json:"args"`
			Env     []EnvVariable `json:"env"`
		}{s.Name, s.Command, args, env})
	}
}

// UnmarshalJSON 认 http、sse 和 stdio 三种写法。acptest 的假 agent 用它读回服务器列表。
func (s *McpServer) UnmarshalJSON(b []byte) error {
	var raw struct {
		Type    string        `json:"type"`
		Name    string        `json:"name"`
		Command string        `json:"command"`
		Args    []string      `json:"args"`
		Env     []EnvVariable `json:"env"`
		URL     string        `json:"url"`
		Headers []HTTPHeader  `json:"headers"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*s = McpServer{Type: raw.Type, Name: raw.Name, Command: raw.Command, Args: raw.Args, Env: raw.Env, URL: raw.URL, Headers: raw.Headers}
	return nil
}

// SystemPrompt 是 session/new 里附加的系统说明。Mode 用 append，接在 agent 自己的系统提示后面。
type SystemPrompt struct {
	Mode string `json:"mode"`
	Text string `json:"text"`
}

// NewSessionRequest 是 session/new 的参数。
type NewSessionRequest struct {
	Cwd                   string        `json:"cwd"`
	McpServers            []McpServer   `json:"mcpServers"`
	AdditionalDirectories []string      `json:"additionalDirectories,omitempty"`
	SystemPrompt          *SystemPrompt `json:"systemPrompt,omitempty"`
}

// ModelInfo 是 agent 能用的一个模型。
type ModelInfo struct {
	ModelID     string `json:"modelId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SessionModelState 是会话当前的模型和可选模型。
type SessionModelState struct {
	CurrentModelID  string      `json:"currentModelId"`
	AvailableModels []ModelInfo `json:"availableModels"`
}

// SessionMode 是一种会话模式。
type SessionMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SessionModeState 是会话当前的模式和可选模式。
type SessionModeState struct {
	CurrentModeID  string        `json:"currentModeId"`
	AvailableModes []SessionMode `json:"availableModes"`
}

// NewSessionResponse 是 session/new 的结果。
type NewSessionResponse struct {
	SessionID string             `json:"sessionId"`
	Modes     *SessionModeState  `json:"modes,omitempty"`
	Models    *SessionModelState `json:"models,omitempty"`
}

// LoadSessionRequest 是 session/load 和 session/resume 的参数。
type LoadSessionRequest struct {
	SessionID             string      `json:"sessionId"`
	Cwd                   string      `json:"cwd"`
	McpServers            []McpServer `json:"mcpServers"`
	AdditionalDirectories []string    `json:"additionalDirectories,omitempty"`
}

// ContentBlock 是一段内容。这里只用到 text，其他种类按原字段留着。
type ContentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Data     string          `json:"data,omitempty"`
	MimeType string          `json:"mimeType,omitempty"`
	URI      string          `json:"uri,omitempty"`
	Name     string          `json:"name,omitempty"`
	Resource json.RawMessage `json:"resource,omitempty"`
}

// TextBlock 返回一段文字内容。
func TextBlock(text string) ContentBlock { return ContentBlock{Type: "text", Text: text} }

// PromptRequest 是 session/prompt 的参数。
type PromptRequest struct {
	SessionID string         `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// StopReason 是一轮结束的原因。
type StopReason string

// 一轮结束的原因。
const (
	StopEndTurn         StopReason = "end_turn"
	StopMaxTokens       StopReason = "max_tokens"
	StopMaxTurnRequests StopReason = "max_turn_requests"
	StopRefusal         StopReason = "refusal"
	StopCancelled       StopReason = "cancelled"
)

// PromptResponse 是 session/prompt 的结果。
type PromptResponse struct {
	StopReason StopReason `json:"stopReason"`
}

// SessionIDParams 是只带会话号的参数，session/cancel 和 session/close 用。
type SessionIDParams struct {
	SessionID string `json:"sessionId"`
}

// Location 是工具调用碰到的位置。
type Location struct {
	Path string `json:"path"`
	Line *int   `json:"line,omitempty"`
}

// ToolCallContent 是工具调用产出的一项：content、diff 或 terminal。
type ToolCallContent struct {
	Type       string        `json:"type"`
	Content    *ContentBlock `json:"content,omitempty"`
	Path       string        `json:"path,omitempty"`
	OldText    *string       `json:"oldText,omitempty"`
	NewText    string        `json:"newText,omitempty"`
	TerminalID string        `json:"terminalId,omitempty"`
}

// ToolCall 是一次工具调用或它的更新。更新里没写的字段保持原值，写了的整项替换。
type ToolCall struct {
	ToolCallID string            `json:"toolCallId"`
	Title      string            `json:"title,omitempty"`
	Name       string            `json:"name,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Status     string            `json:"status,omitempty"`
	Content    []ToolCallContent `json:"content,omitempty"`
	Locations  []Location        `json:"locations,omitempty"`
	RawInput   json.RawMessage   `json:"rawInput,omitempty"`
	RawOutput  json.RawMessage   `json:"rawOutput,omitempty"`
}

// PlanEntry 是计划里的一步。Status 取 pending、in_progress、completed。
type PlanEntry struct {
	Content  string `json:"content"`
	Priority string `json:"priority,omitempty"`
	Status   string `json:"status"`
}

// Cost 是一笔花费。
type Cost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// UsageUpdate 是上下文用量。Used 和 Size 是 token 数。
type UsageUpdate struct {
	Used int   `json:"used"`
	Size int   `json:"size"`
	Cost *Cost `json:"cost,omitempty"`
}

// SessionInfo 是会话信息的更新，比如标题。
type SessionInfo struct {
	Title     string `json:"title,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// 会话更新的种类。
const (
	UpdateUserChunk    = "user_message_chunk"
	UpdateAgentChunk   = "agent_message_chunk"
	UpdateThoughtChunk = "agent_thought_chunk"
	UpdateToolCall     = "tool_call"
	UpdateToolUpdate   = "tool_call_update"
	UpdatePlan         = "plan"
	UpdateUsage        = "usage_update"
	UpdateMode         = "current_mode_update"
	UpdateInfo         = "session_info_update"
)

// SessionUpdate 是一条会话更新。Kind 决定哪个字段有值；认不出的种类只有 Raw。
type SessionUpdate struct {
	Kind     string
	Chunk    *ContentBlock
	ToolCall *ToolCall
	Plan     []PlanEntry
	Usage    *UsageUpdate
	ModeID   string
	Info     *SessionInfo
	Raw      json.RawMessage
}

// UnmarshalJSON 先读 sessionUpdate，再按种类解其余字段。
func (u *SessionUpdate) UnmarshalJSON(b []byte) error {
	var head struct {
		Kind string `json:"sessionUpdate"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return err
	}
	*u = SessionUpdate{Kind: head.Kind, Raw: append(json.RawMessage(nil), b...)}
	switch head.Kind {
	case UpdateUserChunk, UpdateAgentChunk, UpdateThoughtChunk:
		var body struct {
			Content ContentBlock `json:"content"`
		}
		if json.Unmarshal(b, &body) == nil {
			u.Chunk = &body.Content
		}
	case UpdateToolCall, UpdateToolUpdate:
		var body ToolCall
		if json.Unmarshal(b, &body) == nil {
			u.ToolCall = &body
		}
	case UpdatePlan:
		var body struct {
			Entries []PlanEntry `json:"entries"`
		}
		if json.Unmarshal(b, &body) == nil {
			u.Plan = body.Entries
		}
	case UpdateUsage:
		var body UsageUpdate
		if json.Unmarshal(b, &body) == nil {
			u.Usage = &body
		}
	case UpdateMode:
		var body struct {
			ModeID string `json:"currentModeId"`
		}
		if json.Unmarshal(b, &body) == nil {
			u.ModeID = body.ModeID
		}
	case UpdateInfo:
		var body SessionInfo
		if json.Unmarshal(b, &body) == nil {
			u.Info = &body
		}
	}
	return nil
}

// MarshalJSON 写回原文。假 agent 发更新时用 Raw 拼好的对象。
func (u SessionUpdate) MarshalJSON() ([]byte, error) {
	if len(u.Raw) > 0 {
		return u.Raw, nil
	}
	return json.Marshal(map[string]any{"sessionUpdate": u.Kind})
}

// SessionNotification 是 session/update 的参数。
type SessionNotification struct {
	SessionID string        `json:"sessionId"`
	Update    SessionUpdate `json:"update"`
}

// 权限选项的种类。
const (
	PermAllowOnce    = "allow_once"
	PermAllowAlways  = "allow_always"
	PermRejectOnce   = "reject_once"
	PermRejectAlways = "reject_always"
)

// PermissionOption 是 agent 给出的一个选项。
type PermissionOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// PermissionRequest 是 session/request_permission 的参数。
type PermissionRequest struct {
	SessionID string             `json:"sessionId"`
	ToolCall  ToolCall           `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

// PermissionOutcome 是人的选择：selected 带 OptionID，cancelled 不带。
type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

// Selected 返回选中 optionID 的结果。
func Selected(optionID string) PermissionOutcome {
	return PermissionOutcome{Outcome: "selected", OptionID: optionID}
}

// Cancelled 返回这一轮已经取消的结果。
func Cancelled() PermissionOutcome { return PermissionOutcome{Outcome: "cancelled"} }

// PermissionResponse 是 session/request_permission 的结果。
type PermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

// ReadTextFileRequest 是 fs/read_text_file 的参数。Line 从 1 起，Limit 是最多几行。
type ReadTextFileRequest struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Line      *int   `json:"line,omitempty"`
	Limit     *int   `json:"limit,omitempty"`
}

// ReadTextFileResponse 是 fs/read_text_file 的结果。
type ReadTextFileResponse struct {
	Content string `json:"content"`
}

// WriteTextFileRequest 是 fs/write_text_file 的参数。
type WriteTextFileRequest struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

// AuthenticateRequest 是 authenticate 的参数。
type AuthenticateRequest struct {
	MethodID string `json:"methodId"`
}
