// platform.go 把 GOOS/GOARCH 映射成注册表的平台键，并按平台挑二进制归档、判断一项在本机能不能用。
// 测试注入 Platform，不读真实系统。

package buddy

import (
	"path/filepath"
	"runtime"
	"slices"
)

// Platform 是本机的系统、架构和 Linux 的 libc。
type Platform struct {
	GOOS   string
	GOARCH string
	Musl   bool
}

// currentPlatform 返回本机平台。Linux 上用 /lib/ld-musl-* 判断是不是 musl。
func currentPlatform() Platform {
	p := Platform{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	if p.GOOS == "linux" {
		matches, _ := filepath.Glob("/lib/ld-musl-*.so.1")
		p.Musl = len(matches) > 0
	}
	return p
}

// Key 返回注册表的平台键，例如 windows-x86_64、darwin-aarch64、linux-x86_64-musl。
func (p Platform) Key() string {
	arch := p.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}
	key := p.GOOS + "-" + arch
	if p.GOOS == "linux" && p.Musl {
		key += "-musl"
	}
	return key
}

// pickTarget 挑本平台的二进制归档。windows-aarch64 没有对应包时回落到 windows-x86_64（系统自带转译）。
func pickTarget(bin map[string]BinaryTarget, p Platform) (BinaryTarget, string, bool) {
	key := p.Key()
	if t, ok := bin[key]; ok && t.Archive != "" {
		return t, key, true
	}
	if key == "windows-aarch64" {
		if t, ok := bin["windows-x86_64"]; ok && t.Archive != "" {
			return t, "windows-x86_64", true
		}
	}
	return BinaryTarget{}, "", false
}

// declaredUnsupported 在精选表把本系统或本平台键列为不支持时为真。
func declaredUnsupported(e Entry, p Platform) bool {
	return slices.Contains(e.Unsupported, p.GOOS) || slices.Contains(e.Unsupported, p.Key())
}

// installable 判断 e 在本机有没有托管安装的办法：本平台的二进制、npx 或 uvx。
func installable(e Entry, p Platform) bool {
	if declaredUnsupported(e, p) {
		return false
	}
	if _, _, ok := pickTarget(e.Dist.Binary, p); ok {
		return true
	}
	return e.Dist.NPX != nil || e.Dist.UVX != nil
}
