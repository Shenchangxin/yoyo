// helpers_test.go 拥有 buddy 测试共用的假配置、假 vault、可调时钟和建 Manager 的帮手。不含测试本身。

package buddy

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
)

// memSettings 是内存里的配置。
type memSettings struct {
	mu sync.Mutex
	s  Settings
}

// get 返回副本。
func (m *memSettings) get() Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.s.Clone()
}

// save 在副本上改完再换回去。
func (m *memSettings) save(fn func(*Settings)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.s.Clone()
	fn(&c)
	m.s = c
	return nil
}

// memVault 是内存里的 vault。n 数读了几次。
type memVault struct {
	mu sync.Mutex
	m  map[string]string
	n  int
}

// reads 返回 GetStored 被调了几次。
func (v *memVault) reads() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.n
}

// GetStored 取值，没有时报错。
func (v *memVault) GetStored(name string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.n++
	if s := v.m[name]; s != "" {
		return s, nil
	}
	return "", ErrNotFound
}

// Set 存值，空值删除。
func (v *memVault) Set(name, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.m == nil {
		v.m = map[string]string{}
	}
	if value == "" {
		delete(v.m, name)
		return
	}
	v.m[name] = value
}

// clock 是可以拨动的时钟。
type clock struct {
	mu  sync.Mutex
	now time.Time
}

// Now 返回当前时间。
func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// add 拨快 d。
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// offline 拒绝一切请求，测试默认不碰网络。
type offline struct{}

// RoundTrip 总是失败。
func (offline) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline")
}

// fixture 是一套测试用的 Manager 和它的假依赖。
type fixture struct {
	m        *Manager
	settings *memSettings
	vault    *memVault
	clock    *clock
}

// newFixture 建一个不发网络、不找真命令的 Manager。opts 可以再改字段。
func newFixture(t *testing.T, edit func(*Options)) *fixture {
	t.Helper()
	f := &fixture{settings: &memSettings{}, vault: &memVault{}, clock: &clock{now: time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC)}}
	opts := Options{
		Root: t.TempDir(), RuntimeDir: t.TempDir(), Version: "test",
		Platform: &Platform{GOOS: "linux", GOARCH: "amd64"},
		HTTP:     &http.Client{Transport: offline{}},
		LookPath: func(string) (string, error) { return "", errNotFound },
		Run:      func(context.Context, Cmd) ([]byte, error) { return nil, nil },
		Now:      f.clock.Now, Vault: f.vault, Settings: f.settings.get, SaveSettings: f.settings.save,
		Dial: func(context.Context, acp.Options) (*acp.Client, error) {
			return nil, ErrNotInstalled
		},
		BaseEnv:     func() []string { return []string{"PATH=/usr/bin", "HOME=/home/me"} },
		Throttle:    -1,
		HealthDelay: time.Hour,
	}
	if edit != nil {
		edit(&opts)
	}
	f.m = New(opts)
	f.m.paths.shell = nil
	f.m.paths.env = func(name string) string {
		if name == "PATH" {
			return "/usr/bin"
		}
		return ""
	}
	t.Cleanup(f.m.Close)
	return f
}

// pipeDial 让 Dial 连到同进程的假 agent，script 决定它怎么回。
func pipeDial(script acptest.Script) func(context.Context, acp.Options) (*acp.Client, error) {
	return func(ctx context.Context, opts acp.Options) (*acp.Client, error) {
		r, w, _ := acptest.Pipe(script)
		return acp.Connect(ctx, r, w, opts)
	}
}

// withCustom 往配置里加一个能找到的自定义 agent，返回它的编号。
func (f *fixture) withCustom(t *testing.T, id string) string {
	t.Helper()
	_ = f.settings.save(func(s *Settings) {
		s.Custom = append(s.Custom, CustomAgent{ID: id, Name: id, Command: id + "-bin"})
	})
	prev := f.m.opts.LookPath
	f.m.opts.LookPath = func(name string) (string, error) {
		if name == id+"-bin" {
			return "/opt/" + id + "/" + id + "-bin", nil
		}
		return prev(name)
	}
	f.m.mu.Lock()
	f.m.rebuildLocked()
	e, _ := f.m.entryLocked(id)
	f.m.mu.Unlock()
	f.m.detectOne(context.Background(), e)
	return id
}
