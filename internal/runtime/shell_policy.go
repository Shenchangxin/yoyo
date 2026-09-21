package runtime

import (
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// ShellDenied is a deterministic policy jail, not an OS sandbox. Seatbelt /
// bubblewrap are OS-specific; lying about them on Windows is worse than a
// named deny-list plus the existing workspace path jail.
func ShellDenied(command, workspace string, networkAllow []string, extraRoots ...string) error {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return fmt.Errorf("empty command")
	}
	lower := strings.ToLower(cmd)
	for _, pat := range destructiveShell {
		if strings.Contains(lower, pat) {
			return fmt.Errorf("shell policy denied destructive command")
		}
	}
	if looksLikeRemoteNetwork(lower) && !networkPermitted(networkAllow) {
		return fmt.Errorf("shell policy denied network; set policy.network_allow or use MCP")
	}
	if err := denyEscapingAbsPaths(cmd, workspace, extraRoots...); err != nil {
		return err
	}
	if argv := SplitShellArgv(cmd); len(argv) > 0 {
		if err := DenyArgvPaths(argv, workspace, extraRoots...); err != nil {
			return err
		}
	}
	return nil
}

func SplitShellArgv(command string) []string {
	var out []string
	var cur strings.Builder
	quote := rune(0)
	for _, r := range strings.TrimSpace(command) {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t':
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func DenyArgvPaths(argv []string, workspace string, extraRoots ...string) error {
	if workspace == "" && len(extraRoots) == 0 {
		return nil
	}
	roots := append([]string{workspace}, extraRoots...)
	for _, f := range argv {
		f = strings.Trim(f, `"'`)
		if capability.LooksLikeWindowsSwitch(f) {
			continue
		}
		f = capability.CanonicalizeToolPath(f)
		if !filepath.IsAbs(f) {
			continue
		}
		if !pathInRoots(f, roots) {
			return fmt.Errorf("shell policy denied path outside workspace")
		}
	}
	return nil
}

func DenyHomePlanes(argv []string, home string) error {
	if home == "" {
		return nil
	}
	for _, f := range argv {
		f = strings.Trim(f, `"'`)
		if capability.LooksLikeWindowsSwitch(f) {
			continue
		}
		f = capability.CanonicalizeToolPath(f)
		if !filepath.IsAbs(f) {
			f = filepath.Join(home, f)
		}
		if capability.ForbiddenDiagPath(home, f) {
			return fmt.Errorf("shell policy denied diagnostic plane")
		}
	}
	return nil
}

func looksSimpleArgv(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	low := strings.ToLower(argv[0])
	if low == "cmd" || low == "cmd.exe" || low == "powershell" || low == "pwsh" || low == "sh" || low == "bash" {
		return false
	}
	return !strings.ContainsAny(argv[0], `&|;<>`)
}

func shellNeedsWrapper(command string, argv []string) bool {
	if len(argv) == 0 {
		return true
	}
	if strings.ContainsAny(command, "&|;<>\n") {
		return true
	}
	if isShellBuiltin(argv[0]) {
		return true
	}
	return !looksSimpleArgv(argv)
}

func looksPosixUnix(command string) bool {
	if strings.Contains(command, "/dev/null") || strings.Contains(command, "$(") {
		return true
	}
	for _, stage := range splitShellStages(command) {
		argv := SplitShellArgv(stage)
		if len(argv) == 0 {
			continue
		}
		name := strings.ToLower(filepath.Base(argv[0]))
		if name == "find" && isWindowsFindStage(argv) {
			continue
		}
		if isUnixUtil(name) {
			return true
		}
	}
	return false
}

func isUnixUtil(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "ls", "cat", "head", "tail", "find", "pwd", "which", "true", "false",
		"chmod", "chown", "rm", "cp", "mv", "touch", "wc", "tr", "cut", "uniq",
		"xargs", "tee", "basename", "dirname", "realpath", "readlink", "uname",
		"sed", "awk", "test":
		return true
	default:
		return false
	}
}

func isWindowsFindStage(argv []string) bool {
	for _, a := range argv[1:] {
		al := strings.ToLower(strings.TrimSpace(a))
		if al == "/c" || al == "/v" || al == "/n" || al == "/i" || strings.HasPrefix(al, "/off") {
			return true
		}
		if strings.HasPrefix(a, "-") {
			return false
		}
	}
	return false
}

func splitShellStages(command string) []string {
	var out []string
	var cur strings.Builder
	quote := byte(0)
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for i := 0; i < len(command); i++ {
		c := command[i]
		if quote != 0 {
			cur.WriteByte(c)
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			cur.WriteByte(c)
			continue
		}
		if (c == '&' || c == '|') && i+1 < len(command) && command[i+1] == c {
			flush()
			i++
			continue
		}
		if c == '|' || c == ';' || c == '\n' {
			flush()
			continue
		}
		cur.WriteByte(c)
	}
	flush()
	return out
}

func isShellBuiltin(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "cd", "chdir", "set", "export", "unset", "alias", "source", ".",
		"echo", "dir", "copy", "del", "erase", "md", "mkdir", "rd", "rmdir",
		"type", "call", "start", "move", "ren", "rename", "pushd", "popd",
		"cls", "ver", "path", "prompt", "assoc", "ftype", "exit",
		"if", "for", "goto", "shift", "setlocal", "endlocal",
		"eval", "readonly", "wait", "umask", "ulimit", "hash", "command", "builtin":
		return true
	default:
		return false
	}
}

var destructiveShell = []string{
	"rm -rf /", "rm -rf /*", "format c:", "format d:", "mkfs", "diskpart",
	"shutdown", "reboot", "bcdedit", "reg delete",
	":(){:|:&};:", "fork bomb",
	"remove-item -recurse c:\\", "del /s /q c:\\",
}

var networkCmd = regexp.MustCompile(`\b(curl|wget|nc |ncat|ssh |scp |sftp |invoke-webrequest|iwr |start-bitstransfer|certutil\s+-urlcache)\b`)

func looksLikeRemoteNetwork(lower string) bool {
	if !networkCmd.MatchString(lower) {
		return false
	}
	return !networkTargetsAreLoopback(lower)
}

var (
	reURLHost      = regexp.MustCompile(`(?i)https?://(\[[0-9a-f:]+\]|[^/\s:?#]+)`)
	reBareLoopback = regexp.MustCompile(`(?i)\b(localhost|127\.\d+\.\d+\.\d+|0\.0\.0\.0|\[::1\]|::1)(?::\d+)?\b`)
	reBareDNS      = regexp.MustCompile(`(?i)\b(?:https?://)?((?:[a-z0-9-]+\.)+[a-z]{2,})(?::\d+)?\b`)
)

func networkTargetsAreLoopback(lower string) bool {
	hosts := collectNetworkHosts(lower)
	if len(hosts) == 0 {
		return false
	}
	for _, h := range hosts {
		if !isLoopbackHost(h) {
			return false
		}
	}
	return true
}

func collectNetworkHosts(lower string) []string {
	var out []string
	add := func(h string) {
		h = strings.TrimSpace(h)
		if h == "" {
			return
		}
		out = append(out, h)
	}
	for _, m := range reURLHost.FindAllStringSubmatch(lower, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range reBareLoopback.FindAllStringSubmatch(lower, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range reBareDNS.FindAllStringSubmatch(lower, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	return out
}

func isLoopbackHost(h string) bool {
	h = strings.TrimSpace(h)
	h = strings.Trim(h, "[]")
	h = strings.ToLower(h)
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
		h = strings.Trim(h, "[]")
	}
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified()
}

func networkPermitted(allow []string) bool {
	for _, a := range allow {
		if a == "*" {
			return true
		}
		if strings.TrimSpace(a) != "" {
			return true
		}
	}
	return false
}

var absPath = regexp.MustCompile(`(?i)(?:[a-z]:\\|/)(?:windows|etc|usr|system32)[\\/]`)

func denyEscapingAbsPaths(command, workspace string, extraRoots ...string) error {
	if workspace == "" && len(extraRoots) == 0 {
		return nil
	}
	if !absPath.MatchString(command) {
		return nil
	}
	roots := append([]string{workspace}, extraRoots...)
	fields := strings.Fields(command)
	for _, f := range fields {
		f = strings.Trim(f, `"'`)
		if capability.LooksLikeWindowsSwitch(f) {
			continue
		}
		f = capability.CanonicalizeToolPath(f)
		if !filepath.IsAbs(f) {
			continue
		}
		if !pathInRoots(f, roots) {
			return fmt.Errorf("shell policy denied path outside workspace")
		}
	}
	return nil
}

func pathInRoots(p string, roots []string) bool {
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		if capability.WithinWorkspace(root, p) {
			return true
		}
	}
	return false
}
