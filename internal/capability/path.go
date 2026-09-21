package capability

import (
	"regexp"
	"runtime"
	"strings"
)

// msysDrivePath is Git-Bash / MSYS / Cygwin: /c/Users/foo or /cygdrive/c/Users/foo.
// One-letter first segment only — /dev/null and /tmp must not become D:\ev\null.
var msysDrivePath = regexp.MustCompile(`(?i)^(?:/cygdrive)?/+([a-z])/(.*)$`)

// MSYSToWindowsPath translates a POSIX drive path to a Windows path.
// It does not consult GOOS; callers decide when to apply it.
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

// CanonicalizeToolPath rewrites Git-Bash drive paths on Windows so jail
// checks and Join do not treat /c/Users/... as a relative segment.
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

// LooksLikeWindowsSwitch reports cmd.exe / Git-Bash doubled flags such as
// //F, //IM, /C. These are absolute-looking on both Windows (UNC) and Unix
// (leading slash) but are not filesystem paths.
func LooksLikeWindowsSwitch(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	if s == "" {
		return false
	}
	s = strings.ReplaceAll(s, `\`, `/`)
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
