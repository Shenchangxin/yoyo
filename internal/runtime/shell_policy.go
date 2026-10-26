// shell 命令的拒绝名单：破坏性子串、远程网络，以及工作区外的绝对路径。
// 这是确定性策略，不是操作系统沙箱。本文件没有 Windows Job Object，也不调用 Seatbelt 或 bubblewrap。
// 不调用 WorkspaceTools.resolve。相对路径不拼接工作区、不拒绝。tools.go 的 shell 在执行前调用这里。

package runtime

import (
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// ShellDenied 是确定性的命令拒绝名单，不是操作系统沙箱。Seatbelt 和 bubblewrap 取决于操作系统；在 Windows 上把它们说成已经启用，比具名拒绝名单加上工作区路径限制更糟。
// 空命令直接拒绝。破坏性子串按小写包含匹配。远程网络在 networkAllow 未放行时拒绝。绝对路径再经 denyEscapingAbsPaths 和 DenyArgvPaths。
// command 去掉首尾空白，空串直接拒绝；小写后做破坏性子串和远程网络判断，再交给 denyEscapingAbsPaths 和 SplitShellArgv。
// workspace 交给 denyEscapingAbsPaths 和 DenyArgvPaths。
// extraRoots 交给 denyEscapingAbsPaths 和 DenyArgvPaths。
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

// SplitShellArgv 按空白切参数。单引号和双引号成对包裹，引号本身不进入参数。不处理反斜杠转义，也不在分号或管道处切开。
// command 先去掉首尾空白，再按空白切开；单引号和双引号成对包裹，引号本身不进入参数。
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

// DenyArgvPaths 对每个参数调用 denyShellAbsPath。workspace 和 extraRoots 都空时直接放行。
// argv 逐项交给 denyShellAbsPath。
func DenyArgvPaths(argv []string, workspace string, extraRoots ...string) error {
	if workspace == "" && len(extraRoots) == 0 {
		return nil
	}
	roots := append([]string{workspace}, extraRoots...)
	for _, f := range argv {
		if err := denyShellAbsPath(f, roots); err != nil {
			return err
		}
	}
	return nil
}

// DenyHomePlanes 拒绝指向 home 下诊断目录或 vault.json 的参数。home 为空则放行。Windows 开关跳过。相对路径先拼到 home 上，不调用 WorkspaceTools.resolve。
// argv 逐项去掉引号；像 Windows 开关的跳过，相对路径先拼到 home 上再做 ForbiddenDiagPath。
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

// looksLikePowerShell 在命令含常见 cmdlet、$_、以 powershell 或 pwsh 开头，或以 $ 开头且不含 $( 时返回真。
// command 转成小写后看有没有常见 cmdlet；也看原文里的 $_、argv 程序名，以及去掉空白后是否以 $ 开头且不含 $(。
func looksLikePowerShell(command string) bool {
	lower := strings.ToLower(command)
	switch {
	case strings.Contains(lower, "invoke-webrequest"),
		strings.Contains(lower, "start-process"),
		strings.Contains(lower, "get-content"),
		strings.Contains(lower, "get-process"),
		strings.Contains(lower, "get-ciminstance"),
		strings.Contains(lower, "foreach-object"),
		strings.Contains(lower, "select-object"),
		strings.Contains(lower, "where-object"),
		strings.Contains(lower, "convertto-json"),
		strings.Contains(lower, "out-file"),
		strings.Contains(lower, "set-content"),
		strings.Contains(lower, "new-object"):
		return true
	}
	if strings.Contains(command, "$_") {
		return true
	}
	argv := SplitShellArgv(command)
	if len(argv) > 0 {
		name := strings.ToLower(filepath.Base(argv[0]))
		name = strings.TrimSuffix(name, ".exe")
		if name == "powershell" || name == "pwsh" {
			return true
		}
	}
	trim := strings.TrimSpace(command)
	return strings.HasPrefix(trim, "$") && !strings.Contains(trim, "$(")
}

// powershellCommandArg 取出 powershell 或 pwsh 的 -Command、-c、/command 或 /c 后面的正文。不是这类包装时返回原命令。
func powershellCommandArg(command string) string {
	argv := SplitShellArgv(command)
	if len(argv) == 0 {
		return command
	}
	name := strings.ToLower(filepath.Base(argv[0]))
	name = strings.TrimSuffix(name, ".exe")
	if name != "powershell" && name != "pwsh" {
		return command
	}
	for i := 1; i < len(argv); i++ {
		a := strings.ToLower(argv[i])
		if a == "-command" || a == "-c" || a == "/command" || a == "/c" {
			return strings.TrimSpace(strings.Join(argv[i+1:], " "))
		}
	}
	return command
}

// shellHasStatementSep 报告未加引号的分号，这样 Windows 不会把 `node --version; node -p 1` 送进 cmd /C，以免分号被当成开关的一部分。引号里的分号（findstr /C:"a;b"）仍走 cmd。
// 单引号内不把反斜杠当转义。双引号外的反斜杠吃掉下一个字符。
// command 逐个字符看未加引号的分号；双引号外的反斜杠吃掉下一个字符。
func shellHasStatementSep(command string) bool {
	inSingle, inDouble := false, false
	escape := false
	for _, r := range command {
		if escape {
			escape = false
			continue
		}
		if r == '\\' && !inSingle {
			escape = true
			continue
		}
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ';':
			if !inSingle && !inDouble {
				return true
			}
		}
	}
	return false
}

// looksSimpleArgv 判断能否直接执行 argv[0]。cmd、powershell、pwsh、sh、bash，或程序名含 &|;<> 时返回假。
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

// shellNeedsWrapper 在参数为空、命令含 &|;<> 或换行、是 shell 内建，或 looksSimpleArgv 为假时要求外壳包装。
// command 含 &|;<> 或换行时要求包装。
// argv 长度为 0 时要求包装；否则看第一项是不是 shell 内建，以及 looksSimpleArgv。
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

// looksPosixUnix 在命令含 /dev/null 或 $(，或某一段的程序是 unix 工具时返回真。Windows FIND 的 /c、/v 这类开关不算 unix find。
// command 含 /dev/null 或 $( 时返回真；否则按 splitShellStages 分段，看程序名是不是 unix 工具。
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

// unixOutputCap 匹配只用来截断输出的 `| head -N` 或 `| tail -N`，前面可以有 2>&1。运行时已经捕获完整输出。
var unixOutputCap = regexp.MustCompile(`(?i)(?:\s*2>&1)?\s*\|\s*(?:head|tail)(?:\s+-n?\s*\d+|\s+-\d+)?\b`)

// stripUnixOutputPipes 去掉 posix 输出截断，避免 Windows cmd 命令只因为模型接了 head 或 tail 就被拒绝。
// 去掉之后仍像 posix，或没有任何变化，则保持原命令并返回 false。
// command 用 unixOutputCap 去掉 head 或 tail 管道；去掉后为空、与去掉空白后的原文相同，或仍像 posix 时返回原来的 command。
func stripUnixOutputPipes(command string) (string, bool) {
	next := strings.TrimSpace(unixOutputCap.ReplaceAllString(command, ""))
	if next == "" || next == strings.TrimSpace(command) {
		return command, false
	}
	if looksPosixUnix(next) {
		return command, false
	}
	return next, true
}

// isUnixUtil 判断名字是不是 ls、cat、find、sed 这类 posix 工具。比较前去掉空白并转成小写。
// name 去掉空白并转成小写后和上述名单比较。
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

// isWindowsFindStage 在参数里出现 /c、/v、/n、/i 或以 /off 开头时，把 find 当成 Windows FIND。任一参数以 - 开头则返回假。
// argv 从第二项起看；出现 /c、/v、/n、/i 或以 /off 开头则返回真，任一项以 - 开头则返回假。
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

// splitShellStages 按未加引号的 &&、||、|、; 和换行把命令切成段。
// command 按未加引号的 &&、||、|、; 和换行切开，去掉空白后为空的段丢掉。
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

// isShellBuiltin 判断名字是不是 cd、echo、dir 这类需要外壳的内建命令。
// name 去掉空白并转成小写后和上述名单比较。
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

// destructiveShell 是按小写子串拒绝的破坏性命令。匹配的是包含关系，不是整词。
var destructiveShell = []string{
	"rm -rf /", "rm -rf /*", "format c:", "format d:", "mkfs", "diskpart",
	"shutdown", "reboot", "bcdedit", "reg delete",
	":(){:|:&};:", "fork bomb",
	"remove-item -recurse c:\\", "del /s /q c:\\",
}

var networkCmd = regexp.MustCompile(`\b(curl|wget|nc |ncat|ssh |scp |sftp |invoke-webrequest|iwr |start-bitstransfer|certutil\s+-urlcache)\b`)

// looksLikeRemoteNetwork 在命令像 curl、wget、ssh 等且目标不是全部回环时返回真。没有解析出主机时也不算回环，因此光写 curl 会被拒绝。
// lower 先用 networkCmd 匹配；匹配上再看 networkTargetsAreLoopback 是不是假。
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

// networkTargetsAreLoopback 在收集到的主机全部是回环或未指定地址时返回真。一个主机都没有则返回假。
// lower 交给 collectNetworkHosts；没有主机返回假，有一个不是回环也返回假。
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

// collectNetworkHosts 从 URL、裸回环地址和像域名的词里收集主机。解释器模块名和像文件名的词被丢掉。紧挨 -m 的词不当成域名。
// lower 用 URL、裸回环和域名正则收集主机；空串、解释器模块名和像文件名的词丢掉。
func collectNetworkHosts(lower string) []string {
	var out []string
	add := func(h string) {
		h = strings.TrimSpace(h)
		if h == "" || isInterpreterModuleHost(h) {
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
	for _, loc := range reBareDNS.FindAllStringSubmatchIndex(lower, -1) {
		if len(loc) < 4 || loc[2] < 0 {
			continue
		}
		if loc[2] >= 3 && lower[loc[2]-3:loc[2]] == "-m " {
			continue
		}
		h := lower[loc[2]:loc[3]]
		if !isFilenameHost(h) {
			add(h)
		}
	}
	return out
}

// isInterpreterModuleHost 把 http.server、urllib.request 这类模块名排除出主机列表。
func isInterpreterModuleHost(h string) bool {
	h = strings.ToLower(strings.TrimSpace(h))
	switch h {
	case "http.server", "http.client", "http.cookiejar", "simplehttpserver",
		"xml.etree", "urllib.request", "urllib.parse", "urllib.error":
		return true
	default:
		return false
	}
}

var fileExtLooksLikeTLD = map[string]bool{
	"html": true, "htm": true, "xhtml": true, "js": true, "cjs": true, "mjs": true,
	"ts": true, "tsx": true, "jsx": true, "css": true, "json": true, "md": true,
	"txt": true, "png": true, "jpg": true, "jpeg": true, "gif": true, "svg": true,
	"webp": true, "pdf": true, "go": true, "py": true, "rs": true, "xml": true,
	"yml": true, "yaml": true, "toml": true, "lock": true, "map": true, "wasm": true,
	"exe": true, "dll": true, "zip": true, "tar": true, "gz": true, "log": true,
	"csv": true, "vue": true, "scss": true, "less": true,
}

// isFilenameHost 在最后一个点后面的扩展名属于常见文件后缀时返回真，避免把 pelican-bicycle.html 当成域名。
func isFilenameHost(h string) bool {
	h = strings.TrimSpace(strings.ToLower(h))
	if i := strings.LastIndexByte(h, ':'); i > 0 {
		h = h[:i]
	}
	dot := strings.LastIndexByte(h, '.')
	if dot < 0 || dot == len(h)-1 {
		return false
	}
	return fileExtLooksLikeTLD[h[dot+1:]]
}

// isLoopbackHost 判断主机是 localhost、以 .localhost 结尾、回环地址，或未指定地址 0.0.0.0 与 ::。
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

// networkPermitted 在 allow 里出现 * 或任何非空白项时返回真。不按主机名匹配，有一项就放行全部远程命令。
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

// denyEscapingAbsPaths 只在命令里出现 windows、etc、usr 或 system32 的绝对路径样式时，再逐段检查。没有工作区也没有额外根时放行。
// command 不含上述绝对路径样式时直接返回 nil；否则按空白切开，逐段交给 denyShellAbsPath。
// workspace 为空且 extraRoots 长度为 0 时直接返回 nil；否则放进根列表。
// extraRoots 追加进根列表，交给 denyShellAbsPath。
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
		if err := denyShellAbsPath(f, roots); err != nil {
			return err
		}
	}
	return nil
}

// denyShellAbsPath 检查一个参数是否逃出工作区。不调用 WorkspaceTools.resolve，也不解析符号链接。
// Windows 开关直接放行。WSL 的 bash.exe 拒绝。经 CanonicalizeToolPath 之后仍不是绝对路径的，放行。主机解释器的绝对路径放行。其余绝对路径必须落在某个根的 WithinWorkspace 里。
// f 去掉引号后，Windows 开关或经 CanonicalizeToolPath 后仍非绝对路径则放行；是 WSL 桩则拒绝。
// roots 交给 pathInRoots，用来判断其余绝对路径是否落在某个根内。
func denyShellAbsPath(f string, roots []string) error {
	f = strings.Trim(f, `"'`)
	if capability.LooksLikeWindowsSwitch(f) {
		return nil
	}
	if isWSLPosixStub(f) {
		return fmt.Errorf("shell policy denied WSL bash.exe (NAT/mount failures). Use Git Bash, or web_fetch for public HTTP")
	}
	f = capability.CanonicalizeToolPath(f)
	if !filepath.IsAbs(f) {
		return nil
	}
	if looksLikeHostInterpreter(f) {
		return nil
	}
	if !pathInRoots(f, roots) {
		return fmt.Errorf("shell policy denied path outside workspace: %s (interpreters are allowed; data files must stay in the workspace; public HTTP uses web_fetch)", f)
	}
	return nil
}

// looksLikeHostInterpreter 按文件名判断是不是 python、node、go、bash、pwsh 这类宿主解释器。工作区外的这些绝对路径仍放行。
func looksLikeHostInterpreter(p string) bool {
	base := strings.ToLower(filepath.Base(strings.Trim(p, `"'`)))
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "python", "python3", "pythonw", "py", "node", "nodejs", "bun", "deno",
		"go", "ruby", "perl", "php", "bash", "sh", "pwsh", "powershell", "cmd",
		"git", "java", "dotnet", "pip", "pip3", "uv", "conda":
		return true
	default:
		return false
	}
}

// pathInRoots 在任一非空根上 WithinWorkspace 为真时返回真。
// roots 里去掉空白后非空的项交给 WithinWorkspace，有一个为真就返回真。
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
