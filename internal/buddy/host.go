// host.go 拥有 PATH 之外的本机检测：厂商安装器和桌面应用常把命令放在不进 PATH 的地方，精选表按系统列出这些位置，
// 这里把位置里的变量展开，看文件在不在。只看文件和目录，不运行任何东西，也不读凭证文件。
// 找到之后算什么由 detect.go 决定：能直接连的命令当系统版，只是 CLI 的标成缺适配器，只有桌面应用的只记下来给人看。

package buddy

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// HostSpot 的三种 Kind。
const (
	hostACP = "acp"
	hostCLI = "cli"
	hostApp = "app"
)

// hostHit 是在 PATH 之外找到的东西：哪一类、在哪、来自哪个应用。
type hostHit struct {
	Kind string
	Path string
	App  string
}

// reHostVar 匹配位置里的变量：%NAME%、$NAME 和 ${NAME}。Windows 的 ProgramFiles(x86) 名字里带括号。
var reHostVar = regexp.MustCompile(`%[A-Za-z_][A-Za-z0-9_()]*%|\$\{[A-Za-z_][A-Za-z0-9_]*\}|\$[A-Za-z_][A-Za-z0-9_]*`)

// reDigits 匹配一串数字，比版本号时用。
var reDigits = regexp.MustCompile(`\d+`)

// expandHost 把 pattern 里的变量和开头的 ~ 换成 env 给的值。有一个变量是空的就返回空串：
// 少了变量的路径会指到别处去，宁可不找。
func expandHost(pattern string, env func(string) string, goos string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || env == nil {
		return ""
	}
	missing := false
	out := reHostVar.ReplaceAllStringFunc(pattern, func(tok string) string {
		v := env(strings.Trim(tok, "%${}"))
		if v == "" {
			missing = true
		}
		return v
	})
	if missing {
		return ""
	}
	if out == "~" || strings.HasPrefix(out, "~/") || strings.HasPrefix(out, `~\`) {
		home := env("HOME")
		if goos == "windows" {
			home = firstNonEmpty(env("USERPROFILE"), home)
		}
		if home == "" {
			return ""
		}
		out = home + out[1:]
	}
	return filepath.FromSlash(out)
}

// hostPaths 把一处位置展开成路径。带 * 的按通配找出存在的那些，版本高的排前面；不带的原样返回一条，在不在由调用方看。
func hostPaths(pattern string, env func(string) string, goos string) []string {
	p := expandHost(pattern, env, goos)
	if p == "" {
		return nil
	}
	if !strings.Contains(p, "*") {
		return []string{p}
	}
	matches, err := filepath.Glob(p)
	if err != nil {
		return nil
	}
	sort.SliceStable(matches, func(i, j int) bool { return versionLess(matches[j], matches[i]) })
	return matches
}

// versionLess 按路径里的数字比大小：1.9.0 排在 1.10.0 前面。数字都一样时按字面比。
func versionLess(a, b string) bool {
	x, y := reDigits.FindAllString(a, -1), reDigits.FindAllString(b, -1)
	for i := 0; i < len(x) && i < len(y); i++ {
		m, _ := strconv.ParseUint(x[i], 10, 64)
		n, _ := strconv.ParseUint(y[i], 10, 64)
		if m != n {
			return m < n
		}
	}
	if len(x) != len(y) {
		return len(x) < len(y)
	}
	return a < b
}

// hostFind 在 e 的常见位置里找 kind 这一类，按精选表的顺序，返回第一处找到的。
// acp 和 cli 在目录里找对应的命令名，Windows 上按 PATHEXT 补扩展名；app 只看标记文件或 .app 目录在不在。
func (m *Manager) hostFind(e Entry, kind string) (hostHit, bool) {
	env := m.paths.env
	goos := m.plat.GOOS
	for _, spot := range e.Host.Spots(goos) {
		if spot.Kind != kind {
			continue
		}
		if kind == hostApp {
			for _, p := range hostPaths(spot.File, env, goos) {
				if _, err := os.Stat(p); err == nil {
					return hostHit{Kind: kind, Path: p, App: spot.App}, true
				}
			}
			continue
		}
		bins := e.ACPBins
		if kind == hostCLI {
			bins = e.CLIBins
		}
		for _, dir := range hostPaths(spot.Dir, env, goos) {
			for _, bin := range bins {
				// 命令自己就会说 ACP 的，不算「只有 CLI」，和 PATH 上的规则一样。
				if kind == hostCLI && slices.Contains(e.ACPBins, bin) {
					continue
				}
				for _, candidate := range m.paths.candidates(filepath.Join(dir, bin)) {
					if m.paths.runnable(candidate) {
						return hostHit{Kind: kind, Path: candidate, App: spot.App}, true
					}
				}
			}
		}
	}
	return hostHit{}, false
}
