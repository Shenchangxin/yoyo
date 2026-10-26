// payment_rpc.go 按操作系统和架构拼出支付 RPC 后端可能的文件名。

package protocol

import (
	"path"
	"strings"
)

// PaymentRPCBackendCandidates 在 entry 以 backend/ 开头且清理后不变时，给出带系统架构后缀、嵌套目录和原名三种路径。Windows 先给出对应的 .exe。入口、系统或架构为空时返回 nil。
// goos 去掉空白后为空时返回 nil；等于 windows 时先追加三条 .exe 路径。
// goarch 去掉空白后为空时返回 nil，否则和 goos 一起拼进带后缀和嵌套目录的路径。
func PaymentRPCBackendCandidates(entry, goos, goarch string) []string {
	entry = strings.TrimSuffix(strings.TrimSpace(entry), ".exe")
	goos = strings.TrimSpace(goos)
	goarch = strings.TrimSpace(goarch)
	if entry == "" || goos == "" || goarch == "" || path.Clean(entry) != entry || !strings.HasPrefix(entry, "backend/") {
		return nil
	}
	base := path.Base(entry)
	dir := path.Dir(entry)
	tagged := entry + "-" + goos + "-" + goarch
	nested := dir + "/" + goos + "-" + goarch + "/" + base
	candidates := make([]string, 0, 6)
	if goos == "windows" {
		candidates = append(candidates, tagged+".exe", nested+".exe", entry+".exe")
	}
	candidates = append(candidates, tagged, nested, entry)
	return uniqueStrings(candidates)
}

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

// uniqueStrings 按出现顺序去掉重复字符串。
// values 按出现顺序丢掉已经见过的字符串。
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
