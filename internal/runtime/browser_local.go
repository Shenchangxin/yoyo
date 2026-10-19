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

func isRemoteBrowserURL(raw string) bool {
	low := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://")
}

func isDataURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "data:")
}

// workspacePreviewRel maps a browser_open target to a workspace-relative path.
// Remote http(s) and data: URLs are not workspace previews.
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
