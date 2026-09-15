package runtime

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// ShellDenied is a deterministic policy jail, not an OS sandbox. Seatbelt /
// bubblewrap are OS-specific; lying about them on Windows is worse than a
// named deny-list plus the existing workspace path jail.
func ShellDenied(command, workspace string, networkAllow []string) error {
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
	if looksLikeNetwork(lower) && !networkPermitted(networkAllow) {
		return fmt.Errorf("shell policy denied network; set policy.network_allow or use MCP")
	}
	if err := denyEscapingAbsPaths(cmd, workspace); err != nil {
		return err
	}
	return nil
}

var destructiveShell = []string{
	"rm -rf /", "rm -rf /*", "format ", "mkfs", "diskpart",
	"shutdown", "reboot", "bcdedit", "reg delete",
	":(){:|:&};:", "fork bomb",
	"remove-item -recurse c:\\", "del /s /q c:\\",
}

var networkCmd = regexp.MustCompile(`\b(curl|wget|nc |ncat|ssh |scp |sftp |invoke-webrequest|iwr |start-bitstransfer|certutil\s+-urlcache)\b`)

func looksLikeNetwork(lower string) bool {
	return networkCmd.MatchString(lower)
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

func denyEscapingAbsPaths(command, workspace string) error {
	if workspace == "" {
		return nil
	}
	if !absPath.MatchString(command) {
		return nil
	}
	fields := strings.Fields(command)
	for _, f := range fields {
		f = strings.Trim(f, `"'`)
		if !filepath.IsAbs(f) {
			continue
		}
		if !capability.WithinWorkspace(workspace, f) {
			return fmt.Errorf("shell policy denied path outside workspace")
		}
	}
	return nil
}
