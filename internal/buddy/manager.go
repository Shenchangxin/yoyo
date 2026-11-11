// manager.go 拥有 Buddy 的总管：目录、状态记录、检测结果、安装任务和事件发布都挂在 Manager 上。
// 对外两组方法：给界面的列表、查询、刷新、探测、日志、安装、卸载、配置、登录和偏好；给工人的 Launch、Online、Roster、NoteRun、NoteAuthFailure。
// 外部依赖（找命令、跑命令、HTTP、时钟、vault、配置读写、事件发布、ACP 连接、终端）都经 Options 注入。

package buddy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// Options 是 Manager 的全部外部依赖。零值字段用真实实现。
type Options struct {
	// Root 是 buddies 目录，RuntimeDir 是托管 Node 和 uv 的目录。
	Root       string
	RuntimeDir string
	// Version 写进 ACP 的 clientInfo。
	Version string
	// Locale 返回界面语言，决定提示文字用中文还是英文。
	Locale func() string
	// Platform 为 nil 时用本机。
	Platform *Platform
	// LookPath 为 nil 时用扩展过 PATH 的查找器。
	LookPath func(name string) (string, error)
	// Run 跑一条命令直到结束，返回合并输出。
	Run func(ctx context.Context, c Cmd) ([]byte, error)
	// HTTP 是下载和查询注册表用的客户端。
	HTTP *http.Client
	// Now 是时钟。
	Now func() time.Time
	// Vault 存密钥。
	Vault Vault
	// Settings 读配置，SaveSettings 改配置并落盘。
	Settings     func() Settings
	SaveSettings func(func(*Settings)) error
	// Publish 收到一项变化后的视图。
	Publish func(View)
	// Dial 启动并连上 agent。
	Dial func(ctx context.Context, opts acp.Options) (*acp.Client, error)
	// Terminal 在可见终端里跑一条命令。
	Terminal func(ctx context.Context, l acp.Launch) (TerminalResult, error)
	// YoyoKey 按提供商返回 yoyo 自己的钥匙，「从 yoyo 复用」用它。
	YoyoKey func(provider string) (string, bool)
	// YoyoChat 返回 yoyo 当前聊天用的模型、提供商、钥匙和端点。安装完成且这项还没自己配过时用来填默认。
	YoyoChat func() (model, provider, key, endpoint string)
	// Working 返回每个 agent 正在做的工单数。
	Working func() map[string]int
	// BaseEnv 是清洗前的环境，nil 时用 os.Environ。
	BaseEnv func() []string
	// RegistryURL 覆盖注册表地址，NodeMirrors 覆盖 Node 的镜像顺序，NPMRegistries 覆盖 npm 源，测试用。
	RegistryURL   string
	NodeMirrors   []string
	NPMRegistries map[string]string
	// Throttle 是同一项事件的最短间隔，0 时 250ms，负数时不节流。
	Throttle time.Duration
	// HealthDelay 和 HealthEvery 是健康检查的首次延迟和间隔，0 时 20 秒和 10 分钟。
	HealthDelay time.Duration
	HealthEvery time.Duration
}

// Cmd 是一条要跑的命令。Output 非 nil 时边跑边写输出。
type Cmd struct {
	Launch acp.Launch
	Output io.Writer
}

// ErrNotFound 是没有这个 agent。
var ErrNotFound = errors.New("buddy: unknown agent")

// ErrNotInstalled 是 agent 还没装。
var ErrNotInstalled = errors.New("buddy: agent not installed")

// Manager 管理全部 agent。方法都可以并发调用。
type Manager struct {
	opts  Options
	plat  Platform
	paths *PathFinder

	mu      sync.Mutex
	entries []Entry
	recs    map[string]*record
	det     map[string]detection
	jobs    map[string]*job
	icons   map[string]string
	reg     Registry
	started bool
	kicked  bool
	lastPub map[string]time.Time
	pending map[string]bool
	npmPick string
	npmAt   time.Time
	sysNPM  string
	// probing 是每项正在跑的探测数，removing 是正在卸载的项。卸载等探测停下再删目录，卸载中不再起新的探测。
	probing  map[string]int
	removing map[string]bool
	// detected 表示第一次全量检测（只看文件和 PATH）已经做完。
	detected bool

	installSem chan struct{}
	probeSem   chan struct{}
	nodeMu     sync.Mutex
	uvMu       sync.Mutex
	saveMu     sync.Mutex
	logMu      sync.Mutex

	// keyMu 管 keyCache（每把密钥有没有保存）和 yoyoKeys（yoyo 有没有某提供商的钥匙）。可以在持有 mu 时拿。
	keyMu    sync.Mutex
	keyCache map[string]bool
	yoyoKeys map[string]yoyoKeyHit

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New 建一个 Manager。还不读磁盘，Start 之后才开始检测和健康检查。
func New(opts Options) *Manager {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	}
	if opts.Dial == nil {
		opts.Dial = acp.Dial
	}
	if opts.Run == nil {
		opts.Run = defaultRun
	}
	if opts.Terminal == nil {
		opts.Terminal = openTerminal
	}
	if opts.Settings == nil {
		opts.Settings = func() Settings { return Settings{} }
	}
	if opts.BaseEnv == nil {
		opts.BaseEnv = os.Environ
	}
	if opts.Throttle == 0 {
		opts.Throttle = 250 * time.Millisecond
	}
	if opts.HealthDelay == 0 {
		opts.HealthDelay = 20 * time.Second
	}
	if opts.HealthEvery == 0 {
		opts.HealthEvery = 10 * time.Minute
	}
	plat := currentPlatform()
	if opts.Platform != nil {
		plat = *opts.Platform
	}
	m := &Manager{
		opts: opts, plat: plat,
		recs: map[string]*record{}, det: map[string]detection{}, jobs: map[string]*job{}, icons: map[string]string{},
		lastPub: map[string]time.Time{}, pending: map[string]bool{}, probing: map[string]int{}, removing: map[string]bool{},
		installSem: make(chan struct{}, 2), probeSem: make(chan struct{}, 2),
		keyCache: map[string]bool{}, yoyoKeys: map[string]yoyoKeyHit{},
	}
	m.paths = newPathFinder(plat.GOOS, opts.Now)
	if m.opts.LookPath == nil {
		m.opts.LookPath = m.paths.LookPath
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.reg, _ = parseRegistry(registryJSON)
	m.rebuildLocked()
	return m
}

// Start 读缓存的注册表和状态，开始健康检查。重复调用无效。检测、探测和注册表刷新等第一次列出时才做（见 kick）。
func (m *Manager) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	if m.opts.Root != "" {
		m.recs = loadRecords(filepath.Join(m.opts.Root, "state.json"))
		if reg, ok := loadCachedRegistry(m.opts.Root); ok {
			m.reg = reg
		}
		m.loadIconsLocked()
	}
	m.rebuildLocked()
	m.mu.Unlock()
	m.goSafe(func() { m.healthLoop(m.ctx) })
}

// kick 第一次列出时在后台检测全部项、全探还没探过的，并按天刷新注册表。重复调用无效。
func (m *Manager) kick() {
	m.mu.Lock()
	if m.kicked {
		m.mu.Unlock()
		return
	}
	m.kicked = true
	m.mu.Unlock()
	m.goSafe(func() {
		m.detectAll(m.ctx)
		m.probeUnknown(m.ctx)
	})
	m.goSafe(func() { _ = m.refreshRegistry(m.ctx, false) })
}

// Close 停掉后台任务、取消进行中的安装，等它们退出。
func (m *Manager) Close() {
	m.cancel()
	m.mu.Lock()
	for _, j := range m.jobs {
		j.cancel()
	}
	m.mu.Unlock()
	m.wg.Wait()
}

// goSafe 起一个受 Close 管理的后台 goroutine。
func (m *Manager) goSafe(fn func()) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		fn()
	}()
}

// zh 判断提示文字用不用中文。
func (m *Manager) zh() bool {
	if m.opts.Locale == nil {
		return true
	}
	return !strings.HasPrefix(strings.ToLower(m.opts.Locale()), "en")
}

// say 按界面语言挑一句。
func (m *Manager) say(zh, en string) string {
	if m.zh() {
		return zh
	}
	return en
}

// rebuildLocked 按当前注册表和自定义项重建目录。调用方持有 m.mu。
func (m *Manager) rebuildLocked() {
	curated, _ := parseCurated(curatedJSON)
	m.entries = buildCatalog(curated, m.reg, m.opts.Settings().Custom)
}

// entryLocked 按编号找目录项。调用方持有 m.mu。
func (m *Manager) entryLocked(id string) (Entry, bool) {
	for _, e := range m.entries {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}

// working 返回各 agent 正在做的工单数。
func (m *Manager) working() map[string]int {
	if m.opts.Working == nil {
		return map[string]int{}
	}
	if w := m.opts.Working(); w != nil {
		return w
	}
	return map[string]int{}
}

// List 返回全部项和偏好。第一次调用时 Start 并开始后台检测，列表立刻返回，检测结果走事件。
func (m *Manager) List() ([]View, Prefs) {
	m.Start()
	m.kick()
	working := m.working()
	settings := m.opts.Settings()
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]View, 0, len(m.entries))
	for _, e := range m.entries {
		out = append(out, m.viewLocked(e, settings, working))
	}
	return out, prefsOf(settings)
}

// Summary 是给 Doctor 的摘要。和 List 不同，它不 Start、不开始检测：还没人打开过 Buddy 时只返回 started 为假。
func (m *Manager) Summary() Summary {
	m.mu.Lock()
	started := m.started
	m.mu.Unlock()
	if !started {
		return Summary{}
	}
	working := m.working()
	settings := m.opts.Settings()
	m.mu.Lock()
	defer m.mu.Unlock()
	out := Summary{Started: true, Detecting: !m.detected, States: map[State]int{}}
	for _, e := range m.entries {
		v := m.viewLocked(e, settings, working)
		out.States[v.State]++
		if !v.Builtin && v.Install.Kind == "none" {
			continue
		}
		out.Agents = append(out.Agents, SummaryItem{
			ID: v.ID, State: v.State, Install: v.Install.Kind, Version: v.Version,
			Working: v.Working, Unverified: v.Install.Unverified, Detail: redactText(v.Detail),
		})
	}
	return out
}

// Get 返回一项。
func (m *Manager) Get(id string) (View, error) {
	working := m.working()
	settings := m.opts.Settings()
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entryLocked(id)
	if !ok {
		return View{}, ErrNotFound
	}
	return m.viewLocked(e, settings, working), nil
}

// view 取一项当前的视图，找不到时返回零值。
func (m *Manager) view(id string) View {
	v, _ := m.Get(id)
	return v
}

// prefsOf 把配置换成偏好视图，补上默认值。
func prefsOf(s Settings) Prefs {
	return Prefs{
		NPMRegistry: firstNonEmpty(s.NPMRegistry, "auto"), PyIndex: s.PyIndex,
		DefaultAgent: firstNonEmpty(s.DefaultAgent, YoyoID), AutoPick: s.AutoPick, TurnMinutes: s.TurnMinutesOf(),
	}
}

// installedLocked 判断一项装了没有：托管、系统或自定义命令找得到。调用方持有 m.mu。
func (m *Manager) installedLocked(e Entry) bool {
	d := m.det[e.ID]
	return d.Managed != nil || d.SystemPath != ""
}

// viewLocked 把目录项、配置、检测结果、记录和安装任务合成界面上的一项。调用方持有 m.mu。
func (m *Manager) viewLocked(e Entry, settings Settings, working map[string]int) View {
	item := settings.Item(e.ID)
	v := View{
		ID: e.ID, Name: e.Name, Vendor: e.Vendor, Builtin: e.Builtin, Featured: e.Featured && !e.Builtin, Custom: e.Custom,
		Order: e.Order, RegistryID: e.RegistryID, Description: e.Description, Icon: m.icons[e.ID], Docs: e.Docs,
		Latest: e.Version, Aliases: e.Aliases, Working: working[e.ID],
	}
	if e.Builtin {
		v.State = StateOnline
		v.Install = InstallView{Kind: "none"}
		v.Config = ConfigView{Enabled: true, Mode: "managed", Approval: "yoyo", MaxParallel: 0, Strengths: e.Strengths.Pick(m.locale()), Secrets: []SecretView{}}
		return v
	}
	d := m.det[e.ID]
	r := m.recs[e.ID]
	if r != nil && settleLocked(r, m.opts.Now()) {
		m.saveRecordsAsync()
	}
	v.Install = InstallView{Kind: "none"}
	switch {
	case e.Custom && d.SystemPath != "":
		v.Install = InstallView{Kind: "custom", Source: "path", Path: d.SystemPath}
	case d.Managed != nil && (item.Mode != "system" || d.SystemPath == ""):
		v.Install = InstallView{Kind: "managed", Source: d.Managed.Kind, Path: d.ManagedDir, SizeBytes: d.Managed.SizeBytes, Unverified: d.Managed.Kind == "binary" && !d.Managed.Verified}
		v.Version = d.Managed.Version
	case d.SystemPath != "":
		v.Install = InstallView{Kind: "system", Source: "path", Path: d.SystemPath}
	}
	if d.SystemPath != "" && !e.Custom {
		v.System = &SystemView{Path: d.SystemPath, Version: d.SystemVersion}
		if v.Version == "" {
			v.Version = d.SystemVersion
		}
	}
	if r != nil && r.Probe != nil && v.Version == "" {
		v.Version = r.Probe.AgentVersion
	}
	if r != nil && !r.CheckedAt.IsZero() {
		v.CheckedAt = r.CheckedAt.UTC().Format(time.RFC3339)
	}
	v.Config = m.configViewLocked(e, item, d, r)
	v.SystemInstall = m.systemInstallView(e)
	v.CanManage = !e.Custom && installable(e, m.plat)
	if r != nil && r.Probe != nil {
		v.Auth = authView(r.Probe.Auth)
		v.Caps = capsView(r.Probe.Caps)
	}
	if j := m.jobs[e.ID]; j != nil {
		v.State = StateInstalling
		p := j.progress
		v.Progress = &p
		return v
	}
	if !m.installedLocked(e) {
		v.CLIOnly = d.CLIPath != ""
		if r != nil && r.Detail != "" {
			v.Detail = r.Detail
		}
		if !e.Custom && !installable(e, m.plat) && !v.SystemInstall.Available {
			v.State = StateUnsupported
			v.Detail = m.say("此系统不可用", "Not available on this system")
			return v
		}
		v.State = StateNotInstalled
		return v
	}
	switch {
	case r == nil || r.State == "" || r.State == StateNotInstalled || r.State == StateInstalling || r.State == StateUnsupported:
		v.State = StateProbing
	default:
		v.State = r.State
		v.Detail = r.Detail
	}
	if missing := m.missingRequired(e, item); len(missing) > 0 {
		v.State = StateNeedsConfig
		v.Detail = strings.Join(missing, ", ")
	}
	if v.Working > 0 && (v.State == StateRecentlyLost || v.State == StateOffline || v.State == StateError || v.State == StateProbing) {
		v.State = StateOnline
		v.Detail = ""
	}
	return v
}

// locale 返回界面语言，没注入时当中文。
func (m *Manager) locale() string {
	if m.opts.Locale == nil {
		return "zh"
	}
	return m.opts.Locale()
}

// configViewLocked 是一项的设置视图。env_var 型登录方式要的变量也作为密钥列出。调用方持有 m.mu。
func (m *Manager) configViewLocked(e Entry, item ItemConfig, d detection, r *record) ConfigView {
	mode := item.Mode
	if mode == "" {
		mode = "managed"
		if d.Managed == nil && d.SystemPath != "" {
			mode = "system"
		}
	}
	cv := ConfigView{
		Enabled: !item.Disabled, Mode: mode, Model: item.Model, Approval: firstNonEmpty(item.Approval, "yoyo"),
		MaxParallel: item.MaxParallelOf(), Args: item.Args, Env: item.Env,
		Strengths: firstNonEmpty(item.Strengths, e.Strengths.Pick(m.locale())),
		Secrets:   m.secretViews(withEnvMethods(e, r), item),
	}
	if r != nil && r.Probe != nil {
		for _, model := range r.Probe.Models {
			cv.Models = append(cv.Models, ModelView{ID: model.ModelID, Name: firstNonEmpty(model.Name, model.ModelID)})
		}
	}
	return cv
}

// withEnvMethods 把探测到的 env_var 型登录方式补成密钥声明，这样界面能直接填。
func withEnvMethods(e Entry, r *record) Entry {
	if r == nil || r.Probe == nil {
		return e
	}
	out := e
	out.Secrets = slices.Clone(e.Secrets)
	for _, method := range r.Probe.Auth {
		if method.Kind() != "env_var" || method.VarName == "" {
			continue
		}
		if !slices.ContainsFunc(out.Secrets, func(s SecretSpec) bool { return s.Name == method.VarName }) {
			out.Secrets = append(out.Secrets, SecretSpec{Name: method.VarName, Label: method.Name})
		}
	}
	return out
}

// authView 列出 agent 型和 terminal 型的登录方式；env_var 型在配置里填。
func authView(methods []acp.AuthMethod) *AuthView {
	var out []AuthMethodView
	for _, method := range methods {
		kind := method.Kind()
		if kind == "env_var" {
			continue
		}
		out = append(out, AuthMethodView{ID: method.ID, Name: firstNonEmpty(method.Name, method.ID), Type: kind, Description: method.Description})
	}
	if len(out) == 0 {
		return nil
	}
	return &AuthView{Methods: out}
}

// capsView 取出界面关心的能力。
func capsView(c acp.AgentCapabilities) *CapsView {
	return &CapsView{
		Resume: bool(c.SessionCapabilities.Resume), Load: bool(c.LoadSession), Close: bool(c.SessionCapabilities.Close),
		SystemPrompt: bool(c.SessionCapabilities.SystemPrompt), MCPHTTP: bool(c.McpCapabilities.HTTP),
		Image: bool(c.PromptCapabilities.Image), Dirs: bool(c.SessionCapabilities.AdditionalDirectories),
		Logout: c.Auth != nil && bool(c.Auth.Logout),
	}
}

// publish 发出一项的最新视图。同一项 Throttle 内只发一次，多余的合并成一次补发。
func (m *Manager) publish(id string) {
	if m.opts.Publish == nil {
		return
	}
	if m.opts.Throttle < 0 {
		m.opts.Publish(m.view(id))
		return
	}
	m.mu.Lock()
	now := time.Now()
	last := m.lastPub[id]
	if wait := m.opts.Throttle - now.Sub(last); wait > 0 {
		if !m.pending[id] {
			m.pending[id] = true
			time.AfterFunc(wait, func() {
				m.mu.Lock()
				delete(m.pending, id)
				m.lastPub[id] = time.Now()
				m.mu.Unlock()
				m.opts.Publish(m.view(id))
			})
		}
		m.mu.Unlock()
		return
	}
	m.lastPub[id] = now
	m.mu.Unlock()
	m.opts.Publish(m.view(id))
}

// publishAll 发出全部项。
func (m *Manager) publishAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.entries))
	for _, e := range m.entries {
		ids = append(ids, e.ID)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.publish(id)
	}
}

// saveRecordsAsync 在后台把状态写回 state.json。
func (m *Manager) saveRecordsAsync() {
	if m.opts.Root == "" {
		return
	}
	go m.saveRecords()
}

// saveRecords 把状态写回 state.json，串行执行。
func (m *Manager) saveRecords() {
	if m.opts.Root == "" {
		return
	}
	m.saveMu.Lock()
	defer m.saveMu.Unlock()
	m.mu.Lock()
	snapshot := make(map[string]*record, len(m.recs))
	for id, r := range m.recs {
		c := *r
		snapshot[id] = &c
	}
	m.mu.Unlock()
	_ = saveRecords(filepath.Join(m.opts.Root, "state.json"), snapshot)
}

// update 改一项的记录，写回磁盘并发事件。
func (m *Manager) update(id string, fn func(r *record)) {
	m.mu.Lock()
	fn(m.recordLocked(id))
	m.mu.Unlock()
	m.saveRecords()
	m.publish(id)
}

// Settings 返回当前配置的副本。
func (m *Manager) Settings() Settings {
	return m.opts.Settings().Clone()
}

// ConfigPatch 是一次配置修改，nil 字段不动。Secrets 里值为空表示删掉这把密钥。
type ConfigPatch struct {
	Enabled     *bool              `json:"enabled,omitempty"`
	Mode        *string            `json:"mode,omitempty"`
	Model       *string            `json:"model,omitempty"`
	Approval    *string            `json:"approval,omitempty"`
	MaxParallel *int               `json:"maxParallel,omitempty"`
	Args        *[]string          `json:"args,omitempty"`
	Env         *map[string]string `json:"env,omitempty"`
	Secrets     map[string]string  `json:"secrets,omitempty"`
	Reuse       *[]string          `json:"reuse,omitempty"`
	Strengths   *string            `json:"strengths,omitempty"`
}

// Configure 校验并保存一项的设置，密钥写进 vault。保存后在后台全探一次。返回的视图不含密钥值。
func (m *Manager) Configure(id string, p ConfigPatch) (View, error) {
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	m.mu.Unlock()
	if !ok || e.Builtin {
		return View{}, ErrNotFound
	}
	if err := m.validatePatch(p); err != nil {
		return View{}, err
	}
	for name := range p.Secrets {
		if !ValidEnvName(name) {
			return View{}, fmt.Errorf("%s: %s", m.say("密钥名不合法", "invalid key name"), name)
		}
	}
	for name, value := range p.Secrets {
		m.setSecret(id, name, value)
	}
	if m.opts.SaveSettings != nil {
		err := m.opts.SaveSettings(func(s *Settings) {
			if s.Items == nil {
				s.Items = map[string]ItemConfig{}
			}
			item := s.Items[id].Clone()
			applyPatch(&item, p)
			for name, value := range p.Secrets {
				item.Secrets = slices.DeleteFunc(item.Secrets, func(n string) bool { return n == name })
				if strings.TrimSpace(value) != "" {
					item.Secrets = append(item.Secrets, name)
				}
			}
			slices.Sort(item.Secrets)
			s.Items[id] = item
		})
		if err != nil {
			return View{}, err
		}
	}
	m.markProbing(id)
	m.publish(id)
	m.goSafe(func() { _ = m.probe(m.ctx, id, true) })
	return m.view(id), nil
}

// validatePatch 校验取值范围；环境变量名像密钥时拒绝，并提示改放进密钥。
func (m *Manager) validatePatch(p ConfigPatch) error {
	if p.Mode != nil && *p.Mode != "" && *p.Mode != "managed" && *p.Mode != "system" {
		return fmt.Errorf("%s: %s", m.say("模式只能是 managed 或 system", "mode must be managed or system"), *p.Mode)
	}
	if p.Approval != nil && *p.Approval != "" && *p.Approval != "yoyo" && *p.Approval != "ask" {
		return fmt.Errorf("%s: %s", m.say("审批只能是 yoyo 或 ask", "approval must be yoyo or ask"), *p.Approval)
	}
	if p.MaxParallel != nil && (*p.MaxParallel < 1 || *p.MaxParallel > 4) {
		return errors.New(m.say("同时最多 1 到 4 个任务", "max parallel must be between 1 and 4"))
	}
	if p.Strengths != nil && len([]rune(*p.Strengths)) > 200 {
		return errors.New(m.say("擅长写一句话就够了，200 字以内", "keep strengths under 200 characters"))
	}
	if p.Env != nil {
		for name := range *p.Env {
			if !ValidEnvName(name) {
				return fmt.Errorf("%s: %s", m.say("环境变量名不合法", "invalid variable name"), name)
			}
			if SecretLike(name) {
				return fmt.Errorf("%s %s", name, m.say("像是密钥，改放进密钥那一栏", "looks like a secret; save it as a key instead"))
			}
		}
	}
	return nil
}

// applyPatch 把非 nil 的字段写进 item。
func applyPatch(item *ItemConfig, p ConfigPatch) {
	if p.Enabled != nil {
		item.Disabled = !*p.Enabled
	}
	if p.Mode != nil {
		item.Mode = *p.Mode
	}
	if p.Model != nil {
		item.Model = strings.TrimSpace(*p.Model)
	}
	if p.Approval != nil {
		item.Approval = *p.Approval
	}
	if p.MaxParallel != nil {
		item.MaxParallel = *p.MaxParallel
	}
	if p.Args != nil {
		item.Args = slices.DeleteFunc(slices.Clone(*p.Args), func(a string) bool { return strings.TrimSpace(a) == "" })
	}
	if p.Env != nil {
		item.Env = map[string]string{}
		for k, v := range *p.Env {
			item.Env[k] = v
		}
		if len(item.Env) == 0 {
			item.Env = nil
		}
	}
	if p.Reuse != nil {
		item.Reuse = slices.Clone(*p.Reuse)
	}
	if p.Strengths != nil {
		item.Strengths = strings.TrimSpace(*p.Strengths)
	}
}

// PrefsPatch 是一次偏好修改，nil 字段不动。
type PrefsPatch struct {
	NPMRegistry  *string `json:"npmRegistry,omitempty"`
	PyIndex      *string `json:"pyIndex,omitempty"`
	DefaultAgent *string `json:"defaultAgent,omitempty"`
	AutoPick     *bool   `json:"autoPick,omitempty"`
	TurnMinutes  *int    `json:"turnMinutes,omitempty"`
}

// SetPrefs 保存偏好。npm 源只能是 auto、npmjs、npmmirror 或 https 地址；默认 agent 必须在目录里。
func (m *Manager) SetPrefs(p PrefsPatch) (Prefs, error) {
	if p.NPMRegistry != nil {
		v := strings.TrimSpace(*p.NPMRegistry)
		if v != "" && v != "auto" && v != "npmjs" && v != "npmmirror" && !strings.HasPrefix(v, "https://") {
			return Prefs{}, errors.New(m.say("npm 源只能是自动、官方、npmmirror 或 https 地址", "npm registry must be auto, npmjs, npmmirror or an https URL"))
		}
	}
	if p.PyIndex != nil {
		v := strings.TrimSpace(*p.PyIndex)
		if v != "" && !strings.HasPrefix(v, "https://") {
			return Prefs{}, errors.New(m.say("Python 源要是 https 地址", "Python index must be an https URL"))
		}
	}
	if p.DefaultAgent != nil && *p.DefaultAgent != "" && *p.DefaultAgent != YoyoID {
		m.mu.Lock()
		_, ok := m.entryLocked(*p.DefaultAgent)
		m.mu.Unlock()
		if !ok {
			return Prefs{}, ErrNotFound
		}
	}
	if p.TurnMinutes != nil && (*p.TurnMinutes < 5 || *p.TurnMinutes > 600) {
		return Prefs{}, errors.New(m.say("单轮时长上限在 5 到 600 分钟之间", "turn limit must be between 5 and 600 minutes"))
	}
	if m.opts.SaveSettings != nil {
		err := m.opts.SaveSettings(func(s *Settings) {
			if p.NPMRegistry != nil {
				s.NPMRegistry = strings.TrimSpace(*p.NPMRegistry)
			}
			if p.PyIndex != nil {
				s.PyIndex = strings.TrimSpace(*p.PyIndex)
			}
			if p.DefaultAgent != nil {
				s.DefaultAgent = *p.DefaultAgent
				if s.DefaultAgent == YoyoID {
					s.DefaultAgent = ""
				}
			}
			if p.AutoPick != nil {
				s.AutoPick = *p.AutoPick
			}
			if p.TurnMinutes != nil {
				s.TurnMinutes = *p.TurnMinutes
			}
		})
		if err != nil {
			return Prefs{}, err
		}
	}
	m.mu.Lock()
	m.npmPick = ""
	m.mu.Unlock()
	return prefsOf(m.opts.Settings()), nil
}

// Add 加一个自定义 ACP agent。编号从名字生成，不能和已有的重复；命令要找得到。
func (m *Manager) Add(c CustomAgent) (View, error) {
	c.Name = strings.TrimSpace(c.Name)
	c.Command = strings.TrimSpace(c.Command)
	if c.Name == "" || c.Command == "" {
		return View{}, errors.New(m.say("名字和命令都要填", "name and command are required"))
	}
	for name := range c.Env {
		if !ValidEnvName(name) || SecretLike(name) {
			return View{}, fmt.Errorf("%s %s", name, m.say("像是密钥，加好后在密钥那一栏填", "looks like a secret; add it as a key afterwards"))
		}
	}
	if _, err := m.opts.LookPath(c.Command); err != nil {
		return View{}, fmt.Errorf("%s: %s", m.say("找不到这个命令", "command not found"), c.Command)
	}
	id := slugID(firstNonEmpty(c.ID, c.Name))
	m.mu.Lock()
	base := id
	for n := 2; ; n++ {
		if _, taken := m.entryLocked(id); !taken {
			break
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
	m.mu.Unlock()
	c.ID = id
	if m.opts.SaveSettings != nil {
		if err := m.opts.SaveSettings(func(s *Settings) { s.Custom = append(s.Custom, c) }); err != nil {
			return View{}, err
		}
	}
	m.mu.Lock()
	m.rebuildLocked()
	e, _ := m.entryLocked(id)
	m.mu.Unlock()
	m.detectOne(m.ctx, e)
	m.adoptYoyoChat(id)
	m.markProbing(id)
	m.publish(id)
	m.goSafe(func() { _ = m.probe(m.ctx, id, true) })
	return m.view(id), nil
}

// slugID 把名字变成编号：小写字母、数字和连字符。
func slugID(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "agent"
	}
	if out == YoyoID {
		out = "custom-yoyo"
	}
	return out
}

// Ready 判断一项能不能派活：yoyo 永远能；其余要已安装、启用、在线（或正在服务工单）。
func (m *Manager) Ready(id string) bool {
	if id == "" || id == YoyoID {
		return true
	}
	v, err := m.Get(id)
	if err != nil {
		return false
	}
	return v.Config.Enabled && v.State == StateOnline
}

// Online 和 Ready 相同，给工人和路由用。
func (m *Manager) Online(id string) bool { return m.Ready(id) }

// Roster 返回给路由看的名单：yoyo 加上所有已安装且启用的 agent。
func (m *Manager) Roster() []RosterItem {
	views, _ := m.List()
	var out []RosterItem
	for _, v := range views {
		if !v.Builtin && (v.Install.Kind == "none" || !v.Config.Enabled) {
			continue
		}
		out = append(out, RosterItem{
			ID: v.ID, Name: v.Name, Vendor: v.Vendor, Aliases: v.Aliases, State: v.State,
			Ready: v.Builtin || v.State == StateOnline, Working: v.Working, MaxParallel: v.Config.MaxParallel, Strengths: v.Config.Strengths,
		})
	}
	return out
}

// NoteRun 记下工人用这个 agent 跑完一轮的结果：成功就算在线；失败只记详情，留给健康检查判断。
func (m *Manager) NoteRun(id string, err error) {
	if id == "" || id == YoyoID {
		return
	}
	now := m.opts.Now()
	m.update(id, func(r *record) {
		r.CheckedAt = now
		if err == nil {
			r.State = StateOnline
			r.Detail = ""
			r.LostAt = time.Time{}
			return
		}
		r.Detail = redactText(err.Error())
	})
}

// NoteAuthFailure 记下工人发现要重新登录。
func (m *Manager) NoteAuthFailure(id, detail string) {
	if id == "" || id == YoyoID {
		return
	}
	now := m.opts.Now()
	m.update(id, func(r *record) {
		r.State = StateNeedsAuth
		r.Detail = redactText(detail)
		r.CheckedAt = now
	})
}

// MaxParallel 返回一项同时最多几个任务。
func (m *Manager) MaxParallel(id string) int {
	return m.opts.Settings().Item(id).MaxParallelOf()
}

// Approval 返回一项的审批方式：yoyo 或 ask。
func (m *Manager) Approval(id string) string {
	return firstNonEmpty(m.opts.Settings().Item(id).Approval, "yoyo")
}

// Model 返回人为这一项选的模型，没选时为空。
func (m *Manager) Model(id string) string {
	return m.opts.Settings().Item(id).Model
}
