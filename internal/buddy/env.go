// env.go 拥有给 agent 进程的环境变量清洗：只放行白名单，其余一律丢弃（包括 YOYO_* 和任何密钥类变量），再加上 TERM=dumb、NO_COLOR=1。
// 密钥和用户的非密钥变量由 launcher 在清洗之后加上；配置时拒收像密钥的变量名也在这里判断。

package buddy

import (
	"regexp"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// envAllowed 是原样放行的变量名。比较时不分大小写，代理变量的两种写法都能命中。
var envAllowed = map[string]bool{}

func init() {
	for _, name := range []string{
		"PATH", "HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "PROGRAMDATA", "SystemRoot", "windir", "ComSpec", "PATHEXT", "TEMP", "TMP", "TMPDIR",
		"SystemDrive", "HOMEDRIVE", "HOMEPATH", "ProgramFiles", "ProgramFiles(x86)", "ProgramW6432", "CommonProgramFiles", "PROCESSOR_ARCHITECTURE", "NUMBER_OF_PROCESSORS", "OS", "TZ",
		"USER", "USERNAME", "LOGNAME", "LANG", "LANGUAGE", "SHELL",
		"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY", "NODE_EXTRA_CA_CERTS", "SSL_CERT_FILE", "SSL_CERT_DIR",
	} {
		envAllowed[strings.ToUpper(name)] = true
	}
}

// reEnvName 是合法的变量名。
var reEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// BuildEnv 从 base 里只留下白名单变量，再设 TERM=dumb 和 NO_COLOR=1。
func BuildEnv(base []string) []string {
	var out []string
	for _, kv := range base {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		name := kv[:i]
		if envBlocked(name) || !envPassed(name) {
			continue
		}
		out = append(out, kv)
	}
	out = setEnv(out, "TERM", "dumb")
	return setEnv(out, "NO_COLOR", "1")
}

// envPassed 判断变量名在不在白名单里：精确名单，或 LC_、XDG_ 开头。
func envPassed(name string) bool {
	upper := strings.ToUpper(name)
	return envAllowed[upper] || strings.HasPrefix(upper, "LC_") || strings.HasPrefix(upper, "XDG_")
}

// envBlocked 判断变量名必须丢掉：YOYO_ 开头，或者像密钥。
func envBlocked(name string) bool {
	return strings.HasPrefix(strings.ToUpper(name), "YOYO_") || SecretLike(name)
}

// SecretLike 判断变量名像不像密钥：以 _KEY、_TOKEN、_SECRET 结尾，或含 PASSWORD。
func SecretLike(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	return strings.HasSuffix(upper, "_KEY") || strings.HasSuffix(upper, "_TOKEN") || strings.HasSuffix(upper, "_SECRET") ||
		strings.Contains(upper, "PASSWORD") || diaglog.LooksSecretKey(name)
}

// ValidEnvName 判断 name 能不能当环境变量名。
func ValidEnvName(name string) bool {
	return reEnvName.MatchString(name)
}

// setEnv 设置或替换一个变量，名字比较不分大小写（Windows 的规矩，其他平台也不会有只差大小写的两个同名变量要传）。
func setEnv(env []string, name, value string) []string {
	for i, kv := range env {
		if j := strings.IndexByte(kv, '='); j > 0 && strings.EqualFold(kv[:j], name) {
			env[i] = kv[:j] + "=" + value
			return env
		}
	}
	return append(env, name+"="+value)
}

// withoutEnv 去掉 names 里的变量，给人交互用的终端不要 TERM=dumb 和 NO_COLOR。
func withoutEnv(env []string, names ...string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		drop := false
		if j := strings.IndexByte(kv, '='); j > 0 {
			for _, name := range names {
				if strings.EqualFold(kv[:j], name) {
					drop = true
					break
				}
			}
		}
		if !drop {
			out = append(out, kv)
		}
	}
	return out
}

// getEnv 按不分大小写的名字取值。
func getEnv(env []string, name string) string {
	for _, kv := range env {
		if j := strings.IndexByte(kv, '='); j > 0 && strings.EqualFold(kv[:j], name) {
			return kv[j+1:]
		}
	}
	return ""
}
