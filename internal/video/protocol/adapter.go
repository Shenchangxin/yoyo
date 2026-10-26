// adapter.go 定义协议适配器。适配器只翻译请求和响应。

package protocol

import "context"

// Adapter 只做协议翻译。宿主持有凭证、出站安全、轮询租约、计费、结果下载和任务恢复。
type Adapter interface {
	// Metadata 返回这条协议的元数据。
	Metadata() Metadata
	// BuildCreate 把创建请求编成可发送的规格。
	BuildCreate(context.Context, RequestContext) (RequestSpec, error)
	// ParseCreate 解析创建响应。
	ParseCreate(context.Context, []byte) (CreateResult, error)
	// BuildPoll 编出轮询请求。
	BuildPoll(context.Context, PollContext) (RequestSpec, error)
	// ParsePoll 解析轮询响应。
	ParsePoll(context.Context, PollContext, []byte) (PollResult, error)
	// BuildCancel 编出取消请求。
	BuildCancel(context.Context, PollContext) (RequestSpec, error)
}

// AgentAdapter 是可选的文本工具调用面。宿主仍持有凭证、出站策略和计费。插件只把平台的智能体请求映射成供应商载荷，再把响应解析回平台的工具调用约定。
type AgentAdapter interface {
	// BuildAgent 编出智能体请求。
	BuildAgent(context.Context, AgentRequestContext) (RequestSpec, error)
	// ParseAgent 解析智能体响应。
	ParseAgent(context.Context, []byte) (AgentResult, error)
}

// AgentCapability 让宿主区分真正声明了智能体操作的供应商，和只是因为共用清单适配器而凑齐方法集的实现。
type AgentCapability interface {
	// AgentAvailable 报告这条适配器是否声明了智能体操作。
	AgentAvailable() bool
}

// ResultAdapter 在异步任务成功但没有公开媒体地址时，可选地解析需要鉴权的二进制结果。
type ResultAdapter interface {
	// BuildResult 编出拉取该结果的请求。
	BuildResult(context.Context, PollContext) (RequestSpec, error)
}

// ResultCapability 报告适配器有没有结果拉取操作。
type ResultCapability interface {
	// ResultAvailable 报告是否声明了结果拉取。
	ResultAvailable() bool
}

// AgentRequestContext 带上基址、模型名和原始请求。
type AgentRequestContext struct {
	BaseURL string
	Model   string
	Request map[string]any
}

// AgentToolCall 是一次工具调用的标识、名字、参数文本和思考签名。
type AgentToolCall struct {
	ID               string
	Name             string
	Arguments        string
	ThoughtSignature string
}

// AgentResult 是文本、推理和工具调用列表。
type AgentResult struct {
	Text      string
	Reasoning string
	ToolCalls []AgentToolCall
}

// UnavailableAdapter 用元数据应答，翻译方法都返回不可用。
type UnavailableAdapter struct {
	Info Metadata
}

// Metadata 返回构造时放进的元数据。
func (a UnavailableAdapter) Metadata() Metadata { return a.Info }

// BuildCreate 返回协议不可用。
func (a UnavailableAdapter) BuildCreate(context.Context, RequestContext) (RequestSpec, error) {
	return RequestSpec{}, unavailable(a.Info)
}

// ParseCreate 返回协议不可用。
func (a UnavailableAdapter) ParseCreate(context.Context, []byte) (CreateResult, error) {
	return CreateResult{}, unavailable(a.Info)
}

// BuildPoll 返回协议不可用。
func (a UnavailableAdapter) BuildPoll(context.Context, PollContext) (RequestSpec, error) {
	return RequestSpec{}, unavailable(a.Info)
}

// ParsePoll 返回协议不可用。
func (a UnavailableAdapter) ParsePoll(context.Context, PollContext, []byte) (PollResult, error) {
	return PollResult{}, unavailable(a.Info)
}

// BuildCancel 返回协议不可用。
func (a UnavailableAdapter) BuildCancel(context.Context, PollContext) (RequestSpec, error) {
	return RequestSpec{}, unavailable(a.Info)
}
