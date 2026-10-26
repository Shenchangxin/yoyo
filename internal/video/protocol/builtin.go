// builtin.go 登记随二进制带上的协议适配器，并把创建和轮询编成请求规格。不发出网络请求，也不保存密钥。画布插件通过 Builtins 取这份注册表。

package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type builtinAdapter struct {
	info        Metadata
	create      func(GenerationRequest) (RequestSpec, error)
	parseCreate func(map[string]any) (CreateResult, error)
	poll        func(PollContext) (RequestSpec, error)
	parsePoll   func(PollContext, map[string]any) (PollResult, error)
	cancel      func(PollContext) (RequestSpec, error)
}

// Metadata 返回适配器上保存的元数据。
func (a builtinAdapter) Metadata() Metadata { return a.info }

// BuildCreate 没有创建函数时返回不可用，否则用请求里的生成参数调用创建函数。
// c 的 Request 传给 create。
func (a builtinAdapter) BuildCreate(_ context.Context, c RequestContext) (RequestSpec, error) {
	if a.create == nil {
		return RequestSpec{}, unavailable(a.info)
	}
	return a.create(c.Request)
}

// ParseCreate 没有解析函数时返回不可用。否则把响应体解成对象，再交给解析函数。
// body 传给 decodeObject，解出的对象再交给 parseCreate。
func (a builtinAdapter) ParseCreate(_ context.Context, body []byte) (CreateResult, error) {
	if a.parseCreate == nil {
		return CreateResult{}, unavailable(a.info)
	}
	payload, err := decodeObject(body)
	if err != nil {
		return CreateResult{}, err
	}
	return a.parseCreate(payload)
}

// BuildPoll 没有轮询函数时返回不可用，否则用轮询上下文调用轮询函数。
// c 传给 poll。
func (a builtinAdapter) BuildPoll(_ context.Context, c PollContext) (RequestSpec, error) {
	if a.poll == nil {
		return RequestSpec{}, unavailable(a.info)
	}
	return a.poll(c)
}

// ParsePoll 没有解析函数时返回不可用。否则把响应体解成对象，连同轮询上下文交给解析函数。
// c 传给 parsePoll。
// body 传给 decodeObject，解出的对象连同 c 交给 parsePoll。
func (a builtinAdapter) ParsePoll(_ context.Context, c PollContext, body []byte) (PollResult, error) {
	if a.parsePoll == nil {
		return PollResult{}, unavailable(a.info)
	}
	payload, err := decodeObject(body)
	if err != nil {
		return PollResult{}, err
	}
	return a.parsePoll(c, payload)
}

// BuildCancel 没有取消函数时返回不支持取消的错误，否则用轮询上下文调用取消函数。
func (a builtinAdapter) BuildCancel(_ context.Context, c PollContext) (RequestSpec, error) {
	if a.cancel == nil {
		return RequestSpec{}, fmt.Errorf("protocol %s does not support cancellation", a.info.ID)
	}
	return a.cancel(c)
}

var builtinRegistry *Registry
var builtinRegistryMu sync.Mutex

// Builtins 返回进程内的内置协议注册表。尚未建立时用随包适配器建表，建表失败则 panic。
func Builtins() *Registry {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()
	if builtinRegistry != nil {
		return builtinRegistry
	}
	registry, err := NewRegistry(
		openAIChatAdapter(), openAIResponsesAdapter(), claudeAdapter(),
		openAIVideosAdapter(), newAPIChannel1Adapter(), newAPIVideosAdapter(), xAIVideosAdapter(), arkVideosAdapter(), jimengVideosAdapter(), geminiVeoAdapter(), novitaVideosAdapter(), miniMaxVideosAdapter(),
		agnesAdapter(),
	)
	if err != nil {
		panic(err)
	}
	builtinRegistry = registry
	return registry
}

// BundledManifest 按标识从内置注册表取出清单。适配器不存在，或它不是清单适配器时，返回空清单和假。
// id 传给 Builtins 的 Get。
func BundledManifest(id string) (Manifest, bool) {
	adapter, ok := Builtins().Get(id)
	if !ok {
		return Manifest{}, false
	}
	manifest, ok := adapter.(manifestAdapter)
	if !ok {
		return Manifest{}, false
	}
	return manifest.manifest, true
}

// allSurfaces 返回管理端系统频道、用户自定义频道、画布、创作和代理。
func allSurfaces() []Surface {
	return []Surface{SurfaceAdminSystemChannel, SurfaceUserCustomChannel, SurfaceCanvas, SurfaceCreation, SurfaceAgent}
}

// metadata 组装一条元数据，版本写 1.0.0，能力只有传入的那一种，使用面来自 allSurfaces，并标成启用且可安装。
// id 写入 ID。
// name 写入 Name。
// vendor 写入 Vendor。
// capability 作为 Categories 的唯一一项。
// create 写入 Create。
// poll 写入 Poll。
// contentType 写入 ContentType。
func metadata(id, name, vendor string, capability Capability, create, poll, contentType string) Metadata {
	return Metadata{ID: id, Version: "1.0.0", Name: name, Vendor: vendor, Categories: []Capability{capability}, Scopes: allSurfaces(), Create: create, Poll: poll, ContentType: contentType, Enabled: true, Installable: true}
}

// mediaParams 返回 model、prompt、images、imageCount、aspectRatio、resolution 和 quality。model 和 prompt 必填。
func mediaParams() []Parameter {
	return []Parameter{{Name: "model", Type: "string", Required: true, Mapping: "model"}, {Name: "prompt", Type: "string", Required: true, Mapping: "prompt"}, {Name: "images", Type: "media[]", Mapping: "provider-specific"}, {Name: "imageCount", Type: "integer", Mapping: "provider-specific"}, {Name: "aspectRatio", Type: "string", Mapping: "provider-specific"}, {Name: "resolution", Type: "string", Mapping: "provider-specific"}, {Name: "quality", Type: "string", Mapping: "provider-specific"}}
}

// videoParams 返回视频参数，含模型、提示词、图像、视频、音频、时长、画幅、分辨率、是否生成音频和是否加水印。model 和 prompt 必填。
func videoParams() []Parameter {
	return []Parameter{
		{Name: "model", Type: "string", Required: true, Mapping: "model", Description: "上游模型标识。"},
		{Name: "prompt", Type: "string", Required: true, Mapping: "prompt", Description: "视频生成提示词。"},
		{Name: "images", Type: "media[]", Mapping: "provider-specific", Description: "参考图片，由协议适配器转换为上游要求的 URL、Data URL 或 multipart 文件。"},
		{Name: "videos", Type: "media[]", Mapping: "provider-specific", Description: "参考视频；仅在上游协议支持时发送。"},
		{Name: "audios", Type: "media[]", Mapping: "provider-specific", Description: "参考音频或声音；仅在上游协议支持时发送。"},
		{Name: "duration", Type: "integer", Mapping: "provider-specific", Description: "视频时长，单位由上游协议决定。"},
		{Name: "aspectRatio", Type: "string", Mapping: "provider-specific", Description: "画幅比例或上游尺寸字段。"},
		{Name: "resolution", Type: "string", Mapping: "provider-specific", Description: "输出分辨率档位。"},
		{Name: "generateAudio", Type: "boolean", Mapping: "provider-specific", Description: "是否同时生成音频。"},
		{Name: "watermark", Type: "boolean", Mapping: "provider-specific", Description: "是否添加水印。"},
	}
}

// openAIChatAdapter 返回 Chat Completions 适配器。创建请求把提示词放进一条 user 消息，并合并 extra 里点名的字段，再用 parseChatResponse 解析。
func openAIChatAdapter() Adapter {
	info := metadata("chat-completion", "OpenAI Chat Completions", "OpenAI", CapabilityText, "POST /v1/chat/completions", "", "application/json")
	info.Parameters = []Parameter{
		{Name: "model", Type: "string", Required: true, Mapping: "model", Description: "上游模型标识。"},
		{Name: "prompt", Type: "string", Required: true, Mapping: "messages[0].content", Description: "默认转换为一条 user message；extra.messages 可替换完整消息数组。"},
		{Name: "stream", Type: "boolean", Mapping: "stream", Description: "是否请求流式响应。"},
	}
	return builtinAdapter{info: info,
		create: func(r GenerationRequest) (RequestSpec, error) {
			body := map[string]any{"model": r.Model, "messages": []any{map[string]any{"role": "user", "content": r.Prompt}}}
			mergeExtra(body, r.Extra, "messages", "temperature", "top_p", "max_tokens", "stream", "tools", "response_format")
			return jsonSpec(http.MethodPost, "/v1/chat/completions", body), nil
		},
		parseCreate: parseChatResponse,
	}
}

// openAIResponsesAdapter 返回 Responses 适配器。创建请求把提示词放进 input，并合并 extra 里点名的字段。
func openAIResponsesAdapter() Adapter {
	info := metadata("openai-response", "OpenAI Responses", "OpenAI", CapabilityText, "POST /v1/responses", "", "application/json")
	info.Parameters = []Parameter{
		{Name: "model", Type: "string", Required: true, Mapping: "model", Description: "上游模型标识。"},
		{Name: "prompt", Type: "string", Required: true, Mapping: "input", Description: "默认作为 input 字符串；extra.input 可提供结构化输入。"},
		{Name: "stream", Type: "boolean", Mapping: "stream", Description: "是否请求流式响应。"},
	}
	return builtinAdapter{info: info,
		create: func(r GenerationRequest) (RequestSpec, error) {
			input := any(r.Prompt)
			body := map[string]any{"model": r.Model, "input": input}
			mergeExtra(body, r.Extra, "input", "instructions", "temperature", "top_p", "max_output_tokens", "stream", "tools", "text")
			return jsonSpec(http.MethodPost, "/v1/responses", body), nil
		},
		parseCreate: parseResponsesResponse,
	}
}

// claudeAdapter 返回 Claude Messages 适配器。创建请求默认 max_tokens 为 4096，并带上 anthropic-version 2023-06-01。
func claudeAdapter() Adapter {
	info := metadata("claude-api", "Claude API", "Anthropic", CapabilityText, "POST /v1/messages", "", "application/json")
	info.Parameters = []Parameter{
		{Name: "model", Type: "string", Required: true, Mapping: "model", Description: "Anthropic 模型标识。"},
		{Name: "prompt", Type: "string", Required: true, Mapping: "messages[0].content", Description: "默认转换为一条 user message；extra.messages 可替换完整消息数组。"},
		{Name: "stream", Type: "boolean", Mapping: "stream", Description: "是否请求流式响应。"},
	}
	info.LegacyAliases = []string{"anthropic-messages", "claude-messages"}
	return builtinAdapter{info: info,
		create: func(r GenerationRequest) (RequestSpec, error) {
			body := map[string]any{"model": r.Model, "max_tokens": 4096, "messages": []any{map[string]any{"role": "user", "content": r.Prompt}}}
			mergeExtra(body, r.Extra, "max_tokens", "system", "messages", "temperature", "top_p", "top_k", "tools", "tool_choice", "stream")
			spec := jsonSpec(http.MethodPost, "/v1/messages", body)
			spec.Headers = map[string]string{"anthropic-version": "2023-06-01"}
			return spec, nil
		},
		parseCreate: parseClaudeResponse,
	}
}

// openAIVideosAdapter 返回 OpenAI Videos 适配器。时长写入 seconds，画幅写入 size，参考图写入 input_reference，正文为 multipart/form-data。
func openAIVideosAdapter() Adapter {
	info := metadata("newapi", "OpenAI Videos", "OpenAI compatible", CapabilityVideo, "POST /v1/videos", "GET /v1/videos/{task_id}", "multipart/form-data")
	info.LegacyAliases = []string{"openai-video", "openai-videos"}
	info.Parameters = openAIVideoParams()
	return videoAdapter(info, func(r GenerationRequest) (RequestSpec, error) {
		body := map[string]any{"model": r.Model, "prompt": r.Prompt}
		if r.Duration > 0 {
			body["seconds"] = strconv.Itoa(r.Duration)
		}
		if r.AspectRatio != "" {
			body["size"] = r.AspectRatio
		}
		if len(r.Images) > 0 {
			body["input_reference"] = mediaValues(r.Images)
		}
		return RequestSpec{Method: http.MethodPost, Path: "/v1/videos", ContentType: "multipart/form-data", Body: body}, nil
	})
}

// openAIVideoParams 返回 OpenAI Videos 的参数。duration 映射 seconds，aspectRatio 映射 size，视频、音频、分辨率、生成音频和水印标成 unsupported。
func openAIVideoParams() []Parameter {
	return []Parameter{
		{Name: "model", Type: "string", Required: true, Mapping: "model", Description: "Sora 等视频模型标识。"},
		{Name: "prompt", Type: "string", Required: true, Mapping: "prompt", Description: "视频生成提示词。"},
		{Name: "images", Type: "media[]", Mapping: "input_reference", Description: "可选参考图，作为 multipart 文件上传到 input_reference。"},
		{Name: "duration", Type: "integer", Mapping: "seconds", Description: "视频时长，单位为秒。"},
		{Name: "aspectRatio", Type: "string", Mapping: "size", Description: "视频尺寸，例如 1280x720；该协议使用 size 而不是独立的 aspect_ratio。"},
		{Name: "videos", Type: "media[]", Mapping: "unsupported", Description: "OpenAI Videos 当前适配器不发送参考视频。"},
		{Name: "audios", Type: "media[]", Mapping: "unsupported", Description: "OpenAI Videos 当前适配器不发送参考音频。"},
		{Name: "resolution", Type: "string", Mapping: "unsupported", Description: "分辨率通过 size 表达。"},
		{Name: "generateAudio", Type: "boolean", Mapping: "unsupported", Description: "当前适配器不额外发送该字段。"},
		{Name: "watermark", Type: "boolean", Mapping: "unsupported", Description: "当前适配器不额外发送该字段。"},
	}
}

// newAPIVideosAdapter 返回 NewAPI 视频生成适配器。时长缺省 6，画幅缺省 16:9，参考图、视频和音频分别写入对应的 urls 字段。
func newAPIVideosAdapter() Adapter {
	info := metadata("newapi-channel-2", "NewAPI Video Generations", "NewAPI", CapabilityVideo, "POST /v1/video/generations", "GET /v1/video/generations/{task_id}", "application/json")
	info.LegacyAliases = []string{"newapi-video-generations"}
	info.Parameters = videoParams()
	return videoAdapter(info, func(r GenerationRequest) (RequestSpec, error) {
		body := map[string]any{"model": r.Model, "prompt": r.Prompt, "seconds": strconv.Itoa(defaultInt(r.Duration, 6)), "aspect_ratio": defaultValue(r.AspectRatio, "16:9")}
		copyIf(body, "resolution", r.Resolution)
		if r.GenerateAudio {
			body["generate_audio"] = true
		}
		if len(r.Images) > 0 {
			body["image_urls"] = mediaValues(r.Images)
		}
		if len(r.Videos) > 0 {
			body["video_urls"] = mediaValues(r.Videos)
		}
		if len(r.Audios) > 0 {
			body["audio_urls"] = mediaValues(r.Audios)
		}
		return jsonSpec(http.MethodPost, "/v1/video/generations", body), nil
	})
}

// newAPIChannel1Adapter 返回 NewAPI 媒体任务适配器。参考素材收进 input.media，分辨率缺省 720P，比例缺省 16:9，时长缺省 5。
func newAPIChannel1Adapter() Adapter {
	info := metadata("newapi-channel-1", "NewAPI 媒体任务", "NewAPI", CapabilityVideo, "POST /v1/videos", "GET /v1/videos/{task_id}", "application/json")
	info.Parameters = videoParams()
	return videoAdapter(info, func(r GenerationRequest) (RequestSpec, error) {
		media := make([]map[string]string, 0, len(r.Images)+len(r.Videos)+len(r.Audios))
		for _, image := range r.Images {
			media = append(media, map[string]string{"type": mediaRole(image, "reference_image"), "url": mediaValue(image)})
		}
		for _, video := range r.Videos {
			media = append(media, map[string]string{"type": "reference_video", "url": mediaValue(video)})
		}
		for _, audio := range r.Audios {
			media = append(media, map[string]string{"type": "reference_voice", "url": mediaValue(audio)})
		}
		input := map[string]any{"prompt": r.Prompt}
		if len(media) > 0 {
			input["media"] = media
		}
		parameters := map[string]any{"resolution": defaultValue(r.Resolution, "720P"), "ratio": defaultValue(r.AspectRatio, "16:9"), "duration": defaultInt(r.Duration, 5)}
		if r.Watermark {
			parameters["watermark"] = true
		}
		return jsonSpec(http.MethodPost, "/v1/videos", map[string]any{"model": r.Model, "input": input, "parameters": parameters}), nil
	})
}

// xAIVideosAdapter 返回 xAI 视频适配器。时长缺省 6，画幅缺省 16:9，分辨率缺省 720p。首尾帧和角色参考图不能同时出现，首尾帧合计多于 1 张则失败。
func xAIVideosAdapter() Adapter {
	info := metadata("xai-video", "xAI 官方视频", "xAI", CapabilityVideo, "POST /v1/videos/generations", "GET /v1/videos/{request_id}", "application/json")
	info.Parameters = videoParams()
	return videoAdapter(info, func(r GenerationRequest) (RequestSpec, error) {
		body := map[string]any{"model": r.Model, "prompt": r.Prompt, "duration": defaultInt(r.Duration, 6), "aspect_ratio": defaultValue(r.AspectRatio, "16:9"), "resolution": defaultValue(r.Resolution, "720p")}
		frameImages, referenceImages, unspecifiedImages := splitVideoImages(r.Images)
		if len(frameImages) > 0 && len(referenceImages) > 0 {
			return RequestSpec{}, fmt.Errorf("xAI 视频协议不能同时混用首尾帧和角色参考图")
		}
		if len(frameImages) > 1 {
			return RequestSpec{}, fmt.Errorf("xAI 视频协议最多支持 1 张起始图，不支持尾帧输入")
		}
		if len(frameImages) == 1 {
			body["image"] = map[string]any{"url": mediaValue(frameImages[0])}
		} else if len(referenceImages) > 0 {
			refs := make([]any, 0, len(referenceImages))
			for _, image := range referenceImages {
				refs = append(refs, map[string]any{"url": mediaValue(image)})
			}
			body["reference_images"] = refs
		} else if len(unspecifiedImages) == 1 {
			body["image"] = map[string]any{"url": mediaValue(unspecifiedImages[0])}
		} else if len(unspecifiedImages) > 1 {
			refs := make([]any, 0, len(r.Images))
			for _, image := range unspecifiedImages {
				refs = append(refs, map[string]any{"url": mediaValue(image)})
			}
			body["reference_images"] = refs
		}
		return jsonSpec(http.MethodPost, "/v1/videos/generations", body), nil
	})
}

// arkVideosAdapter 返回火山方舟视频适配器。图片多于 9 张、视频或音频多于 3 个，或只有音频没有图像和视频时失败。时长缺省 5，画幅缺省 16:9，分辨率缺省 720p。
func arkVideosAdapter() Adapter {
	info := metadata("volcengine-ark-video", "火山方舟视频", "Volcengine Ark", CapabilityVideo, "POST /api/v3/contents/generations/tasks", "GET /api/v3/contents/generations/tasks/{task_id}", "application/json")
	info.Parameters = videoParams()
	return videoAdapter(info, func(r GenerationRequest) (RequestSpec, error) {
		if len(r.Images) > 9 || len(r.Videos) > 3 || len(r.Audios) > 3 {
			return RequestSpec{}, fmt.Errorf("火山方舟全模态参考最多支持 9 张图片、3 个视频和 3 个音频")
		}
		if len(r.Audios) > 0 && len(r.Images) == 0 && len(r.Videos) == 0 {
			return RequestSpec{}, fmt.Errorf("火山方舟全模态参考不支持纯音频或文本+音频，请同时添加参考图片或参考视频")
		}
		content := []any{map[string]any{"type": "text", "text": r.Prompt}}
		for _, image := range r.Images {
			content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": mediaValue(image)}, "role": mediaRole(image, "reference_image")})
		}
		for _, video := range r.Videos {
			content = append(content, map[string]any{"type": "video_url", "video_url": map[string]any{"url": mediaValue(video)}, "role": "reference_video"})
		}
		for _, audio := range r.Audios {
			content = append(content, map[string]any{"type": "audio_url", "audio_url": map[string]any{"url": mediaValue(audio)}, "role": "reference_audio"})
		}
		body := map[string]any{"model": r.Model, "content": content, "ratio": defaultValue(r.AspectRatio, "16:9"), "resolution": defaultValue(r.Resolution, "720p"), "duration": defaultInt(r.Duration, 5)}
		if r.GenerateAudio {
			body["generate_audio"] = true
		}
		if r.Watermark {
			body["watermark"] = true
		}
		return jsonSpec(http.MethodPost, "/api/v3/contents/generations/tasks", body), nil
	})
}

// jimengVideosAdapter 返回即梦视频适配器。创建请求提交任务，轮询再用模型和任务标识取结果。
func jimengVideosAdapter() Adapter {
	info := metadata("volcengine-jimeng-video", "即梦官方视频", "Volcengine Jimeng", CapabilityVideo, "POST CVSync2AsyncSubmitTask", "POST CVSync2AsyncGetResult", "application/json + AK/SK signature")
	info.Parameters = videoParams()
	return videoAdapterWithPoll(info, func(r GenerationRequest) (RequestSpec, error) {
		body := map[string]any{"req_key": r.Model, "prompt": r.Prompt}
		if len(r.Images) > 0 {
			body["image_urls"] = mediaValues(r.Images)
		}
		mergeExtra(body, r.Extra, "req_key", "prompt", "image_urls", "duration", "ratio", "resolution")
		return jsonSpec(http.MethodPost, "/CVSync2AsyncSubmitTask", body), nil
	}, func(c PollContext) (RequestSpec, error) {
		return jsonSpec(http.MethodPost, "/CVSync2AsyncGetResult", map[string]any{"req_key": c.Model, "task_id": c.TaskID}), nil
	})
}

// geminiVeoAdapter 返回 Gemini Veo 适配器。图片角色为 reference_image 时失败。创建请求只放第一张图。轮询把任务标识补成 operations 路径。
func geminiVeoAdapter() Adapter {
	info := metadata("gemini-veo", "Gemini Veo", "Google", CapabilityVideo, "POST /v1beta/models/{model}:predictLongRunning", "GET /v1beta/{operation_name}", "application/json")
	info.Parameters = videoParams()
	create := func(r GenerationRequest) (RequestSpec, error) {
		if hasVideoImageRole(r.Images, "reference_image") {
			return RequestSpec{}, fmt.Errorf("Gemini Veo 当前不支持角色或风格参考图生视频，请改用支持 reference_to_video 的模型")
		}
		instance := map[string]any{"prompt": r.Prompt}
		if len(r.Images) > 0 {
			image := r.Images[0]
			instance["image"] = map[string]any{"inlineData": map[string]any{"mimeType": dataMIME(image.DataURL), "data": dataPayload(image.DataURL)}}
		}
		body := map[string]any{"instances": []any{instance}, "parameters": compactMap(map[string]any{"aspectRatio": r.AspectRatio, "durationSeconds": r.Duration, "resolution": r.Resolution, "generateAudio": r.GenerateAudio})}
		return jsonSpec(http.MethodPost, "/v1beta/models/"+url.PathEscape(r.Model)+":predictLongRunning", body), nil
	}
	poll := func(c PollContext) (RequestSpec, error) {
		operation := strings.TrimPrefix(c.TaskID, "/")
		if !strings.HasPrefix(operation, "operations/") {
			operation = "operations/" + operation
		}
		return RequestSpec{Method: http.MethodGet, Path: "/v1beta/" + operation}, nil
	}
	return builtinAdapter{info: info, create: create, parseCreate: parseGeminiVeoCreate, poll: poll, parsePoll: parseGeminiVeoPoll}
}

// novitaVideosAdapter 返回 Novita 视频适配器。图片角色为 reference_image 时失败。时长缺省 5，画幅缺省 16:9，只发送第一张图。
func novitaVideosAdapter() Adapter {
	info := metadata("novita-video", "Novita 视频", "Novita", CapabilityVideo, "POST /v3/video/create", "GET /v3/async/task-result?task_id={id}", "application/json")
	info.Parameters = videoParams()
	return videoAdapter(info, func(r GenerationRequest) (RequestSpec, error) {
		if hasVideoImageRole(r.Images, "reference_image") {
			return RequestSpec{}, fmt.Errorf("Novita 视频当前不支持角色或风格参考图生视频，请改用支持 reference_to_video 的模型")
		}
		body := map[string]any{"model_name": r.Model, "prompt": r.Prompt, "duration": defaultInt(r.Duration, 5), "aspect_ratio": defaultValue(r.AspectRatio, "16:9")}
		if len(r.Images) > 0 {
			body["image_url"] = mediaValue(r.Images[0])
		}
		return jsonSpec(http.MethodPost, "/v3/video/create", body), nil
	})
}

// miniMaxVideosAdapter 返回 MiniMax 视频适配器。提示词和参考素材放进 content，时长缺省 6，并合并 extra 里点名的字段。
func miniMaxVideosAdapter() Adapter {
	info := metadata("minimax-video", "MiniMax 视频", "MiniMax", CapabilityVideo, "POST /v2/video_generation", "GET /v2/query/video_generation/{task_id}", "application/json")
	info.Parameters = videoParams()
	return videoAdapter(info, func(r GenerationRequest) (RequestSpec, error) {
		content := []any{map[string]any{"type": "text", "text": r.Prompt}}
		for _, image := range r.Images {
			content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": mediaValue(image)}, "role": mediaRole(image, "reference_image")})
		}
		for _, video := range r.Videos {
			content = append(content, map[string]any{"type": "video_url", "video_url": map[string]any{"url": mediaValue(video)}, "role": "reference_video"})
		}
		for _, audio := range r.Audios {
			content = append(content, map[string]any{"type": "audio_url", "audio_url": map[string]any{"url": mediaValue(audio)}, "role": "reference_audio"})
		}
		body := map[string]any{"model": r.Model, "prompt": r.Prompt, "duration": defaultInt(r.Duration, 6), "content": content}
		mergeExtra(body, r.Extra, "model", "prompt", "duration", "content", "resolution", "aspect_ratio")
		return jsonSpec(http.MethodPost, "/v2/video_generation", body), nil
	})
}

// agnesAdapter 返回 Agnes 视频适配器，版本写成 1.2.0。轮询走 agnesapi，带上 video_id，模型非空时再带 model_name。
func agnesAdapter() Adapter {
	info := metadata("agnes-video", "Agnes 视频", "Agnes AI", CapabilityVideo, "POST /v1/videos", "GET /agnesapi?video_id={id}&model_name={model}", "application/json")
	info.Version = "1.2.0"
	info.Description = "Agnes AI 官方异步视频协议，支持 Agnes Video V2.0、2.5 与 2.5 Flash。"
	info.RequiresPublicMediaURLs = true
	info.Parameters = []Parameter{
		{Name: "model", Type: "string", Required: true, Mapping: "model", Description: "官方模型 ID：agnes-video-v2.0、agnes-video-2.5 或 agnes-video-2.5-flash。"},
		{Name: "prompt", Type: "string", Required: true, Mapping: "prompt", Description: "视频生成提示词。"},
		{Name: "images", Type: "media[]", Mapping: "first_frame/last_frame/images/image/extra_body.image", Description: "2.5 系列按 mode 映射为关键帧或参考图；V2.0 映射为 image 或关键帧数组。"},
		{Name: "videos", Type: "media[]", Mapping: "videos", Description: "仅 Agnes Video 2.5 reference 模式支持，Flash 与 V2.0 不支持。"},
		{Name: "audios", Type: "media[]", Mapping: "audios", Description: "Agnes Video 2.5 系列 reference 模式支持。"},
		{Name: "duration", Type: "integer", Mapping: "seconds/num_frames", Description: "2.5 系列映射为字符串 seconds；V2.0 按帧率换算 num_frames。"},
		{Name: "aspectRatio", Type: "string", Values: []string{"21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}, Mapping: "aspect_ratio", Description: "2.5 系列画幅比例；历史像素尺寸会在插件边界转换为对应比例。"},
		{Name: "resolution", Type: "string", Values: []string{"720P", "960P", "2K"}, Mapping: "size", Description: "通用层分辨率档位，映射为 Agnes 的 size；Flash 固定 720P。"},
	}
	return builtinAdapter{
		info:        info,
		create:      buildAgnesCreate,
		parseCreate: parseAgnesCreate,
		poll: func(c PollContext) (RequestSpec, error) {
			model := strings.TrimSpace(c.Model)
			if model == "" {
				model = strings.TrimSpace(c.Request.Model)
			}
			path := "/agnesapi?video_id=" + url.QueryEscape(c.TaskID)
			if model != "" {
				path += "&model_name=" + url.QueryEscape(model)
			}
			return RequestSpec{Method: http.MethodGet, Path: path, OriginPath: true}, nil
		},
		parsePoll: parseAgnesPoll,
	}
}

// buildAgnesCreate 按模型选择 V2.0 或 2.5 的请求体。其他模型返回不支持。
func buildAgnesCreate(r GenerationRequest) (RequestSpec, error) {
	model := strings.TrimSpace(r.Model)
	switch model {
	case "agnes-video-v2.0":
		body, err := buildAgnesV20Body(r)
		if err != nil {
			return RequestSpec{}, err
		}
		return jsonSpec(http.MethodPost, "/v1/videos", body), nil
	case "agnes-video-2.5", "agnes-video-2.5-flash":
		body, err := buildAgnesV25Body(r)
		if err != nil {
			return RequestSpec{}, err
		}
		return jsonSpec(http.MethodPost, "/v1/videos", body), nil
	default:
		return RequestSpec{}, fmt.Errorf("Agnes 不支持模型 %q", model)
	}
}

// buildAgnesV25Body 组装 Agnes 2.5 的请求体。mode 只能是 text、keyframe 或 reference。seconds 缺省 5，且必须在 4 到 12 之间。Flash 只接受 720P，参考图最多 5 张，且不接受参考视频。
func buildAgnesV25Body(r GenerationRequest) (map[string]any, error) {
	frameImages, referenceImages, _ := splitVideoImages(r.Images)
	if len(frameImages) > 0 && (len(referenceImages) > 0 || len(r.Videos)+len(r.Audios) > 0) {
		return nil, fmt.Errorf("Agnes 2.5 不能同时混用首尾帧和角色、视频或音频参考素材")
	}
	mode := firstString(r.Extra, "mode")
	if mode == "" {
		switch strings.TrimSpace(r.Operation) {
		case "reference_to_video", "audio_to_video":
			mode = "reference"
		case "image_to_video":
			mode = "keyframe"
		default:
			switch {
			case len(r.Videos) > 0 || len(r.Audios) > 0 || len(referenceImages) > 0:
				mode = "reference"
			case len(frameImages) > 0:
				mode = "keyframe"
			case len(r.Images) > 2:
				mode = "reference"
			case len(r.Images) > 0:
				mode = "keyframe"
			default:
				mode = "text"
			}
		}
	}
	if mode != "text" && mode != "keyframe" && mode != "reference" {
		return nil, fmt.Errorf("Agnes 2.5 mode 必须是 text、keyframe 或 reference")
	}
	seconds := defaultInt(r.Duration, 5)
	if seconds < 4 || seconds > 12 {
		return nil, fmt.Errorf("Agnes 2.5 seconds 必须在 4 到 12 秒之间")
	}
	size, err := normalizeAgnesV25Resolution(defaultValue(r.Resolution, firstString(r.Extra, "size")))
	if err != nil {
		return nil, err
	}
	aspectRatio, err := normalizeAgnesV25AspectRatio(defaultValue(r.AspectRatio, firstString(r.Extra, "aspect_ratio")))
	if err != nil {
		return nil, err
	}
	if r.Model == "agnes-video-2.5-flash" {
		if size != "720P" {
			return nil, fmt.Errorf("Agnes Video 2.5 Flash 仅支持 720P")
		}
		if len(r.Images) > 5 {
			return nil, fmt.Errorf("Agnes Video 2.5 Flash 最多支持 5 张参考图")
		}
		if len(r.Videos) > 0 {
			return nil, fmt.Errorf("Agnes Video 2.5 Flash 不支持参考视频")
		}
	} else if size != "720P" && size != "960P" && size != "2K" {
		return nil, fmt.Errorf("Agnes Video 2.5 size 必须是 720P、960P 或 2K")
	}
	body := map[string]any{
		"model":        r.Model,
		"prompt":       r.Prompt,
		"mode":         mode,
		"seconds":      strconv.Itoa(seconds),
		"size":         size,
		"aspect_ratio": aspectRatio,
		"n":            1,
	}
	if seed, ok := r.Extra["seed"]; ok {
		body["seed"] = seed
	}
	switch mode {
	case "text":
		if len(r.Images)+len(r.Videos)+len(r.Audios) > 0 {
			return nil, fmt.Errorf("Agnes 2.5 text 模式不接受参考媒体")
		}
	case "keyframe":
		if len(r.Images) == 0 || len(r.Images) > 2 || len(r.Videos)+len(r.Audios) > 0 {
			return nil, fmt.Errorf("Agnes 2.5 keyframe 模式需要 1 到 2 张图片，且不接受音视频参考")
		}
		_, referenceImages, _ := splitVideoImages(r.Images)
		if len(referenceImages) > 0 {
			return nil, fmt.Errorf("Agnes 2.5 keyframe 模式不能混用角色或风格参考图")
		}
		frameImages := orderedFrameImages(r.Images)
		if len(frameImages) == 0 {
			return nil, fmt.Errorf("Agnes 2.5 keyframe 模式缺少首帧图片")
		}
		body["first_frame"] = mediaValue(frameImages[0])
		if len(frameImages) > 1 {
			body["last_frame"] = mediaValue(frameImages[1])
		}
	case "reference":
		if len(r.Images)+len(r.Videos)+len(r.Audios) == 0 {
			return nil, fmt.Errorf("Agnes 2.5 reference 模式至少需要一种参考媒体")
		}
		if len(r.Images) > 0 {
			body["images"] = mediaValues(r.Images)
		}
		if len(r.Audios) > 0 {
			body["audios"] = mediaValues(r.Audios)
		}
		if len(r.Videos) > 0 {
			videos := make([]any, 0, len(r.Videos))
			for _, video := range r.Videos {
				videos = append(videos, map[string]any{"url": mediaValue(video)})
			}
			body["videos"] = videos
		}
	}
	return compactMap(body), nil
}

// normalizeAgnesV25Resolution 把分辨率收成 720P、960P 或 2K。空值和 auto 收成 720P，其余对不上的写法返回错误。
// value 去掉空白并转小写后，空串、auto、720、720p 返回 720P，960、960p 返回 960P，2k、1440、1440p 返回 2K，其余返回错误。
func normalizeAgnesV25Resolution(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "720", "720p":
		return "720P", nil
	case "960", "960p":
		return "960P", nil
	case "2k", "1440", "1440p":
		return "2K", nil
	default:
		return "", fmt.Errorf("Agnes Video 2.5 分辨率必须是 720P、960P 或 2K")
	}
}

// normalizeAgnesV25AspectRatio 把画幅收成列出的比例。空值和 auto 收成 16:9。像素尺寸两边都大于 0 且不超过 1000000 时，按整数比折回对应比例。
// value 把 × 换成 x 后去掉空白并转小写；空串或 auto 返回 16:9。等于 21:9、16:9、4:3、1:1、3:4、9:16 时原样返回。形如宽x高且两边都大于 0、不超过 1000000 时，按整数比折回对应比例。
func normalizeAgnesV25AspectRatio(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "×", "x")))
	if normalized == "" || normalized == "auto" {
		return "16:9", nil
	}
	ratios := []string{"21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}
	for _, ratio := range ratios {
		if normalized == ratio {
			return ratio, nil
		}
	}
	dimensions := strings.Split(normalized, "x")
	if len(dimensions) == 2 {
		width, widthErr := strconv.ParseInt(dimensions[0], 10, 64)
		height, heightErr := strconv.ParseInt(dimensions[1], 10, 64)
		if widthErr == nil && heightErr == nil && width > 0 && height > 0 && width <= 1_000_000 && height <= 1_000_000 {
			for _, ratio := range ratios {
				parts := strings.Split(ratio, ":")
				ratioWidth, _ := strconv.ParseInt(parts[0], 10, 64)
				ratioHeight, _ := strconv.ParseInt(parts[1], 10, 64)
				if width*ratioHeight == height*ratioWidth {
					return ratio, nil
				}
			}
		}
	}
	return "", fmt.Errorf("Agnes Video 2.5 画幅必须是 21:9、16:9、4:3、1:1、3:4 或 9:16")
}

// buildAgnesV20Body 组装 Agnes V2.0 的请求体。有参考视频、音频或 reference_image 时失败。时长大于 0 时，帧率取 extra 里 1 到 60 的 frame_rate，缺省 24。未自带 num_frames 时按时长乘帧率换算，大于 441 则写成 441。一张图写入 image，多于一张写入 extra_body 的 keyframes。
func buildAgnesV20Body(r GenerationRequest) (map[string]any, error) {
	if len(r.Videos)+len(r.Audios) > 0 {
		return nil, fmt.Errorf("Agnes Video V2.0 不支持参考视频或音频")
	}
	_, referenceImages, _ := splitVideoImages(r.Images)
	if r.Operation == "reference_to_video" || len(referenceImages) > 0 {
		return nil, fmt.Errorf("Agnes Video V2.0 不支持角色或风格参考图模式，请改用 Agnes Video 2.5")
	}
	body := map[string]any{"model": r.Model, "prompt": r.Prompt}
	mergeExtra(body, r.Extra, "mode", "width", "height", "num_frames", "frame_rate", "num_inference_steps", "seed", "negative_prompt")
	if r.Duration > 0 {
		frameRate := 24
		if value, ok := r.Extra["frame_rate"].(float64); ok && value >= 1 && value <= 60 {
			frameRate = int(value)
		}
		if value, ok := r.Extra["frame_rate"].(int); ok && value >= 1 && value <= 60 {
			frameRate = value
		}
		if _, supplied := r.Extra["frame_rate"]; !supplied {
			body["frame_rate"] = frameRate
		}
		if _, supplied := r.Extra["num_frames"]; !supplied {
			target := r.Duration * frameRate
			frames := ((target-1+4)/8)*8 + 1
			if frames > 441 {
				frames = 441
			}
			body["num_frames"] = frames
		}
	}
	orderedImages := orderedFrameImages(r.Images)
	if len(orderedImages) == 1 {
		body["image"] = mediaValue(orderedImages[0])
	} else if len(orderedImages) > 1 {
		extraBody := map[string]any{}
		if configured := object(r.Extra["extra_body"]); configured != nil {
			for key, value := range configured {
				extraBody[key] = value
			}
		}
		extraBody["image"] = mediaValues(orderedImages)
		extraBody["mode"] = "keyframes"
		body["extra_body"] = extraBody
	} else if configured := object(r.Extra["extra_body"]); configured != nil {
		body["extra_body"] = configured
	}
	return compactMap(body), nil
}

// parseAgnesCreate 从 video_id、task_id 或 id 取出任务标识。没有标识则失败。状态为空时写成 pending。
// payload 用 video_id、task_id、id 取任务标识，用 status 取状态；标识为空则失败，状态为空时写成 pending。
func parseAgnesCreate(payload map[string]any) (CreateResult, error) {
	id := firstString(payload, "video_id", "task_id", "id")
	if id == "" {
		return CreateResult{}, fmt.Errorf("Agnes 创建响应没有 video_id")
	}
	status := normalizeStatus(firstString(payload, "status"))
	if status == "" {
		status = StatusPending
	}
	return CreateResult{TaskID: id, Status: status}, nil
}

// parseAgnesPoll 读取状态，空状态写成 processing。失败时带回消息。成功时从 metadata.url 或 url 取视频地址，没有地址则失败。
// payload 用 video_id、task_id 或 id 取标识，没有时改用 c.TaskID；status 为空时写成 processing。失败时从 error、message、detail 取消息。成功时从 metadata 的 url 或 payload 的 url 取视频地址。
func parseAgnesPoll(c PollContext, payload map[string]any) (PollResult, error) {
	id := defaultValue(firstString(payload, "video_id", "task_id", "id"), c.TaskID)
	status := normalizeStatus(firstString(payload, "status"))
	if status == "" {
		status = StatusProcessing
	}
	message := firstString(payload, "message", "detail")
	if failure := object(payload["error"]); failure != nil {
		message = defaultValue(firstString(failure, "message", "detail", "code"), message)
	}
	if status == StatusFailed {
		return PollResult{TaskID: id, Status: status, Message: defaultValue(message, "Agnes 视频生成失败")}, nil
	}
	if status != StatusSucceeded {
		return PollResult{TaskID: id, Status: status, Message: message}, nil
	}
	metadata := object(payload["metadata"])
	videoURL := firstString(metadata, "url")
	if videoURL == "" {
		videoURL = firstString(payload, "url")
	}
	if videoURL == "" {
		return PollResult{}, fmt.Errorf("Agnes 完成响应没有返回视频 URL")
	}
	return PollResult{TaskID: id, Status: status, Result: &Result{Videos: []MediaReference{{URL: videoURL, Kind: string(CapabilityVideo)}}}}, nil
}

// videoAdapter 用给定的创建函数，和按 Poll 路径拼出的 GET 轮询，做成视频异步适配器。路径里的 task_id 和 request_id 做路径转义，id 做查询转义。
// info 传给 asyncMediaAdapterWithPoll，Poll 去掉方法前缀后替换 {task_id}、{request_id} 和 {id}。
// create 传给 asyncMediaAdapterWithPoll。
func videoAdapter(info Metadata, create func(GenerationRequest) (RequestSpec, error)) Adapter {
	return asyncMediaAdapterWithPoll(info, CapabilityVideo, create, func(c PollContext) (RequestSpec, error) {
		path := strings.TrimSpace(info.Poll)
		if index := strings.IndexByte(path, ' '); index > 0 {
			path = strings.TrimSpace(path[index+1:])
		}
		path = strings.Replace(path, "{task_id}", url.PathEscape(c.TaskID), 1)
		path = strings.Replace(path, "{request_id}", url.PathEscape(c.TaskID), 1)
		path = strings.Replace(path, "{id}", url.QueryEscape(c.TaskID), 1)
		return RequestSpec{Method: http.MethodGet, Path: path}, nil
	})
}

// videoAdapterWithPoll 用调用方提供的创建函数和轮询函数做成视频异步适配器。
// info 传给 asyncMediaAdapterWithPoll。
// create 传给 asyncMediaAdapterWithPoll。
// poll 传给 asyncMediaAdapterWithPoll。
func videoAdapterWithPoll(info Metadata, create func(GenerationRequest) (RequestSpec, error), poll func(PollContext) (RequestSpec, error)) Adapter {
	return asyncMediaAdapterWithPoll(info, CapabilityVideo, create, poll)

}

// asyncMediaAdapter 用创建函数和只替换 task_id 的 GET 轮询做成异步媒体适配器。
// info 传给 asyncMediaAdapterWithPoll，Poll 去掉方法前缀后只替换 {task_id}。
// capability 传给 asyncMediaAdapterWithPoll。
// create 传给 asyncMediaAdapterWithPoll。
func asyncMediaAdapter(info Metadata, capability Capability, create func(GenerationRequest) (RequestSpec, error)) Adapter {
	return asyncMediaAdapterWithPoll(info, capability, create, func(c PollContext) (RequestSpec, error) {
		path := strings.TrimSpace(info.Poll)
		if index := strings.IndexByte(path, ' '); index > 0 {
			path = strings.TrimSpace(path[index+1:])
		}
		path = strings.Replace(path, "{task_id}", url.PathEscape(c.TaskID), 1)
		return RequestSpec{Method: http.MethodGet, Path: path}, nil
	})
}

// asyncMediaAdapterWithPoll 把创建、轮询和按能力解析媒体的函数装进 builtinAdapter。创建响应用 parseAsyncCreate。
// info 写入适配器。
// capability 传给 parseAsyncMediaPoll。
// create 写入适配器。
// poll 写入适配器。
func asyncMediaAdapterWithPoll(info Metadata, capability Capability, create func(GenerationRequest) (RequestSpec, error), poll func(PollContext) (RequestSpec, error)) Adapter {
	return builtinAdapter{
		info: info, create: create, parseCreate: parseAsyncCreate, poll: poll,
		parsePoll: func(c PollContext, payload map[string]any) (PollResult, error) {
			return parseAsyncMediaPoll(c, payload, capability)
		},
	}
}

// jsonSpec 返回方法、路径和正文组成的请求规格，内容类型为 application/json。
// method 写入 Method。
// path 写入 Path。
// body 写入 Body。
func jsonSpec(method, path string, body any) RequestSpec {
	return RequestSpec{Method: method, Path: path, ContentType: "application/json", Body: body}
}

// parseChatResponse 取第一条 choice 的 message.content 作为文本。没有 choices 则失败。用量来自 usage。
// payload 取 choices 的第一条 message.content，以及 usage；choices 为空则失败。
func parseChatResponse(payload map[string]any) (CreateResult, error) {
	choices, _ := payload["choices"].([]any)
	if len(choices) == 0 {
		return CreateResult{}, fmt.Errorf("chat response has no choices")
	}
	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)
	return CreateResult{Status: StatusSucceeded, Result: &Result{Text: firstString(message, "content"), Usage: object(payload["usage"])}}, nil
}

// parseResponsesResponse 依次从 output_text、text 和 output 各项里取文本，并带上 usage。
// payload 依次从 output_text、text 和 output 各项的 text、content 取文本，并带上 usage。
func parseResponsesResponse(payload map[string]any) (CreateResult, error) {
	text := firstString(payload, "output_text", "text")
	if text == "" {
		if output, ok := payload["output"].([]any); ok {
			for _, item := range output {
				if m, ok := item.(map[string]any); ok {
					text = firstString(m, "text", "content")
					if text != "" {
						break
					}
				}
			}
		}
	}
	return CreateResult{Status: StatusSucceeded, Result: &Result{Text: text, Usage: object(payload["usage"])}}, nil
}

// parseClaudeResponse 把 content 里 type 为 text 的片段拼成文本，并带上 usage。
// payload 的 content 里 type 为 text 的片段拼成文本，并带上 usage。
func parseClaudeResponse(payload map[string]any) (CreateResult, error) {
	content, _ := payload["content"].([]any)
	text := ""
	for _, item := range content {
		if m, ok := item.(map[string]any); ok && firstString(m, "type") == "text" {
			text += firstString(m, "text")
		}
	}
	return CreateResult{Status: StatusSucceeded, Result: &Result{Text: text, Usage: object(payload["usage"])}}, nil
}

// parseImageResponse 从 data 里收集 url 或 base64 图片。裸 base64 补上 data:image/png;base64, 前缀。一张都没有则失败。
// payload 的 data 里收集 url，或 b64_json、data；不以 data: 开头的补上 data:image/png;base64,。一张都没有则失败。
func parseImageResponse(payload map[string]any) (CreateResult, error) {
	values, _ := payload["data"].([]any)
	images := make([]MediaReference, 0, len(values))
	for _, item := range values {
		media, _ := item.(map[string]any)
		if value := firstString(media, "url"); value != "" {
			images = append(images, MediaReference{URL: value, Kind: "image"})
			continue
		}
		if value := firstString(media, "b64_json", "data"); value != "" {
			if !strings.HasPrefix(value, "data:") {
				value = "data:image/png;base64," + value
			}
			images = append(images, MediaReference{DataURL: value, Kind: "image"})
		}
	}
	if len(images) == 0 {
		return CreateResult{}, fmt.Errorf("image response has no image data")
	}
	return CreateResult{Status: StatusSucceeded, Result: &Result{Images: images}}, nil
}

// parseJiMengCreate 从响应或 data 里取任务标识。没有则失败，状态为 pending。
// payload 以及其中的 data 用 task_id、taskId、id 取任务标识，没有则失败。
func parseJiMengCreate(payload map[string]any) (CreateResult, error) {
	id := firstString(payload, "task_id", "taskId", "id")
	if data := object(payload["data"]); data != nil {
		id = defaultValue(firstString(data, "task_id", "taskId", "id"), id)
	}
	if id == "" {
		return CreateResult{}, fmt.Errorf("Jimeng response has no task id")
	}
	return CreateResult{TaskID: id, Status: StatusPending}, nil
}

// parseJiMengPoll 按图像能力解析即梦的轮询响应。
// c 传给 parseAsyncMediaPoll。
// payload 传给 parseAsyncMediaPoll，能力用 CapabilityImage。
func parseJiMengPoll(c PollContext, payload map[string]any) (PollResult, error) {
	return parseAsyncMediaPoll(c, payload, CapabilityImage)
}

// parseGeminiVeoCreate 用 name 作为任务标识。没有 name 则失败，状态为 pending。
// payload 用 name 作为任务标识，空串则失败。
func parseGeminiVeoCreate(payload map[string]any) (CreateResult, error) {
	id := firstString(payload, "name")
	if id == "" {
		return CreateResult{}, fmt.Errorf("Gemini Veo response has no operation name")
	}
	return CreateResult{TaskID: id, Status: StatusPending}, nil
}

// parseGeminiVeoPoll 在有 error 对象时失败。done 不为真时状态为 processing。done 为真时收集视频并标成 succeeded，没有视频也仍是 succeeded。
// payload 用 name 作为任务标识，没有时改用 c.TaskID；有 error 对象时失败。done 不为真时状态为 processing，为真时把 response 传给 geminiVeoVideos。
func parseGeminiVeoPoll(c PollContext, payload map[string]any) (PollResult, error) {
	id := defaultValue(firstString(payload, "name"), c.TaskID)
	if failure := object(payload["error"]); failure != nil {
		message := firstString(failure, "message", "status")
		if message == "" {
			message = "Gemini Veo operation failed"
		}
		return PollResult{TaskID: id, Status: StatusFailed, Message: message}, nil
	}
	done, _ := payload["done"].(bool)
	if !done {
		return PollResult{TaskID: id, Status: StatusProcessing}, nil
	}
	response := object(payload["response"])
	videos := geminiVeoVideos(response)
	if len(videos) == 0 {
		return PollResult{TaskID: id, Status: StatusSucceeded}, nil
	}
	return PollResult{TaskID: id, Status: StatusSucceeded, Result: &Result{Videos: videos}}, nil
}

// geminiVeoVideos 从响应和嵌套的生成结果里收集带地址的视频。响应为空时返回 nil。
// response 为 nil 时返回 nil；否则连同 generateVideoResponse、generate_video_response 里的 generatedVideos、generated_videos、generatedSamples、generated_samples、videos 收集 uri、url、videoUri、video_url。
func geminiVeoVideos(response map[string]any) []MediaReference {
	if response == nil {
		return nil
	}
	containers := []map[string]any{response}
	for _, key := range []string{"generateVideoResponse", "generate_video_response"} {
		if nested := object(response[key]); nested != nil {
			containers = append(containers, nested)
		}
	}
	result := make([]MediaReference, 0)
	for _, container := range containers {
		for _, key := range []string{"generatedVideos", "generated_videos", "generatedSamples", "generated_samples", "videos"} {
			items, _ := container[key].([]any)
			for _, item := range items {
				entry := object(item)
				video := object(entry["video"])
				if video == nil {
					video = entry
				}
				if value := firstString(video, "uri", "url", "videoUri", "video_url"); value != "" {
					result = append(result, MediaReference{URL: value, Kind: string(CapabilityVideo)})
				}
			}
		}
	}
	return result
}

// parseAsyncCreate 从响应、task 或 data 里取任务标识和状态。没有标识则失败。状态为空时写成 pending。
// payload 以及 task、data 里用 id、task_id、taskId、request_id 取标识，用 status、state 取状态；标识为空则失败，状态为空时写成 pending。
func parseAsyncCreate(payload map[string]any) (CreateResult, error) {
	id := firstString(payload, "id", "task_id", "taskId", "request_id")
	status := normalizeStatus(firstString(payload, "status", "state"))
	if task := object(payload["task"]); task != nil {
		id = defaultValue(firstString(task, "id", "task_id", "taskId", "request_id"), id)
		status = normalizeStatus(defaultValue(firstString(task, "status", "state"), string(status)))
	}
	if data := object(payload["data"]); data != nil {
		id = defaultValue(firstString(data, "id", "task_id", "taskId", "request_id"), id)
		status = normalizeStatus(defaultValue(firstString(data, "status", "state"), string(status)))
	}
	if id == "" {
		return CreateResult{}, fmt.Errorf("async response has no task id")
	}
	if status == "" {
		status = StatusPending
	}
	return CreateResult{TaskID: id, Status: status, Message: firstString(payload, "message", "error")}, nil
}

// parseAsyncMediaPoll 只接受 image、video 和 audio。状态为空时写成 processing。成功但没有媒体时结果为空。
// payload 以及 task、data、content 用来取状态和媒体地址；标识没有时改用 c.TaskID。状态为空时写成 processing。
// capability 不是 image、video、audio 时返回错误；否则作为结果种类，并决定写入 Images、Audios 或 Videos。
func parseAsyncMediaPoll(c PollContext, payload map[string]any, capability Capability) (PollResult, error) {
	if capability != CapabilityImage && capability != CapabilityVideo && capability != CapabilityAudio {
		return PollResult{}, fmt.Errorf("unsupported async media capability %q", capability)
	}
	status := normalizeStatus(firstString(payload, "status", "state", "task_status"))
	if task := object(payload["task"]); task != nil {
		status = normalizeStatus(defaultValue(firstString(task, "status", "state"), string(status)))
		payload = mergeMaps(task, payload)
	}
	if data := object(payload["data"]); data != nil {
		payload = mergeMaps(data, payload)
		status = normalizeStatus(defaultValue(firstString(data, "status", "state"), string(status)))
	}
	if status == "" {
		status = StatusProcessing
	}
	result := &Result{}
	resultKind := string(capability)
	references := mediaFromArray(payload[resultKind+"s"], capability)
	references = append(references, mediaFromArray(payload["data"], capability)...)
	if content := object(payload["content"]); content != nil {
		references = append(references, mediaFromArray([]any{content}, capability)...)
	}
	keys := []string{resultKind + "_url", resultKind + "Url", "result_url", "url"}
	if resultKind == "video" {
		keys = append(keys, "object")
	}
	if value := firstString(payload, keys...); value != "" {
		references = append(references, MediaReference{URL: value, Kind: resultKind})
	}
	switch capability {
	case CapabilityImage:
		result.Images = references
	case CapabilityAudio:
		result.Audios = references
	case CapabilityVideo:
		result.Videos = references
	}
	if status == StatusSucceeded && len(references) == 0 {
		result = nil
	}
	return PollResult{TaskID: defaultValue(firstString(payload, "id", "task_id", "taskId"), c.TaskID), Status: status, Result: result, Message: firstString(payload, "message", "error", "fail_reason")}, nil
}

// parseAudioResponse 从 url、audio_url 或 data 取音频地址。都没有则失败。
// payload 用 url、audio_url、data 取音频地址，都空则失败。
func parseAudioResponse(payload map[string]any) (CreateResult, error) {
	value := firstString(payload, "url", "audio_url", "data")
	if value == "" {
		return CreateResult{}, fmt.Errorf("audio response has no result")
	}
	return CreateResult{Status: StatusSucceeded, Result: &Result{Audios: []MediaReference{{URL: value, Kind: "audio"}}}}, nil
}

// normalizeStatus 把已知状态文本收成 pending、processing、succeeded、cancelled 或 failed。识别不了时返回空串。
// raw 去掉空白并转小写后，queued、pending、created、submitted、in_queue、task_status_queued 收成 pending，running、processing、in_progress、executing、task_status_running 收成 processing，succeeded、success、completed、complete、done、task_status_succeed 收成 succeeded，cancelled、canceled、aborted 收成 cancelled，failed、failure、error、expired 收成 failed，其余返回空串。
func normalizeStatus(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "queued", "pending", "created", "submitted", "in_queue", "task_status_queued":
		return StatusPending
	case "running", "processing", "in_progress", "executing", "task_status_running":
		return StatusProcessing
	case "succeeded", "success", "completed", "complete", "done", "task_status_succeed":
		return StatusSucceeded
	case "cancelled", "canceled", "aborted":
		return StatusCancelled
	case "failed", "failure", "error", "expired":
		return StatusFailed
	default:
		return ""
	}
}

// decodeObject 把响应体解成 JSON 对象。不是 JSON 时返回错误。
// body 交给 json.Unmarshal，解不成对象就返回错误。
func decodeObject(body []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("protocol response is not JSON: %w", err)
	}
	return payload, nil
}

// object 把值当成对象。不是对象时返回 nil。
// value 做 map[string]any 断言，失败返回 nil。
func object(value any) map[string]any { result, _ := value.(map[string]any); return result }

// mergeMaps 先复制 fallback，再用 primary 里的键覆盖。
func mergeMaps(primary, fallback map[string]any) map[string]any {
	result := make(map[string]any, len(fallback)+len(primary))
	for k, v := range fallback {
		result[k] = v
	}
	for k, v := range primary {
		result[k] = v
	}
	return result
}

// firstString 按键的顺序返回第一个去掉空白后非空的字符串。没有则返回空串。
// payload 按 keys 的顺序取字符串，去掉空白后非空就返回。
// keys 决定取值顺序。
func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// defaultValue 在去掉空白后为空时返回后备值，否则返回原值。
// value 去掉空白后为空时返回 fallback，否则返回自身。
// fallback 在 value 去掉空白后为空时返回。
func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// defaultInt 在值小于等于 0 时返回后备值。
// value 小于等于 0 时返回 fallback，否则返回自身。
// fallback 在 value 小于等于 0 时返回。
func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

// copyIf 在值去掉空白后非空时把它写入 body。
// key 作为写入 body 的键。
// value 去掉空白后非空时写入 body。
func copyIf(body map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		body[key] = value
	}
}

// mergeExtra 把 extra 里点名的键抄进 body。没有的键跳过。
// keys 决定从 extra 抄哪些键，extra 里没有的跳过。
func mergeExtra(body map[string]any, extra map[string]any, keys ...string) {
	for _, key := range keys {
		if value, ok := extra[key]; ok {
			body[key] = value
		}
	}
}

// compactMap 丢掉空白字符串、小于等于 0 的整数和 nil。布尔和其他类型保留。
// body 逐项丢掉空白字符串、小于等于 0 的整数和 nil，布尔和其他类型保留。
func compactMap(body map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range body {
		switch item := value.(type) {
		case string:
			if strings.TrimSpace(item) != "" {
				result[key] = value
			}
		case int:
			if item > 0 {
				result[key] = value
			}
		case bool:
			result[key] = value
		case nil:
		default:
			result[key] = item
		}
	}
	return result
}

// mediaValue 优先返回素材的 URL，URL 为空时返回 DataURL。
func mediaValue(media MediaReference) string { return defaultValue(media.URL, media.DataURL) }

// mediaRole 返回素材角色，空白时用后备角色。
// fallback 在 Role 去掉空白后为空时返回。
func mediaRole(media MediaReference, fallback string) string {
	return defaultValue(strings.TrimSpace(media.Role), fallback)
}

// splitVideoImages 按角色把图片分成首尾帧、reference_image 和未标明角色的图。帧的顺序是先首帧再尾帧。
// images 按 Role 分成 first_frame、last_frame、reference_image 和其余；帧的顺序是先首帧再尾帧。
func splitVideoImages(images []MediaReference) (frames []MediaReference, references []MediaReference, unspecified []MediaReference) {
	firstFrames := make([]MediaReference, 0, 1)
	lastFrames := make([]MediaReference, 0, 1)
	for _, image := range images {
		switch strings.TrimSpace(image.Role) {
		case "first_frame":
			firstFrames = append(firstFrames, image)
		case "last_frame":
			lastFrames = append(lastFrames, image)
		case "reference_image":
			references = append(references, image)
		default:
			unspecified = append(unspecified, image)
		}
	}
	frames = append(frames, firstFrames...)
	frames = append(frames, lastFrames...)
	return frames, references, unspecified
}

// orderedFrameImages 有首尾帧时先返回帧，再接上未标明角色的图，否则只返回未标明角色的图。
// images 传给 splitVideoImages；有首尾帧时先返回帧再接上未标明角色的图，否则只返回未标明角色的图。
func orderedFrameImages(images []MediaReference) []MediaReference {
	frames, _, unspecified := splitVideoImages(images)
	if len(frames) == 0 {
		return unspecified
	}
	return append(frames, unspecified...)
}

// hasVideoImageRole 在任意图片的角色等于给定值时返回真。
// images 的 Role 去掉空白后与 role 比较。
// role 用来和每张图的 Role 相等比较。
func hasVideoImageRole(images []MediaReference, role string) bool {
	for _, image := range images {
		if strings.TrimSpace(image.Role) == role {
			return true
		}
	}
	return false
}

// mediaValues 把每条素材收成 URL 或 DataURL，丢掉空串。
// values 逐条传给 mediaValue，空串丢掉。
func mediaValues(values []MediaReference) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if item := mediaValue(value); item != "" {
			result = append(result, item)
		}
	}
	return result
}

// mediaFromArray 从字符串或对象数组里收集媒体。图像能力遇到裸 base64 时补上 data:image/png;base64, 前缀。
// value 当成数组；字符串去掉空白后非空就收成 URL，对象按能力取地址。
// capability 为 image 时，b64_json 或 data 不以 data: 开头就补上 data:image/png;base64,；种类写入 Kind。
func mediaFromArray(value any, capability Capability) []MediaReference {
	values, _ := value.([]any)
	result := make([]MediaReference, 0, len(values))
	kind := string(capability)
	for _, item := range values {
		if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
			result = append(result, MediaReference{URL: strings.TrimSpace(value), Kind: kind})
			continue
		}
		if m, ok := item.(map[string]any); ok {
			if capability == CapabilityImage {
				if value := firstString(m, "b64_json", "data"); value != "" {
					if !strings.HasPrefix(value, "data:") {
						value = "data:image/png;base64," + value
					}
					result = append(result, MediaReference{DataURL: value, Kind: kind})
					continue
				}
			}
			if value := firstString(m, "url", "image_url", "video_url", "audio_url"); value != "" {
				result = append(result, MediaReference{URL: value, Kind: kind})
			}
		}
	}
	return result
}

// dataMIME 从 data URL 里取出分号前的 MIME。不是这种形式时返回 application/octet-stream。
// value 以 data: 开头且分号位置大于 5 时，返回 data: 与分号之间的文本，否则返回 application/octet-stream。
func dataMIME(value string) string {
	if index := strings.Index(value, ";"); strings.HasPrefix(value, "data:") && index > 5 {
		return strings.TrimPrefix(value[5:index], "")
	}
	return "application/octet-stream"
}

// dataPayload 返回 data URL 逗号后面的载荷。不是 data URL 时返回原字符串。
// value 以 data: 开头且含逗号时返回逗号后面的文本，否则返回原字符串。
func dataPayload(value string) string {
	if index := strings.Index(value, ","); strings.HasPrefix(value, "data:") && index >= 0 {
		return value[index+1:]
	}
	return value
}
