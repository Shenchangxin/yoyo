// 本文件在 Windows 上去掉下载文件的 Zone.Identifier 标记。其他系统直接返回。

package pwsh

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// unblockPath 在 Windows 上删除 path 的 Zone.Identifier 备用数据流。路径空白或不是 Windows 时什么都不做。删除错误被丢掉。
func unblockPath(path string) {
	if runtime.GOOS != "windows" || strings.TrimSpace(path) == "" {
		return
	}
	_ = os.Remove(path + ":Zone.Identifier")
}

// unblockTree 对 root 下每个普通文件调用 unblockPath。目录、空白 root 和非 Windows 直接返回。Walk 里的错误被跳过。
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
