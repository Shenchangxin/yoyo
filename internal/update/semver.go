// 比较带可选 v 前缀的版本号。

package update

import (
	"strconv"
	"strings"
)

// canon 去掉首尾空白以及前缀 v 或 V。
func canon(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

type parsed struct {
	major, minor, patch int
	pre                 string
	ok                  bool
}

// parseSemver 解析主、次、补丁和连字符后的预发布段。加号后的构建元数据被丢掉。空串或数字非法时 ok 为 false。
func parseSemver(v string) parsed {
	v = canon(v)
	if v == "" {
		return parsed{}
	}
	core, pre, _ := strings.Cut(v, "-")
	core, _, _ = strings.Cut(core, "+")
	parts := strings.Split(core, ".")
	if len(parts) < 1 {
		return parsed{}
	}
	nums := [3]int{}
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return parsed{}
		}
		nums[i] = n
	}
	return parsed{major: nums[0], minor: nums[1], patch: nums[2], pre: pre, ok: true}
}

// CompareVersions 在 a 小于 b 时返回 -1，相等返回 0，a 大于 b 返回 1。两边都解析失败时按去掉 v 后的字符串比较。只有 a 失败返回 -1，只有 b 失败返回 1。没有预发布段的版本大于有预发布段的版本。
func CompareVersions(a, b string) int {
	pa, pb := parseSemver(a), parseSemver(b)
	if !pa.ok && !pb.ok {
		return strings.Compare(canon(a), canon(b))
	}
	if !pa.ok {
		return -1
	}
	if !pb.ok {
		return 1
	}
	if pa.major != pb.major {
		return cmpInt(pa.major, pb.major)
	}
	if pa.minor != pb.minor {
		return cmpInt(pa.minor, pb.minor)
	}
	if pa.patch != pb.patch {
		return cmpInt(pa.patch, pb.patch)
	}
	if pa.pre == pb.pre {
		return 0
	}
	if pa.pre == "" {
		return 1
	}
	if pb.pre == "" {
		return -1
	}
	return strings.Compare(pa.pre, pb.pre)
}

// cmpInt 比较两个整数，小于返回 -1，大于返回 1，相等返回 0。
func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// IsNewer 在 remote 去掉空白后非空，且 CompareVersions 认为它大于 current 时返回 true。
func IsNewer(remote, current string) bool {
	if strings.TrimSpace(remote) == "" {
		return false
	}
	return CompareVersions(remote, current) > 0
}
