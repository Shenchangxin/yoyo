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
