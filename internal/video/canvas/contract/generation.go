// generation.go 定义不含密钥的画布生成合同。供应商配置属于执行，不写进这里。

package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

//go:generate disabled in yoyo; TS contract is vendored

const (
	// GenerationVersion 是当前合同版本，必须为 1。
	GenerationVersion   = 1
	// MaxNodeTitleRunes 是节点标题上限，240 个字符。
	MaxNodeTitleRunes   = 240
	// MaxPromptRunes 是提示词上限，16000 个字符。
	MaxPromptRunes      = 16000
	// MaxCanvasOperations 是画布操作条数上限，20。
	MaxCanvasOperations = 20
)

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

// FieldError 用路径和消息表示合同字段错误。
type FieldError struct{ Path, Message string }

// Error 返回「路径: 消息」。
func (e *FieldError) Error() string      { return e.Path + ": " + e.Message }
// invalid 构造一条字段错误。
func invalid(path, message string) error { return &FieldError{Path: path, Message: message} }

// Validate 要求 logical 只填 logicalModelId，channel 只填 channelId 和 modelKey。
func (s ModelSelection) Validate() error {
	switch s.Kind {
	case "logical":
		if strings.TrimSpace(s.LogicalModelID) == "" || s.ChannelID != "" || s.ModelKey != "" {
			return invalid("modelSelection", "é»è¾æ¨¡åå¿
é¡»ä¸åªè½æå® logicalModelId")
		}
	case "channel":
		if strings.TrimSpace(s.ChannelID) == "" || strings.TrimSpace(s.ModelKey) == "" || s.LogicalModelID != "" {
			return invalid("modelSelection", "æ¸ éæ¨¡åå¿
é¡»ä¸åªè½æå® channelId å modelKey")
		}
	default:
		return invalid("modelSelection.kind", "æªç¥æ¨¡åéæ©ç±»å")
	}
	return nil
}

// Validate 要求版本为 1，模式是 image、video 或 audio。提示词不超过上限。选项字段的 modes 标签必须包含当前模式。时长不能为负，数量必须为正。引用的标识和顺序要唯一，来源不能空，临时来源不能再混用节点或资源。分辨率是 latest 或 snapshot。source-text 必须是文本，首帧、尾帧和遮罩必须是图片。
func (s GenerationSpec) Validate() error {
	if s.Version != GenerationVersion {
		return invalid("generationSpec.version", "çæååçæ¬ä¸åæ¯æ")
	}
	if s.Mode != "image" && s.Mode != "video" && s.Mode != "audio" {
		return invalid("generationSpec.mode", "æªç¥çææ¨¡å¼")
	}
	if utf8.RuneCountInString(s.Prompt) > MaxPromptRunes {
		return invalid("generationSpec.prompt", "æç¤ºè¯è¶
è¿é¿åº¦éå¶")
	}
	if s.ModelSelection != nil {
		if err := s.ModelSelection.Validate(); err != nil {
			return err
		}
	}
	if s.TextInputMode != "prompt-only" && s.TextInputMode != "append-sources" {
		return invalid("generationSpec.textInputMode", "æªç¥ææ¬è¾å
¥æ¨¡å¼")
	}
	value, typ := reflect.ValueOf(s.Options), reflect.TypeOf(s.Options)
	for i := 0; i < typ.NumField(); i++ {
		if value.Field(i).IsNil() {
			continue
		}
		field := typ.Field(i)
		if !strings.Contains(","+field.Tag.Get("modes")+",", ","+s.Mode+",") {
			return invalid("options."+strings.Split(field.Tag.Get("json"), ",")[0], "è¯¥éé¡¹ä¸éç¨äºå½åçææ¨¡å¼")
		}
	}
	if s.Options.DurationSeconds != nil && *s.Options.DurationSeconds < 0 {
		return invalid("options.durationSeconds", "æ¶é¿ä¸è½ä¸ºè´æ°")
	}
	if s.Options.Count != nil && *s.Options.Count < 1 {
		return invalid("options.count", "æ°éå¿
é¡»ä¸ºæ­£æ´æ°")
	}
	ids := map[string]bool{}
	orders := map[int]bool{}
	for i, binding := range s.ReferenceBindings {
		path := fmt.Sprintf("referenceBindings[%d]", i)
		if binding.ID == "" || ids[binding.ID] || binding.Order < 0 || orders[binding.Order] {
			return invalid(path, "å¼ç¨èº«ä»½åé¡ºåºå¿
é¡»ææä¸å¯ä¸")
		}
		ids[binding.ID], orders[binding.Order] = true, true
		if binding.NodeID == "" && binding.ResourceID == "" && binding.TransientID == "" {
			return invalid(path, "å¼ç¨ç¼ºå°æ¥æº")
		}
		if binding.TransientID != "" && (binding.NodeID != "" || binding.ResourceID != "") {
			return invalid(path, "ä¸´æ¶å¼ç¨ä¸è½æ··ç¨å
¶ä»æ¥æº")
		}
		if binding.Resolution != "latest" && binding.Resolution != "snapshot" {
			return invalid(path+".resolution", "æªç¥å¼ç¨è§£æç­ç¥")
		}
		if binding.MediaType != "image" && binding.MediaType != "video" && binding.MediaType != "audio" && binding.MediaType != "text" {
			return invalid(path+".mediaType", "æªç¥åèç±»å")
		}
		if binding.Role != "reference" && binding.Role != "source-text" && binding.Role != "first-frame" && binding.Role != "last-frame" && binding.Role != "mask" {
			return invalid(path+".role", "æªç¥åèè§è²")
		}
		if binding.Role == "source-text" && binding.MediaType != "text" {
			return invalid(path+".role", "ææ¬æ¥æºå¿
é¡»å¼ç¨ææ¬")
		}
		if (binding.Role == "first-frame" || binding.Role == "last-frame" || binding.Role == "mask") && binding.MediaType != "image" {
			return invalid(path+".role", "å¸§åé®ç½©å¿
é¡»å¼ç¨å¾ç")
		}
	}
	return nil
}

// Decode 用 DisallowUnknownFields 解析单个 JSON 对象，再调用 Validate。
func Decode(data []byte) (GenerationSpec, error) {
	var spec GenerationSpec
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return spec, invalid("generationSpec", err.Error())
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return spec, invalid("generationSpec", "å¿
é¡»æ¯åä¸ªå¯¹è±¡")
	}
	return spec, spec.Validate()
}

// OptionsFromTaskConfig 只复制上面声明过的公开字段。任务配置里多出来的键，包括密钥和供应商路由，不会被收进来。字符串、布尔和整数按字段类型解析。
func OptionsFromTaskConfig(mode string, config map[string]any) (Options, error) {
	options := Options{}
	value, typ := reflect.ValueOf(&options).Elem(), reflect.TypeOf(options)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !strings.Contains(","+field.Tag.Get("modes")+",", ","+mode+",") {
			continue
		}
		raw, exists := config[field.Tag.Get("task")]
		if !exists || raw == nil || raw == "" {
			continue
		}
		parsed := reflect.New(field.Type.Elem())
		switch field.Type.Elem().Kind() {
		case reflect.String:
			text, ok := raw.(string)
			if !ok {
				return options, invalid("options."+field.Tag.Get("task"), "å¿
é¡»æ¯å­ç¬¦ä¸²")
			}
			parsed.Elem().SetString(text)
		case reflect.Bool:
			boolean, err := strconv.ParseBool(fmt.Sprint(raw))
			if err != nil {
				return options, invalid("options."+field.Tag.Get("task"), "å¿
é¡»æ¯å¸å°å¼")
			}
			parsed.Elem().SetBool(boolean)
		case reflect.Int:
			number, err := strconv.Atoi(fmt.Sprint(raw))
			if err != nil {
				return options, invalid("options."+field.Tag.Get("task"), "å¿
é¡»æ¯æ´æ°")
			}
			parsed.Elem().SetInt(int64(number))
		}
		value.Field(i).Set(parsed)
	}
	return options, nil
}

// TaskConfig 把非空选项按 task 标签写成字符串。
func (o Options) TaskConfig() map[string]any {
	result := map[string]any{}
	value, typ := reflect.ValueOf(o), reflect.TypeOf(o)
	for i := 0; i < typ.NumField(); i++ {
		if !value.Field(i).IsNil() {
			result[typ.Field(i).Tag.Get("task")] = fmt.Sprint(value.Field(i).Elem().Interface())
		}
	}
	return result
}

// NodeMetadata 把通过校验的规格写成画布元数据。generationSpec 保存整份规格，prompt 和 composerContent 都写提示词。count 保留数字，其他选项按 node 标签写成字符串。逻辑模型写 logicalModelId，渠道模型写 channelId、channelModelKey，以及 channelId::modelKey。
func (s GenerationSpec) NodeMetadata() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var stored map[string]any
	if err := json.Unmarshal(encoded, &stored); err != nil {
		return nil, err
	}
	result := map[string]any{"generationSpec": stored, "prompt": s.Prompt, "composerContent": s.Prompt}
	value, typ := reflect.ValueOf(s.Options), reflect.TypeOf(s.Options)
	for i := 0; i < typ.NumField(); i++ {
		if value.Field(i).IsNil() {
			continue
		}
		fieldValue := value.Field(i).Elem().Interface()
		if typ.Field(i).Tag.Get("node") != "count" {
			fieldValue = fmt.Sprint(fieldValue)
		}
		result[typ.Field(i).Tag.Get("node")] = fieldValue
	}
	if selection := s.ModelSelection; selection != nil {
		if selection.Kind == "logical" {
			result["logicalModelId"] = selection.LogicalModelID
		} else {
			result["channelId"], result["channelModelKey"] = selection.ChannelID, selection.ModelKey
			result["model"] = selection.ChannelID + "::" + selection.ModelKey
		}
	}
	return result, nil
}

// NodeGenerationProjection 抽出生成要读的字段。锁、任务绑定和授权由调用方另外检查。有 generationSpec 时先 Decode。已知的合同字段原样留下。位置、尺寸和选中状态这类展示字段丢掉。其余键放进 extensions，避免新的生成输入绕过校验，也避免移动节点改掉已经准备好的生成。
func NodeGenerationProjection(metadata map[string]any) (map[string]any, error) {
	result := map[string]any{}
	if raw, exists := metadata["generationSpec"]; exists {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		spec, err := Decode(encoded)
		if err != nil {
			return nil, err
		}
		result["generationSpec"] = spec
	}
	known := map[string]struct{}{
		"generationSpec": {}, "prompt": {}, "composerContent": {}, "model": {},
		"logicalModelId": {}, "channelId": {}, "channelModelKey": {},
		"videoStartFrameNodeId": {}, "videoEndFrameNodeId": {},
		"videoEditOperation": {}, "referenceNodeIds": {},
	}
	typ := reflect.TypeOf(Options{})
	for i := 0; i < typ.NumField(); i++ {
		known[typ.Field(i).Tag.Get("node")] = struct{}{}
	}
	for key, value := range metadata {
		if _, exists := known[key]; exists {
			if key != "generationSpec" {
				result[key] = value
			}
			continue
		}
		if isCanvasPresentationMetadataKey(key) {
			continue
		}
		extensions, _ := result["extensions"].(map[string]any)
		if extensions == nil {
			extensions = map[string]any{}
			result["extensions"] = extensions
		}
		extensions[key] = value
	}
	return result, nil
}

// isCanvasPresentationMetadataKey 判断键是不是位置、尺寸、选中、颜色或布局这类展示字段。
func isCanvasPresentationMetadataKey(key string) bool {
	switch key {
	case "position", "x", "y", "width", "height", "createdAt", "updatedAt", "selected", "collapsed", "zIndex", "color", "background", "displayName", "label", "ui", "presentation", "layout", "screenPosition", "isSelected", "dragging", "resizing":
		return true
	default:
		return false
	}
}
