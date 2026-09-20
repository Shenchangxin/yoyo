package desktop

import (
	"os"
	"path/filepath"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
)

func RevealPath(path string) error {
	if path == "" {
		return nil
	}
	return hostopen.Reveal(path)
}

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
