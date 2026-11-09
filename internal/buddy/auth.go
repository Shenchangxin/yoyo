// auth.go 拥有登录和退出登录。agent 型方法调 authenticate（最多等 5 分钟）后全探；terminal 型方法用同一个启动器加上方法的参数和环境，
// 打开可见终端，结束后（或每 5 秒，最多 10 分钟）全探。对方没声明方法时用精选表的兜底命令。env_var 型方法到配置里填密钥。

package buddy

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// TerminalResult 是开终端的结果。Waited 为真表示终端里的命令已经结束；Copy 非空表示没有可用终端，要人自己复制运行。
type TerminalResult struct {
	Waited bool
	Copy   string
}

// LoginResult 是一次登录的回执。登录在后台进行，结果走事件。
type LoginResult struct {
	Started  bool   `json:"started"`
	Terminal bool   `json:"terminal,omitempty"`
	Copy     string `json:"copy,omitempty"`
	Buddy    View   `json:"buddy"`
}

// authTimeout 是 agent 型登录的上限。
const authTimeout = 5 * time.Minute

// Login 用 methodID（空时取第一个可用方法）登录 id。
func (m *Manager) Login(ctx context.Context, id, methodID string) (LoginResult, error) {
	settings := m.opts.Settings()
	item := settings.Item(id)
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	d := m.det[id]
	var methods []acp.AuthMethod
	if r := m.recs[id]; r != nil && r.Probe != nil {
		methods = r.Probe.Auth
	}
	m.mu.Unlock()
	if !ok || e.Builtin {
		return LoginResult{}, ErrNotFound
	}
	l, secrets, err := m.launchFor(e, item, d)
	if err != nil {
		return LoginResult{}, err
	}
	method, found := pickMethod(methods, methodID)
	switch {
	case found && method.Kind() == "env_var":
		return LoginResult{}, errors.New(m.say("这个 agent 用密钥登录，到配置里填", "this agent signs in with a key; fill it in under Configure"))
	case found && method.Kind() == "agent":
		m.update(id, func(r *record) {
			r.Detail = m.say("等你在弹出的页面里完成登录", "finish signing in on the page that opened")
		})
		m.goSafe(func() {
			err := m.authenticate(id, l, method.ID, secrets)
			if err != nil {
				m.update(id, func(r *record) {
					r.State = StateNeedsAuth
					r.Detail = shortDetail(redactText(err.Error(), secrets...))
				})
				return
			}
			_ = m.probe(m.ctx, id, true)
		})
		return LoginResult{Started: true, Buddy: m.view(id)}, nil
	case found:
		term := l.WithArgs(method.Args...)
		for _, kv := range method.EnvPairs() {
			if i := strings.IndexByte(kv, '='); i > 0 {
				term.Env = setEnv(term.Env, kv[:i], kv[i+1:])
			}
		}
		return m.terminalLogin(ctx, id, term)
	}
	for _, spec := range e.Login {
		if len(spec.Cmd) == 0 {
			continue
		}
		path, err := m.opts.LookPath(spec.Cmd[0])
		if err != nil {
			continue
		}
		term := commandFor(path, spec.Cmd[1:], m.nodePath)
		term.Env = l.Env
		return m.terminalLogin(ctx, id, term)
	}
	return LoginResult{}, errors.New(m.say("它没有声明登录方式，按它的文档在终端里登录", "it declares no sign-in method; sign in from a terminal as its docs describe"))
}

// pickMethod 按编号找登录方式；编号为空时取第一个 agent 或 terminal 型的方法。
func pickMethod(methods []acp.AuthMethod, id string) (acp.AuthMethod, bool) {
	for _, method := range methods {
		if id != "" && method.ID == id {
			return method, true
		}
	}
	if id != "" {
		return acp.AuthMethod{}, false
	}
	for _, method := range methods {
		if method.Kind() != "env_var" {
			return method, true
		}
	}
	for _, method := range methods {
		return method, true
	}
	return acp.AuthMethod{}, false
}

// authenticate 启动 agent，握手后调 authenticate，最多等 5 分钟。
func (m *Manager) authenticate(id string, l acp.Launch, methodID string, secrets []string) error {
	ctx, cancel := context.WithTimeout(m.ctx, authTimeout)
	defer cancel()
	opts := l.Options()
	opts.Dir = m.probeDir(id)
	opts.Version = m.opts.Version
	opts.Name = id
	opts.InitTimeout = probeTimeout
	c, err := m.opts.Dial(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		_ = c.Close()
		m.writeProbeLog(id, c.Stderr().Tail(4000), secrets)
	}()
	return c.Authenticate(ctx, methodID)
}

// terminalLogin 在可见终端里跑登录命令。终端结束后全探一次；拿不到结束信号时每 5 秒全探，最多 10 分钟。
func (m *Manager) terminalLogin(ctx context.Context, id string, l acp.Launch) (LoginResult, error) {
	l.Dir = m.probeDir(id)
	l.Env = withoutEnv(l.Env, "TERM", "NO_COLOR")
	m.update(id, func(r *record) {
		r.Detail = m.say("在终端窗口里按提示登录，完成后回来", "follow the prompts in the terminal window, then come back")
	})
	m.goSafe(func() {
		res, err := m.opts.Terminal(m.ctx, l)
		if err != nil {
			m.update(id, func(r *record) { r.Detail = shortDetail(redactText(err.Error())) })
			return
		}
		if res.Copy != "" {
			return
		}
		if res.Waited {
			_ = m.probe(m.ctx, id, true)
			return
		}
		deadline := time.Now().Add(10 * time.Minute)
		for time.Now().Before(deadline) {
			select {
			case <-m.ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			_ = m.probe(m.ctx, id, true)
			if m.view(id).State == StateOnline {
				return
			}
		}
	})
	return LoginResult{Started: true, Terminal: true, Buddy: m.view(id)}, nil
}

// Logout 让 agent 退出登录，然后全探。对方没声明 logout 能力时报错。
func (m *Manager) Logout(ctx context.Context, id string) (View, error) {
	settings := m.opts.Settings()
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	d := m.det[id]
	canLogout := false
	if r := m.recs[id]; r != nil && r.Probe != nil && r.Probe.Caps.Auth != nil {
		canLogout = bool(r.Probe.Caps.Auth.Logout)
	}
	m.mu.Unlock()
	if !ok || e.Builtin {
		return View{}, ErrNotFound
	}
	if !canLogout {
		return View{}, errors.New(m.say("它不支持从 yoyo 退出登录", "it does not support signing out from yoyo"))
	}
	l, secrets, err := m.launchFor(e, settings.Item(id), d)
	if err != nil {
		return View{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	opts := l.Options()
	opts.Dir = m.probeDir(id)
	opts.Version = m.opts.Version
	opts.Name = id
	c, err := m.opts.Dial(ctx, opts)
	if err != nil {
		return View{}, errors.New(shortDetail(redactText(err.Error(), secrets...)))
	}
	err = c.Logout(ctx)
	_ = c.Close()
	if err != nil {
		return View{}, errors.New(shortDetail(redactText(err.Error(), secrets...)))
	}
	_ = m.probe(ctx, id, true)
	return m.view(id), nil
}
