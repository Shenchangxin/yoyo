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

const recordBinding = "yoyoRecord"

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

func newNonce() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func (h *Host) recording() bool {
	if h == nil || h.rec == nil {
		return false
	}
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	return h.rec.active
}

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
