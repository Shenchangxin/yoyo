package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const chromeReadyWait = 20 * time.Second
const chromeHeadedReadyWait = 8 * time.Second

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

func waitDevToolsPort(profile string, deadline time.Time) (int, error) {
	return waitChromeDebug(profile, 0, deadline)
}

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

func waitProfileUnlocked(profile string, d time.Duration) {
	if d <= 0 {
		d = 3 * time.Second
	}
	deadline := time.Now().Add(d)
	locks := []string{
		filepath.Join(profile, "SingletonLock"),
		filepath.Join(profile, "SingletonSocket"),
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
