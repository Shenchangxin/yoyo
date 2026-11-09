//go:build windows

// shim_windows.go 把 PATH 上找到的 .cmd/.bat 改写成能直接启动的命令：npm 垫片换成 node.exe 加入口，
// 解析不了的经 cmd.exe /d /s /c 原样启动。避免控制台闪烁和 cmd 的参数转义问题。

package buddy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// commandFor 把可执行文件 path 和参数 args 变成启动命令。node 在垫片旁边没有 node.exe 时提供 node 的路径。
func commandFor(path string, args []string, node func() (string, error)) acp.Launch {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		return acp.Launch{Command: path, Args: args}
	}
	if raw, err := os.ReadFile(path); err == nil {
		if entry, ok := parseShim(string(raw), filepath.Dir(path)); ok && fileExists(entry) {
			nodeExe := filepath.Join(filepath.Dir(path), "node.exe")
			if !fileExists(nodeExe) {
				nodeExe = ""
				if node != nil {
					if found, err := node(); err == nil {
						nodeExe = found
					}
				}
			}
			if nodeExe != "" {
				return acp.Launch{Command: nodeExe, Args: append([]string{entry}, args...)}
			}
		}
	}
	return acp.Launch{Command: path, Args: args, ViaCmd: true}
}
