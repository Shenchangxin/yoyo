// 偶像的聊天、语音、识别、图像和立绘快照。聊天优先用会话里的文本模型。媒体密钥先用偶像保险库里的那把，没有再按同能力、同厂商和同主机借用。

package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/idol"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

const (
	// idolSpeakTimeout、idolHearTimeout、idolPaintTimeout 是一句合成、一段识别和一张图最多等多久。
	idolSpeakTimeout = 45 * time.Second
	idolHearTimeout  = 60 * time.Second
	idolPaintTimeout = 150 * time.Second
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
// payload 解出服务、用户卡和四把密钥。
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
// name 传给 Vault.Set。
// value 先 TrimSpace，空串则返回，减号则把该键设成空串，否则写入 vault。
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
// name 传给 Vault.Get。
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
// s 先 TrimSpace，空串返回空串；不足 8 个 rune 时返回已保存，否则返回末四位。
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

// idolSnapshot 先 EnsureDaily 再返回钉住的偶像、立绘、好感（补上回暖）、圈子和各媒体是否就绪。立绘优先用当前服装 blob，否则用肖像 blob，读出后做 base64。
func (a *App) idolSnapshot() (string, error) {
	now := time.Now()
	_ = a.Idols.Update(func(live *idol.State) error {
		live.EnsureDaily(now)
		return nil
	})
	st := a.Idols.View()
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
	var aff any
	if ok {
		aff = st.AffinityOf(pinned.ID, now)
	}
	return idolJSON(map[string]any{
		"pinned":       pinnedOrNil(ok, pinned),
		"portrait":     portrait,
		"affinity":     aff,
		"user":         st.User,
		"idols":        st.Idols,
		"circles":      st.Circles,
		"affinities":   st.Affinities(now),
		"lore":         st.Lore,
		"chat_ready":   ready.ChatReady,
		"speech_ready": ready.SpeechReady,
		"asr_ready":    ready.ASRReady,
		"image_ready":  ready.ImageReady,
		"text_only":    st.Services.TextOnly,
		"phone":        st.Services.PhoneTexture,
		"reach_out":    st.Services.ReachOut,
		"work":         st.Services.WorkEnabled,
		"barge_in":     !st.Services.NoBargeIn,
	})
}

// pinnedOrNil 在没有钉住偶像时返回 nil，否则返回该偶像。
// ok 为假时返回 nil。
// it 在 ok 为真时返回。
func pinnedOrNil(ok bool, it idol.Idol) any {
	if !ok {
		return nil
	}
	return it
}

// idolSave 解析并保存一个偶像。
// payload 解成偶像后传给 SaveIdol。
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
// payload 解成圈子后传给 SaveCircle。
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
// payload 解成设定后传给 SaveLore。
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
// idolID 传给 State.Idol。
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

// idolClient 在 kind 为 judge、memory 或 title（后台的活）且 JudgeModel 非空时用评判模型，否则用 ChatModel。先走 idolSessionClient。得到 errNoChatConn 且 ChatBaseURL 非空时改用旧端点，否则 idolConfigClient。
func (a *App) idolClient(kind string) (*runtime.OpenAIClient, string, error) {
	if a == nil || a.Idols == nil {
		return nil, "", fmt.Errorf("idol store missing")
	}
	svc := a.Idols.View().Services
	picked := strings.TrimSpace(svc.ChatModel)
	switch kind {
	case "judge", "memory", "title":
		if strings.TrimSpace(svc.JudgeModel) != "" {
			picked = strings.TrimSpace(svc.JudgeModel)
		}
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
// picked 传给 splitSessionModel；连接 id 为空时再传给 sessionModelName。
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
// picked 先 TrimSpace，按 :: 切开；切不开或任一侧去掉空白后为空时返回 picked，否则返回右侧模型名。
func sessionModelName(picked string) string {
	picked = strings.TrimSpace(picked)
	left, right, ok := strings.Cut(picked, "::")
	if !ok || strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return picked
	}
	return strings.TrimSpace(right)
}

// idolLegacyClient 用传入的旧端点和 idol.chat 密钥，密钥空则 fallbackChatKey。模型空则用 gpt-4.1-mini。密钥仍空则返回带 chat model 的错误。
// base 传给 NewOpenAIClient。
// picked 传给 sessionModelName，得到的模型名为空时改用 Config.Model，仍为空则用 gpt-4.1-mini。
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
// picked 传给 sessionModelName，得到的模型名为空时改用 Config.Model，仍为空则用 gpt-4.1-mini。
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

// idolSpeak 用这位偶像的音色把一句话念出来，返回音频和 Content-Type。读音表先换掉书面写法；语气提示 cue 交给认得它的厂商。
// 按设置里的语音预设或地址认出厂商，各说各的协议。ctx 超时或取消时请求随之结束。
func (a *App) idolSpeak(ctx context.Context, idolID, text, cue string) ([]byte, string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", fmt.Errorf("empty speech")
	}
	base, key, model, err := a.idolSpeechCreds("")
	if err != nil {
		return nil, "", err
	}
	st := a.Idols.View()
	spokenLang := "zh"
	if it, ok := st.Idol(idolID); ok {
		text = idol.ApplyReadings(text, it.Readings)
		spokenLang = it.SpokenLang
	}
	provider := idol.NormalizeServices(st.Services).SpeechProvider
	return synthesize(ctx, speechCall{
		route: mediaRoute(provider, base), base: base, key: key, model: model,
		voice: a.idolVoice(idolID), text: text, cue: cue, lang: spokenLang,
	})
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

// idolTranscribe 识别一段录音。暂停时返回 paused。按设置里的识别预设或地址认出厂商，各说各的协议；hint 是热词。
func (a *App) idolTranscribe(ctx context.Context, filename string, audio []byte, hint string) (string, error) {
	if a.IsPaused() {
		return "", fmt.Errorf("paused")
	}
	if len(audio) == 0 {
		return "", fmt.Errorf("empty audio")
	}
	base, key, model, err := a.idolASRCreds()
	if err != nil {
		return "", err
	}
	provider := idol.NormalizeServices(a.Idols.View().Services).ASRProvider
	return transcribe(ctx, transcribeCall{
		route: mediaRoute(provider, base), base: base, key: key, model: model,
		filename: firstNonEmpty(filename, "audio.webm"), audio: audio, hint: hint,
	})
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

// idolPaint 按提示画一张图。按设置里的生图预设或地址认出厂商，各说各的协议；ctx 管整次请求和下载。
func (a *App) idolPaint(ctx context.Context, prompt string) ([]byte, error) {
	base, key, model, err := a.idolImageCreds()
	if err != nil {
		return nil, err
	}
	provider := idol.NormalizeServices(a.Idols.View().Services).ImageProvider
	return paint(ctx, paintCall{route: mediaRoute(provider, base), base: base, key: key, model: model, prompt: prompt})
}

// idolCanPaint 判断有没有配好能用的生图服务。没有时她会说现在发不了照片，而不是答应了却发不出来。
func (a *App) idolCanPaint() bool {
	_, _, _, err := a.idolImageCreds()
	return err == nil
}

// idolMediaFields 按 speech、asr 或其他种类返回对应的地址、vault 密钥、模型和厂商。
// kind 为 speech 或 asr 时返回对应字段，否则返回图像字段。
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
// cap 传给 leaseMediaKey。
// provider 传给 idol.Preset。
// endpoint 在 provider 没有预设时作为租用地址。
// key 去掉首尾空白后非空则原样返回。
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
// cap 去掉首尾空白后为空或连接库为空则返回空串，否则传给 List。
// vendor 去掉首尾空白后用来和连接的 Vendor 比较。
// endpoint 去掉首尾空白后用来和连接端点比较。
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
// r 为 nil 时返回空串，否则用来 Lease。
// c 传给 Lease。
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
// cap 传给 Active 和 ModelFor。
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

// idolPing 试通一项服务。speech 念一句你好，asr 识别一小段静音，image 画一只不带字的茶杯（最多等 150 秒），其他种类让聊天只回复 OK。
// payload 解出 kind。
func (a *App) idolPing(payload string) (string, error) {
	var p struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal([]byte(payload), &p)
	timeout := 40 * time.Second
	if p.Kind == "image" {
		timeout = idolPaintTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	switch p.Kind {
	case "speech":
		_, _, err := a.idolSpeak(ctx, "", "你好", "")
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "asr":
		if _, err := a.idolTranscribe(ctx, "ping.wav", silentWAV(), ""); err != nil {
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
