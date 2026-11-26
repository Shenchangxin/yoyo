// payment_rpc.go 按操作系统和架构拼出支付 RPC 后端可能的文件名。

package protocol

import (
	"path"
	"strings"
)

// HasAnyPaymentRPCBackend 查看文件表里有没有入口本身、入口加 .exe、以入口加横线开头的名字，或嵌在系统架构目录下的同名文件。
// entry 去掉首尾空白和 .exe 后缀，空串时返回假。
// files 按 entry、entry 加 .exe、entry 加横线前缀，以及目录下同名文件查找。
func HasAnyPaymentRPCBackend(entry string, files map[string][]byte) bool {
	entry = strings.TrimSuffix(strings.TrimSpace(entry), ".exe")
	if entry == "" {
		return false
	}
	if _, ok := files[entry]; ok {
		return true
	}
	if _, ok := files[entry+".exe"]; ok {
		return true
	}
	prefix := entry + "-"
	nestedRoot := path.Dir(entry) + "/"
	base := path.Base(entry)
	for name := range files {
		if strings.HasPrefix(name, prefix) {
			return true
		}
		rest, ok := strings.CutPrefix(name, nestedRoot)
		if !ok {
			continue
		}
		slash := strings.IndexByte(rest, '/')
		if slash <= 0 {
			continue
		}
		leaf := rest[slash+1:]
		if leaf == base || leaf == base+".exe" {
			return true
		}
	}
	return false
}
