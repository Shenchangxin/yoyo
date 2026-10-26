// 本文件钉住 PowerShell 版本、压缩包上限、环境变量和官方 SHA-256。下载顺序在 http.go。

package pwsh

const (
	// Version 是智能体 shell 钉住的 PowerShell 版本，值是 7.6.7。
	Version = "7.6.7"

	// maxZipBytes 是单个 zip 的字节上限，200<<20。downloadURL 读过这个数就失败。extractZip 里单个条目的 UncompressedSize64 超过它也失败。
	maxZipBytes = 200 << 20
	// maxUncompressedBytes 是解压后累计字节上限，512<<20。extractZip 累加超过它就失败。
	maxUncompressedBytes = 512 << 20
	// userAgent 是下载请求的 User-Agent。
	userAgent = "Mozilla/5.0 (compatible; Yoyo-Workstation)"

	// EnvBin 是环境变量 YOYO_PWSH。Look 优先用它指向的文件或目录。
	EnvBin = "YOYO_PWSH"
	// EnvEndpoint 是环境变量 YOYO_PWSH_ENDPOINT，覆盖默认的 ModelScope 站点。
	EnvEndpoint = "YOYO_PWSH_ENDPOINT"
	// EnvRepo 是环境变量 YOYO_PWSH_REPO，覆盖默认数据集仓库。
	EnvRepo = "YOYO_PWSH_REPO"
	// EnvRevision 是环境变量 YOYO_PWSH_REVISION，覆盖默认修订。
	EnvRevision = "YOYO_PWSH_REVISION"
	// EnvPrefix 是环境变量 YOYO_PWSH_PREFIX，覆盖数据集里的目录前缀。
	EnvPrefix = "YOYO_PWSH_PREFIX"
	// EnvURL 是环境变量 YOYO_PWSH_URL。非空时 candidateURLs 把它放在第一位。
	EnvURL = "YOYO_PWSH_URL"
)

const (
	// defaultEndpoint 是 ModelScope 站点 https://www.modelscope.cn。
	defaultEndpoint = "https://www.modelscope.cn"
	// defaultRepo 是数据集仓库 NanoModel/ffmpeg。
	defaultRepo = "NanoModel/ffmpeg"
	// defaultRevision 是修订 master。
	defaultRevision = "master"
	// defaultPrefix 是数据集路径前缀 ps7。
	defaultPrefix = "ps7"
)

// officialSHA256 是两个官方 zip 的 SHA-256。键是 archiveName 返回的文件名。wantSHA 为空时 downloadURL 不校验。
var officialSHA256 = map[string]string{
	"PowerShell-7.6.7-win-x64.zip":   "d2f952ebdbf24386cb8923553aa58e41a846db35a6155ea04c58142a4b16dbb7",
	"PowerShell-7.6.7-win-arm64.zip": "119bd9221254ef3637523cb1990797618f5fd68848b57de40f12b561d743149e",
}

// archiveName 只为 windows/amd64 返回 PowerShell-<Version>-win-x64.zip，为 windows/arm64 返回 PowerShell-<Version>-win-arm64.zip。其他组合返回 errUnsupported。
func archiveName(goos, goarch string) (string, error) {
	if goos != "windows" {
		return "", errUnsupported(goos, goarch)
	}
	switch goarch {
	case "amd64":
		return "PowerShell-" + Version + "-win-x64.zip", nil
	case "arm64":
		return "PowerShell-" + Version + "-win-arm64.zip", nil
	default:
		return "", errUnsupported(goos, goarch)
	}
}

// githubURL 拼出 https://github.com/PowerShell/PowerShell/releases/download/v<Version>/<archive>。
func githubURL(archive string) string {
	return "https://github.com/PowerShell/PowerShell/releases/download/v" + Version + "/" + archive
}
