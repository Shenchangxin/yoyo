// buddy.go 拥有 app 对 Buddy 的接线：建 buddy.Manager（buddies 目录、vault、config.yaml 的 buddies 段、Hub 事件、yoyo 自己的钥匙），
// 以及 buddy.* RPC 的包装方法。安装、探测、登录的细节在 internal/buddy；手机上的禁用和确认在 internal/remote/policy.go。
// Manager 第一次被列出时才开始检测和健康检查，单元测试打开 App 不会去碰本机装的 agent。

package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/version"
)

// errBuddyOff 是 Buddy 没有接上（没有主目录）时的错误。
var errBuddyOff = errors.New("buddy is not running")

// buddyVault 让 Manager 每次都用 App 当前的 vault。没有 vault 时读不到、写不进。
type buddyVault struct{ a *App }

// GetStored 读一把已保存的密钥。
func (v buddyVault) GetStored(name string) (string, error) {
	if v.a.Vault == nil {
		return "", errBuddyOff
	}
	return v.a.Vault.GetStored(name)
}

// Set 保存或删除一把密钥。
func (v buddyVault) Set(name, value string) {
	if v.a.Vault != nil {
		v.a.Vault.Set(name, value)
	}
}

// configureBuddies 在 Home 为空时直接返回，否则建 Manager。不在这里 Start。
func (a *App) configureBuddies() {
	if a == nil || a.Home == nil {
		return
	}
	a.buddies = buddy.New(buddy.Options{
		Root:         a.Home.Buddies(),
		RuntimeDir:   a.Home.Runtime(),
		Version:      version.Version,
		Locale:       func() string { return a.Config.Locale },
		Vault:        buddyVault{a},
		Settings:     a.buddySettings,
		SaveSettings: a.saveBuddySettings,
		Publish:      a.publishBuddy,
		YoyoKey:      a.yoyoKey,
	})
}

// Buddies 返回 Buddy 管理器，没有主目录时为 nil。工人和路由从这里查名单和启动方式。
func (a *App) Buddies() *buddy.Manager {
	if a == nil {
		return nil
	}
	return a.buddies
}

// closeBuddies 停掉后台检测、健康检查和进行中的安装。
func (a *App) closeBuddies() {
	if a != nil && a.buddies != nil {
		a.buddies.Close()
	}
}

// buddySettings 返回 buddies 段的副本。
func (a *App) buddySettings() buddy.Settings {
	a.buddyMu.Lock()
	defer a.buddyMu.Unlock()
	return a.Config.Buddies.Clone()
}

// saveBuddySettings 在副本上改完换回去，再写 config.yaml。
func (a *App) saveBuddySettings(fn func(*buddy.Settings)) error {
	a.buddyMu.Lock()
	s := a.Config.Buddies.Clone()
	fn(&s)
	a.Config.Buddies = s
	a.buddyMu.Unlock()
	return a.SaveConfig()
}

// buddyNoticeLine 是事件里进度行的最多字数。整条事件要小到能内联进桌面通知（liveInlineBytes），否则界面只收到一个空壳。
const buddyNoticeLine = 160

// publishBuddy 把一项的变化发到 Hub：system 类事件，来源和种类都是 buddy。
// 载荷只有编号、状态、进度和在做的任务数；界面要全量视图时再调 buddy.get。
func (a *App) publishBuddy(v buddy.View) {
	if a == nil || a.Hub == nil {
		return
	}
	payload := map[string]any{"id": v.ID, "state": string(v.State)}
	if v.Working > 0 {
		payload["working"] = v.Working
	}
	if p := v.Progress; p != nil {
		payload["progress"] = map[string]any{"phase": p.Phase, "percent": p.Percent, "line": clipRunes(p.Line, buddyNoticeLine)}
	}
	a.Hub.Publish(trace.Event{TS: time.Now().UTC(), Type: trace.TypeSystem, Source: "buddy", ItemKind: "buddy", Payload: payload})
}

// yoyoKey 按提供商找 yoyo 自己的聊天钥匙，给人打开了「复用」的 agent 用。只认官方端点上的启用连接，代理端点的钥匙不外借。
func (a *App) yoyoKey(provider string) (string, bool) {
	r := a.connReg()
	if r == nil {
		return "", false
	}
	list, err := r.List(connection.CapChat)
	if err != nil {
		return "", false
	}
	for _, c := range list {
		if !c.Active || !officialFor(provider, c) {
			continue
		}
		if k, err := r.Lease(c); err == nil && strings.TrimSpace(k) != "" {
			return strings.TrimSpace(k), true
		}
	}
	return "", false
}

// officialFor 判断连接是不是 provider 的官方端点（端点为空算官方）。
func officialFor(provider string, c connection.Connection) bool {
	vendor := strings.ToLower(strings.TrimSpace(c.Vendor))
	ep := strings.ToLower(strings.TrimSpace(c.Endpoint))
	switch provider {
	case "anthropic":
		return (vendor == "claude" || vendor == "anthropic") && (ep == "" || strings.Contains(ep, "api.anthropic.com"))
	case "openai":
		return vendor == "openai" && (ep == "" || strings.Contains(ep, "api.openai.com"))
	case "gemini":
		return (vendor == "gemini" || vendor == "google") && (ep == "" || strings.Contains(ep, "generativelanguage.googleapis.com"))
	}
	return false
}

// buddyManager 返回管理器，没有时报错。
func (a *App) buddyManager() (*buddy.Manager, error) {
	if a == nil || a.buddies == nil {
		return nil, errBuddyOff
	}
	return a.buddies, nil
}

// BuddyList 返回 {buddies, prefs}。第一次调用时开始后台检测。
func (a *App) BuddyList() (map[string]any, error) {
	m, err := a.buddyManager()
	if err != nil {
		return nil, err
	}
	views, prefs := m.List()
	return map[string]any{"buddies": views, "prefs": prefs}, nil
}

// BuddyGet 返回一项。
func (a *App) BuddyGet(id string) (buddy.View, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.View{}, err
	}
	return m.Get(id)
}

// BuddyLogTail 返回 {text}：安装日志和探测日志的最后几行，已打码。
func (a *App) BuddyLogTail(id string, lines int) (map[string]any, error) {
	m, err := a.buddyManager()
	if err != nil {
		return nil, err
	}
	text, err := m.LogTail(id, lines)
	if err != nil {
		return nil, err
	}
	return map[string]any{"text": text}, nil
}

// BuddyRefresh 在后台重新检测和探测，force 时先刷新注册表；立刻返回当前列表。
func (a *App) BuddyRefresh(force bool) (map[string]any, error) {
	m, err := a.buddyManager()
	if err != nil {
		return nil, err
	}
	m.Refresh(force)
	return a.BuddyList()
}

// BuddyProbe 对一项做全探，返回探测后的视图。
func (a *App) BuddyProbe(ctx context.Context, id string) (buddy.View, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.View{}, err
	}
	m.Start()
	v, err := m.Probe(ctx, id)
	if errors.Is(err, buddy.ErrNotFound) {
		return v, err
	}
	return v, nil
}

// BuddyInstall 开始安装，返回安装中的视图。
func (a *App) BuddyInstall(id, mode string) (buddy.View, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.View{}, err
	}
	m.Start()
	return m.Install(id, mode)
}

// BuddyCancel 取消进行中的安装。
func (a *App) BuddyCancel(id string) (map[string]any, error) {
	m, err := a.buddyManager()
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, m.Cancel(id)
}

// BuddyUninstall 卸载一项。
func (a *App) BuddyUninstall(ctx context.Context, id string) (buddy.View, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.View{}, err
	}
	m.Start()
	return m.Uninstall(ctx, id)
}

// BuddyConfigure 保存一项的设置，密钥进 vault。返回的视图只有 hasKey。
func (a *App) BuddyConfigure(id string, p buddy.ConfigPatch) (buddy.View, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.View{}, err
	}
	m.Start()
	return m.Configure(id, p)
}

// BuddyLogin 开始登录，结果走事件。
func (a *App) BuddyLogin(ctx context.Context, id, method string) (buddy.LoginResult, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.LoginResult{}, err
	}
	m.Start()
	return m.Login(ctx, id, method)
}

// BuddyLogout 退出登录。
func (a *App) BuddyLogout(ctx context.Context, id string) (buddy.View, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.View{}, err
	}
	m.Start()
	return m.Logout(ctx, id)
}

// BuddyPrefs 保存偏好，返回保存后的偏好。
func (a *App) BuddyPrefs(p buddy.PrefsPatch) (buddy.Prefs, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.Prefs{}, err
	}
	return m.SetPrefs(p)
}

// BuddyAdd 添加自定义 ACP agent。
func (a *App) BuddyAdd(c buddy.CustomAgent) (buddy.View, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.View{}, err
	}
	m.Start()
	return m.Add(c)
}

// BuddyDisk 统计 Buddy 占的磁盘。
func (a *App) BuddyDisk() (buddy.DiskUsage, error) {
	m, err := a.buddyManager()
	if err != nil {
		return buddy.DiskUsage{}, err
	}
	return m.Disk(), nil
}

// BuddyClearCache 清掉能重新下载的缓存，返回 {freed} 字节数。
func (a *App) BuddyClearCache(ctx context.Context) (map[string]any, error) {
	m, err := a.buddyManager()
	if err != nil {
		return nil, err
	}
	m.Start()
	freed, err := m.ClearCache(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"freed": freed}, nil
}
