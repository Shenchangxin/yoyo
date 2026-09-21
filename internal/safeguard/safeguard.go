package safeguard

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// RedactText is the shared redactor used by diagnostic logs and support bundles.
func RedactText(s string) string {
	return diaglog.Redact(s)
}

// LooksExfil is a conservative heuristic for connector/browser data leaving
// the machine via an unexpected channel. False positives pause the loop;
// false negatives are the real risk, so this is deny-on-suspicion in AutoSafe.
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

func isShellTool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "shell", "bash", "sh", "cmd", "powershell", "pwsh", "run_command", "execute_command", "exec":
		return true
	default:
		return false
	}
}

func LooksSecretPaste(args string) bool {
	low := strings.ToLower(args)
	return strings.Contains(low, "sk-") || strings.Contains(low, "api_key") || strings.Contains(low, "-----begin")
}

func PreTool(name, arguments string) (deny bool, reason string) {
	// File bodies are not shell commands. Scanning write_file/apply_patch
	// arguments for "format " blocked ordinary Go ("export format is").
	if isShellTool(name) && LooksDelete(name, arguments, "") {
		return true, "destructive delete requires an explicit operator path"
	}
	if ok, why := LooksExfil(name, "", "", arguments); ok {
		return true, why
	}
	if name == "browser_type" || name == "browser_fill" || strings.HasPrefix(name, "computer_") {
		if LooksSecretPaste(arguments) {
			return true, "refusing to paste secrets into the browser/desktop"
		}
	}
	return false, ""
}

func RequestUnsafe(req capability.Request) (bool, string) {
	if isShellTool(req.Action) && LooksDelete(req.Action, req.Command, "") {
		return true, "destructive delete"
	}
	return LooksExfil(req.Action, req.Command, req.Path, "")
}

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
