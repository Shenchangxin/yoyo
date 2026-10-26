// 本文件在隔离 Chrome 里录一段操作。绑定名是 yoyoRecord。
// 页面脚本经 Runtime.addBinding 送事件；主框架导航也记一条。密码和地址里的令牌在记下之前先打码。最多留 2000 条，10 毫秒内的连续滚动丢掉。

package browser

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// recordBinding 是页面里调用的绑定名，值是 yoyoRecord。
const recordBinding = "yoyoRecord"

// RecordTarget 是一次录制事件点中的元素，带角色、无障碍名、选择器候选和框架路径。
type RecordTarget struct {
	Tag                string   `json:"tag,omitempty"`
	Role               string   `json:"role,omitempty"`
	Type               string   `json:"type,omitempty"`
	Name               string   `json:"name,omitempty"`
	Label              string   `json:"label,omitempty"`
	Placeholder        string   `json:"placeholder,omitempty"`
	ID                 string   `json:"id,omitempty"`
	NameAttr           string   `json:"nameAttr,omitempty"`
	Text               string   `json:"text,omitempty"`
	TestID             string   `json:"testId,omitempty"`
	Href               string   `json:"href,omitempty"`
	SelectorCandidates []string `json:"selectorCandidates,omitempty"`
	FramePath          []string `json:"framePath,omitempty"`
	ShadowPath         []string `json:"shadowPath,omitempty"`
}

// RecordEvent 是录下来的一条。Kind 取 click、input、change、keydown、scroll、navigation。
type RecordEvent struct {
	Kind   string       `json:"kind"`
	TS     string       `json:"ts"`
	URL    string       `json:"url,omitempty"`
	Title  string       `json:"title,omitempty"`
	Key    string       `json:"key,omitempty"`
	Value  string       `json:"value,omitempty"`
	DeltaY float64      `json:"deltaY,omitempty"`
	Target RecordTarget `json:"target,omitempty"`
}

// RecordSession 是停录之后的一份结果。Steps 和 SkillBody 由 collapseRecordEvents 与 generateRecordedSkill 填上。Dir 是写到磁盘的目录。
type RecordSession struct {
	ID        string         `json:"id"`
	Started   time.Time      `json:"started"`
	Stopped   time.Time      `json:"stopped,omitempty"`
	StartURL  string         `json:"start_url,omitempty"`
	Events    []RecordEvent  `json:"events"`
	Steps     []SemanticStep `json:"steps,omitempty"`
	Inputs    []SkillInput   `json:"inputs,omitempty"`
	SkillName string         `json:"skill_name,omitempty"`
	SkillBody string         `json:"skill_body,omitempty"`
	Dir       string         `json:"dir,omitempty"`
}

// recorder 是进行中的一次录制。nonce 用来丢掉不是这次注入的事件。
type recorder struct {
	mu       sync.Mutex
	active   bool
	id       string
	nonce    string
	started  time.Time
	startURL string
	events   []RecordEvent
	lastAt   time.Time
	dropped  int
}

// newNonce 返回 8 字节随机数的十六进制。读随机数失败时用当前纳秒。
func newNonce() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// recording 在宿主上有一场还没停的录制时返回真。
func (h *Host) recording() bool {
	if h == nil || h.rec == nil {
		return false
	}
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	return h.rec.active
}

// StartRecord 确保 Chrome 已启动，记下 id 形如 rec_20060102_150405，并在当前连接上装上录制钩子。返回这次的 id。
func (h *Host) StartRecord() (string, error) {
	if h == nil {
		return "", fmt.Errorf("browser: no host")
	}
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", err
	}
	rec := &recorder{
		active:   true,
		id:       "rec_" + time.Now().UTC().Format("20060102_150405"),
		nonce:    newNonce(),
		started:  time.Now().UTC(),
		startURL: h.liveURL(),
	}
	h.mu.Lock()
	h.rec = rec
	c := h.cdp
	h.mu.Unlock()
	if c != nil {
		h.applyRecordHooks(c)
	}
	h.record("record", "start "+rec.id)
	return rec.id, nil
}

// applyRecordHooks 在正在录制时加上绑定、新文档脚本，并立刻在当前页执行一次注入脚本。
func (h *Host) applyRecordHooks(c *cdpConn) {
	if h == nil || c == nil || !h.recording() {
		return
	}
	_, _ = c.call("Runtime.addBinding", map[string]any{"name": recordBinding})
	js := h.recordInjectJS()
	if js == "" {
		return
	}
	_, _ = c.call("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": js})
	_, _ = c.call("Runtime.evaluate", map[string]any{"expression": js, "returnByValue": true})
}

// recordInjectJS 用当前 nonce 和录制 id 生成注入脚本。没在录时返回空字符串。
func (h *Host) recordInjectJS() string {
	if h == nil || h.rec == nil {
		return ""
	}
	h.rec.mu.Lock()
	nonce := h.rec.nonce
	id := h.rec.id
	h.rec.mu.Unlock()
	return recorderJS(nonce, id)
}

// StopRecord 结束录制，把事件收成步骤并生成技能草稿，写到 recordings/<id>。没在录时返回 not recording。
func (h *Host) StopRecord() (*RecordSession, error) {
	if h == nil || h.rec == nil {
		return nil, fmt.Errorf("browser: not recording")
	}
	h.rec.mu.Lock()
	if !h.rec.active {
		h.rec.mu.Unlock()
		return nil, fmt.Errorf("browser: not recording")
	}
	h.rec.active = false
	sess := &RecordSession{
		ID:       h.rec.id,
		Started:  h.rec.started,
		Stopped:  time.Now().UTC(),
		StartURL: h.rec.startURL,
		Events:   append([]RecordEvent(nil), h.rec.events...),
	}
	h.rec.mu.Unlock()
	doc := collapseRecordEvents(sess.Events, sess.StartURL)
	sess.Steps = doc.Steps
	sess.Inputs = doc.Inputs
	sess.SkillName = doc.SkillName
	sess.SkillBody = generateRecordedSkill(doc)
	if dir, err := h.persistRecord(sess); err == nil {
		sess.Dir = dir
	}
	h.record("record", "stop "+sess.ID)
	return sess, nil
}

// RecordStatus 返回没在录，或一行带 id、事件数和开始时间的状态。
func (h *Host) RecordStatus() string {
	if h == nil || h.rec == nil {
		return "not recording"
	}
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	if !h.rec.active {
		return "not recording"
	}
	return fmt.Sprintf("recording %s events=%d started=%s", h.rec.id, len(h.rec.events), h.rec.started.Format(time.RFC3339))
}

// persistRecord 把 manifest.json、events.json、steps.json 和 draft.skill.md 写到目录下的 recordings/<id>，权限 0o600。目录本身是 0o700。
// sess 的 ID 用来拼 recordings 下的目录。manifest 写入 ID、Started、Stopped 和 StartURL。Events、Steps 分别写成 events.json 和 steps.json。SkillBody 非空时写成 draft.skill.md。
func (h *Host) persistRecord(sess *RecordSession) (string, error) {
	dir := filepath.Join(h.dir, "recordings", sess.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	write := func(name string, v any) error {
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, name), raw, 0o600)
	}
	if err := write("manifest.json", map[string]any{
		"id": sess.ID, "started": sess.Started, "stopped": sess.Stopped, "start_url": sess.StartURL,
	}); err != nil {
		return "", err
	}
	_ = write("events.json", sess.Events)
	_ = write("steps.json", sess.Steps)
	if sess.SkillBody != "" {
		_ = os.WriteFile(filepath.Join(dir, "draft.skill.md"), []byte(sess.SkillBody), 0o600)
	}
	return dir, nil
}

// onCDPEvent 在主框架导航时清掉 ref，录制中再记一条打过码的 navigation。Runtime.bindingCalled 交给 noteRecordBinding。
// method 为 Page.frameNavigated 或 Runtime.bindingCalled 时才处理。
// params 在 Page.frameNavigated 时传给 isMainFrameNav。主框架且正在录制时再解出 frame.url。为 Runtime.bindingCalled 时传给 noteRecordBinding。
func (h *Host) onCDPEvent(method string, params json.RawMessage) {
	switch method {
	case "Page.frameNavigated":
		if isMainFrameNav(params) {
			h.invalidateRefs()
			if h.recording() {
				var p struct {
					Frame struct {
						URL string `json:"url"`
					} `json:"frame"`
				}
				_ = json.Unmarshal(params, &p)
				h.noteRecordEvent(RecordEvent{
					Kind: "navigation",
					TS:   time.Now().UTC().Format(time.RFC3339Nano),
					URL:  redactRecordURL(p.Frame.URL),
				})
			}
		}
	case "Runtime.bindingCalled":
		h.noteRecordBinding(params)
	}
}

// isMainFrameNav 在帧没有 parentId 时返回真，用来只记主框架。
// params 解出 frame.parentId。Unmarshal 失败时返回假。parentId 去掉空白后为空时返回真。
func isMainFrameNav(params json.RawMessage) bool {
	var p struct {
		Frame struct {
			ParentID string `json:"parentId"`
			URL      string `json:"url"`
		} `json:"frame"`
	}
	if json.Unmarshal(params, &p) != nil {
		return false
	}
	return strings.TrimSpace(p.Frame.ParentID) == ""
}

// noteRecordBinding 解析 yoyoRecord 的载荷。nonce 对上且仍在录时才收下事件。
// params 先解出 name 和 payload。Unmarshal 失败或 name 不是 recordBinding 时直接返回。payload 再解出 nonce 和 event。
func (h *Host) noteRecordBinding(params json.RawMessage) {
	var p struct {
		Name    string `json:"name"`
		Payload string `json:"payload"`
	}
	if json.Unmarshal(params, &p) != nil || p.Name != recordBinding {
		return
	}
	var wrap struct {
		Nonce string      `json:"nonce"`
		Event RecordEvent `json:"event"`
	}
	if json.Unmarshal([]byte(p.Payload), &wrap) != nil {
		return
	}
	if h.rec == nil {
		return
	}
	h.rec.mu.Lock()
	ok := h.rec.active && wrap.Nonce == h.rec.nonce
	h.rec.mu.Unlock()
	if !ok {
		return
	}
	h.noteRecordEvent(wrap.Event)
}

// noteRecordEvent 打码地址和值后追加一条。距上一条不足 10 毫秒的 scroll，或已经有 2000 条时，只增加 dropped。
// ev 的 TS 为空时写成当前 UTC 时间。URL 传给 redactRecordURL。Target 和 Value 传给 redactRecordValue。Kind 为 scroll 且距上一条不足 10 毫秒，或 events 已有 2000 条时只增加 dropped，否则追加 ev。
func (h *Host) noteRecordEvent(ev RecordEvent) {
	if h == nil || h.rec == nil {
		return
	}
	if ev.TS == "" {
		ev.TS = time.Now().UTC().Format(time.RFC3339Nano)
	}
	ev.URL = redactRecordURL(ev.URL)
	ev.Value = redactRecordValue(ev.Target, ev.Value)
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	if !h.rec.active {
		return
	}
	now := time.Now()
	if !h.rec.lastAt.IsZero() && now.Sub(h.rec.lastAt) < 10*time.Millisecond && ev.Kind == "scroll" {
		h.rec.dropped++
		return
	}
	if len(h.rec.events) >= 2000 {
		h.rec.dropped++
		return
	}
	h.rec.lastAt = now
	h.rec.events = append(h.rec.events, ev)
}
