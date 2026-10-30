package im

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// SecretStore 按名字存取密钥。名字由本包拼成 im.<账号>.<字段>。
type SecretStore interface {
	Get(name string) (string, error)
	Set(name, value string)
}

type fatalError struct{ error }

func fatalf(format string, args ...any) error {
	return fatalError{fmt.Errorf(format, args...)}
}

type liveSocket struct {
	mu sync.Mutex
	c  *websocket.Conn
}

func (s *liveSocket) write(ctx context.Context, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	c := s.c
	s.mu.Unlock()
	if c == nil {
		return fmt.Errorf("not connected")
	}
	return c.Write(ctx, websocket.MessageText, b)
}

// Runtime 保存账号并拉起启用的适配器。
type Runtime struct {
	mu         sync.Mutex
	path       string
	vault      SecretStore
	settings   Settings
	accounts   []Account
	bindings   map[string]string
	status     map[string]Status
	rates      map[string]rateWindow
	handler    func(Inbound)
	root       context.Context
	rootCancel context.CancelFunc
	stop       map[string]context.CancelFunc
	hook       *http.Server
	hookErr    string
	seen       map[string]time.Time
	sockets    sync.Map
	bases      map[string]string
	cursors    map[string]string
	httpClient *http.Client
	tokens     map[string]struct {
		value string
		until time.Time
	}
}

// Open 读取 dir/im.json。目录不存在时创建。文件还不存在时用默认规则。
func Open(dir string, vault SecretStore) (*Runtime, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	rt := &Runtime{
		path:     filepath.Join(dir, "im.json"),
		vault:    vault,
		status:   map[string]Status{},
		bindings: map[string]string{},
		bases:    map[string]string{},
		cursors:  map[string]string{},
		tokens: map[string]struct {
			value string
			until time.Time
		}{},
	}
	if err := rt.load(); err != nil {
		return nil, err
	}
	return rt, nil
}

// SetHandler 在 Start 之前设置。每条已通过平台解析的消息都会进来，过滤在 Admit。
func (rt *Runtime) SetHandler(fn func(Inbound)) {
	rt.mu.Lock()
	rt.handler = fn
	rt.mu.Unlock()
}

// Start 拉起已启用的账号。可以重复调用，后一次会先停掉上一次。
func (rt *Runtime) Start() {
	rt.Close()
	ctx, cancel := context.WithCancel(context.Background())
	rt.mu.Lock()
	rt.root = ctx
	rt.rootCancel = cancel
	rt.mu.Unlock()
	rt.reload()
}

// Close 停掉适配器和回调监听。
func (rt *Runtime) Close() {
	rt.mu.Lock()
	cancel := rt.rootCancel
	rt.rootCancel = nil
	rt.root = nil
	stops := rt.stop
	rt.stop = nil
	hook := rt.hook
	rt.hook = nil
	rt.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	for _, stop := range stops {
		stop()
	}
	if hook != nil {
		ctx, c := context.WithTimeout(context.Background(), 2*time.Second)
		_ = hook.Shutdown(ctx)
		c()
	}
}

// Snapshot 返回目录、规则、账号和状态。密钥原文不在里面。
func (rt *Runtime) Snapshot() Snapshot {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	accounts := make([]Account, len(rt.accounts))
	for i, a := range rt.accounts {
		accounts[i] = rt.decorateLocked(a)
	}
	statuses := make([]Status, 0, len(rt.status))
	for _, a := range rt.accounts {
		if st, ok := rt.status[a.ID]; ok {
			statuses = append(statuses, st)
		}
	}
	return Snapshot{
		Catalog:  Catalog(),
		Settings: rt.settings,
		Accounts: accounts,
		Statuses: statuses,
		HookBase: fmt.Sprintf("http://%s:%d/im/", rt.settings.ListenHost, rt.settings.ListenPort),
	}
}

// SaveSettings 写下规则并重载正在运行的账号。
func (rt *Runtime) SaveSettings(s Settings) error {
	rt.mu.Lock()
	rt.settings = s.normalized()
	err := rt.flush()
	running := rt.root != nil
	rt.mu.Unlock()
	if err != nil {
		return err
	}
	if running {
		rt.reload()
	}
	return nil
}

// SaveAccount 新增或替换账号。secrets 里的空字符串表示保留原密钥。
func (rt *Runtime) SaveAccount(a Account, secrets map[string]string) (Account, error) {
	if _, ok := specByID(a.Platform); !ok {
		return Account{}, fmt.Errorf("unknown platform %s", a.Platform)
	}
	a = applyDefaults(a)
	a.ID = cleanID(a.ID)
	if a.ID == "" {
		a.ID = newAccountID(a.Platform)
	}
	a.Label = strings.TrimSpace(a.Label)
	if a.Label == "" {
		a.Label = a.Platform
	}
	rt.mu.Lock()
	replaced := false
	for i := range rt.accounts {
		if rt.accounts[i].ID == a.ID {
			rt.accounts[i] = a
			replaced = true
			break
		}
	}
	if !replaced {
		rt.accounts = append(rt.accounts, a)
	}
	err := rt.flush()
	running := rt.root != nil
	rt.mu.Unlock()
	if err != nil {
		return Account{}, err
	}
	for k, v := range secrets {
		if strings.TrimSpace(k) == "" || v == "" {
			continue
		}
		if rt.vault != nil {
			rt.vault.Set(secretName(a.ID, k), v)
		}
	}
	if running {
		rt.reload()
	}
	return rt.publicAccount(a.ID), nil
}

// DeleteAccount 停掉并忘掉一个账号，同时清掉它的密钥和会话绑定。
func (rt *Runtime) DeleteAccount(id string) error {
	id = cleanID(id)
	spec, _ := rt.account(id)
	rt.mu.Lock()
	out := rt.accounts[:0]
	for _, a := range rt.accounts {
		if a.ID == id {
			continue
		}
		out = append(out, a)
	}
	rt.accounts = out
	for origin := range rt.bindings {
		if strings.HasPrefix(origin, id+":") {
			delete(rt.bindings, origin)
		}
	}
	delete(rt.status, id)
	err := rt.flush()
	running := rt.root != nil
	rt.mu.Unlock()
	if err != nil {
		return err
	}
	if rt.vault != nil {
		if p, ok := specByID(spec.Platform); ok {
			for _, field := range p.Fields {
				if field.Secret {
					rt.vault.Set(secretName(id, field.Key), "")
				}
			}
		}
	}
	if running {
		rt.reload()
	}
	return nil
}

// Binding 返回这个来源已经绑定的会话 id。
func (rt *Runtime) Binding(origin string) string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.bindings[origin]
}

// Bind 记住来源和会话。
func (rt *Runtime) Bind(origin, sessionID string) {
	if origin == "" || sessionID == "" {
		return
	}
	rt.mu.Lock()
	if rt.bindings == nil {
		rt.bindings = map[string]string{}
	}
	rt.bindings[origin] = sessionID
	_ = rt.flush()
	rt.mu.Unlock()
}

// Push 向一个已知会话主动发文本。账号必须已经启用。
func (rt *Runtime) Push(ctx context.Context, accountID, chatID, text string) error {
	acct, ok := rt.account(accountID)
	if !ok {
		return fmt.Errorf("unknown account")
	}
	if !acct.Enabled {
		return fmt.Errorf("account is stopped")
	}
	text = strings.TrimSpace(text)
	chatID = strings.TrimSpace(chatID)
	if text == "" || chatID == "" {
		return fmt.Errorf("chat id and text are required")
	}
	return rt.push(ctx, acct, chatID, text)
}

func (rt *Runtime) reload() {
	rt.mu.Lock()
	stops := rt.stop
	rt.stop = map[string]context.CancelFunc{}
	accounts := append([]Account(nil), rt.accounts...)
	parent := rt.root
	rt.mu.Unlock()
	for _, stop := range stops {
		stop()
	}
	rt.restartHook()
	if parent == nil {
		return
	}
	for _, acct := range accounts {
		if !acct.Enabled {
			rt.setStatus(acct.ID, "stopped", "")
			continue
		}
		ctx, cancel := context.WithCancel(parent)
		rt.mu.Lock()
		if rt.stop == nil {
			rt.stop = map[string]context.CancelFunc{}
		}
		rt.stop[acct.ID] = cancel
		rt.mu.Unlock()
		go rt.supervise(ctx, acct)
	}
}

func (rt *Runtime) supervise(ctx context.Context, acct Account) {
	run, ok := runners[acct.Platform]
	if !ok {
		rt.setStatus(acct.ID, "error", "unknown platform")
		return
	}
	for {
		if ctx.Err() != nil {
			rt.setStatus(acct.ID, "stopped", "")
			return
		}
		rt.setStatus(acct.ID, "running", rt.runningDetail(acct))
		err := run(rt, ctx, acct)
		if ctx.Err() != nil {
			rt.setStatus(acct.ID, "stopped", "")
			return
		}
		if err == nil {
			rt.setStatus(acct.ID, "stopped", "")
			return
		}
		var fatal fatalError
		rt.setStatus(acct.ID, "error", err.Error())
		if errors.As(err, &fatal) {
			<-ctx.Done()
			rt.setStatus(acct.ID, "stopped", "")
			return
		}
		select {
		case <-ctx.Done():
			rt.setStatus(acct.ID, "stopped", "")
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func (rt *Runtime) runningDetail(acct Account) string {
	if usesHook(acct) {
		return rt.hookURL(acct.ID)
	}
	return ""
}

func (rt *Runtime) restartHook() {
	rt.mu.Lock()
	old := rt.hook
	rt.hook = nil
	need := false
	for _, a := range rt.accounts {
		if a.Enabled && usesHook(a) {
			need = true
			break
		}
	}
	settings := rt.settings
	rt.mu.Unlock()
	if old != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = old.Shutdown(ctx)
		cancel()
	}
	if !need {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/im/", rt.onHook)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", settings.ListenHost, settings.ListenPort))
	if err != nil {
		rt.mu.Lock()
		rt.hookErr = err.Error()
		rt.mu.Unlock()
		for _, a := range rt.copyAccounts() {
			if a.Enabled && usesHook(a) {
				rt.setStatus(a.ID, "error", err.Error())
			}
		}
		return
	}
	rt.mu.Lock()
	rt.hook = srv
	rt.hookErr = ""
	rt.mu.Unlock()
	go func() { _ = srv.Serve(ln) }()
}

func usesHook(a Account) bool {
	switch a.Platform {
	case "line", "weixin_official_account", "wecom", "qq_official_webhook", "whatsapp", "vocechat":
		return true
	case "lark":
		return modeOf(a, "webhook") == "webhook"
	case "slack":
		return modeOf(a, "socket") == "webhook"
	case "dingtalk":
		return modeOf(a, "stream") == "webhook"
	case "wecom_ai_bot":
		return modeOf(a, "long") == "webhook"
	default:
		return false
	}
}

func (rt *Runtime) hookURL(id string) string {
	rt.mu.Lock()
	s := rt.settings
	rt.mu.Unlock()
	return fmt.Sprintf("http://%s:%d/im/%s", s.ListenHost, s.ListenPort, id)
}

func (rt *Runtime) emit(in Inbound) {
	if in.MessageID != "" && !rt.fresh(in.AccountID+"\n"+in.MessageID) {
		return
	}
	rt.mu.Lock()
	h := rt.handler
	rt.mu.Unlock()
	if h == nil {
		return
	}
	if in.Origin == "" {
		kind := in.Kind
		if kind == "" {
			kind = "direct"
		}
		in.Origin = in.AccountID + ":" + kind + ":" + in.ChatID
		in.Kind = kind
	}
	go h(in)
}

func (rt *Runtime) fresh(key string) bool {
	now := time.Now()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.seen == nil {
		rt.seen = map[string]time.Time{}
	}
	for k, at := range rt.seen {
		if now.Sub(at) > 2*time.Minute {
			delete(rt.seen, k)
		}
	}
	if _, ok := rt.seen[key]; ok {
		return false
	}
	rt.seen[key] = now
	return true
}

func (rt *Runtime) hookProblem() string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.hook != nil {
		return ""
	}
	if rt.hookErr != "" {
		return rt.hookErr
	}
	return "callback listener is not up"
}

func (rt *Runtime) setStatus(id, state, detail string) {
	rt.mu.Lock()
	if rt.status == nil {
		rt.status = map[string]Status{}
	}
	rt.status[id] = Status{ID: id, State: state, Detail: detail}
	rt.mu.Unlock()
}

func (rt *Runtime) secret(id, key string) string {
	if rt.vault == nil || id == "" || key == "" {
		return ""
	}
	v, err := rt.vault.Get(secretName(id, key))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func (rt *Runtime) account(id string) (Account, bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for _, a := range rt.accounts {
		if a.ID == id {
			return a, true
		}
	}
	return Account{}, false
}

func (rt *Runtime) accountByPath(id string) (Account, bool) {
	return rt.account(cleanID(id))
}

func (rt *Runtime) copyAccounts() []Account {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return append([]Account(nil), rt.accounts...)
}

func (rt *Runtime) publicAccount(id string) Account {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for _, a := range rt.accounts {
		if a.ID == id {
			return rt.decorateLocked(a)
		}
	}
	return Account{}
}

func (rt *Runtime) decorateLocked(a Account) Account {
	spec, ok := specByID(a.Platform)
	if !ok {
		return a
	}
	has := map[string]bool{}
	for _, field := range spec.Fields {
		if field.Secret && rt.secret(a.ID, field.Key) != "" {
			has[field.Key] = true
		}
	}
	if len(has) > 0 {
		a.HasSecret = has
	}
	return a
}

func (rt *Runtime) client() *http.Client {
	if rt.httpClient != nil {
		return rt.httpClient
	}
	return &http.Client{Timeout: 40 * time.Second}
}

func (rt *Runtime) rootURL(name, fallback string) string {
	rt.mu.Lock()
	b := rt.bases[name]
	rt.mu.Unlock()
	if strings.TrimSpace(b) == "" {
		b = fallback
	}
	return strings.TrimRight(b, "/")
}

func (rt *Runtime) postJSON(ctx context.Context, url string, headers map[string]string, body any) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, r)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return rt.do(req)
}

func (rt *Runtime) get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return rt.do(req)
}

func (rt *Runtime) do(req *http.Request) ([]byte, int, error) {
	resp, err := rt.client().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return raw, resp.StatusCode, nil
}

func (rt *Runtime) putSocket(id string, c *websocket.Conn) *liveSocket {
	s := &liveSocket{c: c}
	rt.sockets.Store(id, s)
	return s
}

func (rt *Runtime) cursor(id string) string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.cursors[id]
}

func (rt *Runtime) setCursor(id, v string) {
	rt.mu.Lock()
	if rt.cursors == nil {
		rt.cursors = map[string]string{}
	}
	rt.cursors[id] = v
	rt.mu.Unlock()
}

func (rt *Runtime) socket(id string) *liveSocket {
	v, ok := rt.sockets.Load(id)
	if !ok {
		return nil
	}
	s, _ := v.(*liveSocket)
	return s
}
