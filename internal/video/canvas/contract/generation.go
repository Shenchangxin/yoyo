// generation.go 定义不含密钥的画布生成合同。供应商配置属于执行，不写进这里。
// 这里只有类型：video/canvas_protocol.go 把节点上的 generationSpec JSON 解成 GenerationSpec，再按字段取提示词、选项和参考。
// 除 json 以外的结构体 tag（node、task、modes、enum）只作文档，记录画布节点字段、任务字段、适用模式和取值范围的对应关系，
// 运行时没有代码通过反射读取它们。改字段或取值时要同步前端画布的生成表单，tag 也随之更新。

package contract

// ModelSelection 选择逻辑模型或渠道模型。kind 为 logical 或 channel。
type ModelSelection struct {
	Kind           string `json:"kind" enum:"logical,channel"`
	LogicalModelID string `json:"logicalModelId,omitempty"`
	ChannelID      string `json:"channelId,omitempty"`
	ModelKey       string `json:"modelKey,omitempty"`
}

// Options 是用户看到的公开选项，不是已经解析好的供应商配置。node 和 task 标签是画布字段和任务字段之间仅有的对应关系。
type Options struct {
	Size                  *string `json:"size,omitempty" node:"size" task:"size" modes:"image,video"`
	Quality               *string `json:"quality,omitempty" node:"quality" task:"quality" modes:"image"`
	Count                 *int    `json:"count,omitempty" node:"count" task:"count" modes:"image"`
	TransparentBackground *bool   `json:"transparentBackground,omitempty" node:"transparentBackground" task:"transparentBackground" modes:"image"`
	DurationSeconds       *int    `json:"durationSeconds,omitempty" node:"seconds" task:"videoSeconds" modes:"video"`
	Resolution            *string `json:"resolution,omitempty" node:"vquality" task:"vquality" modes:"video"`
	GenerateAudio         *bool   `json:"generateAudio,omitempty" node:"generateAudio" task:"videoGenerateAudio" modes:"video"`
	Watermark             *bool   `json:"watermark,omitempty" node:"watermark" task:"videoWatermark" modes:"video"`
	AudioVoice            *string `json:"audioVoice,omitempty" node:"audioVoice" task:"audioVoice" modes:"audio"`
	AudioFormat           *string `json:"audioFormat,omitempty" node:"audioFormat" task:"audioFormat" modes:"audio"`
	AudioSpeed            *string `json:"audioSpeed,omitempty" node:"audioSpeed" task:"audioSpeed" modes:"audio"`
	AudioInstructions     *string `json:"audioInstructions,omitempty" node:"audioInstructions" task:"audioInstructions" modes:"audio"`
}

// ReferenceBinding 是一条参考，来源可以是节点、资源或临时标识，并带媒体类型、角色和顺序。
type ReferenceBinding struct {
	ID          string `json:"id"`
	NodeID      string `json:"nodeId,omitempty"`
	ResourceID  string `json:"resourceId,omitempty"`
	TransientID string `json:"transientId,omitempty"`
	MediaType   string `json:"mediaType" enum:"image,video,audio,text"`
	Role        string `json:"role" enum:"reference,source-text,first-frame,last-frame,mask"`
	Order       int    `json:"order"`
	Resolution  string `json:"resolution" enum:"latest,snapshot"`
}

// GenerationSpec 是一次生成的版本、模式、提示词、模型、选项、参考和文本输入方式。
type GenerationSpec struct {
	Version           int                `json:"version"`
	Mode              string             `json:"mode" enum:"image,video,audio"`
	Prompt            string             `json:"prompt"`
	ModelSelection    *ModelSelection    `json:"modelSelection,omitempty"`
	Options           Options            `json:"options"`
	ReferenceBindings []ReferenceBinding `json:"referenceBindings"`
	TextInputMode     string             `json:"textInputMode" enum:"prompt-only,append-sources"`
}
