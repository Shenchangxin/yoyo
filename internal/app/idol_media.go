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

func pinnedOrNil(ok bool, it idol.Idol) any {
	if !ok {
		return nil
	}
	return it
}

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

// errNoChatConn is returned when no session chat connection can be resolved.
// The text includes "chat model" so the idol desk can point at settings.
var errNoChatConn = errors.New("configure a chat model in Settings → Idol or Models")

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
	// A saved legacy endpoint still works until a text model from Settings →
	// Models is available. The settings page clears it on the next save.
	if strings.TrimSpace(svc.ChatBaseURL) != "" {
		return a.idolLegacyClient(svc.ChatBaseURL, picked)
	}
	return a.idolConfigClient(picked)
}

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

func sessionModelName(picked string) string {
	picked = strings.TrimSpace(picked)
	left, right, ok := strings.Cut(picked, "::")
	if !ok || strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return picked
	}
	return strings.TrimSpace(right)
}

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

// borrowMediaKey keeps an idol vault key. Otherwise it reuses a generation
// connection of the same kind, then a chat connection on the same host and
// vendor. It does not borrow a key from a different provider.
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

func encodeB64Raw(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
