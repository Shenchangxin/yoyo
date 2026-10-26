// 本文件拥有配置名和用户目录路径。命名配置把 cookie 放在目录下的 profiles/<name>，避免一份登录态进到工作区 HTML 用的那份配置。
// 配置名 attached 被拒绝。有头和无头各用一份目录：default 是 profile 与 profile-headed，其他名字是 profiles/<name> 与 profiles/<name>-headed。

package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// SanitizeProfileName 接受空名（当成 default）、字母、数字、下划线和减号。长于 40 个字符，或名字是 attached（不区分大小写），返回错误。
// name 先去掉空白。空则返回 default。长度大于 40、与 attached 不区分大小写相等，或含有字母、数字、下划线和减号以外的字符时返回错误。
func SanitizeProfileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default", nil
	}
	if len(name) > 40 {
		return "", fmt.Errorf("browser: profile name longer than 40 characters")
	}
	if strings.EqualFold(name, "attached") {
		return "", fmt.Errorf("browser: profile name %q is reserved", name)
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return "", fmt.Errorf("browser: profile name %q must be letters, digits, _ or -", name)
	}
	return name, nil
}

// ProfileName 返回当前配置名。宿主为空或名字空白时是 default。
func (h *Host) ProfileName() string {
	if h == nil {
		return "default"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.profileName == "" {
		return "default"
	}
	return h.profileName
}

// UseProfile 停掉当前 Chrome，换成另一份用户目录。名字相同且 Chrome 还活着时什么也不做。
// name 传给 SanitizeProfileName。与当前配置名相同且 chromeAlive 为真时直接返回。否则写入 profileName，并作为 profile 的 detail 传给 record。
func (h *Host) UseProfile(name string) error {
	name, err := SanitizeProfileName(name)
	if err != nil {
		return err
	}
	if h == nil {
		return fmt.Errorf("browser: no host")
	}
	if h.ProfileName() == name && h.chromeAlive() {
		return nil
	}
	h.stopChrome()
	h.invalidateRefs()
	h.mu.Lock()
	h.profileName = name
	h.profile = h.profileDirLocked(false)
	h.mu.Unlock()
	h.record("profile", name)
	return nil
}

// profileDirLocked 按配置名和是否有头返回用户目录。调用方须已持有 mu。
// headed 为真且配置名是空或 default 时返回 dir 下的 profile-headed，为假时返回 profile。其他名字为真时返回 profiles 下该名字加 -headed，为假时返回 profiles 下该名字。
func (h *Host) profileDirLocked(headed bool) string {
	name := h.profileName
	if name == "" || name == "default" {
		if headed {
			return filepath.Join(h.dir, "profile-headed")
		}
		return filepath.Join(h.dir, "profile")
	}
	if headed {
		return filepath.Join(h.dir, "profiles", name+"-headed")
	}
	return filepath.Join(h.dir, "profiles", name)
}

// ListProfiles 返回 default，再加上 profiles 目录里名字合法、尚未见过的子目录。带 -headed 后缀的目录并进同名配置。
func (h *Host) ListProfiles() []string {
	if h == nil {
		return []string{"default"}
	}
	out := []string{"default"}
	root := filepath.Join(h.dir, "profiles")
	ents, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	seen := map[string]bool{"default": true}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), "-headed")
		if _, err := SanitizeProfileName(name); err != nil || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
