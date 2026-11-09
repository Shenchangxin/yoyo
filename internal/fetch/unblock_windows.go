//go:build windows

// unblock_windows.go 拥有 Windows 上去掉下载标记：删掉每个文件的 Zone.Identifier 备用流，SmartScreen 才不拦托管安装的程序。
// 不拥有解压。buddy 安装完成后调用 Unblock。

package fetch

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Unblock 删掉 dir 下所有文件的 Zone.Identifier。dir 是单个文件时只处理它。尽力而为，不返回错误。
func Unblock(dir string) {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		_ = os.Remove(p + ":Zone.Identifier")
		return nil
	})
}
