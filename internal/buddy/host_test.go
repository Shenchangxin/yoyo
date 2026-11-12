// host_test.go 锁住 PATH 之外的本机检测：厂商安装器放的命令算系统版、能直接连；桌面应用带的 CLI 只算「缺适配器」，
// 并记下来自哪个应用；只有桌面应用时不算装了，只记一笔；PATH 上的优先；变量没设的位置不找；按版本分的目录取最高的；
// 检测全程不运行任何命令。也锁住适配器启动时拿到本机 CLI 的路径，以及关掉、人自己写了、文件没了这三种不交的情况。

package buddy

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
)

// hostFixture 建一个按本机系统看文件的 Manager。env 是它看得到的环境变量；runs 数跑了几条命令。
func hostFixture(t *testing.T, env map[string]string) (*fixture, *atomic.Int32) {
	t.Helper()
	var runs atomic.Int32
	f := newFixture(t, func(o *Options) {
		o.Platform = &Platform{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
		o.Run = func(context.Context, Cmd) ([]byte, error) {
			runs.Add(1)
			return nil, nil
		}
	})
	f.m.paths.env = func(name string) string { return env[name] }
	return f, &runs
}

// setHost 把 id 的常见位置换成 spots（三个系统都用这一组），可选地改它认的环境变量，返回改好的目录项。
func (f *fixture) setHost(t *testing.T, id, cliEnv string, spots ...HostSpot) Entry {
	t.Helper()
	f.m.mu.Lock()
	defer f.m.mu.Unlock()
	for i := range f.m.entries {
		if f.m.entries[i].ID != id {
			continue
		}
		f.m.entries[i].Host = HostSpec{Windows: spots, Darwin: spots, Linux: spots}
		f.m.entries[i].CLIEnv = cliEnv
		return f.m.entries[i]
	}
	t.Fatalf("no entry %q", id)
	return Entry{}
}

// fakeBin 在 dir 里放一个找得到、但跑不起来的命令，返回它的路径。
func fakeBin(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("not a program"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// put 把检测结果记进 Manager，之后 Get 和 launch 读它。
func (f *fixture) put(id string, d detection) {
	f.m.mu.Lock()
	f.m.det[id] = d
	f.m.mu.Unlock()
}

// TestExpandHost 锁住位置里的变量怎么展开，以及少一个变量就整条不要。
func TestExpandHost(t *testing.T) {
	env := func(name string) string {
		return map[string]string{"LOCALAPPDATA": "/data/local", "HOME": "/home/me", "USERPROFILE": "/users/me", "ProgramFiles(x86)": "/pf86"}[name]
	}
	for _, c := range []struct{ in, goos, want string }{
		{"%LOCALAPPDATA%/cursor-agent", "windows", "/data/local/cursor-agent"},
		{"%ProgramFiles(x86)%/App/app.exe", "windows", "/pf86/App/app.exe"},
		{"$HOME/.opencode/bin", "linux", "/home/me/.opencode/bin"},
		{"${HOME}/bin", "linux", "/home/me/bin"},
		{"~/.opencode/bin", "darwin", "/home/me/.opencode/bin"},
		{"~/.local/bin", "windows", "/users/me/.local/bin"},
		{"/Applications/Cursor.app", "darwin", "/Applications/Cursor.app"},
		{"%APPDATA%/Claude", "windows", ""},
		{"$NOPE/bin", "linux", ""},
		{"", "linux", ""},
	} {
		if got := filepath.ToSlash(expandHost(c.in, env, c.goos)); got != c.want {
			t.Errorf("expandHost(%q, %s) = %q, want %q", c.in, c.goos, got, c.want)
		}
	}
	if got := expandHost("~/bin", func(string) string { return "" }, "linux"); got != "" {
		t.Errorf("no home should give nothing, got %q", got)
	}
}

// TestVersionLess 锁住按版本分的目录取最高的那个，不是按字面排。
func TestVersionLess(t *testing.T) {
	dirs := []string{"/x/2.1.9", "/x/2.1.10", "/x/2.0.30", "/x/10.0.0", "/x/9.9.9"}
	slices.SortStableFunc(dirs, func(a, b string) int {
		if versionLess(b, a) {
			return -1
		}
		if versionLess(a, b) {
			return 1
		}
		return 0
	})
	if want := []string{"/x/10.0.0", "/x/9.9.9", "/x/2.1.10", "/x/2.1.9", "/x/2.0.30"}; !slices.Equal(dirs, want) {
		t.Fatalf("sorted %v", dirs)
	}
}

// TestHostACPBinaryIsASystemInstall 锁住厂商安装器放在 PATH 之外的命令：算系统版、视图里写明是在哪找到的，检测不跑命令。
func TestHostACPBinaryIsASystemInstall(t *testing.T) {
	local := t.TempDir()
	bin := fakeBin(t, filepath.Join(local, "cursor-agent"), "cursor-agent")
	f, runs := hostFixture(t, map[string]string{"LOCALAPPDATA": local})
	e := f.setHost(t, "cursor", "",
		HostSpot{Kind: hostACP, Dir: "%LOCALAPPDATA%/cursor-agent"},
		HostSpot{Kind: hostApp, File: "%LOCALAPPDATA%/Programs/cursor/Cursor.exe", App: "Cursor"},
	)
	d := f.m.detectFast(e)
	if runs.Load() != 0 {
		t.Fatalf("detection ran %d commands", runs.Load())
	}
	if d.SystemPath != bin || d.Host == nil || d.Host.Kind != hostACP || d.CLIPath != "" {
		t.Fatalf("detection %+v host %+v", d, d.Host)
	}
	f.put("cursor", d)
	v, err := f.m.Get("cursor")
	if err != nil {
		t.Fatal(err)
	}
	if v.Install.Kind != "system" || v.Install.Path != bin || v.Host == nil || v.Host.Kind != hostACP || v.Host.Path != bin || v.CLIOnly {
		t.Fatalf("view install %+v host %+v cliOnly %v", v.Install, v.Host, v.CLIOnly)
	}
	if v.State == StateNotInstalled || v.State == StateUnsupported {
		t.Fatalf("a found binary is listed as %s", v.State)
	}
	l, _, err := f.m.launch("cursor")
	if err != nil || l.Command == "" {
		t.Fatalf("launch %+v %v", l, err)
	}
}

// TestHostAppMarkerIsNotAnInstall 锁住只装了桌面应用时：不算装了、不能启动，只记下来自哪个应用。
func TestHostAppMarkerIsNotAnInstall(t *testing.T) {
	local := t.TempDir()
	app := filepath.Join(local, "Programs", "cursor", "Cursor.exe")
	_ = os.MkdirAll(filepath.Dir(app), 0o755)
	if err := os.WriteFile(app, []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, runs := hostFixture(t, map[string]string{"LOCALAPPDATA": local})
	e := f.setHost(t, "cursor", "",
		HostSpot{Kind: hostACP, Dir: "%LOCALAPPDATA%/cursor-agent"},
		HostSpot{Kind: hostApp, File: "%LOCALAPPDATA%/Programs/cursor/Cursor.exe", App: "Cursor"},
	)
	d := f.m.detectFast(e)
	if runs.Load() != 0 {
		t.Fatalf("detection ran %d commands", runs.Load())
	}
	if d.SystemPath != "" || d.CLIPath != "" || d.Host == nil || d.Host.Kind != hostApp || d.Host.App != "Cursor" {
		t.Fatalf("detection %+v host %+v", d, d.Host)
	}
	f.put("cursor", d)
	v, _ := f.m.Get("cursor")
	if v.Install.Kind != "none" || v.CLIOnly || v.Host == nil || v.Host.Kind != hostApp || v.Host.Path != app {
		t.Fatalf("view install %+v host %+v cliOnly %v", v.Install, v.Host, v.CLIOnly)
	}
	if v.State != StateNotInstalled && v.State != StateUnsupported {
		t.Fatalf("an app marker made the agent %s", v.State)
	}
	if _, _, err := f.m.launch("cursor"); err == nil {
		t.Fatal("an app marker is launchable")
	}
}

// TestHostCLIFromDesktopApp 锁住桌面应用带的 CLI：算「只有 CLI、缺适配器」，按版本目录取最高的，并记下来自哪个应用。
func TestHostCLIFromDesktopApp(t *testing.T) {
	roaming := t.TempDir()
	base := filepath.Join(roaming, "Claude", "claude-code")
	fakeBin(t, filepath.Join(base, "2.1.9"), "claude")
	want := fakeBin(t, filepath.Join(base, "2.1.10"), "claude")
	if err := os.MkdirAll(filepath.Join(base, "3.0.0"), 0o755); err != nil { // 空目录，里面没有命令
		t.Fatal(err)
	}
	f, runs := hostFixture(t, map[string]string{"APPDATA": roaming})
	e := f.setHost(t, "claude", "CLAUDE_CODE_EXECUTABLE", HostSpot{Kind: hostCLI, Dir: "%APPDATA%/Claude/claude-code/*", App: "Claude"})
	d := f.m.detectFast(e)
	if runs.Load() != 0 {
		t.Fatalf("detection ran %d commands", runs.Load())
	}
	if d.CLIPath != want || d.SystemPath != "" || d.Host == nil || d.Host.Kind != hostCLI || d.Host.App != "Claude" {
		t.Fatalf("detection %+v host %+v", d, d.Host)
	}
	f.put("claude", d)
	v, _ := f.m.Get("claude")
	if !v.CLIOnly || v.Install.Kind != "none" || v.Host == nil || v.Host.Path != want {
		t.Fatalf("view cliOnly %v install %+v host %+v", v.CLIOnly, v.Install, v.Host)
	}
	if v.Config.HostCLI == nil || v.Config.HostCLI.Path != want || !v.Config.HostCLI.On {
		t.Fatalf("host cli option %+v", v.Config.HostCLI)
	}
}

// TestPathWinsOverHostSpots 锁住 PATH 上找得到时不看别处，也不标成是在别处找到的；变量没设的位置直接跳过。
func TestPathWinsOverHostSpots(t *testing.T) {
	local := t.TempDir()
	fakeBin(t, filepath.Join(local, "cursor-agent"), "cursor-agent")
	f, _ := hostFixture(t, map[string]string{"LOCALAPPDATA": local})
	f.m.opts.LookPath = func(name string) (string, error) {
		if name == "cursor-agent" {
			return "/usr/bin/cursor-agent", nil
		}
		return "", errNotFound
	}
	e := f.setHost(t, "cursor", "", HostSpot{Kind: hostACP, Dir: "%LOCALAPPDATA%/cursor-agent"})
	if d := f.m.detectFast(e); d.SystemPath != "/usr/bin/cursor-agent" || d.Host != nil {
		t.Fatalf("PATH did not win: %+v host %+v", d, d.Host)
	}

	g, _ := hostFixture(t, map[string]string{})
	e = g.setHost(t, "cursor", "", HostSpot{Kind: hostACP, Dir: "%LOCALAPPDATA%/cursor-agent"}, HostSpot{Kind: hostApp, File: "%LOCALAPPDATA%/Programs/cursor/Cursor.exe"})
	if d := g.m.detectFast(e); d.SystemPath != "" || d.Host != nil {
		t.Fatalf("a spot with an unset variable matched: %+v host %+v", d, d.Host)
	}
}

// TestAdapterGetsTheHostCLI 锁住适配器启动时拿到本机 CLI 的路径；关掉开关、人自己写了同名变量、文件没了，这三种都不交。
func TestAdapterGetsTheHostCLI(t *testing.T) {
	const name = "CLAUDE_CODE_EXECUTABLE"
	roaming := t.TempDir()
	cli := fakeBin(t, filepath.Join(roaming, "Claude", "claude-code", "2.1.10"), "claude")
	f, _ := hostFixture(t, map[string]string{"APPDATA": roaming})
	e := f.setHost(t, "claude", name, HostSpot{Kind: hostCLI, Dir: "%APPDATA%/Claude/claude-code/*", App: "Claude"})
	adapter := fakeBin(t, filepath.Join(f.m.opts.Root, "claude", "pkg"), "claude-agent-acp")
	if err := f.m.writeInstall("claude", &InstallRecord{Kind: "binary", Version: "1", Command: adapter, Dir: filepath.Dir(adapter), Verified: true}); err != nil {
		t.Fatal(err)
	}
	d := f.m.detectFast(e)
	if d.Managed == nil || d.CLIPath != cli {
		t.Fatalf("detection %+v", d)
	}
	f.put("claude", d)
	envOf := func() string {
		t.Helper()
		l, _, err := f.m.launch("claude")
		if err != nil {
			t.Fatal(err)
		}
		return getEnv(l.Env, name)
	}
	if got := envOf(); got != cli {
		t.Fatalf("adapter got %q, want the host CLI %q", got, cli)
	}

	// 关掉：用适配器自带的。
	off := false
	if _, err := f.m.Configure("claude", ConfigPatch{HostCLI: &off}); err != nil {
		t.Fatal(err)
	}
	if got := envOf(); got != "" {
		t.Fatalf("switched off, adapter still got %q", got)
	}
	if v, _ := f.m.Get("claude"); v.Config.HostCLI == nil || v.Config.HostCLI.On {
		t.Fatalf("option after switching off: %+v", v.Config.HostCLI)
	}

	// 打开，但人在环境变量里自己写了一个：用人写的。
	on := true
	own := map[string]string{name: "/opt/my/claude"}
	if _, err := f.m.Configure("claude", ConfigPatch{HostCLI: &on, Env: &own}); err != nil {
		t.Fatal(err)
	}
	if got := envOf(); got != "/opt/my/claude" {
		t.Fatalf("the person's own value lost: %q", got)
	}

	// 人写的拿掉，本机 CLI 的文件也没了：不交，适配器退回自带的。
	none := map[string]string{}
	if _, err := f.m.Configure("claude", ConfigPatch{Env: &none}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cli); err != nil {
		t.Fatal(err)
	}
	if got := envOf(); got != "" {
		t.Fatalf("a missing CLI was still handed over: %q", got)
	}
}

// TestCuratedHostSpotsAreWellFormed 锁住精选表里写的位置都合规矩：种类认得、该写目录的写目录、该写文件的写文件，
// 用到的变量是这个系统上真有的那几个，写了 app 的才给应用名；也锁住 Windows 上常见目录里有 ~/.local/bin。
func TestCuratedHostSpotsAreWellFormed(t *testing.T) {
	curated, err := parseCurated(curatedJSON)
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string][]string{
		"windows": {"APPDATA", "LOCALAPPDATA", "USERPROFILE", "ProgramFiles", "ProgramFiles(x86)"},
		"darwin":  {"HOME"},
		"linux":   {"HOME"},
	}
	spots := 0
	for _, c := range curated {
		for goos, allowed := range vars {
			for _, s := range c.Host.Spots(goos) {
				spots++
				where := c.ID + "/" + goos
				switch s.Kind {
				case hostACP, hostCLI:
					if s.Dir == "" || s.File != "" {
						t.Errorf("%s: %s spot needs dir only: %+v", where, s.Kind, s)
					}
				case hostApp:
					if s.File == "" || s.Dir != "" || s.App == "" {
						t.Errorf("%s: app spot needs file and app: %+v", where, s)
					}
				default:
					t.Errorf("%s: unknown kind %q", where, s.Kind)
				}
				if s.Kind == hostCLI && len(c.CLIBins) == 0 || s.Kind == hostACP && len(c.ACPBins) == 0 {
					t.Errorf("%s: %s spot but no command names", where, s.Kind)
				}
				for _, tok := range reHostVar.FindAllString(s.Dir+s.File, -1) {
					if !slices.Contains(allowed, strings.Trim(tok, "%${}")) {
						t.Errorf("%s: variable %s is not one this system sets", where, tok)
					}
				}
				if goos == "windows" && strings.HasPrefix(s.Dir+s.File, "~") {
					t.Errorf("%s: use %%USERPROFILE%% on Windows: %+v", where, s)
				}
			}
		}
		if c.CLIEnv != "" && len(c.CLIBins) == 0 {
			t.Errorf("%s: cliEnv without cliBins", c.ID)
		}
	}
	if spots == 0 {
		t.Fatal("curated.json has no host spots")
	}
	p := &PathFinder{goos: "windows", env: func(name string) string {
		return map[string]string{"APPDATA": `C:\Users\me\AppData\Roaming`, "USERPROFILE": `C:\Users\me`, "LOCALAPPDATA": `C:\Users\me\AppData\Local`}[name]
	}}
	dirs := p.commonDirs()
	for _, want := range []string{filepath.Join(`C:\Users\me\AppData\Roaming`, "npm"), filepath.Join(`C:\Users\me`, ".local", "bin")} {
		if !slices.Contains(dirs, want) {
			t.Errorf("windows common dirs lack %s: %v", want, dirs)
		}
	}
}
