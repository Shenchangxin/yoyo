package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// SanitizeProfileName accepts default, letters, digits, _ and -. Named
// profiles keep cookies in ~/.yoyo/browser/profiles/<name> so a GitHub
// login does not leak into the workspace-HTML game profile.
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
