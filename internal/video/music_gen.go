// music_gen.go 按已选配乐协议向供应商要音频字节。

package video

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// scoreMusic 按协议分发配乐请求。提示词为空时返回「配乐没有描述」。上下文为空时用 Background，超时 4 分钟。minimax-music、mureka、elevenlabs-sound、stability-audio 会真正发请求。google-lyria、suno 和其他协议返回 presetOnly，说明这行预设还不能直接发请求。
// ctx 为 nil 时换成 Background，再加 4 分钟超时后传给具体配乐函数。
// protocol 去掉空白并转成小写后与 minimax-music、mureka、elevenlabs-sound、stability-audio、google-lyria、suno 比较。
// key 传给 minimaxMusic、murekaMusic、elevenMusic 或 stabilityMusic。
// prompt 去掉空白后为空则返回配乐没有描述，否则传给具体配乐函数。
// model 先与供应商模型传给 first，再按协议与 music-3.0、auto、eleven_text_to_sound_v2 或 stable-audio-2.5 传给 first。
func (e *Engine) scoreMusic(ctx context.Context, p Provider, protocol, key, prompt, model string) ([]byte, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("配乐没有描述")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	model = first(model, p.Model)
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "minimax-music":
		return e.minimaxMusic(ctx, p, key, prompt, first(model, "music-3.0"))
	case "mureka":
		return e.murekaMusic(ctx, p, key, prompt, first(model, "auto"))
	case "elevenlabs-sound":
		return e.elevenMusic(ctx, p, key, prompt, first(model, "eleven_text_to_sound_v2"))
	case "stability-audio":
		return e.stabilityMusic(ctx, p, key, prompt, first(model, "stable-audio-2.5"))
	case "google-lyria":
		return nil, presetOnly("配乐连接已选中。Google Lyria 要走 Vertex 项目，这行预设还不能直接发请求")
	case "suno":
		return nil, presetOnly("配乐连接已选中。Suno 兼容网关还没有统一请求体")
	default:
		return nil, presetOnly("配乐连接已选中，这条供应商还没有配乐请求")
	}
}

// minimaxMusic 向 /v1/music_generation 发送纯器乐请求，采样率 44100、比特率 256000、格式 mp3，输出 hex。base_resp.status_code 非 0 时失败。data.status 为 1 时返回「生成还在进行」。音频以 http 开头则下载，否则按十六进制解码。
// ctx 设到请求上。
// key 传给 jsonReq。
// prompt 写入请求的 prompt。
// model 写入请求的 model。
func (e *Engine) minimaxMusic(ctx context.Context, p Provider, key, prompt, model string) ([]byte, error) {
	body := map[string]any{
		"model":           model,
		"prompt":          prompt,
		"is_instrumental": true,
		"stream":          false,
		"output_format":   "hex",
		"audio_setting": map[string]any{
			"sample_rate": 44100,
			"bitrate":     256000,
			"format":      "mp3",
		},
	}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/v1", "/music_generation"), key, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	m, _, _, err := e.doJSON(req)
	if err != nil {
		return nil, err
	}
	if br, ok := m["base_resp"].(map[string]any); ok {
		if code := intAny(br["status_code"]); code != 0 {
			msg := strAny(br, "status_msg")
			if msg == "" {
				msg = "minimax music failed"
			}
			return nil, fmt.Errorf("%s", msg)
		}
	}
	data, _ := m["data"].(map[string]any)
	if intAny(data["status"]) == 1 {
		return nil, presetOnly("生成还在进行")
	}
	audio := strAny(data, "audio")
	if strings.HasPrefix(audio, "http") {
		return e.DownloadURL(audio)
	}
	raw, err := hex.DecodeString(strings.TrimSpace(audio))
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("minimax music returned no audio")
	}
	return raw, nil
}

// murekaMusic 用歌词占位 [Instrumental] 调用 /v1/song/generate，再每秒查询 /v1/song/query。succeeded、success、completed 时下载音频地址。failed、error、timeouted、cancelled、canceled 时返回错误。没有任务标识时失败。
// ctx 传给 doJSON，查询循环里取消则返回。
// key 传给 jsonReq。
// prompt 写入请求的 prompt。
// model 写入请求的 model。
func (e *Engine) murekaMusic(ctx context.Context, p Provider, key, prompt, model string) ([]byte, error) {
	body := map[string]any{
		"lyrics": "[Instrumental]",
		"model":  model,
		"prompt": prompt,
	}
	req, err := jsonReq(http.MethodPost, joinURL(p.BaseURL, "/v1", "/song/generate"), key, body)
	if err != nil {
		return nil, err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	id := strAny(m, "id", "task_id")
	if id == "" {
		return nil, fmt.Errorf("mureka returned no task")
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		q, err := jsonReq(http.MethodGet, joinURL(p.BaseURL, "/v1", "/song/query/"+id), key, nil)
		if err != nil {
			return nil, err
		}
		got, _, _, err := e.doJSON(q.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		status := strings.ToLower(strAny(got, "status"))
		switch status {
		case "succeeded", "success", "completed":
			if u := murekaAudioURL(got); u != "" {
				return e.DownloadURL(u)
			}
			return nil, fmt.Errorf("mureka returned no audio")
		case "failed", "error", "timeouted", "cancelled", "canceled":
			msg := strAny(got, "error", "message")
			if msg == "" {
				msg = "mureka music failed"
			}
			return nil, fmt.Errorf("%s", msg)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// murekaAudioURL 先看 url 和 audio_url，再看 choices 里的 url、flac_url 和 audio_url。
func murekaAudioURL(m map[string]any) string {
	if u := strAny(m, "url", "audio_url"); u != "" {
		return u
	}
	choices, _ := m["choices"].([]any)
	for _, item := range choices {
		row, _ := item.(map[string]any)
		if u := strAny(row, "url", "flac_url", "audio_url"); u != "" {
			return u
		}
	}
	return ""
}

// elevenMusic 默认向 /sound-generation 发送 12 秒请求。模型名含 music 时改走 /music，长度为 20000 毫秒。基址不含 /v1 时补上这个前缀。鉴权头是 xi-api-key。
// ctx 传给 NewRequestWithContext。
// prompt 在模型名含 music 时写入 prompt，否则写入 text。
// model 含 music 时改走 /music 且长度为 20000 毫秒，否则写入 model_id 且时长为 12 秒。
func (e *Engine) elevenMusic(ctx context.Context, p Provider, key, prompt, model string) ([]byte, error) {
	path := "/sound-generation"
	body := map[string]any{"text": prompt, "model_id": model, "duration_seconds": 12}
	if strings.Contains(model, "music") {
		path = "/music"
		body = map[string]any{"prompt": prompt, "music_length_ms": 20000}
	}
	prefix := ""
	if !strings.Contains(p.BaseURL, "/v1") {
		prefix = "/v1"
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(p.BaseURL, prefix, path), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", key)
	return e.readAudioBody(req)
}

// stabilityMusic 用 multipart 向 Stable Audio 要 20 秒 mp3。模型名含 stable-audio-3 时改走 /v2beta/audio/stable-audio/text-to-audio，否则走 stable-audio-2。鉴权是 Bearer，Accept 为 audio/*。
// ctx 传给 NewRequestWithContext。
// key 写成 Authorization Bearer。
// prompt 写入表单字段 prompt。
// model 含 stable-audio-3 时改走 /v2beta/audio/stable-audio/text-to-audio，非空时再写入表单字段 model。
func (e *Engine) stabilityMusic(ctx context.Context, p Provider, key, prompt, model string) ([]byte, error) {
	path := "/v2beta/audio/stable-audio-2/text-to-audio"
	if strings.Contains(model, "stable-audio-3") {
		path = "/v2beta/audio/stable-audio/text-to-audio"
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("prompt", prompt)
	_ = w.WriteField("output_format", "mp3")
	_ = w.WriteField("duration", "20")
	if model != "" {
		_ = w.WriteField("model", model)
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(p.BaseURL, "", path), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "audio/*")
	return e.readAudioBody(req)
}

// readAudioBody 最多读 48 MiB。空正文失败。类型含 json 或正文以 { 开头时，有 http 地址就下载，否则返回 message 或 error，都没有则报 music returned no audio。其他正文原样返回。
// req 传给 doRaw。
func (e *Engine) readAudioBody(req *http.Request) ([]byte, error) {
	_, ctype, raw, err := e.doRaw(req, 48<<20)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("music returned no audio")
	}
	if strings.Contains(ctype, "json") || raw[0] == '{' {
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			if u := strAny(m, "url", "audio_url"); strings.HasPrefix(u, "http") {
				return e.DownloadURL(u)
			}
			if s := strAny(m, "message", "error"); s != "" {
				return nil, fmt.Errorf("%s", s)
			}
		}
		return nil, fmt.Errorf("music returned no audio")
	}
	return raw, nil
}
