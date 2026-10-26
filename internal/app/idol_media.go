// 偶像的聊天、语音、识别、图像和立绘快照。聊天优先用会话里的文本模型。媒体密钥先用偶像保险库里的那把，没有再按同能力、同厂商和同主机借用。

package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/idol"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// idolMediaFallback 是设置页上语音、识别和图像的当前活动连接名称、模型和预设 id。
type idolMediaFallback struct {
	SpeechPreset string `json:"speech_preset,omitempty"`
	SpeechName   string `json:"speech_name,omitempty"`
	SpeechModel  string `json:"speech_model,omitempty"`
	ASRPreset    string `json:"asr_preset,omitempty"`
	ASRName      string `json:"asr_name,omitempty"`
	ASRModel     string `json:"asr_model,omitempty"`
	ImagePreset  string `json:"image_preset,omitempty"`
	ImageName    string `json:"image_name,omitempty"`
	ImageModel   string `json:"image_model,omitempty"`
}

// idolSettingsView 是偶像设置页的视图。密钥字段只放 keyHint，ChatReady 等表示对应客户端能否配齐。
type idolSettingsView struct {
	idol.Services
	User        idol.UserCard        `json:"user"`
	Presets     []idol.ServicePreset `json:"presets"`
	Fallback    idolMediaFallback    `json:"fallback"`
	ChatKey     string               `json:"chat_key_hint"`
	SpeechKey   string               `json:"speech_key_hint"`
	ASRKey      string               `json:"asr_key_hint"`
	ImageKey    string               `json:"image_key_hint"`
	ChatReady   bool                 `json:"chat_ready"`
	SpeechReady bool                 `json:"speech_ready"`
	ASRReady    bool                 `json:"asr_ready"`
	ImageReady  bool                 `json:"image_ready"`
}

// idolSettingsView 组装服务、用户卡、预设和回落连接，并用 idol.chat、idol.speech、idol.asr、idol.image 的密钥做提示。就绪与否来自对应的客户端函数。
func (a *App) idolSettingsView() idolSettingsView {
	st := a.Idols.View()
	v := idolSettingsView{
		Services: idol.NormalizeServices(st.Services),
		User:     st.User,
		Presets:  idol.ServicePresets(),
		Fallback: a.idolMediaFallback(),
	}
	v.ChatKey = keyHint(a.vaultIdol("idol.chat"))
	v.SpeechKey = keyHint(a.vaultIdol("idol.speech"))
	v.ASRKey = keyHint(a.vaultIdol("idol.asr"))
	v.ImageKey = keyHint(a.vaultIdol("idol.image"))
	_, _, err := a.idolClient("chat")
	v.ChatReady = err == nil
	_, _, _, serr := a.idolSpeechCreds("")
	v.SpeechReady = serr == nil
	_, _, _, aerr := a.idolASRCreds()
	v.ASRReady = aerr == nil
	_, _, _, ierr := a.idolImageCreds()
	v.ImageReady = ierr == nil
	return v
}

// idolSaveSettings 保存服务和用户卡。四把密钥经 putIdolKey 写入 vault，空字符串不改，减号清空。
func (a *App) idolSaveSettings(payload string) (string, error) {
	var p struct {
		idol.Services
		User      idol.UserCard `json:"user"`
		ChatKey   string        `json:"chat_key"`
		SpeechKey string        `json:"speech_key"`
		ASRKey    string        `json:"asr_key"`
		ImageKey  string        `json:"image_key"`
	}
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return "", err
	}
	p.Services = idol.NormalizeServices(p.Services)
	if err := a.Idols.SaveServices(p.Services); err != nil {
		return "", err
	}
	_ = a.Idols.Update(func(st *idol.State) error {
		st.User = p.User
		return nil
	})
	a.putIdolKey("idol.chat", p.ChatKey)
	a.putIdolKey("idol.speech", p.SpeechKey)
	a.putIdolKey("idol.asr", p.ASRKey)
	a.putIdolKey("idol.image", p.ImageKey)
	return idolJSON(a.idolSettingsView())
}

// putIdolKey 把非空密钥写入 vault。值为减号时把该键设成空串。vault 为空则返回。
func (a *App) putIdolKey(name, value string) {
	if a.Vault == nil {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if value == "-" {
		a.Vault.Set(name, "")
		return
	}
	a.Vault.Set(name, value)
}

// vaultIdol 读取一把偶像密钥。vault 为空或读取失败返回空串。
func (a *App) vaultIdol(name string) string {
	if a == nil || a.Vault == nil {
		return ""
	}
	v, err := a.Vault.Get(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

// keyHint 在密钥不足 8 个 rune 时返回已保存，否则返回末四位，前面用圆点。空串返回空串。
func keyHint(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) < 8 {
		return "已保存"
	}
	return "••••" + string(r[len(r)-4:])
}

// idolSnapshot 先 EnsureDaily 再返回钉住的偶像、立绘、好感、圈子和各媒体是否就绪。立绘优先用当前服装 blob，否则用肖像 blob，读出后做 base64。
func (a *App) idolSnapshot() (string, error) {
	st := a.Idols.View()
	_ = a.Idols.Update(func(live *idol.State) error {
		live.EnsureDaily(time.Now())
		return nil
	})
	st = a.Idols.View()
	pinned, ok := st.Pinned()
	var portrait string
	if ok {
		blob := pinned.Portrait
		for _, o := range pinned.Outfits {
			if o.ID == pinned.OutfitID && o.Blob != "" {
				blob = o.Blob
			}
		}
		if b, err := a.Idols.ReadBlob(blob); err == nil && len(b) > 0 {
			portrait = encodeB64Raw(b)
		}
	}
	ready := a.idolSettingsView()
	return idolJSON(map[string]any{
		"pinned":       pinnedOrNil(ok, pinned),
		"portrait":     portrait,
		"affinity":     affinityOrNil(st, pinned.ID, ok),
		"user":         st.User,
		"idols":        st.Idols,
		"circles":      st.Circles,
		"affinities":   st.Affinity,
		"lore":         st.Lore,
		"chat_ready":   ready.ChatReady,
		"speech_ready": ready.SpeechReady,
		"asr_ready":    ready.ASRReady,
		"image_ready":  ready.ImageReady,
		"text_only":    st.Services.TextOnly,
		"phone":        st.Services.PhoneTexture,
		"reach_out":    st.Services.ReachOut,
		"work":         st.Services.WorkEnabled,
	})
}

// pinnedOrNil 在没有钉住偶像时返回 nil，否则返回该偶像。
func pinnedOrNil(ok bool, it idol.Idol) any {
	if !ok {
		return nil
	}
	return it
}

// affinityOrNil 返回该偶像的好感并填上 Tier。没有记录时分数为 50，等级为 TierTrust。未钉住时返回 nil。
func affinityOrNil(st idol.State, id string, ok bool) any {
	if !ok {
		return nil
	}
	for _, a := range st.Affinity {
		if a.IdolID == id {
			a.Tier = idol.TierOf(a.Score)
			return a
		}
	}
	return idol.Affinity{IdolID: id, Score: 50, Tier: idol.TierTrust}
}

// idolSave 解析并保存一个偶像。
func (a *App) idolSave(payload string) (string, error) {
	var in idol.Idol
	if err := json.Unmarshal([]byte(payload), &in); err != nil {
		return "", err
	}
	out, err := a.Idols.SaveIdol(in, time.Now())
	if err != nil {
		return "", err
	}
	return idolJSON(out)
}

// idolSaveCircle 解析并保存一个圈子。
func (a *App) idolSaveCircle(payload string) (string, error) {
	var c idol.Circle
	if err := json.Unmarshal([]byte(payload), &c); err != nil {
		return "", err
	}
	out, err := a.Idols.SaveCircle(c)
	if err != nil {
		return "", err
	}
	return idolJSON(out)
}

// idolSaveLore 解析并保存一条设定。
func (a *App) idolSaveLore(payload string) (string, error) {
	var doc idol.Lore
	if err := json.Unmarshal([]byte(payload), &doc); err != nil {
		return "", err
	}
	out, err := a.Idols.SaveLore(doc)
	if err != nil {
		return "", err
	}
	return idolJSON(out)
}

// idolVoice 优先用该偶像的 VoiceID，其次用服务里的 SpeechVoice，都空则返回 alloy。
func (a *App) idolVoice(idolID string) string {
	if a == nil || a.Idols == nil {
		return "alloy"
	}
	st := a.Idols.View()
	if it, ok := st.Idol(idolID); ok && strings.TrimSpace(it.VoiceID) != "" {
		return strings.TrimSpace(it.VoiceID)
	}
	svc := idol.NormalizeServices(st.Services)
	if v := strings.TrimSpace(svc.SpeechVoice); v != "" {
		return v
	}
	return "alloy"
}

// errNoChatConn 表示没有解析到会话聊天连接。错误文本含 chat model，方便偶像桌面指向设置。
var errNoChatConn = errors.New("configure a chat model in Settings → Idol or Models")

// idolClient 在 kind 为 judge 且 JudgeModel 非空时用裁判模型，否则用 ChatModel。先走 idolSessionClient。得到 errNoChatConn 且 ChatBaseURL 非空时改用旧端点，否则 idolConfigClient。
func (a *App) idolClient(kind string) (*runtime.OpenAIClient, string, error) {
	if a == nil || a.Idols == nil {
		return nil, "", fmt.Errorf("idol store missing")
	}
	svc := a.Idols.View().Services
	picked := strings.TrimSpace(svc.ChatModel)
	if kind == "judge" && strings.TrimSpace(svc.JudgeModel) != "" {
		picked = strings.TrimSpace(svc.JudgeModel)
	}
	client, model, err := a.idolSessionClient(picked)
	if err == nil {
		return client, model, nil
	}
	if !errors.Is(err, errNoChatConn) {
		return nil, "", err
	}
	// 会话连接返回 errNoChatConn 且 ChatBaseURL 非空时，改用这份旧端点。
	if strings.TrimSpace(svc.ChatBaseURL) != "" {
		return a.idolLegacyClient(svc.ChatBaseURL, picked)
	}
	return a.idolConfigClient(picked)
}

// idolSessionClient 按会话模型解析 chat 连接。没有连接时返回 errNoChatConn。模型名空则用连接的 chat 模型，再空用 Config.Model。
func (a *App) idolSessionClient(picked string) (*runtime.OpenAIClient, string, error) {
	connID, model := a.splitSessionModel(picked)
	if connID == "" {
		model = sessionModelName(picked)
	}
	c, err := a.resolveChatConnection(SessionMeta{ConnectionID: connID, Model: model})
	if err != nil || strings.TrimSpace(c.ID) == "" {
		return nil, "", errNoChatConn
	}
	client := a.openAIForConn(c)
	if client == nil || strings.TrimSpace(client.APIKey) == "" {
		return nil, "", fmt.Errorf("chat connection has no key")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = connection.ModelFor(c, connection.CapChat)
	}
	if model == "" {
		model = strings.TrimSpace(a.Config.Model)
	}
	if model == "" {
		return nil, "", fmt.Errorf("configure a chat model in Settings → Idol or Models")
	}
	return client, model, nil
}

// sessionModelName 在文本是 connectionId::model 时返回右边的模型名，否则返回整段。
func sessionModelName(picked string) string {
	picked = strings.TrimSpace(picked)
	left, right, ok := strings.Cut(picked, "::")
	if !ok || strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return picked
	}
	return strings.TrimSpace(right)
}

// idolLegacyClient 用传入的旧端点和 idol.chat 密钥，密钥空则 fallbackChatKey。模型空则用 gpt-4.1-mini。密钥仍空则返回带 chat model 的错误。
func (a *App) idolLegacyClient(base, picked string) (*runtime.OpenAIClient, string, error) {
	key := a.vaultIdol("idol.chat")
	if key == "" {
		key = a.fallbackChatKey()
	}
	model := sessionModelName(picked)
	if model == "" {
		model = strings.TrimSpace(a.Config.Model)
	}
	if model == "" {
		model = "gpt-4.1-mini"
	}
	if key == "" {
		return nil, "", fmt.Errorf("configure a chat model in Settings → Idol or Models")
	}
	return runtime.NewOpenAIClient(base, key), model, nil
}

// idolConfigClient 用 Config.BaseURL 和 fallbackChatKey。模型空则用 gpt-4.1-mini。密钥或地址为空时返回 errNoChatConn。
func (a *App) idolConfigClient(picked string) (*runtime.OpenAIClient, string, error) {
	key := a.fallbackChatKey()
	model := sessionModelName(picked)
	if model == "" {
		model = strings.TrimSpace(a.Config.Model)
	}
	if model == "" {
		model = "gpt-4.1-mini"
	}
	if key == "" || strings.TrimSpace(a.Config.BaseURL) == "" {
		return nil, "", errNoChatConn
	}
	return runtime.NewOpenAIClient(a.Config.BaseURL, key), model, nil
}

// fallbackChatKey 依次用 Vault.Get 读取 default、openai、api_key。default 和 openai 的 Get 会回落环境变量。都没有返回空串。
func (a *App) fallbackChatKey() string {
	if a == nil || a.Vault == nil {
		return ""
	}
	for _, name := range []string{"default", "openai", "api_key"} {
		if v, err := a.Vault.Get(name); err == nil && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// idolSpeechCreds 读取语音服务字段。地址为空时改用 speechCreds。然后 borrowMediaKey。填了厂商但仍然没有密钥时返回错误。
func (a *App) idolSpeechCreds(_ string) (base, key, model string, err error) {
	base, key, model, provider := a.idolMediaFields("speech")
	if base == "" {
		ep, k, m, e := a.speechCreds()
		if e != nil {
			return "", "", "", e
		}
		return ep, firstNonEmpty(key, k), firstNonEmpty(model, m), nil
	}
	key = a.borrowMediaKey(connection.CapSpeech, provider, base, key)
	if provider != "" && key == "" {
		return "", "", "", fmt.Errorf("add a speech key in Settings → Idol, or a matching speech connection under Generation")
	}
	return base, key, model, nil
}

// idolSpeak 把文本 POST 到音频 speech 地址。声音空则用 idolVoice。模型空则用 gpt-4o-mini-tts。响应体最多读 8 MiB。
func (a *App) idolSpeak(text, voice string) ([]byte, string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", fmt.Errorf("empty speech")
	}
	base, key, model, err := a.idolSpeechCreds(voice)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(voice) == "" {
		voice = a.idolVoice("")
	}
	payload, _ := json.Marshal(map[string]any{
		"model": firstNonEmpty(model, "gpt-4o-mini-tts"),
		"input": text,
		"voice": voice,
	})
	req, err := http.NewRequest(http.MethodPost, idol.MediaURL(base, "audio/speech"), bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("tts %s: %s", resp.Status, bytes.TrimSpace(b))
	}
	ctype := resp.Header.Get("Content-Type")
	if ctype == "" {
		ctype = "audio/mpeg"
	}
	return b, ctype, nil
}

// idolASRCreds 读取识别字段。地址为空时先找活动的 asr 连接，再退到 speechCreds。speechCreds 失败时错误为 speech recognition is not configured。
func (a *App) idolASRCreds() (base, key, model string, err error) {
	base, key, model, provider := a.idolMediaFields("asr")
	if base == "" {
		if r := a.connReg(); r != nil {
			if c, e := r.Active(connection.CapASR, ""); e == nil {
				base = strings.TrimSpace(c.Endpoint)
				model = firstNonEmpty(model, connection.ModelFor(c, connection.CapASR))
				if k, e2 := r.Lease(c); e2 == nil {
					key = firstNonEmpty(key, k)
				}
			}
		}
	}
	if base == "" {
		ep, k, m, e := a.speechCreds()
		if e != nil {
			return "", "", "", fmt.Errorf("speech recognition is not configured")
		}
		return ep, firstNonEmpty(key, k), firstNonEmpty(model, m), nil
	}
	key = a.borrowMediaKey(connection.CapASR, provider, base, key)
	if provider != "" && key == "" {
		return "", "", "", fmt.Errorf("add an asr key in Settings → Idol, or a matching transcription connection under Generation")
	}
	return base, key, model, nil
}

// idolTranscribe 在暂停时返回 paused。模型空则用 gpt-4o-mini-transcribe。响应体最多读 1 MiB。JSON 没有 text 时返回正文本身。
func (a *App) idolTranscribe(filename string, audio []byte) (string, error) {
	if a.IsPaused() {
		return "", fmt.Errorf("paused")
	}
	base, key, model, err := a.idolASRCreds()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", firstNonEmpty(model, "gpt-4o-mini-transcribe"))
	fw, err := w.CreateFormFile("file", firstNonEmpty(filename, "audio.webm"))
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(audio); err != nil {
		return "", err
	}
	_ = w.Close()
	req, err := http.NewRequest(http.MethodPost, idol.MediaURL(base, "audio/transcriptions"), &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("stt %s: %s", resp.Status, bytes.TrimSpace(b))
	}
	var out struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(b, &out) != nil {
		return strings.TrimSpace(string(b)), nil
	}
	return strings.TrimSpace(out.Text), nil
}

// idolImageCreds 读取图像字段。地址为空时用活动的 image 连接。仍没有地址则返回 image connection not configured。
func (a *App) idolImageCreds() (base, key, model string, err error) {
	base, key, model, provider := a.idolMediaFields("image")
	if base == "" {
		if r := a.connReg(); r != nil {
			if c, e := r.Active(connection.CapImage, ""); e == nil {
				base = strings.TrimSpace(c.Endpoint)
				model = firstNonEmpty(model, connection.ModelFor(c, connection.CapImage))
				if k, e2 := r.Lease(c); e2 == nil {
					key = firstNonEmpty(key, k)
				}
			}
		}
	}
	if base == "" {
		return "", "", "", fmt.Errorf("image connection not configured")
	}
	key = a.borrowMediaKey(connection.CapImage, provider, base, key)
	if provider != "" && key == "" {
		return "", "", "", fmt.Errorf("add an image key in Settings → Idol, or a matching image connection under Generation")
	}
	return base, key, model, nil
}

// idolPaint 请求 images/generations，默认模型 gpt-image-1，尺寸 1024x1024。优先用 b64_json，否则再 GET 返回的 URL。两段响应体上限分别是 8 MiB 和 6 MiB。
func (a *App) idolPaint(ctx context.Context, prompt string) ([]byte, error) {
	base, key, model, err := a.idolImageCreds()
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]any{
		"model":  firstNonEmpty(model, "gpt-image-1"),
		"prompt": prompt,
		"size":   "1024x1024",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, idol.MediaURL(base, "images/generations"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image %s", resp.Status)
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
	ireq, err := http.NewRequestWithContext(ctx, http.MethodGet, out.Data[0].URL, nil)
	if err != nil {
		return nil, err
	}
	ires, err := http.DefaultClient.Do(ireq)
	if err != nil {
		return nil, err
	}
	defer ires.Body.Close()
	return io.ReadAll(io.LimitReader(ires.Body, 6<<20))
}

// idolMediaFields 按 speech、asr 或其他种类返回对应的地址、vault 密钥、模型和厂商。
func (a *App) idolMediaFields(kind string) (base, key, model, provider string) {
	var svc idol.Services
	if a != nil && a.Idols != nil {
		svc = idol.NormalizeServices(a.Idols.View().Services)
	}
	switch kind {
	case "speech":
		return strings.TrimSpace(svc.SpeechBaseURL), a.vaultIdol("idol.speech"), strings.TrimSpace(svc.SpeechModel), strings.TrimSpace(svc.SpeechProvider)
	case "asr":
		return strings.TrimSpace(svc.ASRBaseURL), a.vaultIdol("idol.asr"), strings.TrimSpace(svc.ASRModel), strings.TrimSpace(svc.ASRProvider)
	default:
		return strings.TrimSpace(svc.ImageBaseURL), a.vaultIdol("idol.image"), strings.TrimSpace(svc.ImageModel), strings.TrimSpace(svc.ImageProvider)
	}
}

// borrowMediaKey 已有偶像 vault 密钥时原样返回。否则先租同能力的连接，再在厂商和地址都非空时按严格主机加厂商租一把 chat 密钥。不借用另一家厂商的密钥。
func (a *App) borrowMediaKey(cap, provider, endpoint, key string) string {
	if strings.TrimSpace(key) != "" {
		return strings.TrimSpace(key)
	}
	vendor, root := "", endpoint
	if p, ok := idol.Preset(provider); ok {
		vendor, root = p.Vendor, p.Endpoint
	}
	if borrowed := a.leaseMediaKey(cap, vendor, root, false); borrowed != "" {
		return borrowed
	}
	if vendor != "" && root != "" {
		if borrowed := a.leaseMediaKey(connection.CapChat, vendor, root, true); borrowed != "" {
			return borrowed
		}
	}
	return ""
}

// leaseMediaKey 在活动连接里找密钥。strict 为真时厂商和主机必须同时匹配。否则先要主机匹配，没有再退到只匹配厂商的第一条。
func (a *App) leaseMediaKey(cap, vendor, endpoint string, strict bool) string {
	r := a.connReg()
	if r == nil || strings.TrimSpace(cap) == "" {
		return ""
	}
	list, err := r.List(cap)
	if err != nil {
		return ""
	}
	vendor = strings.TrimSpace(vendor)
	endpoint = strings.TrimSpace(endpoint)
	var vendorOnly connection.Connection
	var sawVendor bool
	for _, c := range list {
		if !c.Active {
			continue
		}
		vendorOK := vendor != "" && strings.EqualFold(strings.TrimSpace(c.Vendor), vendor)
		hostOK := endpoint != "" && idol.SameEndpoint(c.Endpoint, endpoint)
		if strict && !(vendorOK && hostOK) {
			continue
		}
		if hostOK && (vendor == "" || vendorOK || !strict) {
			if key := leasedKey(r, c); key != "" {
				return key
			}
		}
		if !strict && vendorOK && !sawVendor {
			vendorOnly = c
			sawVendor = true
		}
	}
	if sawVendor {
		return leasedKey(r, vendorOnly)
	}
	return ""
}

// leasedKey 向连接库租用该连接的密钥。失败或库为空返回空串。
func leasedKey(r *connection.Registry, c connection.Connection) string {
	if r == nil {
		return ""
	}
	k, err := r.Lease(c)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(k)
}

// idolMediaFallback 用语音、识别和图像的活动连接填名称、模型和预设 id。
func (a *App) idolMediaFallback() idolMediaFallback {
	var out idolMediaFallback
	name, vendor, endpoint, model := a.activeMedia(connection.CapSpeech)
	out.SpeechName, out.SpeechModel = name, model
	out.SpeechPreset = idol.MatchPreset("speech", vendor, endpoint)
	name, vendor, endpoint, model = a.activeMedia(connection.CapASR)
	out.ASRName, out.ASRModel = name, model
	out.ASRPreset = idol.MatchPreset("asr", vendor, endpoint)
	name, vendor, endpoint, model = a.activeMedia(connection.CapImage)
	out.ImageName, out.ImageModel = name, model
	out.ImagePreset = idol.MatchPreset("image", vendor, endpoint)
	return out
}

// activeMedia 返回该能力的活动连接名称、厂商、端点和模型。没有库或没有活动连接时返回空串。
func (a *App) activeMedia(cap string) (name, vendor, endpoint, model string) {
	r := a.connReg()
	if r == nil {
		return
	}
	c, err := r.Active(cap, "")
	if err != nil {
		return
	}
	name = strings.TrimSpace(c.Name)
	if name == "" {
		name = strings.TrimSpace(c.Vendor)
	}
	return name, c.Vendor, c.Endpoint, connection.ModelFor(c, cap)
}

// idolPing 用 40 秒超时试通。speech 朗读你好，image 画一只不带字的茶杯，其他种类让聊天只回复 OK。
func (a *App) idolPing(payload string) (string, error) {
	var p struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal([]byte(payload), &p)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	switch p.Kind {
	case "speech":
		_, _, err := a.idolSpeak("你好", "")
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "image":
		_, err := a.idolPaint(ctx, "a simple all-ages teacup, no text")
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	default:
		_, err := a.idolEngine().Brain.Chat(ctx, "title", "", []idol.Msg{
			{Role: "user", Content: "只回复 OK"},
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	}
}

// encodeB64Raw 用标准 base64 编码字节。
func encodeB64Raw(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
