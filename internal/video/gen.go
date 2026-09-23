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
