// Git Bash 盘符翻译、cmd 开关识别，以及 YOYO_HOME 下诊断目录和 vault.json 的拒绝。
// 不做 Abs 再 Rel 的工作区判断，那在 broker.go 的 WithinWorkspace。runtime 的工具、shell 策略，以及 WithinWorkspace 会调用这里。

package capability

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// msysDrivePath 匹配 Git Bash、MSYS、Cygwin 的盘符路径：/c/Users/foo 或 /cygdrive/c/Users/foo。
// 盘符只认一个字母，且后面必须有斜杠，因此 /dev/null 和 /tmp 不会变成 D:\ev\null 这类路径。
var msysDrivePath = regexp.MustCompile(`(?i)^(?:/cygdrive)?/+([a-z])/(.*)$`)

// MSYSToWindowsPath 把 POSIX 盘符路径译成 Windows 路径，不看 GOOS，由调用方决定何时使用。
// 先去掉首尾空白，把反斜杠换成斜杠。/c/Users/foo 得到 C:\Users\foo；/cygdrive/c/Users/foo 同样处理，盘符改为大写，剩余斜杠换成反斜杠。
// 盘符后没有剩余路径时得到 C:\。/dev/null、/tmp 和 //F 匹配失败，返回空串和 false。//F 在字母后面没有斜杠，不会被当成盘符。
func MSYSToWindowsPath(p string) (string, bool) {
	s := strings.TrimSpace(p)
	if s == "" {
		return "", false
	}
	s = strings.ReplaceAll(s, `\`, `/`)
	m := msysDrivePath.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	drive := strings.ToUpper(m[1])
	rest := strings.TrimPrefix(m[2], "/")
	if rest == "" {
		return drive + `:\`, true
	}
	return drive + `:\` + strings.ReplaceAll(rest, `/`, `\`), true
}

// CanonicalizeToolPath 去掉首尾空白。GOOS 为 windows 时，若 MSYSToWindowsPath 匹配成功，把 Git Bash 的 /c/Users/foo 收成 C:\Users\foo：盘符大写，其余斜杠换成反斜杠，/cygdrive/c/... 同样处理。这样后续的路径判断和 Join 不会把 /c/Users 当成相对片段。
// //F 这类开关在盘符字母后面没有斜杠，MSYSToWindowsPath 匹配失败，本函数原样返回，不把它收成盘符路径。开关由 LooksLikeWindowsSwitch 识别。非 Windows 只做 TrimSpace。
func CanonicalizeToolPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if runtime.GOOS == "windows" {
		if w, ok := MSYSToWindowsPath(p); ok {
			return w
		}
	}
	return p
}

// LooksLikeWindowsSwitch 判断是不是 cmd.exe 或 Git Bash 的开关，例如 //F、//IM、/C。它们在 Windows 上看似 UNC，在 Unix 上以斜杠开头，但不是文件系统路径。
// 去掉首尾空白和两端的单引号、双引号，把反斜杠换成斜杠。以 // 开头先去掉一个斜杠，于是 //F 按 /F 判断。
// 剩下的必须是一个斜杠加一段标记：标记里不能再有斜杠，长度不超过 16，首字符是字母，其余是字母或数字。冒号及其后的内容不参与判断。因此 /c/Users 不是开关。
func LooksLikeWindowsSwitch(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	if s == "" {
		return false
	}
	s = strings.ReplaceAll(s, `\`, `/`)
	// //F 先去掉一个斜杠，再按 /F 判断。
	if strings.HasPrefix(s, "//") {
		s = s[1:]
	}
	if !strings.HasPrefix(s, "/") {
		return false
	}
	body := s[1:]
	if i := strings.IndexByte(body, ':'); i >= 0 {
		body = body[:i]
	}
	if body == "" || strings.ContainsAny(body, `/`) {
		return false
	}
	if len(body) > 16 {
		return false
	}
	for i, r := range body {
		if i == 0 {
			if r < 'A' || (r > 'Z' && r < 'a') || r > 'z' {
				return false
			}
			continue
		}
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !ok {
			return false
		}
	}
	return true
}

// diagPlaneNames 是紧挨 YOYO_HOME 的拒绝目录：logs、journal、observe、pages、learn、profiles。
var diagPlaneNames = []string{"logs", "journal", "observe", "pages", "learn", "profiles"}

// ForbiddenDiagPath 判断 p 是不是 homeRoot 下不该被代理读取的位置。
// homeRoot 或 p 去掉首尾空白后为空，或 Abs 失败，则返回 false。两边先经 CanonicalizeToolPath 再 Abs。
// 与 homeRoot 下的 vault.json 相同则返回 true。logs、journal、observe、pages、learn、profiles 这些目录本身及其内部路径也返回 true，目录内部用 WithinWorkspace 判断。
func ForbiddenDiagPath(homeRoot, p string) bool {
	if strings.TrimSpace(homeRoot) == "" || strings.TrimSpace(p) == "" {
		return false
	}
	homeAbs, err := filepath.Abs(CanonicalizeToolPath(homeRoot))
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(CanonicalizeToolPath(p))
	if err != nil {
		return false
	}
	vault := filepath.Join(homeAbs, "vault.json")
	if samePath(pathAbs, vault) {
		return true
	}
	for _, name := range diagPlaneNames {
		root := filepath.Join(homeAbs, name)
		if samePath(pathAbs, root) || WithinWorkspace(root, pathAbs) {
			return true
		}
	}
	return false
}

// samePath 在两边都能 Stat 时用 os.SameFile，否则比较 filepath.Clean 之后的字符串。
// b 能 Stat 且 a 也能 Stat 时，和 a 的 FileInfo 一起交给 os.SameFile；否则用 filepath.Clean 之后的字符串和 a 比较。
func samePath(a, b string) bool {
	ai, err := os.Stat(a)
	if err == nil {
		if bi, err2 := os.Stat(b); err2 == nil {
			return os.SameFile(ai, bi)
		}
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
