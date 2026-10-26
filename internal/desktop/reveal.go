package desktop

import (
	"os"
	"path/filepath"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
)

// RevealPath 在路径为空时返回 nil。否则交给 hostopen.Reveal。
// path 为空时返回 nil。否则传给 hostopen.Reveal。
func RevealPath(path string) error {
	if path == "" {
		return nil
	}
	return hostopen.Reveal(path)
}

// executablePath 返回可执行文件的真实路径。符号链接解析失败时返回 os.Executable 的原路径。连原路径都取不到时返回空字符串。
func executablePath() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}
