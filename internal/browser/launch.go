// 本文件拥有 Chrome 启动参数、DevToolsActivePort 等待，以及用户目录锁。
// 隔离启动的调试端口由调用方传入，通常是 freePort 得到的端口，写进 --remote-debugging-port。参数里有 --remote-allow-origins=*，不设置 --remote-debugging-address。
// 无头等待 20 秒，有头等待 8 秒。

package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// chromeReadyWait 是无头 Chrome 等到调试端口起来的时间，20 秒。
const chromeReadyWait = 20 * time.Second

// chromeHeadedReadyWait 是有头 Chrome 等到调试端口起来的时间，8 秒。
const chromeHeadedReadyWait = 8 * time.Second

// chromeLaunchArgs 拼出用户目录、调试端口和禁用项。headed 为假时加 --headless=new 和 --disable-gpu；为真时加新窗口并关掉崩溃恢复气泡。startURL 空白时用 about:blank。
func chromeLaunchArgs(profile string, port int, headed bool, startURL string) []string {
	args := []string{
		"--user-data-dir=" + profile,
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--remote-allow-origins=*",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-sync",
		"--disable-extensions",
		"--disable-background-networking",
		"--disable-component-update",
		"--disable-default-apps",
		"--disable-hang-monitor",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-client-side-phishing-detection",
		"--disable-features=Translate,MediaRouter,OptimizationHints",
		"--metrics-recording-only",
		"--password-store=basic",
		"--use-mock-keychain",
		"--no-service-autorun",
	}
	if !headed {
		args = append(args, "--headless=new", "--disable-gpu")
	} else {
		args = append(args,
			"--new-window",
			"--disable-session-crashed-bubble",
			"--hide-crash-restore-bubble",
		)
	}
	if strings.TrimSpace(startURL) == "" {
		startURL = "about:blank"
	}
	return append(args, startURL)
}

// resetChromeSession 删掉用户目录和 Default 子目录里的当前会话、上次会话和 Sessions，避免有头窗口恢复旧标签。
func resetChromeSession(profile string) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return
	}
	names := []string{
		"Current Session", "Current Tabs", "Last Session", "Last Tabs", "Visited Links",
	}
	roots := []string{profile, filepath.Join(profile, "Default")}
	for _, root := range roots {
		for _, name := range names {
			_ = os.Remove(filepath.Join(root, name))
		}
		_ = os.RemoveAll(filepath.Join(root, "Sessions"))
	}
}

// waitDevToolsPort 等到 DevToolsActivePort 里的端口在本机上可访问。等价于 hint 为 0 的 waitChromeDebug。
func waitDevToolsPort(profile string, deadline time.Time) (int, error) {
	return waitChromeDebug(profile, 0, deadline)
}

// waitChromeDebug 在截止时间前等调试端口。hint 大于 0 时先看 127.0.0.1:hint 是否已在听；否则读 DevToolsActivePort，并确认那个端口真的在听。超时错误文案是 chrome debug port did not come up。
func waitChromeDebug(profile string, hint int, deadline time.Time) (int, error) {
	path := filepath.Join(profile, "DevToolsActivePort")
	var last error
	for time.Now().Before(deadline) {
		if hint > 0 {
			if chromeDebugUp("127.0.0.1:" + strconv.Itoa(hint)) {
				return hint, nil
			}
			last = fmt.Errorf("debug port %d not listening", hint)
		}
		port, err := readDevToolsActivePort(path)
		if err == nil && port > 0 {
			if chromeDebugUp("127.0.0.1:" + strconv.Itoa(port)) {
				return port, nil
			}
			last = fmt.Errorf("DevToolsActivePort %d not listening", port)
		} else if err != nil {
			last = err
		}
		time.Sleep(80 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("missing DevToolsActivePort")
	}
	return 0, fmt.Errorf("browser: chrome debug port did not come up: %w", last)
}

// readDevToolsActivePort 读文件第一行作为端口。不是正整数时返回错误。
func readDevToolsActivePort(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	line := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	port, err := strconv.Atoi(line)
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("invalid DevToolsActivePort %q", line)
	}
	return port, nil
}

// waitProfileUnlocked 等到 SingletonLock、SingletonSocket、SingletonCookie 和 lockfile 都消失。d 小于等于 0 时等 3 秒。到点仍在就删掉这些锁文件。
func waitProfileUnlocked(profile string, d time.Duration) {
	if d <= 0 {
		d = 3 * time.Second
	}
	deadline := time.Now().Add(d)
	locks := []string{
		filepath.Join(profile, "SingletonLock"),
		filepath.Join(profile, "SingletonSocket"),
		filepath.Join(profile, "SingletonCookie"),
		filepath.Join(profile, "lockfile"),
	}
	for time.Now().Before(deadline) {
		busy := false
		for _, p := range locks {
			if _, err := os.Stat(p); err == nil {
				busy = true
				break
			}
		}
		if !busy {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, p := range locks {
		_ = os.Remove(p)
	}
}
