// detect.go 拥有本机检测：依次看托管安装（buddies/<id>/install.json 且入口还在）、PATH 上的 ACP 命令、只有 CLI 没有适配器三种情况；
// PATH 上没有的，再看厂商安装器和桌面应用常放的位置（host.go）。
// 再用 --version 取系统版本（超时 10 秒、并发 8，文件没变就用缓存）。检测时不读任何凭证文件。

package buddy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// detection 是一项在本机的安装情况。Host 非 nil 表示有东西是在 PATH 之外的常见位置找到的（见 host.go）。
type detection struct {
	Managed       *InstallRecord
	ManagedDir    string
	SystemPath    string
	SystemVersion string
	CLIPath       string
	Host          *hostHit
}

// InstallRecord 是 buddies/<id>/install.json：托管安装的启动方式和来历。
type InstallRecord struct {
	Kind        string            `json:"kind"`
	Version     string            `json:"version"`
	Package     string            `json:"package,omitempty"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	ViaCmd      bool              `json:"viaCmd,omitempty"`
	Runtime     string            `json:"runtime,omitempty"`
	Entry       string            `json:"entry,omitempty"`
	Dir         string            `json:"dir"`
	SHA256      string            `json:"sha256,omitempty"`
	Verified    bool              `json:"verified,omitempty"`
	SizeBytes   int64             `json:"sizeBytes,omitempty"`
	InstalledAt time.Time         `json:"installedAt"`
}

// reVersion 从 --version 的输出里取版本号。
var reVersion = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?(?:[-+.][0-9A-Za-z.]+)?`)

// detectAll 检测全部项，再在后台补上系统版本，最后发出全部视图。
func (m *Manager) detectAll(ctx context.Context) {
	m.mu.Lock()
	entries := slices.Clone(m.entries)
	m.mu.Unlock()
	npm := ""
	if p, err := m.opts.LookPath("npm"); err == nil {
		npm = p
	}
	results := make(map[string]detection, len(entries))
	for _, e := range entries {
		if ctx.Err() != nil {
			return
		}
		results[e.ID] = m.detectFast(e)
	}
	m.mu.Lock()
	m.sysNPM = npm
	for id, d := range results {
		if prev, ok := m.det[id]; ok && prev.SystemPath == d.SystemPath {
			d.SystemVersion = prev.SystemVersion
		}
		m.det[id] = d
	}
	m.detected = true
	m.mu.Unlock()
	m.publishAll()
	m.fillVersions(ctx, entries)
}

// Detecting 表示第一次检测还没做完。这时列表里的「未安装」还不可信，界面先显示骨架。
func (m *Manager) Detecting() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return !m.detected
}

// detectOne 重新检测一项，装完、卸完或加了自定义 agent 之后用。
func (m *Manager) detectOne(ctx context.Context, e Entry) {
	d := m.detectFast(e)
	m.mu.Lock()
	if prev, ok := m.det[e.ID]; ok && prev.SystemPath == d.SystemPath {
		d.SystemVersion = prev.SystemVersion
	}
	m.det[e.ID] = d
	m.mu.Unlock()
	m.fillVersions(ctx, []Entry{e})
}

// Prepare 让 id 在第一次列出之前也能判断在线和启动：还没检测过时只按文件和 PATH 检测这一项，不跑命令。
// 工人续接和派活前调用；已经检测过的不重做。
func (m *Manager) Prepare(id string) {
	if id == "" || id == YoyoID {
		return
	}
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	_, seen := m.det[id]
	m.mu.Unlock()
	if !ok || seen {
		return
	}
	d := m.detectFast(e)
	m.mu.Lock()
	if _, seen := m.det[id]; !seen {
		m.det[id] = d
	}
	m.mu.Unlock()
}

// detectFast 只看文件和 PATH，不跑任何命令。
func (m *Manager) detectFast(e Entry) detection {
	var d detection
	if e.Builtin {
		return d
	}
	if e.Custom {
		if p, err := m.opts.LookPath(e.Command); err == nil {
			d.SystemPath = p
		}
		return d
	}
	if rec, dir := m.readInstall(e.ID); rec != nil {
		d.Managed = rec
		d.ManagedDir = dir
	}
	for _, bin := range e.ACPBins {
		if p, err := m.opts.LookPath(bin); err == nil {
			d.SystemPath = p
			break
		}
	}
	// PATH 上没有时，再看厂商安装器和桌面应用常放的位置。找到能直接连的命令，就和 PATH 上的系统版一样用。
	if d.SystemPath == "" {
		if hit, ok := m.hostFind(e, hostACP); ok {
			d.SystemPath = hit.Path
			d.Host = &hit
		}
	}
	// 没有能直接连的命令时找 CLI，用来提示「装了 CLI，还缺适配器」。
	// 适配器能改用本机 CLI 的（CLIEnv），不管适配器在不在都找，启动时把路径交给它。
	if d.SystemPath == "" || e.CLIEnv != "" {
		for _, bin := range e.CLIBins {
			if slices.Contains(e.ACPBins, bin) {
				continue
			}
			if p, err := m.opts.LookPath(bin); err == nil {
				d.CLIPath = p
				break
			}
		}
		if d.CLIPath == "" {
			if hit, ok := m.hostFind(e, hostCLI); ok {
				d.CLIPath = hit.Path
				if d.Host == nil {
					d.Host = &hit
				}
			}
		}
	}
	// 什么命令都没有、也没托管安装时，看有没有只装了桌面应用。它自己不是 agent，只用来告诉人还差什么。
	if d.SystemPath == "" && d.CLIPath == "" && d.Managed == nil {
		if hit, ok := m.hostFind(e, hostApp); ok {
			d.Host = &hit
		}
	}
	return d
}

// installDir 是一项的托管目录。
func (m *Manager) installDir(id string) string {
	return filepath.Join(m.opts.Root, id)
}

// readInstall 读 install.json；入口文件已经不在时当作没装。
func (m *Manager) readInstall(id string) (*InstallRecord, string) {
	if m.opts.Root == "" {
		return nil, ""
	}
	dir := m.installDir(id)
	raw, err := os.ReadFile(filepath.Join(dir, "install.json"))
	if err != nil {
		return nil, ""
	}
	var rec InstallRecord
	if json.Unmarshal(raw, &rec) != nil || rec.Command == "" {
		return nil, ""
	}
	if rec.Entry != "" {
		if !fileExists(rec.Entry) {
			return nil, ""
		}
	} else if !fileExists(rec.Command) {
		return nil, ""
	}
	return &rec, dir
}

// writeInstall 写 install.json。
func (m *Manager) writeInstall(id string, rec *InstallRecord) error {
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	dir := m.installDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(dir, "install.json.tmp")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "install.json"))
}

// fillVersions 给有系统路径的项跑 --version。并发 8，单个 10 秒；文件路径和修改时间没变就用缓存。
func (m *Manager) fillVersions(ctx context.Context, entries []Entry) {
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, e := range entries {
		m.mu.Lock()
		d := m.det[e.ID]
		r := m.recs[e.ID]
		m.mu.Unlock()
		if d.SystemPath == "" || e.Custom {
			continue
		}
		info, err := os.Stat(d.SystemPath)
		if err != nil {
			continue
		}
		if r != nil && r.SysPath == d.SystemPath && r.SysMod.Equal(info.ModTime()) && r.SysVersion != "" {
			m.setVersion(e.ID, d.SystemPath, info.ModTime(), r.SysVersion)
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(id, path string, mod time.Time) {
			defer wg.Done()
			defer func() { <-sem }()
			v := m.versionOf(ctx, path)
			m.setVersion(id, path, mod, v)
			m.publish(id)
		}(e.ID, d.SystemPath, info.ModTime())
	}
	wg.Wait()
	m.saveRecords()
}

// setVersion 记下系统版本号和缓存键。
func (m *Manager) setVersion(id, path string, mod time.Time, version string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d := m.det[id]
	if d.SystemPath != path {
		return
	}
	d.SystemVersion = version
	m.det[id] = d
	r := m.recordLocked(id)
	r.SysPath, r.SysMod, r.SysVersion = path, mod, version
}

// versionOf 跑 path --version，取输出里第一个像版本号的词。
func (m *Manager) versionOf(ctx context.Context, path string) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	l := commandFor(path, []string{"--version"}, m.nodePath)
	l.Env = m.baseEnv(nil)
	out, _ := m.opts.Run(ctx, Cmd{Launch: l})
	for _, line := range strings.Split(string(out), "\n") {
		if v := reVersion.FindString(line); v != "" {
			return v
		}
	}
	return ""
}

// fileExists 判断 path 是不是存在的普通文件。
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
