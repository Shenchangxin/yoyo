// gen.go 按供应商提交和轮询图像、视频生成任务。

package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
)

// submitImage 按任务的图像供应商提交。openai、gemini、volcengine 各走自己的请求，其他供应商返回 unsupported image provider。
func (e *Engine) submitImage(ctx context.Context, j *Job) error {
	p, err := e.providerForJob(*j, "image")
	if err != nil {
		return err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return err
	}
	params := parseParams(j.Params)
	switch strings.ToLower(p.Provider) {
	case "openai":
		return e.openaiImage(ctx, p, key, j, params)
	case "gemini":
		return e.geminiImage(ctx, p, key, j, params)
	case "volcengine":
		return e.seedreamImage(ctx, p, key, j, params)
	default:
		return fmt.Errorf("unsupported image provider %s", p.Provider)
	}
}

// pollImage 用 GET /api/v3/images/generations/<远程标识> 查询。succeeded、completed、success 时下载地址并收成图片。failed 或 error 返回错误文本。其余把状态留在 polling。
func (e *Engine) pollImage(ctx context.Context, j *Job) error {
	p, err := e.providerForJob(*j, "image")
	if err != nil {
		return err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return err
	}
	url := joinURL(p.BaseURL, "/api/v3", "/images/generations/"+j.RemoteID)
	req, err := jsonReq(http.MethodGet, url, key, nil)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	st := strings.ToLower(strAny(m, "status", "task_status"))
	if st == "succeeded" || st == "completed" || st == "success" {
		u := firstURL(m["data"])
		if u == "" {
			u = strAny(pickData(m), "url", "image_url")
		}
		if u == "" {
			return fmt.Errorf("image completed without url")
		}
		raw, err := e.DownloadURL(u)
		if err != nil {
			return err
		}
		return e.finishMedia(j, raw, false)
	}
	if st == "failed" || st == "error" {
		return fmt.Errorf("%s", strAny(m, "error", "message"))
	}
	j.Status = "polling"
	return nil
}

// submitVideo 按视频供应商提交。volcengine 走 seedance，minimax 走 MiniMax，aliyun 走万相。其他供应商返回 unsupported video provider。
func (e *Engine) submitVideo(ctx context.Context, j *Job) error {
	p, err := e.providerForJob(*j, "video")
	if err != nil {
		return err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return err
	}
	params := parseParams(j.Params)
	switch strings.ToLower(p.Provider) {
	case "volcengine":
		return e.seedanceSubmit(ctx, p, key, j, params)
	case "minimax":
		return e.minimaxSubmit(ctx, p, key, j, params)
	case "aliyun":
		return e.wanSubmit(ctx, p, key, j, params)
	default:
		return fmt.Errorf("unsupported video provider %s", p.Provider)
	}
}

// pollVideo 按同一组视频供应商查询远程任务。
func (e *Engine) pollVideo(ctx context.Context, j *Job) error {
	p, err := e.providerForJob(*j, "video")
	if err != nil {
		return err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return err
	}
	switch strings.ToLower(p.Provider) {
	case "volcengine":
		return e.seedancePoll(ctx, p, key, j)
	case "minimax":
		return e.minimaxPoll(ctx, p, key, j)
	case "aliyun":
		return e.wanPoll(ctx, p, key, j)
	default:
		return fmt.Errorf("unsupported video provider %s", p.Provider)
	}
}

// providerForJob 先按金库键找启用的供应商，再按供应商名找。都没有时用该种类的当前供应商。
func (e *Engine) providerForJob(j Job, kind string) (Provider, error) {
	list, err := e.ListProviders(kind)
	if err != nil {
		return Provider{}, err
	}
	if j.VaultKey != "" {
		for _, p := range list {
			if p.VaultKey == j.VaultKey && p.IsActive {
				return p, nil
			}
		}
	}
	for _, p := range list {
		if p.Provider == j.Provider && p.IsActive {
			return p, nil
		}
	}
	return e.ActiveProvider(kind, "")
}

// openaiImage 有参考图时改走编辑。否则 POST /v1/images/generations，尺寸默认 1024x1024，张数为 1。
func (e *Engine) openaiImage(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	model := first(j.Model, p.Model)
	if len(params.ReferenceImages) > 0 {
		return e.openaiEdit(ctx, p, key, j, params, model)
	}
	body := map[string]any{"model": model, "prompt": j.Prompt, "n": 1, "size": OpenAIImageSize(first(params.Size, "1024x1024"))}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/v1", "/images/generations"), key, body)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	return e.takeImageResult(j, m)
}

// openaiEdit 用 multipart 把提示、尺寸和参考图 POST 到 /v1/images/edits。读不到的参考图跳过。
func (e *Engine) openaiEdit(ctx context.Context, p Provider, key string, j *Job, params JobParams, model string) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", model)
	_ = w.WriteField("prompt", j.Prompt)
	_ = w.WriteField("size", OpenAIImageSize(first(params.Size, "1024x1024")))
	for i, ref := range params.ReferenceImages {
		raw, err := e.refBytes(ref)
		if err != nil {
			continue
		}
		part, err := w.CreateFormFile("image[]", fmt.Sprintf("ref-%d.jpg", i))
		if err != nil {
			continue
		}
		_, _ = part.Write(raw)
	}
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(p.BaseURL, "/v1", "/images/edits"), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+key)
	m, _, _, err := e.doJSON(req)
	if err != nil {
		return err
	}
	return e.takeImageResult(j, m)
}

// geminiImage 在官方域名且模型名同时含 image 和 gemini-3 时走 /v1beta/interactions。否则把参考图当成 jpeg 的 inline_data，POST generateContent，并设置 x-goog-api-key。
func (e *Engine) geminiImage(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	model := first(j.Model, p.Model)
	official := strings.Contains(p.BaseURL, "generativelanguage.googleapis.com")
	if official && strings.Contains(model, "image") && strings.Contains(model, "gemini-3") {
		body := map[string]any{
			"model":           strings.TrimPrefix(model, "models/"),
			"input":           j.Prompt,
			"response_format": map[string]any{"type": "image", "aspect_ratio": GeminiAspectFromSize(params.Size, params.AspectRatio), "image_size": "2K"},
		}
		req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/v1beta", "/interactions"), key, body)
		if err != nil {
			return err
		}
		req.Header.Set("x-goog-api-key", key)
		m, _, _, err := e.doJSON(req.WithContext(ctx))
		if err != nil {
			return err
		}
		return e.takeGeminiImage(j, m)
	}
	if !strings.HasPrefix(model, "models/") {
		model = "models/" + model
	}
	parts := []any{}
	for _, ref := range params.ReferenceImages {
		raw, err := e.refBytes(ref)
		if err != nil {
			continue
		}
		parts = append(parts, map[string]any{"inline_data": map[string]any{"mime_type": "image/jpeg", "data": base64.StdEncoding.EncodeToString(raw)}})
	}
	parts = append(parts, map[string]any{"text": j.Prompt})
	body := map[string]any{
		"contents":         []any{map[string]any{"parts": parts}},
		"generationConfig": map[string]any{"responseModalities": []string{"IMAGE", "TEXT"}},
	}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/v1beta", "/"+model+":generateContent"), key, body)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-api-key", key)
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	return e.takeGeminiImage(j, m)
}

// seedreamImage POST /api/v3/images/generations。尺寸能拆成宽高就写上。有任务标识且还没有图片地址时把任务标成 polling。
func (e *Engine) seedreamImage(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	body := map[string]any{"model": first(j.Model, p.Model), "prompt": j.Prompt}
	if w, h, ok := splitSize(params.Size); ok {
		body["width"], body["height"] = w, h
	}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/api/v3", "/images/generations"), key, body)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	if id := strAny(m, "task_id", "id"); id != "" && firstURL(m["data"]) == "" {
		j.RemoteID = id
		j.Status = "polling"
		return nil
	}
	return e.takeImageResult(j, m)
}

// takeImageResult 先取 data 里的 b64_json 或 url，再取顶层 url。都没有时交给 takeGeminiImage。
func (e *Engine) takeImageResult(j *Job, m map[string]any) error {
	if data, ok := m["data"].([]any); ok && len(data) > 0 {
		if row, ok := data[0].(map[string]any); ok {
			if s, _ := row["b64_json"].(string); s != "" {
				raw, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					return err
				}
				return e.finishMedia(j, raw, false)
			}
			if u := strAny(row, "url"); u != "" {
				raw, err := e.DownloadURL(u)
				if err != nil {
					return err
				}
				return e.finishMedia(j, raw, false)
			}
		}
	}
	if u := strAny(m, "url"); u != "" {
		raw, err := e.DownloadURL(u)
		if err != nil {
			return err
		}
		return e.finishMedia(j, raw, false)
	}
	return e.takeGeminiImage(j, m)
}

// takeGeminiImage 在 JSON 文本里找第一段 "data":"..." 的 base64。解开后超过 32 字节才收成图片。
func (e *Engine) takeGeminiImage(j *Job, m map[string]any) error {
	rawJSON, _ := json.Marshal(m)
	s := string(rawJSON)
	const needle = `"data":"`
	if i := strings.Index(s, needle); i >= 0 {
		rest := s[i+len(needle):]
		if end := strings.Index(rest, `"`); end > 0 {
			b, err := base64.StdEncoding.DecodeString(rest[:end])
			if err == nil && len(b) > 32 {
				return e.finishMedia(j, b, false)
			}
		}
	}
	return fmt.Errorf("no image in response")
}

// seedanceSubmit 在模型不是 doubao-seedance-2-0 开头时改用 doubao-seedance-2-0-mini-260615。参考图角色是 reference_image。POST /api/v3/contents/generations/tasks。有 id 时进入 polling。
func (e *Engine) seedanceSubmit(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	model := first(j.Model, p.Model)
	if !strings.HasPrefix(strings.ToLower(model), "doubao-seedance-2-0") {
		model = "doubao-seedance-2-0-mini-260615"
	}
	content := []any{}
	if j.Prompt != "" {
		content = append(content, map[string]any{"type": "text", "text": j.Prompt})
	}
	for _, u := range params.ReferenceImageURLs {
		content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": u}, "role": "reference_image"})
	}
	body := map[string]any{
		"model": model, "content": content,
		"generate_audio": params.GenerateAudio, "ratio": first(params.AspectRatio, "16:9"),
		"duration":   ClampProviderDuration(params.Duration, "volcengine"),
		"resolution": NormalizeVideoResolution(params.Resolution, "volcengine"),
		"watermark":  false,
	}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/api/v3", "/contents/generations/tasks"), key, body)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	if id := strAny(m, "id"); id != "" {
		j.RemoteID = id
		j.Status = "polling"
		return nil
	}
	return e.takeVideoURL(j, m)
}

// seedancePoll 查询 contents/generations/tasks。succeeded 收视频地址，failed 返回 videoErr，其余保持 polling。
func (e *Engine) seedancePoll(ctx context.Context, p Provider, key string, j *Job) error {
	req, err := jsonReq(http.MethodGet, joinURL(p.BaseURL, "/api/v3", "/contents/generations/tasks/"+j.RemoteID), key, nil)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	st := strings.ToLower(strAny(m, "status"))
	if st == "succeeded" {
		return e.takeVideoURL(j, m)
	}
	if st == "failed" {
		return fmt.Errorf("%s", videoErr(m))
	}
	j.Status = "polling"
	return nil
}

// minimaxSubmit 在模型名不含 minimax-h3 时改用 MiniMax-H3。没有首帧时写入比例，adaptive 收成 16:9。POST /v2/video_generation。有 task_id 时进入 polling。
func (e *Engine) minimaxSubmit(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	model := first(j.Model, p.Model)
	if !strings.Contains(strings.ToLower(model), "minimax-h3") {
		model = "MiniMax-H3"
	}
	content := []any{map[string]any{"type": "text", "text": j.Prompt}}
	for _, u := range params.ReferenceImageURLs {
		content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": u}, "role": "reference_image"})
	}
	body := map[string]any{
		"model": model, "content": content,
		"duration":       ClampProviderDuration(params.Duration, "minimax"),
		"resolution":     NormalizeVideoResolution(params.Resolution, "minimax"),
		"generate_audio": params.GenerateAudio,
	}
	if params.FirstFrameURL == "" {
		ratio := first(params.AspectRatio, "16:9")
		if ratio == "adaptive" {
			ratio = "16:9"
		}
		body["ratio"] = ratio
	}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "", "/v2/video_generation"), key, body)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	if id := strAny(m, "task_id"); id != "" {
		j.RemoteID = id
		j.Status = "polling"
		return nil
	}
	return e.takeVideoURL(j, m)
}

// minimaxPoll 查询 /v2/query/video_generation。优先读 task 对象。success 或 succeeded 收地址，failed 或 fail 返回错误。
func (e *Engine) minimaxPoll(ctx context.Context, p Provider, key string, j *Job) error {
	req, err := jsonReq(http.MethodGet, joinURL(p.BaseURL, "", "/v2/query/video_generation/"+j.RemoteID), key, nil)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	task, _ := m["task"].(map[string]any)
	if task == nil {
		task = m
	}
	st := strings.ToLower(strAny(task, "status"))
	if st == "success" || st == "succeeded" {
		return e.takeVideoURL(j, task)
	}
	if st == "failed" || st == "fail" {
		return fmt.Errorf("%s", videoErr(task))
	}
	j.Status = "polling"
	return nil
}

// wanSubmit 只接受 wan3.0-video 和 wan3.0-video-prime，其他模型名改成 wan3.0-video。提示里的 @图片 换成「图」。请求带上 X-DashScope-Async: enable。
func (e *Engine) wanSubmit(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	model := first(j.Model, p.Model)
	if model != "wan3.0-video" && model != "wan3.0-video-prime" {
		model = "wan3.0-video"
	}
	prompt := strings.ReplaceAll(j.Prompt, "@图片", "图")
	media := []any{}
	for _, u := range params.ReferenceImageURLs {
		media = append(media, map[string]any{"type": "reference_image", "url": u})
	}
	input := map[string]any{"prompt": prompt}
	if len(media) > 0 {
		input["media"] = media
	}
	body := map[string]any{
		"model": model,
		"input": input,
		"parameters": map[string]any{
			"duration":   ClampProviderDuration(params.Duration, "aliyun"),
			"resolution": NormalizeVideoResolution(params.Resolution, "aliyun"),
			"ratio":      first(params.AspectRatio, "16:9"),
			"audio":      params.GenerateAudio,
		},
	}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/api/v1", "/services/aigc/video-generation/video-synthesis"), key, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-DashScope-Async", "enable")
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	out := pickData(m)
	if id := strAny(out, "task_id"); id != "" {
		j.RemoteID = id
		j.Status = "polling"
		return nil
	}
	return e.takeVideoURL(j, m)
}

// wanPoll 查询 /api/v1/tasks。succeeded 或 success 收地址，failed 返回错误。
func (e *Engine) wanPoll(ctx context.Context, p Provider, key string, j *Job) error {
	req, err := jsonReq(http.MethodGet, joinURL(p.BaseURL, "/api/v1", "/tasks/"+j.RemoteID), key, nil)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	out := pickData(m)
	st := strings.ToLower(strAny(out, "task_status", "status"))
	if st == "succeeded" || st == "success" {
		return e.takeVideoURL(j, out)
	}
	if st == "failed" {
		return fmt.Errorf("%s", videoErr(out))
	}
	j.Status = "polling"
	return nil
}

// takeVideoURL 从 data、content 或 data 列表里找视频地址并下载，收成视频。没有地址时返回 video completed without url。
func (e *Engine) takeVideoURL(j *Job, m map[string]any) error {
	u := strAny(pickData(m), "video_url", "url")
	if u == "" {
		if c, ok := m["content"].(map[string]any); ok {
			u = strAny(c, "url", "video_url")
		}
	}
	if u == "" {
		u = firstURL(m["data"])
	}
	if u == "" {
		return fmt.Errorf("video completed without url")
	}
	raw, err := e.DownloadURL(u)
	if err != nil {
		return err
	}
	return e.finishMedia(j, raw, true)
}

// refBytes 对 data: 地址解码。长度为 64 或 32 时先按内容寻址读取。否则按 URL 下载。
func (e *Engine) refBytes(ref string) ([]byte, error) {
	if strings.HasPrefix(ref, "data:") {
		return decodeDataURL(ref)
	}
	if len(ref) == 64 || len(ref) == 32 {
		if b, err := e.GetBytes(ref); err == nil {
			return b, nil
		}
	}
	return e.DownloadURL(ref)
}

// videoErr 优先返回 [code] message。没有结构化错误时用 message 或 error 字段。都没有时返回 generation failed。
func videoErr(m map[string]any) string {
	if e, ok := m["error"].(map[string]any); ok {
		code, _ := e["code"].(string)
		msg, _ := e["message"].(string)
		if code != "" {
			return "[" + code + "] " + msg
		}
		return msg
	}
	if s := strAny(m, "message", "error"); s != "" {
		return s
	}
	return "generation failed"
}

// splitSize 把小写的 WxH 拆成正整数宽高。不是两段或不是正整数时返回假。
func splitSize(s string) (int, int, bool) {
	parts := strings.Split(strings.ToLower(s), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	var w, h int
	_, err1 := fmt.Sscanf(parts[0], "%d", &w)
	_, err2 := fmt.Sscanf(parts[1], "%d", &h)
	return w, h, err1 == nil && err2 == nil && w > 0 && h > 0
}
