// Package safeguard 在工具调用前做保守拦截。
// PreTool 只对 shell、bash、sh、cmd、powershell、pwsh、run_command、execute_command、exec 跑 LooksDelete。
// write_file、str_replace、apply_patch、office_create、office_update 不跑 LooksExfil。
// browser_type、browser_fill，以及 computer_ 前缀，会在 LooksSecretPaste 命中时拒绝。
package safeguard

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// RedactText 交给 diaglog.Redact，供诊断日志和支持包共用。
// s 传给 diaglog.Redact。
func RedactText(s string) string {
	return diaglog.Redact(s)
}

// LooksExfil 用整段小写文本做外传猜测。命中时第二个返回值是原因。
// 含 webhook.site、requestbin 或 pipedream.net 时原因是 exfil sink host。
// @ 至少出现 8 次且含 http:// 或 https:// 时原因是 bulk addresses posted to a URL。
// 同时含 authorization: 和 http 时原因是 authorization header on an outbound URL。
// 抽到的第一个 URL 旁边出现 contacts、mailbox 或 password，且主机不在 trustedDest 里时，原因是 personal graph posted off-connector。
// action 转成小写后与 command、path、args 用空格拼成待查文本。
// command 转成小写后与 action、path、args 用空格拼成待查文本。
// path 转成小写后与 action、command、args 用空格拼成待查文本。
// args 转成小写后与 action、command、path 用空格拼成待查文本。
func LooksExfil(action, command, path, args string) (bool, string) {
	blob := strings.ToLower(action + " " + command + " " + path + " " + args)
	if strings.Contains(blob, "webhook.site") || strings.Contains(blob, "requestbin") || strings.Contains(blob, "pipedream.net") {
		return true, "exfil sink host"
	}
	if strings.Count(blob, "@") >= 8 && (strings.Contains(blob, "http://") || strings.Contains(blob, "https://")) {
		return true, "bulk addresses posted to a URL"
	}
	if strings.Contains(blob, "authorization:") && strings.Contains(blob, "http") {
		return true, "authorization header on an outbound URL"
	}
	dest := firstURL(blob)
	if dest != "" && (strings.Contains(blob, "contacts") || strings.Contains(blob, "mailbox") || strings.Contains(blob, "password")) {
		if !trustedDest(dest) {
			return true, "personal graph posted off-connector"
		}
	}
	return false, ""
}

// LooksDelete 在小写拼接文本里查找固定的破坏性片段：rm -rf /、rm -rf /*、del /s /q c:、format c:、format d:、rmdir /s /q c:、remove-item -recurse -force c:。
// action 转成小写后与 command、args 用空格拼成待查文本。
// command 转成小写后与 action、args 用空格拼成待查文本。
// args 转成小写后与 action、command 用空格拼成待查文本。
func LooksDelete(action, command, args string) bool {
	blob := strings.ToLower(action + " " + command + " " + args)
	for _, tok := range []string{
		"rm -rf /", "rm -rf /*", "del /s /q c:", "format c:", "format d:",
		"rmdir /s /q c:", "remove-item -recurse -force c:",
	} {
		if strings.Contains(blob, tok) {
			return true
		}
	}
	return false
}

// isShellTool 在名字去掉空白并小写后是否为 shell、bash、sh、cmd、powershell、pwsh、run_command、execute_command 或 exec。
// name 去掉空白并转成小写后，与 shell、bash、sh、cmd、powershell、pwsh、run_command、execute_command、exec 比较。
func isShellTool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "shell", "bash", "sh", "cmd", "powershell", "pwsh", "run_command", "execute_command", "exec":
		return true
	default:
		return false
	}
}

// LooksSecretPaste 在小写参数里查找 sk-、api_key 或 -----begin。
// args 转成小写后查找 sk-、api_key 或 -----begin。
func LooksSecretPaste(args string) bool {
	low := strings.ToLower(args)
	return strings.Contains(low, "sk-") || strings.Contains(low, "api_key") || strings.Contains(low, "-----begin")
}

// isWorkspaceWrite 在名字去掉空白并小写后是否为 write_file、str_replace、apply_patch、office_create 或 office_update。
// name 去掉空白并转成小写后，与 write_file、str_replace、apply_patch、office_create、office_update 比较。
func isWorkspaceWrite(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "write_file", "str_replace", "apply_patch", "office_create", "office_update":
		return true
	default:
		return false
	}
}

// PreTool 在执行工具前决定是否拒绝。文件正文不当成 shell：只有 isShellTool 为真才跑 LooksDelete，命中时原因是 destructive delete requires an explicit operator path。
// 工作区写入不跑 LooksExfil。其余名字若 LooksExfil 命中就拒绝。
// 名字是 browser_type、browser_fill，或以 computer_ 开头时，LooksSecretPaste 命中就拒绝，原因是 refusing to paste secrets into the browser/desktop。
// name 传给 isShellTool、LooksDelete 和 isWorkspaceWrite。非工作区写入时再传给 LooksExfil。等于 browser_type、browser_fill，或以 computer_ 开头时再查粘贴。
// arguments 在 shell 工具时作为 LooksDelete 的 command。非工作区写入时作为 LooksExfil 的最后一段，并传给 LooksSecretPaste。
func PreTool(name, arguments string) (deny bool, reason string) {
	// 文件正文不是 shell 命令。若在 write_file、apply_patch 的参数里扫 "format "，普通 Go 注释也会被挡住。
	if isShellTool(name) && LooksDelete(name, arguments, "") {
		return true, "destructive delete requires an explicit operator path"
	}
	// 写进工作区不是外传。接口说明里的 Authorization: 和示例 https URL 要等后面的网络工具才会离开本机。
	if !isWorkspaceWrite(name) {
		if ok, why := LooksExfil(name, "", "", arguments); ok {
			return true, why
		}
	}
	if name == "browser_type" || name == "browser_fill" || strings.HasPrefix(name, "computer_") {
		if LooksSecretPaste(arguments) {
			return true, "refusing to paste secrets into the browser/desktop"
		}
	}
	return false, ""
}

// RequestUnsafe 对能力请求做同样的破坏性删除和外传判断。shell 类且 LooksDelete 命中时原因是 destructive delete。
// 否则把 Action、Command、Path 交给 LooksExfil，第四段参数是空字符串。
// req 的 Action 传给 isShellTool。命中时 Action 和 Command 传给 LooksDelete，第三段是空字符串。否则 Action、Command、Path 传给 LooksExfil，第四段是空字符串。
func RequestUnsafe(req capability.Request) (bool, string) {
	if isShellTool(req.Action) && LooksDelete(req.Action, req.Command, "") {
		return true, "destructive delete"
	}
	return LooksExfil(req.Action, req.Command, req.Path, "")
}

// firstURL 返回第一个以 http:// 或 https:// 开头的字段，并去掉两端的引号。
// 没有这样的字段时，若整段是 JSON，再依次看 url、endpoint、href。
func firstURL(s string) string {
	for _, p := range strings.Fields(s) {
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			return strings.Trim(p, `"'`)
		}
	}
	var raw map[string]any
	if json.Unmarshal([]byte(s), &raw) == nil {
		for _, k := range []string{"url", "endpoint", "href"} {
			if v, ok := raw[k].(string); ok {
				return v
			}
		}
	}
	return ""
}

// trustedDest 在解析出的主机名等于或结尾为下列后缀时返回 true：googleapis.com、google.com、microsoft.com、office.com、outlook.com、github.com、slack.com、feishu.cn、dingtalk.com、larksuite.com。解析失败返回 false。
// raw 交给 url.Parse。失败返回 false。否则取主机名的小写，与这些后缀相等或以点加后缀结尾时返回 true。
func trustedDest(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, suf := range []string{"googleapis.com", "google.com", "microsoft.com", "office.com", "outlook.com", "github.com", "slack.com", "feishu.cn", "dingtalk.com", "larksuite.com"} {
		if h == suf || strings.HasSuffix(h, "."+suf) {
			return true
		}
	}
	return false
}
