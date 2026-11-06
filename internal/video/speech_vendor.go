// speech_vendor.go 按语音连接认出厂商，走它自己的合成接口。OpenAI 兼容的走 /audio/speech；
// MiniMax、火山引擎、通义、腾讯云、讯飞、ElevenLabs、Azure 各有各的请求格式和鉴权，不能都当成 /audio/speech。
// 长口播按厂商一次能合成的长度按句切段，逐段合成，再拼成一条 mp3。

package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/coder/websocket"
)

// speechCall 是合成一段要用的连接信息。Route 决定请求格式，Catalog 决定声音目录和分段长度，Voice 已经换成这家认得的声音。
type speechCall struct {
	Route    string
	Catalog  string
	Endpoint string
	Key      string
	Model    string
	Voice    speechVoice
	Speed    float64
	Settings map[string]any
}

// speechOut 是一次配音的结果。Engine 是 cloud 或 edge，Vendor 是厂商中文名，Notes 是要告诉操作者的改动。
type speechOut struct {
	Audio  []byte
	Engine string
	Vendor string
	Voice  speechVoice
	Notes  []string
}

// speechRoute 认出语音连接该走哪家的接口。登记了专门协议的按协议；老连接都登记成 openai-audio，再看地址的主机名；
// 地址空着才看厂商名。都认不出时走 OpenAI 兼容接口，网关和自建服务多半是这种。
func speechRoute(c connection.Connection) string {
	switch strings.ToLower(strings.TrimSpace(c.Protocol)) {
	case "minimax-t2a":
		return "minimax"
	case "volcengine-tts":
		return "volcengine"
	case "dashscope-tts":
		return "dashscope"
	case "tencent-tts":
		return "tencent"
	case "xfyun-tts":
		return "spark"
	case "elevenlabs-tts":
		return "elevenlabs"
	case "azure-tts":
		return "azure"
	}
	host := endpointHost(c.Endpoint)
	switch {
	case strings.Contains(host, "minimax"):
		return "minimax"
	case strings.Contains(host, "openspeech.bytedance.com"):
		return "volcengine"
	case strings.Contains(host, "dashscope"), strings.HasSuffix(host, "maas.aliyuncs.com"):
		return "dashscope"
	case strings.Contains(host, "tencentcloudapi.com"), strings.Contains(host, "tts.cloud.tencent.com"):
		return "tencent"
	case strings.Contains(host, "xfyun.cn"), strings.Contains(host, "xf-yun.com"):
		return "spark"
	case strings.Contains(host, "elevenlabs.io"):
		return "elevenlabs"
	case strings.Contains(host, "speech.microsoft.com"):
		return "azure"
	}
	if host == "" {
		switch strings.ToLower(strings.TrimSpace(c.Vendor)) {
		case "minimax":
			return "minimax"
		case "volcengine":
			return "volcengine"
		case "aliyun", "dashscope", "qwen":
			return "dashscope"
		case "tencent":
			return "tencent"
		case "spark", "xfyun", "iflytek":
			return "spark"
		case "elevenlabs":
			return "elevenlabs"
		case "azure", "microsoft":
			return "azure"
		}
	}
	return "openai"
}

// endpointHost 取地址里的主机名，小写。没写协议头的地址也认。
func endpointHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// endpointOrigin 取地址的协议和主机部分，比如 https://api.minimax.io。地址空着或解析不了时用 fallback。
func endpointOrigin(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimRight(fallback, "/")
	}
	return u.Scheme + "://" + u.Host
}

// prepareSpeechCall 认出路由、换好声音，并记下要提前告诉操作者的改动。
// CosyVoice 的 HTTP 接口只在业务空间地址上开放；连接填的是 DashScope 通用地址时改用千问 TTS，同一个 API Key 就能用。
// Azure 和腾讯云的模板把声音写在模型里；工程没选这家的声音时，模型像声音就用模型。
func prepareSpeechCall(c connection.Connection, key, voice string, speed float64) (speechCall, []string) {
	route := speechRoute(c)
	model := strings.TrimSpace(connection.ModelFor(c, connection.CapSpeech))
	catalog := route
	var notes []string
	if route == "dashscope" {
		if strings.HasPrefix(strings.ToLower(model), "cosyvoice") {
			if strings.HasSuffix(endpointHost(c.Endpoint), "maas.aliyuncs.com") {
				catalog = "cosyvoice"
			} else {
				model = "qwen3-tts-flash"
				notes = append(notes, "CosyVoice 的 HTTP 接口要填业务空间地址（https://<WorkspaceId>.cn-beijing.maas.aliyuncs.com），这次改用千问 TTS 配音。")
			}
		}
		if catalog == "dashscope" && speed > 0 && math.Abs(speed-1) > 0.05 {
			notes = append(notes, "千问 TTS 不接受语速，已按原速配音。")
		}
	}
	v := resolveSpeechVoice(catalog, voice)
	if _, own := voiceInRoute(catalog, strings.TrimSpace(voice)); !own && modelIsVoice(route, model) {
		v = resolveSpeechVoice(catalog, model)
	}
	return speechCall{
		Route: route, Catalog: catalog, Endpoint: c.Endpoint, Key: strings.TrimSpace(key),
		Model: model, Voice: v, Speed: speed, Settings: c.Settings,
	}, notes
}

// modelIsVoice 判断模板把声音写在模型字段里的那几家，模型这一栏是不是一个声音。
func modelIsVoice(route, model string) bool {
	switch route {
	case "azure":
		return strings.HasSuffix(model, "Neural")
	case "tencent":
		_, err := strconv.Atoi(model)
		return err == nil
	}
	return false
}

// speechLimits 是每家一次能合成的长度：字数上限和 UTF-8 字节上限，0 表示不限。
func speechLimits(catalog string) (int, int) {
	switch catalog {
	case "openai":
		return 3800, 0
	case "minimax":
		return 2800, 0
	case "volcengine":
		return 280, 1000
	case "dashscope":
		return 500, 0
	case "cosyvoice":
		return 1800, 0
	case "tencent":
		return 140, 0
	case "spark":
		return 1500, 7000
	case "elevenlabs":
		return 4500, 0
	case "azure":
		return 3000, 0
	}
	return 0, 0
}

// speechChunks 把口播按句切成每段不超过 maxRunes 个字、maxBytes 个字节。maxRunes 为 0 或整段放得下时整段返回。
// 一句本身超长时再按逗号、分号、顿号切，还长就按字数硬切。
func speechChunks(text string, maxRunes, maxBytes int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxRunes <= 0 || fitsSpeech(text, maxRunes, maxBytes) {
		return []string{text}
	}
	var pieces []string
	for _, s := range splitSpeak(text) {
		pieces = append(pieces, splitLongSpeech(s, maxRunes, maxBytes)...)
	}
	var out []string
	cur := ""
	for _, p := range pieces {
		next := joinSpeechText(cur, p)
		if cur != "" && !fitsSpeech(next, maxRunes, maxBytes) {
			out = append(out, cur)
			cur = p
			continue
		}
		cur = next
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// fitsSpeech 判断一段放不放得进一次合成。
func fitsSpeech(s string, maxRunes, maxBytes int) bool {
	return utf8.RuneCountInString(s) <= maxRunes && (maxBytes <= 0 || len(s) <= maxBytes)
}

// splitLongSpeech 把超长的一句先按逗号类标点切开再合并，单块还长就按字数硬切。
func splitLongSpeech(s string, maxRunes, maxBytes int) []string {
	if fitsSpeech(s, maxRunes, maxBytes) {
		return []string{s}
	}
	var parts []string
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(r)
		if strings.ContainsRune("，,；;、：:", r) {
			parts = append(parts, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	var out []string
	cur := ""
	for _, p := range parts {
		if !fitsSpeech(p, maxRunes, maxBytes) {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			out = append(out, hardSplitSpeech(p, maxRunes, maxBytes)...)
			continue
		}
		if cur != "" && !fitsSpeech(cur+p, maxRunes, maxBytes) {
			out = append(out, cur)
			cur = p
			continue
		}
		cur += p
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// hardSplitSpeech 按字数和字节数硬切。
func hardSplitSpeech(s string, maxRunes, maxBytes int) []string {
	var out []string
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n+1 > maxRunes || (maxBytes > 0 && b.Len()+utf8.RuneLen(r) > maxBytes) {
			out = append(out, b.String())
			b.Reset()
			n = 0
		}
		b.WriteRune(r)
		n++
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// joinSpeechText 接上两句。两边都是拉丁字母、数字或半角标点时中间补一个空格，其余直接接上。
func joinSpeechText(a, b string) string {
	if a == "" {
		return b
	}
	ra, _ := utf8.DecodeLastRuneInString(a)
	rb, _ := utf8.DecodeRuneInString(b)
	if ra < utf8.RuneSelf && rb < utf8.RuneSelf && ra != ' ' && rb != ' ' {
		return a + " " + b
	}
	return a + b
}

// audioFormat 按文件头认音频格式：mp3、wav、ogg，认不出返回空串。
func audioFormat(b []byte) string {
	switch {
	case len(b) >= 3 && string(b[:3]) == "ID3":
		return "mp3"
	case len(b) >= 2 && b[0] == 0xFF && b[1]&0xE0 == 0xE0:
		return "mp3"
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WAVE":
		return "wav"
	case len(b) >= 4 && string(b[:4]) == "OggS":
		return "ogg"
	}
	return ""
}

// joinSpeech 把分段合成的音频收成一条。
// 只有一段时：mp3 或认不出格式的原样返回；wav、ogg 在本机有 ffmpeg 时转成 mp3，转不成也原样返回。
// 多段时：本机有 ffmpeg 且格式都认得就用它拼接并统一编成 mp3；否则只要没有 wav、ogg 就直接接上字节，有就报错。
func (e *Engine) joinSpeech(parts [][]byte) ([]byte, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("speech returned empty audio")
	}
	allMP3, known, byteSafe := true, true, true
	for _, p := range parts {
		switch audioFormat(p) {
		case "mp3":
		case "":
			allMP3, known = false, false
		default:
			allMP3, byteSafe = false, false
		}
	}
	ff := strings.TrimSpace(e.FFMPEG)
	if len(parts) == 1 {
		if allMP3 || !known || ff == "" {
			return parts[0], nil
		}
		if out, err := e.concatSpeech(ff, parts); err == nil {
			return out, nil
		}
		return parts[0], nil
	}
	if ff != "" && known {
		out, err := e.concatSpeech(ff, parts)
		if err == nil || !allMP3 {
			return out, err
		}
	}
	if byteSafe {
		return bytes.Join(parts, nil), nil
	}
	return nil, fmt.Errorf("分段配音要用 ffmpeg 拼成一条。先到剪辑台装好 ffmpeg")
}

// concatSpeech 用 ffmpeg 的 concat 把几段音频接成一条 128k 单声道 mp3。
func (e *Engine) concatSpeech(ff string, parts [][]byte) ([]byte, error) {
	dir, err := os.MkdirTemp(e.Dir, "speech-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	var list strings.Builder
	for i, p := range parts {
		ext := audioFormat(p)
		if ext == "" {
			ext = "bin"
		}
		name := filepath.Join(dir, fmt.Sprintf("part-%03d.%s", i, ext))
		if err := os.WriteFile(name, p, 0o600); err != nil {
			return nil, err
		}
		fmt.Fprintf(&list, "file '%s'\n", escapeConcat(filepath.ToSlash(name)))
	}
	listPath := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(listPath, []byte(list.String()), 0o600); err != nil {
		return nil, err
	}
	out := filepath.Join(dir, "speech.mp3")
	cmd := quiet(exec.Command(ff, "-y", "-f", "concat", "-safe", "0", "-i", listPath, "-ac", "1", "-c:a", "libmp3lame", "-b:a", "128k", out))
	if b, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("拼接配音失败：%s", ffmpegErr(b, err))
	}
	return os.ReadFile(out)
}

// activeSpeech 返回当前启用、而且填了密钥的语音连接。
func (e *Engine) activeSpeech() (connection.Connection, string, bool) {
	if e.Conn == nil {
		return connection.Connection{}, "", false
	}
	c, err := e.Conn.Active(connection.CapSpeech, "")
	if err != nil {
		return connection.Connection{}, "", false
	}
	key, err := e.Conn.Lease(c)
	if err != nil || strings.TrimSpace(key) == "" {
		return connection.Connection{}, "", false
	}
	return c, key, true
}

// cloudSpeech 用这条语音连接配音：按厂商切段、逐段合成，再拼成一条。任何一段失败都返回错误，不留半截音频。
// c 和 key 是语音连接和它的密钥。text 是口播。voice 是工程选的声音，会换成这家认得的。speed 是语速，1 是原速。
func (e *Engine) cloudSpeech(c connection.Connection, key, text, voice string, speed float64) (speechOut, error) {
	call, notes := prepareSpeechCall(c, key, voice, speed)
	maxRunes, maxBytes := speechLimits(call.Catalog)
	chunks := speechChunks(text, maxRunes, maxBytes)
	if len(chunks) == 0 {
		return speechOut{}, fmt.Errorf("还没有口播可配")
	}
	parts := make([][]byte, 0, len(chunks))
	ignored := false
	for _, chunk := range chunks {
		ctx, cancel := context.WithTimeout(e.speechCtx(), 3*time.Minute)
		raw, speedIgnored, err := e.speechOnce(ctx, call, chunk)
		cancel()
		if err != nil {
			return speechOut{}, err
		}
		if len(raw) < 32 {
			return speechOut{}, fmt.Errorf("%s 返回了空音频", speechVendorNames[call.Catalog])
		}
		ignored = ignored || speedIgnored
		parts = append(parts, raw)
	}
	audio, err := e.joinSpeech(parts)
	if err != nil {
		return speechOut{}, err
	}
	if ignored {
		notes = append(notes, "这个语音接口不接受语速，已按原速配音。")
	}
	return speechOut{Audio: audio, Engine: "cloud", Vendor: speechVendorNames[call.Catalog], Voice: call.Voice, Notes: notes}, nil
}

// speechCtx 是合成请求的上级上下文：引擎关闭时一起取消。
func (e *Engine) speechCtx() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

// speechOnce 按路由合成一段。第二个结果为真表示对方不接受语速、已按原速合成。
func (e *Engine) speechOnce(ctx context.Context, call speechCall, text string) ([]byte, bool, error) {
	var raw []byte
	var err error
	switch call.Route {
	case "minimax":
		raw, err = e.speechMiniMax(ctx, call, text)
	case "volcengine":
		raw, err = e.speechVolcengine(ctx, call, text)
	case "dashscope":
		raw, err = e.speechDashScope(ctx, call, text)
	case "tencent":
		raw, err = e.speechTencent(ctx, call, text)
	case "spark":
		raw, err = e.speechSpark(ctx, call, text)
	case "elevenlabs":
		raw, err = e.speechElevenLabs(ctx, call, text)
	case "azure":
		raw, err = e.speechAzure(ctx, call, text)
	default:
		model := call.Model
		if model == "" {
			model = "gpt-4o-mini-tts"
		}
		return e.ttsOpenAI(call.Endpoint, call.Key, model, text, call.Voice.ID, call.Speed)
	}
	return raw, false, err
}

// speechPost 发一次 JSON 请求，返回响应正文。
func (e *Engine) speechPost(ctx context.Context, u string, headers map[string]string, body any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return e.speechDo(req)
}

// speechDo 发出请求并读完正文，最多 48 MiB。状态码不是 2xx 时把正文开头写进错误。
func (e *Engine) speechDo(req *http.Request) ([]byte, error) {
	client := e.HTTP
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Minute}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 48<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, &httpStatusError{Code: res.StatusCode, Msg: fmt.Sprintf("speech %d: %s", res.StatusCode, clipBody(raw, 300))}
	}
	return raw, nil
}

// clipBody 取正文开头最多 n 个字，当错误文本用。
func clipBody(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

// speedIn 把语速收进 [lo, hi]。小于等于 0 当原速。
func speedIn(speed, lo, hi float64) float64 {
	if speed <= 0 {
		return 1
	}
	return math.Max(lo, math.Min(hi, speed))
}

// speechMiniMax 走 MiniMax 的 /v1/t2a_v2：Bearer 鉴权，音频以十六进制放在 data.audio，base_resp.status_code 非 0 是失败。
// 连接设置里有 group_id 时带上 GroupId，老的 api.minimax.chat 账号要它。
func (e *Engine) speechMiniMax(ctx context.Context, call speechCall, text string) ([]byte, error) {
	u := endpointOrigin(call.Endpoint, "https://api.minimax.io") + "/v1/t2a_v2"
	if g := settingStr(call.Settings, "group_id", "GroupId", "groupId"); g != "" {
		u += "?GroupId=" + url.QueryEscape(g)
	}
	body := map[string]any{
		"model": firstNonEmpty(call.Model, "speech-2.6-hd"), "text": text, "stream": false,
		"output_format": "hex", "language_boost": "auto",
		"voice_setting": map[string]any{"voice_id": call.Voice.ID, "speed": speedIn(call.Speed, 0.5, 2), "vol": 1, "pitch": 0},
		"audio_setting": map[string]any{"sample_rate": 32000, "bitrate": 128000, "format": "mp3", "channel": 1},
	}
	raw, err := e.speechPost(ctx, u, map[string]string{"Authorization": "Bearer " + call.Key}, body)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("MiniMax 返回的不是 JSON：%s", clipBody(raw, 200))
	}
	if out.BaseResp.Code != 0 {
		return nil, fmt.Errorf("MiniMax 配音失败：%s（%d）", out.BaseResp.Msg, out.BaseResp.Code)
	}
	audio, err := hex.DecodeString(out.Data.Audio)
	if err != nil {
		return nil, fmt.Errorf("MiniMax 音频解不开：%w", err)
	}
	return audio, nil
}

// volcSpeechCred 取火山语音的 AppID 和 Access Token：连接设置里有 app_id 时密钥就是 Token，否则密钥写成 AppID:Token。
func volcSpeechCred(call speechCall) (string, string) {
	if app := settingStr(call.Settings, "app_id", "appid", "appId"); app != "" {
		return app, call.Key
	}
	if p := splitCred(call.Key, 2); p != nil {
		return p[0], p[1]
	}
	return "", ""
}

// speechVolcengine 走火山引擎（豆包语音）的 HTTP 一次性合成 /api/v1/tts。鉴权头是 "Bearer;" 接 Token，正文里也要带 AppID。
// code 3000 是成功，音频以 base64 放在 data。集群默认 volcano_tts，连接设置里的 cluster 或以 volcano 开头的模型名优先。
func (e *Engine) speechVolcengine(ctx context.Context, call speechCall, text string) ([]byte, error) {
	appID, token := volcSpeechCred(call)
	if appID == "" || token == "" {
		return nil, fmt.Errorf("火山语音要 AppID 和 Access Token。在这个语音连接的密钥框里填「AppID:AccessToken」（英文冒号隔开）")
	}
	cluster := settingStr(call.Settings, "cluster")
	if cluster == "" {
		cluster = "volcano_tts"
		if strings.HasPrefix(call.Model, "volcano") {
			cluster = call.Model
		}
	}
	body := map[string]any{
		"app":   map[string]any{"appid": appID, "token": token, "cluster": cluster},
		"user":  map[string]any{"uid": "yoyo"},
		"audio": map[string]any{"voice_type": call.Voice.ID, "encoding": "mp3", "speed_ratio": speedIn(call.Speed, 0.2, 3)},
		"request": map[string]any{
			"reqid": NewID(), "text": text, "text_type": "plain", "operation": "query",
		},
	}
	u := endpointOrigin(call.Endpoint, "https://openspeech.bytedance.com") + "/api/v1/tts"
	raw, err := e.speechPost(ctx, u, map[string]string{"Authorization": "Bearer;" + token}, body)
	if err != nil {
		return nil, err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("火山语音返回的不是 JSON：%s", clipBody(raw, 200))
	}
	if out.Code != 3000 {
		return nil, fmt.Errorf("火山语音配音失败：%s（%d）。确认这个声音已在控制台开通", out.Message, out.Code)
	}
	return base64.StdEncoding.DecodeString(out.Data)
}

// speechDashScope 走阿里云百炼。千问 TTS 用 /api/v1/services/aigc/multimodal-generation/generation；
// CosyVoice 只在业务空间地址上用 /api/v1/services/audio/tts/SpeechSynthesizer。两种都返回一个 24 小时有效的音频地址，有时直接给 base64。
func (e *Engine) speechDashScope(ctx context.Context, call speechCall, text string) ([]byte, error) {
	origin := endpointOrigin(call.Endpoint, "https://dashscope.aliyuncs.com")
	var u string
	var body map[string]any
	if call.Catalog == "cosyvoice" {
		u = origin + "/api/v1/services/audio/tts/SpeechSynthesizer"
		body = map[string]any{"model": call.Model, "input": map[string]any{
			"text": text, "voice": call.Voice.ID, "format": "mp3", "sample_rate": 24000, "rate": speedIn(call.Speed, 0.5, 2),
		}}
	} else {
		u = origin + "/api/v1/services/aigc/multimodal-generation/generation"
		body = map[string]any{"model": firstNonEmpty(call.Model, "qwen3-tts-flash"), "input": map[string]any{
			"text": text, "voice": call.Voice.ID, "language_type": "Chinese",
		}}
	}
	raw, err := e.speechPost(ctx, u, map[string]string{"Authorization": "Bearer " + call.Key}, body)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("通义语音返回的不是 JSON：%s", clipBody(raw, 200))
	}
	if out.Output.Audio.Data != "" {
		return base64.StdEncoding.DecodeString(out.Output.Audio.Data)
	}
	if out.Output.Audio.URL != "" {
		return e.DownloadURL(out.Output.Audio.URL)
	}
	return nil, fmt.Errorf("通义语音没有返回音频：%s %s", out.Code, out.Message)
}

// tencentCred 取腾讯云的 SecretId 和 SecretKey：连接设置里有 secret_id 时密钥就是 SecretKey，否则密钥写成 SecretId:SecretKey。
func tencentCred(call speechCall) (string, string) {
	if id := settingStr(call.Settings, "secret_id", "secretId", "SecretId"); id != "" {
		return id, call.Key
	}
	if p := splitCred(call.Key, 2); p != nil {
		return p[0], p[1]
	}
	return "", ""
}

// tencentVoiceType 把声音 id 收成腾讯云的音色编号。认不出时用 101001（智瑜）。
func tencentVoiceType(id string) int {
	if n, err := strconv.Atoi(strings.TrimSpace(id)); err == nil && n >= 0 {
		return n
	}
	return 101001
}

// tencentSpeed 把倍速换成腾讯云的语速档：0.6 倍是 -2，0.8 倍是 -1，1 倍是 0，1.2 倍是 1，1.5 倍是 2，2.5 倍是 6，中间按直线插值。
func tencentSpeed(speed float64) float64 {
	if speed <= 0 {
		return 0
	}
	pts := [][2]float64{{0.6, -2}, {0.8, -1}, {1, 0}, {1.2, 1}, {1.5, 2}, {2.5, 6}}
	if speed <= pts[0][0] {
		return pts[0][1]
	}
	for i := 1; i < len(pts); i++ {
		if speed <= pts[i][0] {
			a, b := pts[i-1], pts[i]
			v := a[1] + (speed-a[0])/(b[0]-a[0])*(b[1]-a[1])
			return math.Round(v*100) / 100
		}
	}
	return pts[len(pts)-1][1]
}

// speechTencent 走腾讯云 TextToVoice（2019-08-23），TC3-HMAC-SHA256 签名。一次最多 150 个汉字，音频以 base64 放在 Response.Audio。
// 模板里的 tts.cloud.tencent.com 不是 API 地址，遇到它或地址空着时改用 tts.tencentcloudapi.com。
func (e *Engine) speechTencent(ctx context.Context, call speechCall, text string) ([]byte, error) {
	id, secret := tencentCred(call)
	if id == "" || secret == "" {
		return nil, fmt.Errorf("腾讯云语音要 SecretId 和 SecretKey。在这个语音连接的密钥框里填「SecretId:SecretKey」（英文冒号隔开）")
	}
	origin := endpointOrigin(call.Endpoint, "https://tts.tencentcloudapi.com")
	if h := endpointHost(origin); h == "tts.cloud.tencent.com" || h == "" {
		origin = "https://tts.tencentcloudapi.com"
	}
	body, _ := json.Marshal(map[string]any{
		"Text": text, "SessionId": NewID(), "VoiceType": tencentVoiceType(call.Voice.ID), "Codec": "mp3",
		"Speed": tencentSpeed(call.Speed), "SampleRate": 16000, "PrimaryLanguage": 1, "ModelType": 1,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	signTC3(req, body, id, secret, "tts", "TextToVoice", "2019-08-23", settingStr(call.Settings, "region"), time.Now())
	raw, err := e.speechDo(req)
	if err != nil {
		return nil, err
	}
	var out struct {
		Response struct {
			Audio string `json:"Audio"`
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("腾讯云语音返回的不是 JSON：%s", clipBody(raw, 200))
	}
	if out.Response.Error != nil {
		return nil, fmt.Errorf("腾讯云语音配音失败：%s（%s）", out.Response.Error.Message, out.Response.Error.Code)
	}
	return base64.StdEncoding.DecodeString(out.Response.Audio)
}

// sparkCred 取讯飞的 APPID、APIKey 和 APISecret，密钥写成 APPID:APIKey:APISecret。
func sparkCred(call speechCall) []string {
	if app := settingStr(call.Settings, "app_id", "appid"); app != "" {
		if p := splitCred(call.Key, 2); p != nil {
			return []string{app, p[0], p[1]}
		}
	}
	return splitCred(call.Key, 3)
}

// sparkSpeed 把倍速换成讯飞的 0 到 100，50 是原速。
func sparkSpeed(speed float64) int {
	s := speedIn(speed, 0.5, 2)
	if s >= 1 {
		return int(math.Round(50 + (s-1)*50))
	}
	return int(math.Round(50 - (1-s)*100))
}

// speechSpark 走讯飞在线语音合成的 WebSocket 接口 wss://tts-api.xfyun.cn/v2/tts。
// 鉴权写在地址的 authorization、date、host 三个参数里，签名是对 host、date 和请求行做 HMAC-SHA256。音频分帧以 base64 回来，status 为 2 是最后一帧。
func (e *Engine) speechSpark(ctx context.Context, call speechCall, text string) ([]byte, error) {
	cred := sparkCred(call)
	if cred == nil {
		return nil, fmt.Errorf("讯飞语音要 APPID、APIKey 和 APISecret。在这个语音连接的密钥框里填「APPID:APIKey:APISecret」（英文冒号隔开）")
	}
	scheme, host, path := "wss", "tts-api.xfyun.cn", "/v2/tts"
	if u, err := url.Parse(strings.TrimSpace(call.Endpoint)); err == nil && (u.Scheme == "ws" || u.Scheme == "wss") && u.Host != "" {
		scheme, host = u.Scheme, u.Host
		if u.Path != "" {
			path = u.Path
		}
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	sig := base64.StdEncoding.EncodeToString(hmacSHA256([]byte(cred[2]), []byte("host: "+host+"\ndate: "+date+"\nGET "+path+" HTTP/1.1")))
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`, cred[1], sig)))
	q := url.Values{"authorization": {auth}, "date": {date}, "host": {host}}
	conn, _, err := websocket.Dial(ctx, scheme+"://"+host+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("连不上讯飞语音：%w", err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(16 << 20)
	frame, _ := json.Marshal(map[string]any{
		"common": map[string]any{"app_id": cred[0]},
		"business": map[string]any{
			"aue": "lame", "sfl": 1, "auf": "audio/L16;rate=16000", "vcn": call.Voice.ID, "speed": sparkSpeed(call.Speed), "tte": "UTF8",
		},
		"data": map[string]any{"status": 2, "text": base64.StdEncoding.EncodeToString([]byte(text))},
	})
	if err := conn.Write(ctx, websocket.MessageText, frame); err != nil {
		return nil, err
	}
	var audio bytes.Buffer
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			return nil, fmt.Errorf("讯飞语音中途断开：%w", err)
		}
		var res struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Audio  string `json:"audio"`
				Status int    `json:"status"`
			} `json:"data"`
		}
		if err := json.Unmarshal(msg, &res); err != nil {
			return nil, fmt.Errorf("讯飞语音返回的不是 JSON：%s", clipBody(msg, 200))
		}
		if res.Code != 0 {
			return nil, fmt.Errorf("讯飞语音配音失败：%s（%d）", res.Message, res.Code)
		}
		if res.Data.Audio != "" {
			chunk, err := base64.StdEncoding.DecodeString(res.Data.Audio)
			if err != nil {
				return nil, err
			}
			audio.Write(chunk)
		}
		if res.Data.Status == 2 {
			break
		}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
	return audio.Bytes(), nil
}

// speechElevenLabs 走 ElevenLabs 的 /v1/text-to-speech/{voice_id}，密钥放在 xi-api-key，直接回 mp3。
func (e *Engine) speechElevenLabs(ctx context.Context, call speechCall, text string) ([]byte, error) {
	base := strings.TrimRight(firstNonEmpty(strings.TrimSpace(call.Endpoint), "https://api.elevenlabs.io/v1"), "/")
	if !versionTail.MatchString(base) {
		base += "/v1"
	}
	u := base + "/text-to-speech/" + url.PathEscape(call.Voice.ID) + "?output_format=mp3_44100_128"
	body := map[string]any{"text": text, "model_id": firstNonEmpty(call.Model, "eleven_multilingual_v2")}
	if call.Speed > 0 && math.Abs(call.Speed-1) > 0.05 {
		body["voice_settings"] = map[string]any{"speed": speedIn(call.Speed, 0.7, 1.2)}
	}
	return e.speechPost(ctx, u, map[string]string{"xi-api-key": call.Key, "Accept": "audio/mpeg"}, body)
}

// azureSSMLEscape 转义 SSML 里不能直接出现的字符。
var azureSSMLEscape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

// speechAzure 走 Azure 语音的 /cognitiveservices/v1：密钥放在 Ocp-Apim-Subscription-Key，正文是 SSML，语速写成 prosody 的百分比。
func (e *Engine) speechAzure(ctx context.Context, call speechCall, text string) ([]byte, error) {
	origin := endpointOrigin(call.Endpoint, "https://eastus.tts.speech.microsoft.com")
	voice := call.Voice.ID
	lang := "zh-CN"
	if parts := strings.SplitN(voice, "-", 3); len(parts) == 3 {
		lang = parts[0] + "-" + parts[1]
	}
	inner := azureSSMLEscape.Replace(text)
	if call.Speed > 0 && math.Abs(call.Speed-1) > 0.05 {
		inner = fmt.Sprintf(`<prosody rate="%+d%%">%s</prosody>`, int(math.Round((speedIn(call.Speed, 0.5, 2)-1)*100)), inner)
	}
	ssml := fmt.Sprintf(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="%s"><voice name="%s">%s</voice></speak>`, lang, azureSSMLEscape.Replace(voice), inner)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/cognitiveservices/v1", strings.NewReader(ssml))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("Ocp-Apim-Subscription-Key", call.Key)
	req.Header.Set("X-Microsoft-OutputFormat", "audio-24khz-96kbitrate-mono-mp3")
	req.Header.Set("User-Agent", "yoyo")
	return e.speechDo(req)
}
