// Package boot 是窗口进程（根目录 main.go）和命令行（cmd/yoyo）共用的启动路径解析。
// 两个入口以前各写一份，查找顺序不一致；这里是唯一的规则。
package boot

import (
	"net/http"
	"os"
	"path/filepath"
)

// ExeDir 返回当前可执行文件所在目录。取不到时返回 "."。
func ExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// EvalsDir 返回评测目录。顺序：YOYO_EVALS；可执行文件旁的 evals；其上一级的 evals；工作目录下的 evals。
// 可执行文件旁的优先，安装包不会被启动时所在目录里的同名文件夹带偏；go run 时可执行文件在临时目录，自然落到工作目录。
// 命中的目录转成绝对路径。都不存在时返回相对路径 evals。
func EvalsDir() string {
	if v := os.Getenv("YOYO_EVALS"); v != "" {
		return v
	}
	exe := ExeDir()
	candidates := []string{
		filepath.Join(exe, "evals"),
		filepath.Join(exe, "..", "evals"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "evals"))
	}
	for _, c := range candidates {
		if isDir(c) {
			if abs, err := filepath.Abs(c); err == nil {
				return abs
			}
			return c
		}
	}
	return "evals"
}

// FrontendDir 返回磁盘上已构建的前端目录：先看工作目录下的 frontend/dist，再看可执行文件旁的。
// 目录里 index.html 必须是文件才算；只有占位文件的空目录不算。都没有时返回空串。
func FrontendDir() string {
	for _, dir := range []string{"frontend/dist", filepath.Join(ExeDir(), "frontend", "dist")} {
		if st, err := os.Stat(filepath.Join(dir, "index.html")); err == nil && !st.IsDir() {
			return dir
		}
	}
	return ""
}

// DiskFrontend 用 FrontendDir 建静态文件处理器。没有磁盘前端时返回 nil，调用方自行退回内嵌资源或不挂静态页。
func DiskFrontend() http.Handler {
	dir := FrontendDir()
	if dir == "" {
		return nil
	}
	return http.FileServer(http.Dir(dir))
}

// isDir 在路径存在且是目录时返回真。
func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
