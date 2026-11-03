package initiative

import (
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LocalFacts 收集仓库、电量、网络、前台程序和离开时长。不读窗口标题，不读文件正文。
func LocalFacts(roots []string) ([]Fact, time.Duration) {
	var facts []Fact
	m := machine()
	idle := time.Duration(0)
	if m.IdleKnown {
		idle = m.Idle
		if idle > 12*time.Hour {
			idle = 0
		} else if idle >= 2*time.Minute {
			facts = append(facts, Fact{
				ID: "idle", Code: "idle", Class: "idle", Name: "idle",
				Detail: idle.Truncate(time.Minute).String(), State: "idle", Sig: "idle", Rank: 8,
			})
		}
	}
	if m.PowerKnown {
		state := "ok"
		eligible := false
		if m.Percent < 15 && !m.Charging {
			state = "low"
			eligible = true
		}
		detail := "battery"
		if m.Charging {
			detail = "charging"
		}
		facts = append(facts, Fact{
			ID: "power", Code: "power", Class: "power", Name: itoa(m.Percent),
			Detail: detail, State: state, Sig: state, Rank: 4, Eligible: eligible,
		})
	}
	if name := strings.TrimSpace(m.Focus); name != "" {
		facts = append(facts, Fact{
			ID: "focus", Code: "focus", Class: "focus", Name: name, State: "foreground", Sig: name, Rank: 8,
		})
	}
	state := "online"
	eligible := false
	if !Online() {
		state = "offline"
		eligible = true
	}
	facts = append(facts, Fact{
		ID: "net", Code: "net", Class: "net", Name: "network", State: state, Sig: state, Rank: 4, Eligible: eligible,
	})
	for _, repo := range GitRepos(roots) {
		gitState := "clean"
		if repo.Dirty {
			gitState = "dirty"
		}
		facts = append(facts, Fact{
			ID: "git:" + strings.ToLower(repo.Name), Code: "git", Class: "git", Name: repo.Name,
			Detail: repo.Branch, State: gitState, Sig: repo.Branch + "/" + gitState, Rank: 5, Eligible: true,
		})
	}
	return facts, idle
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [4]byte
	i := len(buf)
	for n > 0 && i > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// Online 表示有一块非回环网卡带着地址并且是起来的。不读取网络名。
func Online() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		return true
	}
	return false
}

// GitFact 是一个仓库的分支和干净与否。不列出文件名。
type GitFact struct {
	Name   string
	Branch string
	Dirty  bool
}

// GitRepos 看这些根目录自己和下一层有没有仓库。最多八个。不读文件正文。
func GitRepos(roots []string) []GitFact {
	var out []GitFact
	seen := map[string]bool{}
	for _, root := range roots {
		if len(out) >= 8 {
			break
		}
		consider := []string{root}
		entries, err := osReadDir(root)
		if err == nil {
			for _, name := range entries {
				if len(consider) >= 24 {
					break
				}
				if name == "" || strings.HasPrefix(name, ".") || name == "node_modules" {
					continue
				}
				consider = append(consider, filepath.Join(root, name))
			}
		}
		for _, dir := range consider {
			if len(out) >= 8 {
				break
			}
			if seen[dir] || !isRepo(dir) {
				continue
			}
			seen[dir] = true
			branch, dirty, ok := gitState(dir)
			if !ok || scrub(branch) == "" {
				continue
			}
			name := filepath.Base(dir)
			if scrub(name) == "" {
				continue
			}
			out = append(out, GitFact{Name: name, Branch: branch, Dirty: dirty})
		}
	}
	return out
}

func isRepo(dir string) bool {
	info, err := osStat(filepath.Join(dir, ".git"))
	return err == nil && info != nil
}

func gitState(dir string) (string, bool, bool) {
	branch := strings.TrimSpace(gitOutput(dir, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch == "" || branch == "HEAD" {
		branch = "detached"
	}
	if strings.ContainsAny(branch, "\n\r") {
		return "", false, false
	}
	out := gitOutput(dir, "status", "--porcelain")
	dirty := strings.TrimSpace(out) != ""
	return branch, dirty, true
}

func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	timer := time.AfterFunc(3*time.Second, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	defer timer.Stop()
	b, err := cmd.Output()
	if err != nil && len(b) == 0 {
		return ""
	}
	return string(b)
}
