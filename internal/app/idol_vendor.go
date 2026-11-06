// 偶像的语音、识别和生图按厂商说各自的协议。OpenAI 兼容的走 /audio/speech、/audio/transcriptions、/images/generations；
// MiniMax 的语音是 /v1/t2a_v2，音频以十六进制返回；阿里云百炼的千问语音和千问生图走原生的 multimodal-generation 接口，
// 千问识别（qwen3-asr-flash）只能经兼容模式的 chat/completions 带 input_audio 调用。把它们都当成 OpenAI 协议会直接 404。
// 每个请求都带上下文，超时或挂断时随之取消。

package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/idol"
)

const (
	routeOpenAI    = "openai"
	routeMiniMax   = "minimax"
	routeDashScope = "dashscope"
)

// mediaRoute 按预设 id 和地址认出厂商。认不出的都按 OpenAI 兼容。
func mediaRoute(provider, base string) string {
	switch provider {
	case "minimax-speech":
		return routeMiniMax
	case "qwen-speech", "qwen-asr", "qwen-image":
		return routeDashScope
	}
	host := strings.ToLower(endpointHostOf(base))
	switch {
	case strings.Contains(host, "minimax"):
		return routeMiniMax
	case strings.Contains(host, "dashscope"), strings.HasSuffix(host, "maas.aliyuncs.com"):
		return routeDashScope
	}
	return routeOpenAI
}

// endpointHostOf 取地址的主机名，解析不了时返回空串。
func endpointHostOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// endpointOriginOf 取地址的协议和主机，例如 https://dashscope.aliyuncs.com。解析不了时用 fallback。
func endpointOriginOf(raw, fallback string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fallback
	}
	return u.Scheme + "://" + u.Host
}

// idolPost 发一个 JSON 请求，返回响应体（最多 limit 字节）。状态码不是 2xx 时把正文前一段写进错误。
func idolPost(ctx context.Context, u, key string, body any, limit int64) ([]byte, string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return idolDo(req, limit)
}

// idolDo 发请求并读回正文。
func idolDo(req *http.Request, limit int64) ([]byte, string, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("%s: %s", resp.Status, clipText(string(bytes.TrimSpace(b)), 300))
	}
	return b, resp.Header.Get("Content-Type"), nil
}

// idolGet 下载一个地址（厂商返回的音频或图片链接）。
func idolGet(ctx context.Context, u string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	b, _, err := idolDo(req, limit)
	return b, err
}

// clipText 把 s 截到 n 个字符以内。
func clipText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// audioType 按文件头认出音频格式，认不出时用 fallback。
func audioType(b []byte, fallback string) string {
	switch {
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WAVE":
		return "audio/wav"
	case len(b) >= 4 && string(b[:4]) == "OggS":
		return "audio/ogg"
	case len(b) >= 4 && string(b[:4]) == "fLaC":
		return "audio/flac"
	case len(b) >= 3 && string(b[:3]) == "ID3", len(b) >= 2 && b[0] == 0xff && b[1]&0xe0 == 0xe0:
		return "audio/mpeg"
	}
	if fallback == "" || strings.HasPrefix(fallback, "application/") {
		return "audio/mpeg"
	}
	return fallback
}

// speechLang 把偶像的口语语言码换成千问语音认的 language_type。
func speechLang(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "ja":
		return "Japanese"
	case "en":
		return "English"
	case "ko":
		return "Korean"
	case "fr":
		return "French"
	case "de":
		return "German"
	case "es":
		return "Spanish"
	case "ru":
		return "Russian"
	default:
		return "Chinese"
	}
}

// minimaxEmotion 把语气提示归到 MiniMax 认的情绪，认不出时返回空串（不传）。
func minimaxEmotion(cue string) string {
	switch idol.CueEmotion(cue) {
	case "happy":
		return "happy"
	case "sad":
		return "sad"
	case "angry":
		return "angry"
	case "surprised":
		return "surprised"
	}
	return ""
}

// speechCall 是一次合成要用的东西。
type speechCall struct {
	route, base, key, model, voice, text, cue, lang string
}

// synthesize 按厂商合成一句话，返回音频和 Content-Type。
func synthesize(ctx context.Context, c speechCall) ([]byte, string, error) {
	switch c.route {
	case routeMiniMax:
		return speakMiniMax(ctx, c)
	case routeDashScope:
		return speakDashScope(ctx, c)
	default:
		return speakOpenAI(ctx, c)
	}
}

// speakOpenAI 走 OpenAI 兼容的 /audio/speech。gpt-4o 系列的语音模型认 instructions，语气提示写进去。
func speakOpenAI(ctx context.Context, c speechCall) ([]byte, string, error) {
	model := firstNonEmpty(c.model, "gpt-4o-mini-tts")
	body := map[string]any{"model": model, "input": c.text, "voice": c.voice}
	if c.cue != "" && strings.Contains(model, "gpt-4o") {
		body["instructions"] = "Speak " + c.cue + "."
	}
	b, ctype, err := idolPost(ctx, idol.MediaURL(c.base, "audio/speech"), c.key, body, 8<<20)
	if err != nil {
		return nil, "", fmt.Errorf("tts %w", err)
	}
	return b, audioType(b, ctype), nil
}

// speakMiniMax 走 MiniMax 的 /v1/t2a_v2：音频以十六进制放在 data.audio，base_resp.status_code 非 0 是失败。语气提示能对上时带上 emotion。
func speakMiniMax(ctx context.Context, c speechCall) ([]byte, string, error) {
	voice := map[string]any{"voice_id": c.voice, "speed": 1, "vol": 1, "pitch": 0}
	if e := minimaxEmotion(c.cue); e != "" {
		voice["emotion"] = e
	}
	body := map[string]any{
		"model": firstNonEmpty(c.model, "speech-2.6-turbo"), "text": c.text, "stream": false,
		"output_format": "hex", "language_boost": "auto", "voice_setting": voice,
		"audio_setting": map[string]any{"sample_rate": 32000, "bitrate": 128000, "format": "mp3", "channel": 1},
	}
	raw, _, err := idolPost(ctx, endpointOriginOf(c.base, "https://api.minimax.io")+"/v1/t2a_v2", c.key, body, 16<<20)
	if err != nil {
		return nil, "", fmt.Errorf("tts %w", err)
	}
	var out struct {
		Data struct {
			Audio string `json:"audio"`
		} `json:"data"`
		BaseResp struct {
			Code int    `json:"status_code"`
			Msg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", fmt.Errorf("tts: MiniMax returned %s", clipText(string(raw), 200))
	}
	if out.BaseResp.Code != 0 {
		return nil, "", fmt.Errorf("tts: MiniMax %s (%d)", out.BaseResp.Msg, out.BaseResp.Code)
	}
	audio, err := hex.DecodeString(out.Data.Audio)
	if err != nil || len(audio) == 0 {
		return nil, "", fmt.Errorf("tts: MiniMax audio could not be decoded")
	}
	return audio, "audio/mpeg", nil
}

// speakDashScope 走百炼的千问语音 multimodal-generation 接口。返回一个音频地址或 base64，地址要再下载。
func speakDashScope(ctx context.Context, c speechCall) ([]byte, string, error) {
	body := map[string]any{"model": firstNonEmpty(c.model, "qwen3-tts-flash"), "input": map[string]any{
		"text": c.text, "voice": c.voice, "language_type": speechLang(c.lang),
	}}
	raw, _, err := idolPost(ctx, endpointOriginOf(c.base, "https://dashscope.aliyuncs.com")+"/api/v1/services/aigc/multimodal-generation/generation", c.key, body, 4<<20)
	if err != nil {
		return nil, "", fmt.Errorf("tts %w", err)
	}
	var out struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Output  struct {
			Audio struct {
				URL  string `json:"url"`
				Data string `json:"data"`
			} `json:"audio"`
		} `json:"output"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, "", fmt.Errorf("tts: DashScope returned %s", clipText(string(raw), 200))
	}
	var audio []byte
	switch {
	case out.Output.Audio.Data != "":
		audio, err = base64.StdEncoding.DecodeString(out.Output.Audio.Data)
	case out.Output.Audio.URL != "":
		audio, err = idolGet(ctx, out.Output.Audio.URL, 16<<20)
	default:
		return nil, "", fmt.Errorf("tts: DashScope returned no audio: %s %s", out.Code, out.Message)
	}
	if err != nil {
		return nil, "", fmt.Errorf("tts: %w", err)
	}
	return audio, audioType(audio, "audio/wav"), nil
}

// transcribeCall 是一次识别要用的东西。hint 是热词（她的名字、称呼），帮识别写对专有名词。
type transcribeCall struct {
	route, base, key, model, filename, hint string
	audio                                   []byte
}

// transcribe 按厂商识别一段录音。
func transcribe(ctx context.Context, c transcribeCall) (string, error) {
	if c.route == routeDashScope {
		return transcribeDashScope(ctx, c)
	}
	return transcribeOpenAI(ctx, c)
}

// transcribeOpenAI 走 OpenAI 兼容的 /audio/transcriptions（multipart）。热词写进 prompt 字段。JSON 没有 text 时返回正文本身。
func transcribeOpenAI(ctx context.Context, c transcribeCall) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", firstNonEmpty(c.model, "gpt-4o-mini-transcribe"))
	if c.hint != "" {
		_ = w.WriteField("prompt", c.hint)
	}
	fw, err := w.CreateFormFile("file", firstNonEmpty(c.filename, "audio.webm"))
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(c.audio); err != nil {
		return "", err
	}
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, idol.MediaURL(c.base, "audio/transcriptions"), &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	b, _, err := idolDo(req, 1<<20)
	if err != nil {
		return "", fmt.Errorf("stt %w", err)
	}
	var out struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(b, &out) != nil {
		return strings.TrimSpace(string(b)), nil
	}
	return strings.TrimSpace(out.Text), nil
}

// transcribeDashScope 走千问识别：兼容模式的 chat/completions，录音以 data URI 放进 input_audio，识别结果在回复正文里。
// 热词放在第一条 system 消息里，千问识别把它当作背景文本和词表。
func transcribeDashScope(ctx context.Context, c transcribeCall) (string, error) {
	mime := "audio/webm"
	switch {
	case strings.HasSuffix(c.filename, ".wav"):
		mime = "audio/wav"
	case strings.HasSuffix(c.filename, ".mp3"):
		mime = "audio/mpeg"
	case strings.HasSuffix(c.filename, ".ogg"):
		mime = "audio/ogg"
	}
	messages := []any{}
	if c.hint != "" {
		messages = append(messages, map[string]any{"role": "system", "content": []any{map[string]any{"text": c.hint}}})
	}
	messages = append(messages, map[string]any{"role": "user", "content": []any{map[string]any{
		"type": "input_audio", "input_audio": map[string]any{"data": "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(c.audio)},
	}}})
	body := map[string]any{
		"model":       firstNonEmpty(c.model, "qwen3-asr-flash"),
		"messages":    messages,
		"stream":      false,
		"asr_options": map[string]any{"enable_itn": false},
	}
	base := c.base
	if !strings.Contains(base, "compatible-mode") {
		base = endpointOriginOf(base, "https://dashscope.aliyuncs.com") + "/compatible-mode/v1"
	}
	raw, _, err := idolPost(ctx, idol.MediaURL(base, "chat/completions"), c.key, body, 1<<20)
	if err != nil {
		return "", fmt.Errorf("stt %w", err)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("stt: DashScope returned %s", clipText(string(raw), 200))
	}
	switch v := out.Choices[0].Message.Content.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case []any:
		var b strings.Builder
		for _, part := range v {
			if m, ok := part.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					b.WriteString(t)
				}
			}
		}
		return strings.TrimSpace(b.String()), nil
	}
	return "", nil
}

// paintCall 是一次生图要用的东西。
type paintCall struct {
	route, base, key, model, prompt string
}

// paint 按厂商画一张图，返回图片字节。
func paint(ctx context.Context, c paintCall) ([]byte, error) {
	if c.route == routeDashScope {
		return paintDashScope(ctx, c)
	}
	return paintOpenAI(ctx, c)
}

// paintOpenAI 走 OpenAI 兼容的 /images/generations，1024x1024。优先用 b64_json，否则下载返回的地址（智谱这类只给地址）。
func paintOpenAI(ctx context.Context, c paintCall) ([]byte, error) {
	b, _, err := idolPost(ctx, idol.MediaURL(c.base, "images/generations"), c.key, map[string]any{
		"model": firstNonEmpty(c.model, "gpt-image-1"), "prompt": c.prompt, "size": "1024x1024",
	}, 32<<20)
	if err != nil {
		return nil, fmt.Errorf("image %w", err)
	}
	var out struct {
		Data []struct {
			B64 string `json:"b64_json"`
			URL string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(b, &out) != nil || len(out.Data) == 0 {
		return nil, fmt.Errorf("image response was not a picture")
	}
	if out.Data[0].B64 != "" {
		return decodeIdolB64(out.Data[0].B64)
	}
	if out.Data[0].URL == "" {
		return nil, fmt.Errorf("image response was empty")
	}
	return idolGet(ctx, out.Data[0].URL, 24<<20)
}

// paintDashScope 走百炼千问生图的同步 multimodal-generation 接口，返回的是 24 小时有效的图片地址，再下载。
// qwen-image-2.0 系列用 1024*1024，其余型号用它们支持的 1328*1328。
func paintDashScope(ctx context.Context, c paintCall) ([]byte, error) {
	model := firstNonEmpty(c.model, "qwen-image-2.0")
	size := "1328*1328"
	if strings.Contains(model, "2.0") {
		size = "1024*1024"
	}
	body := map[string]any{
		"model": model,
		"input": map[string]any{"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"text": c.prompt}}}}},
		"parameters": map[string]any{
			"size": size, "watermark": false, "prompt_extend": true, "n": 1,
			"negative_prompt": "低分辨率，低画质，肢体畸形，手指畸形，文字，水印，裸露",
		},
	}
	raw, _, err := idolPost(ctx, endpointOriginOf(c.base, "https://dashscope.aliyuncs.com")+"/api/v1/services/aigc/multimodal-generation/generation", c.key, body, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("image %w", err)
	}
	var out struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Output  struct {
			Choices []struct {
				Message struct {
					Content []struct {
						Image string `json:"image"`
					} `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("image: DashScope returned %s", clipText(string(raw), 200))
	}
	for _, ch := range out.Output.Choices {
		for _, part := range ch.Message.Content {
			if part.Image != "" {
				return idolGet(ctx, part.Image, 24<<20)
			}
		}
	}
	return nil, fmt.Errorf("image: DashScope returned no picture: %s %s", out.Code, out.Message)
}

// silentWAV 是 0.4 秒 16kHz 单声道的静音 WAV，试识别连接时用。
func silentWAV() []byte {
	const rate, samples = 16000, 6400
	data := samples * 2
	b := make([]byte, 44+data)
	copy(b[0:], "RIFF")
	putLE32(b[4:], uint32(36+data))
	copy(b[8:], "WAVEfmt ")
	putLE32(b[16:], 16)
	putLE16(b[20:], 1)
	putLE16(b[22:], 1)
	putLE32(b[24:], rate)
	putLE32(b[28:], rate*2)
	putLE16(b[32:], 2)
	putLE16(b[34:], 16)
	copy(b[36:], "data")
	putLE32(b[40:], uint32(data))
	return b
}

func putLE16(b []byte, v uint16) { b[0], b[1] = byte(v), byte(v>>8) }
func putLE32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}
