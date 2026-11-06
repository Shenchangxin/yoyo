package connector

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/poi"
)

func missingToken(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no token") || strings.Contains(msg, "empty token")
}

func entryBy(provider string) (CatalogEntry, bool) {
	for _, entry := range Catalog() {
		if entry.Provider == provider {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}

func osPrefix() string {
	switch runtime.GOOS {
	case "windows":
		return "win-"
	case "darwin":
		return "mac-"
	case "linux":
		return "linux-"
	default:
		return ""
	}
}

// readSurface 读这台电脑上的桌面、文档、下载或命令行。不是本机条目时 handled 为假。
func readSurface(provider string) ([]map[string]string, bool, error) {
	entry, ok := entryBy(provider)
	if !ok || entry.Access != "local" {
		return nil, false, nil
	}
	if !strings.HasPrefix(provider, osPrefix()) {
		return nil, true, fmt.Errorf("connector: 这台电脑读不了 %s", provider)
	}
	if strings.HasSuffix(provider, "-shell") {
		shell := os.Getenv("ComSpec")
		if shell == "" {
			shell = "shell"
		}
		return []map[string]string{{"provider": provider, "title": "命令行", "text": shell}}, true, nil
	}
	var roots []string
	for _, root := range poi.DefaultRoots() {
		base := strings.ToLower(filepath.Base(root))
		switch {
		case strings.HasSuffix(provider, "-desktop") && base == "desktop":
			roots = append(roots, root)
		case strings.HasSuffix(provider, "-docs") && base == "documents":
			roots = append(roots, root)
		case strings.HasSuffix(provider, "-downloads") && base == "downloads":
			roots = append(roots, root)
		case strings.HasSuffix(provider, "-files"):
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		return nil, true, fmt.Errorf("connector: 没有找到对应的文件夹")
	}
	var items []map[string]string
	for _, obs := range poi.ScanRoots(roots, time.Now()) {
		title := strings.TrimSpace(obs.Name)
		if title == "" {
			continue
		}
		items = append(items, map[string]string{"provider": provider, "title": title, "text": roots[0]})
		if len(items) == 20 {
			break
		}
	}
	if len(items) == 0 {
		items = append(items, map[string]string{"provider": provider, "title": filepath.Base(roots[0]), "text": roots[0]})
	}
	return items, true, nil
}

func filterItems(items []map[string]string, query string) []map[string]string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items
	}
	var out []map[string]string
	for _, item := range items {
		blob := strings.ToLower(item["title"] + " " + item["text"] + " " + item["subject"])
		if strings.Contains(blob, q) {
			out = append(out, item)
		}
	}
	return out
}
