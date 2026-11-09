// install_npm.go 拥有 npm 分发的托管安装：用托管 Node 自带的 npm-cli.js 装进 buddies/<id>/pkg-<版本>，不走任何 .cmd，
// 装完读包的 bin 写 install.json（node 加 JS 入口）。npm 源为 auto 时并发问 npmjs 和 npmmirror，取先回的那个，结果缓存 24 小时。

package buddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// npm 源的固定地址。
const (
	npmjsURL     = "https://registry.npmjs.org/"
	npmmirrorURL = "https://registry.npmmirror.com/"
)

// npmRegistries 返回可选的 npm 源，测试可以覆盖。
func (m *Manager) npmRegistries() map[string]string {
	if m.opts.NPMRegistries != nil {
		return m.opts.NPMRegistries
	}
	return map[string]string{"npmjs": npmjsURL, "npmmirror": npmmirrorURL}
}

// npmRegistryFor 选 npm 源：偏好里写死的直接用；auto 时并发查这个版本的元数据，各 3 秒，取先成功的。都失败时用 npmjs。
func (m *Manager) npmRegistryFor(ctx context.Context, name, version string) string {
	regs := m.npmRegistries()
	pref := strings.TrimSpace(m.opts.Settings().NPMRegistry)
	switch {
	case pref == "npmjs" || pref == "npmmirror":
		return regs[pref]
	case strings.HasPrefix(pref, "https://"):
		return withSlash(pref)
	}
	m.mu.Lock()
	if m.npmPick != "" && m.opts.Now().Sub(m.npmAt) < 24*time.Hour {
		pick := m.npmPick
		m.mu.Unlock()
		if m.npmHasVersion(ctx, pick, name, version) {
			return pick
		}
		return regs["npmjs"]
	}
	m.mu.Unlock()
	type answer struct{ url string }
	ch := make(chan answer, 2)
	keys := []string{"npmmirror", "npmjs"}
	for _, key := range keys {
		go func(u string) {
			if m.npmHasVersion(ctx, u, name, version) {
				ch <- answer{u}
				return
			}
			ch <- answer{}
		}(regs[key])
	}
	pick := ""
	for range keys {
		if a := <-ch; a.url != "" {
			pick = a.url
			break
		}
	}
	if pick == "" {
		return regs["npmjs"]
	}
	m.mu.Lock()
	m.npmPick, m.npmAt = pick, m.opts.Now()
	m.mu.Unlock()
	return pick
}

// npmHasVersion 问 registry 有没有 name 的 version，3 秒超时。只取很小的 dist-tags：某个标签就是这个版本，
// 或者它不比 latest 新，就算有（镜像只会落后于新版本）。npmmirror 不支持按版本查元数据，所以不用那个接口。
func (m *Manager) npmHasVersion(ctx context.Context, registry, name, version string) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	segments := strings.Split(name, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	target := withSlash(registry) + "-/package/" + strings.Join(segments, "/") + "/dist-tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false
	}
	resp, err := m.opts.HTTP.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var tags map[string]string
	if json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&tags) != nil || tags["latest"] == "" {
		return false
	}
	if version == "" || version == "latest" {
		return true
	}
	for _, v := range tags {
		if v == version {
			return true
		}
	}
	return versionAtMost(version, tags["latest"])
}

// versionAtMost 判断 v 的主、次、修订号不大于 limit 的。认不出数字时按不满足处理。
func versionAtMost(v, limit string) bool {
	a, okA := versionCore(v)
	b, okB := versionCore(limit)
	if !okA || !okB {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return true
}

// versionCore 取出 x.y.z 三个数，去掉前导 v、预发布和构建后缀。
func versionCore(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// withSlash 给地址补上结尾的斜杠。
func withSlash(u string) string {
	if strings.HasSuffix(u, "/") {
		return u
	}
	return u + "/"
}

// npmInstallArgs 拼出 npm install 的参数。
func npmInstallArgs(npmCLI, prefix, registry, spec string) []string {
	return []string{npmCLI, "install", "--prefix", prefix, "--no-audit", "--no-fund", "--loglevel=error", "--registry", registry, spec}
}

// npmEnv 是跑 npm 的环境：清洗后的白名单，PATH 最前面放 Node 的目录，缓存放在 buddies/.npm-cache。
func (m *Manager) npmEnv(nodeDir string) []string {
	env := m.baseEnv([]string{nodeDir})
	env = setEnv(env, "npm_config_cache", filepath.Join(m.opts.Root, ".npm-cache"))
	env = setEnv(env, "npm_config_update_notifier", "false")
	return env
}

// installNPM 装一个 npm 包。镜像上失败时换 npmjs 再试一次；失败时删掉半成品目录。
func (m *Manager) installNPM(ctx context.Context, e Entry, pkg PackageTarget, prog func(Progress), log io.Writer) (*InstallRecord, error) {
	name, version := splitPackage(pkg.Package)
	if version == "" {
		version = e.Version
	}
	node, npmCLI, err := m.ensureNode(ctx, prog, log)
	if err != nil {
		return nil, err
	}
	prog(Progress{Phase: "deps", Percent: 40})
	registry := m.npmRegistryFor(ctx, name, version)
	dir := filepath.Join(m.installDir(e.ID), "pkg-"+safeName(version))
	spec := name
	if version != "" {
		spec += "@" + version
	}
	run := func(reg string) error {
		_ = os.RemoveAll(dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		fmt.Fprintf(log, "npm install %s --registry %s\n", spec, reg)
		l := acp.Launch{Command: node, Args: npmInstallArgs(npmCLI, dir, reg, spec), Env: m.npmEnv(filepath.Dir(node)), Dir: dir}
		_, err := m.opts.Run(ctx, Cmd{Launch: l, Output: log})
		return err
	}
	err = run(registry)
	if err != nil && ctx.Err() == nil && registry != m.npmRegistries()["npmjs"] {
		fmt.Fprintln(log, "retrying with npmjs")
		err = run(m.npmRegistries()["npmjs"])
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("npm install: %w", err)
	}
	rec, err := npmRecord(e, pkg, name, version, dir, node, m.plat.GOOS)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return rec, nil
}

// npmRecord 读装好的包的 bin，决定怎么启动：node 脚本用 node 加入口；别的脚本走 .bin 里 npm 生成的启动文件。
func npmRecord(e Entry, pkg PackageTarget, name, version, dir, node, goos string) (*InstallRecord, error) {
	pkgDir := filepath.Join(dir, "node_modules", filepath.FromSlash(name))
	raw, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("npm install: package.json missing: %w", err)
	}
	var manifest struct {
		Name string          `json:"name"`
		Bin  json.RawMessage `json:"bin"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	bins := map[string]string{}
	var single string
	if json.Unmarshal(manifest.Bin, &single) == nil && single != "" {
		base := name
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		bins[base] = single
	} else {
		_ = json.Unmarshal(manifest.Bin, &bins)
	}
	binName := pickBin(bins, e.ACPBins, name)
	if binName == "" {
		return nil, errors.New("npm install: the package has no executable")
	}
	entry := filepath.Join(pkgDir, filepath.FromSlash(strings.TrimPrefix(bins[binName], "./")))
	rec := &InstallRecord{Kind: "npm", Version: version, Package: name, Env: pkg.Env, Runtime: "node", Dir: dir}
	head := make([]byte, 256)
	if f, err := os.Open(entry); err == nil {
		n, _ := io.ReadFull(f, head)
		head = head[:n]
		_ = f.Close()
	} else {
		return nil, fmt.Errorf("npm install: entry missing: %w", err)
	}
	if nodeShebang(head) {
		rec.Command = node
		rec.Entry = entry
		rec.Args = append([]string{entry}, pkg.Args...)
		return rec, nil
	}
	shim := filepath.Join(dir, "node_modules", ".bin", binName)
	if goos == "windows" {
		rec.Command = shim + ".cmd"
		rec.ViaCmd = true
	} else {
		rec.Command = shim
	}
	rec.Args = append([]string(nil), pkg.Args...)
	return rec, nil
}

// pickBin 选要启动的那个 bin：先按精选表的 ACP 命令名，其次和包名同名的，最后按名字排第一个。
func pickBin(bins map[string]string, prefer []string, pkg string) string {
	for _, name := range prefer {
		if _, ok := bins[name]; ok {
			return name
		}
	}
	base := pkg
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if _, ok := bins[base]; ok {
		return base
	}
	keys := make([]string, 0, len(bins))
	for k := range bins {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

// safeName 把版本号变成能做目录名的样子。
func safeName(s string) string {
	if s == "" {
		return "latest"
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || slices.Contains([]rune{'.', '-', '_'}, r) {
			return r
		}
		return '_'
	}, s)
}
