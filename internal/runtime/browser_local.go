// 把 browser_open 的目标收成工作区相对路径，并拆出 lane 与 profile。
// 工作区预览调用 WorkspaceTools.resolve。http、https 和 data: 不是工作区预览。不启动 Chrome。

package runtime

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// parseBrowserOpen 从目标文本里拆出 lane 和 profile。默认 lane 是 isolated。lane=attached 改成 attached。
// raw 去掉空白后按空白拆开，从中取 lane= 和 profile=，其余拼回目标。
func parseBrowserOpen(raw string) (target, lane, profile string) {
	target = strings.TrimSpace(raw)
	lane = "isolated"
	var kept []string
	for _, f := range strings.Fields(target) {
		low := strings.ToLower(f)
		switch {
		case low == "lane=attached":
			lane = "attached"
		case low == "lane=isolated":
			lane = "isolated"
		case strings.HasPrefix(low, "profile="):
			profile = strings.TrimSpace(f[len("profile="):])
		default:
			kept = append(kept, f)
		}
	}
	return strings.Join(kept, " "), lane, profile
}

// browserHandle 优先用 ref，并补上 ref= 前缀。没有 ref 时返回 selector。
// args 为 nil 时返回空串，否则读取 ref，去掉空白后为空再读 selector。
func browserHandle(args map[string]any) string {
	if args == nil {
		return ""
	}
	if ref := strings.TrimSpace(str(args["ref"])); ref != "" {
		if strings.HasPrefix(strings.ToLower(ref), "ref=") || strings.HasPrefix(strings.ToLower(ref), "ref:") {
			return ref
		}
		return "ref=" + ref
	}
	return strings.TrimSpace(str(args["selector"]))
}

// isRemoteBrowserURL 判断目标是不是 http 或 https。
// raw 去掉空白并转小写后看是否以 http:// 或 https:// 开头。
func isRemoteBrowserURL(raw string) bool {
	low := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://")
}

// isDataURL 判断目标是不是 data:。
// raw 去掉空白并转小写后看是否以 data: 开头。
func isDataURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "data:")
}

// workspacePreviewRel 把 browser_open 的目标收成相对工作区的路径。远程 http(s) 和 data: 不是工作区预览。
// workspace:// 去掉前缀，file: 先译成路径。然后调用 WorkspaceTools.resolve，文件必须已经存在。
// raw 去掉空白后为空、是远程 URL 或 data: 时返回错误；workspace:// 和 file: 先剥掉再交给 resolve。
func (t *WorkspaceTools) workspacePreviewRel(raw string) (string, error) {
	if t == nil || strings.TrimSpace(t.Workspace) == "" {
		return "", fmt.Errorf("browser_open: no workspace")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || isRemoteBrowserURL(raw) || isDataURL(raw) {
		return "", fmt.Errorf("browser_open: not a workspace path")
	}
	rel := raw
	low := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(low, "workspace://"):
		rel = raw[len("workspace://"):]
	case strings.HasPrefix(low, "file:"):
		p, err := fileURLToPath(raw)
		if err != nil {
			return "", err
		}
		rel = p
	}
	abs, err := t.resolve(rel)
	if err != nil {
		return "", err
	}
	out, err := filepath.Rel(t.Workspace, abs)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("browser_open: %s not found (write_file it first)", filepath.ToSlash(out))
	}
	return filepath.ToSlash(out), nil
}

// fileURLToPath 把 file: URL 收成路径。Windows 上去掉盘符前多余的斜杠，再经 CanonicalizeToolPath。
// raw 交给 url.Parse，取 Path，Path 为空时改用 Opaque。
func fileURLToPath(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("browser_open: invalid file URL")
	}
	p := u.Path
	if u.Opaque != "" && p == "" {
		p = u.Opaque
	}
	if decoded, err := url.PathUnescape(p); err == nil {
		p = decoded
	}
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
			p = p[1:]
		}
		p = strings.ReplaceAll(p, "/", `\`)
	}
	return capability.CanonicalizeToolPath(p), nil
}
