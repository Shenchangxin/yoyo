//go:build !windows

// alive_unix_test.go 给测试判断进程是否还活着：信号 0 能送到且不是僵尸。

package procgroup

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// alive 在 pid 对应的进程还在跑时为真。Linux 上读 /proc 排除僵尸进程。
func alive(pid int) bool {
	if syscall.Kill(pid, 0) != nil {
		return false
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	fields := strings.Fields(string(raw))
	return len(fields) < 3 || fields[2] != "Z"
}
