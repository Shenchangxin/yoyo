// types.go 定义协议清单、生成请求和供应商投影用的数据结构。

package protocol

import (
	"encoding/json"
	"fmt"
)

// Capability 是协议适配器处理的模型操作种类。
type Capability string

// 协议能力只有这四种取值。
const (
	// CapabilityText 是文本能力 text。
	CapabilityText  Capability = "text"
	// CapabilityImage 是图像能力 image。
	CapabilityImage Capability = "image"
	// CapabilityVideo 是视频能力 video。
	CapabilityVideo Capability = "video"
	// CapabilityAudio 是音频能力 audio。
	CapabilityAudio Capability = "audio"
)

// Surface 是已安装协议可以被选中的位置。
type Surface string

// 使用面是协议可以被选中的位置取值。
const (
	// SurfaceAdminSystemChannel 是管理端系统频道 admin.system-channel。
	SurfaceAdminSystemChannel Surface = "admin.system-channel"
	// SurfaceUserCustomChannel 是用户自定义频道 user.custom-channel。
	SurfaceUserCustomChannel  Surface = "user.custom-channel"
	// SurfaceCanvas 是画布使用面 canvas。
	SurfaceCanvas             Surface = "canvas"
	// SurfaceCreation 是创作使用面 creation。
	SurfaceCreation           Surface = "creation"
	// SurfaceAgent 是代理使用面 agent。
	SurfaceAgent              Surface = "agent"
)

// Status 是生成任务的状态文本。
type Status string

// 任务状态使用这五种文本。
const (
	// StatusPending 是排队状态 pending。
	StatusPending    Status = "pending"
	// StatusProcessing 是处理中 processing。
	StatusProcessing Status = "processing"
	// StatusSucceeded 是成功 succeeded。
	StatusSucceeded  Status = "succeeded"
	// StatusFailed 是失败 failed。
	StatusFailed     Status = "failed"
	// StatusCancelled 是取消 cancelled。
	StatusCancelled  Status = "cancelled"
)

// MediaReference 是一条素材引用，可以是地址、data URL 或带顺序和权重的附件。
type MediaReference struct {
	ID        string         `json:"id,omitempty"`
	URL       string         `json:"url,omitempty"`
	DataURL   string         `json:"dataUrl,omitempty"`
	Kind      string         `json:"kind,omitempty"`
	Role      string         `json:"role,omitempty"`
	MIMEType  string         `json:"mimeType,omitempty"`
	Name      string         `json:"name,omitempty"`
	Order     int            `json:"order,omitempty"`
	Weight    float64        `json:"weight,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Ephemeral bool           `json:"ephemeral,omitempty"`
}

// OutputOptions 是与供应商无关的输出约定。迁移期间，时长和画幅等旧字段仍留在 GenerationRequest 上。清单会同时看到两套投影，新插件应优先读 request.output。
type OutputOptions struct {
	Count         int            `json:"count,omitempty"`
	Duration      int            `json:"duration,omitempty"`
	AspectRatio   string         `json:"aspectRatio,omitempty"`
	Width         int            `json:"width,omitempty"`
	Height        int            `json:"height,omitempty"`
	Resolution    string         `json:"resolution,omitempty"`
	Quality       string         `json:"quality,omitempty"`
	FPS           float64        `json:"fps,omitempty"`
	GenerateAudio bool           `json:"generateAudio,omitempty"`
	Watermark     bool           `json:"watermark,omitempty"`
	Format        string         `json:"format,omitempty"`
	Options       map[string]any `json:"options,omitempty"`
}

// Message 是一条对话，含角色和内容。
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// GenerationRequest 是画布、创作和代理共用的请求。供应商专有字段放在 Extra，宿主不解释它们。
type GenerationRequest struct {
	Capability      Capability                `json:"capability,omitempty"`
	Model           string                    `json:"model"`
	Prompt          string                    `json:"prompt,omitempty"`
	Instructions    string                    `json:"instructions,omitempty"`
	Messages        []Message                 `json:"messages,omitempty"`
	Inputs          []MediaReference          `json:"inputs,omitempty"`
	Images          []MediaReference          `json:"images,omitempty"`
	Videos          []MediaReference          `json:"videos,omitempty"`
	Audios          []MediaReference          `json:"audios,omitempty"`
	Duration        int                       `json:"duration,omitempty"`
	AspectRatio     string                    `json:"aspectRatio,omitempty"`
	Resolution      string                    `json:"resolution,omitempty"`
	Quality         string                    `json:"quality,omitempty"`
	GenerateAudio   bool                      `json:"generateAudio,omitempty"`
	Watermark       bool                      `json:"watermark,omitempty"`
	ImageCount      int                       `json:"imageCount,omitempty"`
	Operation       string                    `json:"operation,omitempty"`
	Output          OutputOptions             `json:"output,omitempty"`
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
	Extra           map[string]any            `json:"extra,omitempty"`
}

// RequestContext 带上基址和这一次生成请求。
type RequestContext struct {
	BaseURL string
	Request GenerationRequest
}

// PollContext 带上基址、模型、原请求、任务标识和元数据，用来查询。
type PollContext struct {
	BaseURL  string
	Model    string
	Request  GenerationRequest
	TaskID   string
	Metadata map[string]any
}

// RequestSpec 是一次出站请求的方法、路径、头、查询、正文、文件和鉴权。
type RequestSpec struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	OriginPath  bool                `json:"originPath,omitempty"`
	ContentType string              `json:"contentType"`
	Headers     map[string]string   `json:"headers,omitempty"`
	Query       map[string][]string `json:"query,omitempty"`
	Body        any                 `json:"body,omitempty"`
	Files       []RequestFilePart   `json:"files,omitempty"`
	Auth        ManifestAuth        `json:"auth,omitempty"`
}

// RequestFilePart 是 multipart 里的一个文件段，指向一条素材引用。
type RequestFilePart struct {
	Name      string         `json:"name"`
	Filename  string         `json:"filename,omitempty"`
	MIMEType  string         `json:"mimeType,omitempty"`
	Reference MediaReference `json:"reference"`
}

// CreateResult 是创建任务的返回，含任务标识、状态、可选结果和消息。
type CreateResult struct {
	TaskID  string  `json:"taskId,omitempty"`
	Status  Status  `json:"status"`
	Result  *Result `json:"result,omitempty"`
	Message string  `json:"message,omitempty"`
}

// PollResult 是轮询的返回，字段与创建结果相同。
type PollResult struct {
	TaskID  string  `json:"taskId,omitempty"`
	Status  Status  `json:"status"`
	Result  *Result `json:"result,omitempty"`
	Message string  `json:"message,omitempty"`
}

// Result 是生成结果，可以含图像、视频、音频、文本、推理和用量。
type Result struct {
	Images    []MediaReference `json:"images,omitempty"`
	Videos    []MediaReference `json:"videos,omitempty"`
	Audios    []MediaReference `json:"audios,omitempty"`
	Text      string           `json:"text,omitempty"`
	Reasoning string           `json:"reasoning,omitempty"`
	Usage     map[string]any   `json:"usage,omitempty"`
}

// Parameter 是清单里的一个参数，含类型、是否必填、可选值和映射。
type Parameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	Description string   `json:"description,omitempty"`
	Values      []string `json:"values,omitempty"`
	Mapping     string   `json:"mapping,omitempty"`
}

// Metadata 是适配器在进程内的投影。创建、轮询、取消路径和别名不写入 JSON。
type Metadata struct {
	ID                      string       `json:"id"`
	Version                 string       `json:"version"`
	Name                    string       `json:"name"`
	Vendor                  string       `json:"vendor"`
	Description             string       `json:"description,omitempty"`
	Categories              []Capability `json:"categories"`
	Scopes                  []Surface    `json:"scopes"`
	Parameters              []Parameter  `json:"parameters,omitempty"`
	Create                  string       `json:"-"`
	Poll                    string       `json:"-"`
	Cancel                  string       `json:"-"`
	ContentType             string       `json:"-"`
	Documentation           string       `json:"documentation,omitempty"`
	LegacyAliases           []string     `json:"-"`
	Enabled                 bool         `json:"enabled"`
	Installable             bool         `json:"installable"`
	Execution               string       `json:"-"`
	RequiresPublicMediaURLs bool         `json:"-"`
	UnavailableReason       string       `json:"-"`
}

// Manifest 是唯一的插件合同。同一个包可以贡献供应商、工作流、画布节点和其他能力。运行时按每项贡献选择执行方式，公开清单的形状不变。
type Manifest struct {
	APIVersion    string                `json:"apiVersion"`
	Metadata      Metadata              `json:"-"`
	Entry         string                `json:"entry,omitempty"`
	Surfaces      []string              `json:"surfaces,omitempty"`
	Runtime       ManifestRuntime       `json:"runtime,omitempty"`
	Permissions   []string              `json:"permissions,omitempty"`
	Configuration ManifestConfiguration `json:"configuration,omitempty"`
	Contributes   ManifestContributions `json:"contributes"`
	// 下面这些字段是运行时用的供应商投影。它们不参与序列化。供应商怎么执行只写在 contributes.providers 里。
	Create          ManifestOperation      `json:"-"`
	Agent           *ManifestOperation     `json:"-"`
	Poll            *ManifestOperation     `json:"-"`
	Cancel          *ManifestOperation     `json:"-"`
	ResultOperation *ManifestOperation     `json:"-"`
	Response        ManifestResponse       `json:"-"`
	AgentResponse   *ManifestAgentResponse `json:"-"`
	Auth            ManifestAuth           `json:"-"`
	Validations     []ManifestValidation   `json:"-"`
}

// MarshalJSON 写出公开清单。id、name、version、author 来自 Metadata，author 用的是 Vendor。创建、轮询等投影字段不写出。Metadata 不是另一套线上格式。
func (m Manifest) MarshalJSON() ([]byte, error) {
	type wire struct {
		APIVersion    string                `json:"apiVersion"`
		ID            string                `json:"id"`
		Name          string                `json:"name"`
		Version       string                `json:"version"`
		Author        string                `json:"author,omitempty"`
		Description   string                `json:"description,omitempty"`
		Documentation string                `json:"documentation,omitempty"`
		Entry         string                `json:"entry,omitempty"`
		Surfaces      []string              `json:"surfaces,omitempty"`
		Enabled       bool                  `json:"enabled"`
		Installable   bool                  `json:"installable,omitempty"`
		Runtime       ManifestRuntime       `json:"runtime,omitempty"`
		Permissions   []string              `json:"permissions,omitempty"`
		Configuration ManifestConfiguration `json:"configuration,omitempty"`
		Contributes   ManifestContributions `json:"contributes"`
	}
	return json.Marshal(wire{APIVersion: m.APIVersion, ID: m.Metadata.ID, Name: m.Metadata.Name, Version: m.Metadata.Version, Author: m.Metadata.Vendor, Description: m.Metadata.Description, Documentation: m.Metadata.Documentation, Entry: m.Entry, Surfaces: m.Surfaces, Enabled: m.Metadata.Enabled, Installable: m.Metadata.Installable, Runtime: m.Runtime, Permissions: m.Permissions, Configuration: m.Configuration, Contributes: m.Contributes})
}

// UnmarshalJSON 读公开 JSON。enabled 缺省时为真。author 写入 Metadata.Vendor。不在这里展开 contributes.providers。
func (m *Manifest) UnmarshalJSON(data []byte) error {
	type wire struct {
		APIVersion    string                `json:"apiVersion"`
		ID            string                `json:"id"`
		Name          string                `json:"name"`
		Version       string                `json:"version"`
		Author        string                `json:"author"`
		Description   string                `json:"description"`
		Documentation string                `json:"documentation"`
		Entry         string                `json:"entry"`
		Surfaces      []string              `json:"surfaces"`
		Enabled       *bool                 `json:"enabled"`
		Installable   bool                  `json:"installable"`
		Runtime       ManifestRuntime       `json:"runtime"`
		Permissions   []string              `json:"permissions"`
		Configuration ManifestConfiguration `json:"configuration"`
		Contributes   ManifestContributions `json:"contributes"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	enabled := true
	if value.Enabled != nil {
		enabled = *value.Enabled
	}
	m.APIVersion, m.Entry, m.Surfaces, m.Runtime, m.Permissions, m.Configuration, m.Contributes = value.APIVersion, value.Entry, value.Surfaces, value.Runtime, value.Permissions, value.Configuration, value.Contributes
	m.Metadata = Metadata{ID: value.ID, Name: value.Name, Version: value.Version, Vendor: value.Author, Description: value.Description, Documentation: value.Documentation, Enabled: enabled, Installable: value.Installable}
	return nil
}

// ManifestRuntime 记录后端种类、后端入口和网页运行取值。
type ManifestRuntime struct {
	Backend      string `json:"backend,omitempty"`
	BackendEntry string `json:"backendEntry,omitempty"`
	Web          string `json:"web,omitempty"`
}

// ManifestConfiguration 是安装时要填的字段列表。
type ManifestConfiguration struct {
	Fields []ManifestField `json:"fields,omitempty"`
}

// ManifestField 是一个配置项，可以标成密钥并带默认值和可选值。
type ManifestField struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Label       string   `json:"label,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
	Default     any      `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Values      []string `json:"values,omitempty"`
}

// ManifestContributions 是插件贡献的供应商、工作流、画布节点、变换、命令和其他名字列表。
type ManifestContributions struct {
	SMSProviders     []ManifestSMSProvider     `json:"smsProviders,omitempty"`
	Providers        []ManifestProvider        `json:"providers,omitempty"`
	PaymentProviders []ManifestPaymentProvider `json:"paymentProviders,omitempty"`
	Workflows        []ManifestWorkflow        `json:"workflows,omitempty"`
	CanvasNodes      []ManifestCanvasNode      `json:"canvasNodes,omitempty"`
	Transforms       []ManifestTransform       `json:"transforms,omitempty"`
	Commands         []ManifestCommand         `json:"commands,omitempty"`
	AssetSources     []string                  `json:"assetSources,omitempty"`
	UsageObservers   []string                  `json:"usageObservers,omitempty"`
	AICapabilities   []string                  `json:"aiCapabilities,omitempty"`
	Agents           []string                  `json:"agents,omitempty"`
	ImportExport     []string                  `json:"importExport,omitempty"`
}

// ManifestSMSProvider 是一条短信供应商，含标识和标签。
type ManifestSMSProvider struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ManifestPaymentProvider 是一条支付供应商，含结账方式、过期策略和通知响应。
type ManifestPaymentProvider struct {
	ID                  string                      `json:"id"`
	Label               string                      `json:"label"`
	Icon                string                      `json:"icon"`
	CheckoutMode        string                      `json:"checkoutMode"`
	ExpiryPolicy        ManifestPaymentExpiryPolicy `json:"expiryPolicy"`
	IdentityFields      []string                    `json:"identityFields,omitempty"`
	NotificationSuccess ManifestPaymentResponse     `json:"notificationSuccess,omitempty"`
	NotificationFailure ManifestPaymentResponse     `json:"notificationFailure,omitempty"`
}

// ManifestPaymentExpiryPolicy 是支付单的默认、最短和最长分钟数。
type ManifestPaymentExpiryPolicy struct {
	DefaultMinutes int `json:"defaultMinutes"`
	MinMinutes     int `json:"minMinutes"`
	MaxMinutes     int `json:"maxMinutes"`
}

// ManifestPaymentResponse 是通知成功或失败时返回的状态码、类型和正文。
type ManifestPaymentResponse struct {
	Status      int    `json:"status,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Body        string `json:"body,omitempty"`
}

// ManifestProvider 是一条生成供应商，含能力、使用面、鉴权、参数和创建、轮询、取消操作。
type ManifestProvider struct {
	ID                      string                 `json:"id"`
	Label                   string                 `json:"label"`
	Capabilities            []Capability           `json:"capabilities"`
	Scopes                  []Surface              `json:"scopes"`
	BaseURL                 string                 `json:"baseUrl,omitempty"`
	RequiresPublicMediaURLs bool                   `json:"requiresPublicMediaUrls,omitempty"`
	Auth                    ManifestAuth           `json:"auth,omitempty"`
	Parameters              []Parameter            `json:"parameters,omitempty"`
	Validations             []ManifestValidation   `json:"validations,omitempty"`
	Create                  ManifestOperation      `json:"create"`
	Agent                   *ManifestOperation     `json:"agent,omitempty"`
	Poll                    *ManifestOperation     `json:"poll,omitempty"`
	Cancel                  *ManifestOperation     `json:"cancel,omitempty"`
	Result                  *ManifestOperation     `json:"result,omitempty"`
	Response                ManifestResponse       `json:"response"`
	AgentResponse           *ManifestAgentResponse `json:"agentResponse,omitempty"`
}

// ManifestAuth 描述鉴权放在头、查询还是用户名，以及密钥字段、前缀、服务和区域。
type ManifestAuth struct {
	Type        string `json:"type"`
	Field       string `json:"field"`
	SecretField string `json:"secretField,omitempty"`
	Header      string `json:"header,omitempty"`
	Prefix      string `json:"prefix,omitempty"`
	Query       string `json:"query,omitempty"`
	Username    string `json:"username,omitempty"`
	Service     string `json:"service,omitempty"`
	Region      string `json:"region,omitempty"`
}

// ManifestValidation 是一条断言表达式和失败时的消息。
type ManifestValidation struct {
	Assert  any    `json:"assert"`
	Message string `json:"message"`
}

// ManifestWorkflow 把一个供应商和能力绑成工作流，并带参数和默认值。
type ManifestWorkflow struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	ProviderID string         `json:"providerId"`
	Capability Capability     `json:"capability"`
	Parameters []Parameter    `json:"parameters"`
	Defaults   map[string]any `json:"defaults,omitempty"`
}

// ManifestCanvasNode 是画布节点的标识、标题、默认尺寸、结构和渲染器。
type ManifestCanvasNode struct {
	ID           string         `json:"id"`
	Label        string         `json:"label"`
	DefaultTitle string         `json:"defaultTitle"`
	DefaultSize  map[string]int `json:"defaultSize"`
	Schema       map[string]any `json:"schema"`
	Renderer     string         `json:"renderer"`
}

// ManifestTransform 声明一种输入到输出的变换及其运行取值。
type ManifestTransform struct {
	ID      string `json:"id"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Runtime string `json:"runtime"`
}

// ManifestCommand 是一条命令的标识和标签。
type ManifestCommand struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Validate 要求方法和路径非空。带文件段时 ContentType 必须是 multipart/form-data。
func (r RequestSpec) Validate() error {
	if r.Method == "" || r.Path == "" {
		return fmt.Errorf("protocol request spec is incomplete")
	}
	if len(r.Files) > 0 && r.ContentType != "multipart/form-data" {
		return fmt.Errorf("protocol file parts require multipart/form-data")
	}
	return nil
}
