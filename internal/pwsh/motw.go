package pwsh

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func unblockPath(path string) {
	if runtime.GOOS != "windows" || strings.TrimSpace(path) == "" {
		return
	}
	_ = os.Remove(path + ":Zone.Identifier")
}

func unblockTree(root string) {
	if runtime.GOOS != "windows" || strings.TrimSpace(root) == "" {
		return
	}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		unblockPath(p)
		return nil
	})
}
