// 拼 ModelScope 数据集上的清单地址和平台键。

package update

import (
	"net/url"
	"os"
	"runtime"
	"strings"
)

const (
	// DefaultEndpoint 是国际站 ModelScope，值为 https://www.modelscope.ai。中国站 modelscope.cn 是另一套目录，令牌和数据集不通用。
	DefaultEndpoint = "https://www.modelscope.ai"
	// DefaultRepo 是默认数据集 yoyo-agent/yoyo。
	DefaultRepo = "yoyo-agent/yoyo"
	// DefaultRevision 是默认修订 master。
	DefaultRevision = "master"
	// UpdaterUA 是下载请求的 User-Agent。
	UpdaterUA = "Yoyo-Updater"
	// UpdaterReferer 是默认 Referer。实际请求用 Referer 按当前 Endpoint 现算。
	UpdaterReferer = "https://www.modelscope.ai/"
)

const (
	// EnvEndpoint 是覆盖站点的环境变量名 YOYO_MODELSCOPE_ENDPOINT。
	EnvEndpoint = "YOYO_MODELSCOPE_ENDPOINT"
	// EnvRepo 是覆盖数据集的环境变量名 YOYO_MODELSCOPE_REPO。
	EnvRepo = "YOYO_MODELSCOPE_REPO"
	// EnvRevision 是覆盖修订的环境变量名 YOYO_MODELSCOPE_REVISION。
	EnvRevision = "YOYO_MODELSCOPE_REVISION"
)

// Endpoint 返回 YOYO_MODELSCOPE_ENDPOINT，空则用 DefaultEndpoint。结果会经过 normalizeEndpoint。
func Endpoint() string {
	if v := strings.TrimSpace(os.Getenv(EnvEndpoint)); v != "" {
		return normalizeEndpoint(v)
	}
	return DefaultEndpoint
}

// Referer 返回当前 Endpoint 去掉末尾斜杠后再加一个斜杠，供 resolve 下载使用。
func Referer() string {
	return strings.TrimRight(Endpoint(), "/") + "/"
}

// normalizeEndpoint 去掉末尾斜杠。主机名是 modelscope.ai 时改成 https://www.modelscope.ai。解析失败或没有主机则原样返回去掉斜杠后的字符串。
func normalizeEndpoint(raw string) string {
	raw = strings.TrimSpace(strings.TrimRight(raw, "/"))
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	if strings.EqualFold(u.Host, "modelscope.ai") {
		u.Scheme = "https"
		u.Host = "www.modelscope.ai"
		u.Path = strings.TrimRight(u.Path, "/")
		if u.Path == "/" {
			u.Path = ""
		}
		return u.String()
	}
	return raw
}

// RepoID 返回 YOYO_MODELSCOPE_REPO，空则用 DefaultRepo。两侧斜杠会被去掉。
func RepoID() string {
	if v := strings.TrimSpace(os.Getenv(EnvRepo)); v != "" {
		return strings.Trim(v, "/")
	}
	return DefaultRepo
}

// Revision 返回 YOYO_MODELSCOPE_REVISION，空则用 DefaultRevision。
func Revision() string {
	if v := strings.TrimSpace(os.Getenv(EnvRevision)); v != "" {
		return v
	}
	return DefaultRevision
}

// NormalizeChannel 忽略传入值，总是返回 DefaultChannel。
func NormalizeChannel(ch string) Channel {
	_ = ch
	return DefaultChannel
}

// FeedURL 忽略通道参数，返回 channels/latest.json 在当前数据集上的 resolve 地址。
func FeedURL(ch Channel) string {
	_ = ch
	return ResolveURL("channels/" + string(DefaultChannel) + ".json")
}

// ResolveURL 拼出 Endpoint/datasets/<repo>/resolve/<revision>/<file>。
func ResolveURL(file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	return Endpoint() + "/datasets/" + RepoID() + "/resolve/" + Revision() + "/" + file
}

// APIFileURL 拼出数据集 API：/api/v1/datasets/<repo>/repo，查询参数 Revision 和 FilePath。
func APIFileURL(file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	u, err := url.Parse(Endpoint() + "/api/v1/datasets/" + RepoID() + "/repo")
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("Revision", Revision())
	q.Set("FilePath", file)
	u.RawQuery = q.Encode()
	return u.String()
}

// SigURL 在清单地址末尾加上 .sig。
func SigURL(manifestURL string) string {
	return strings.TrimRight(manifestURL, "/") + ".sig"
}

// DesktopOS 在 YOYO_DESKTOP 或 YOYO_WORKER 为 "1" 时返回 true。
func DesktopOS() bool {
	return os.Getenv("YOYO_DESKTOP") == "1" || os.Getenv("YOYO_WORKER") == "1"
}

// DesktopPlatform 是安装包矩阵的键：windows 为 windows-amd64，darwin 为 darwin-universal，linux 为 linux-amd64，其余是 GOOS-GOARCH。
func DesktopPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "windows-amd64"
	case "darwin":
		return "darwin-universal"
	case "linux":
		return "linux-amd64"
	default:
		return runtime.GOOS + "-" + runtime.GOARCH
	}
}

// CLIPlatform 返回 GOOS-GOARCH-cli。
func CLIPlatform() string {
	return runtime.GOOS + "-" + runtime.GOARCH + "-cli"
}

// SelectAsset 在 desktop 为真时先找 DesktopPlatform。没有再找 CLIPlatform。再没有且清单自身 URL 非空时，返回 kind 为 bin、文件名为 yoyo.staging 的资源。都没有则返回 false。
func (m Manifest) SelectAsset(desktop bool) (Asset, bool) {
	if desktop {
		if a, ok := m.Platforms[DesktopPlatform()]; ok && a.URL != "" {
			return a, true
		}
	}
	if a, ok := m.Platforms[CLIPlatform()]; ok && a.URL != "" {
		return a, true
	}
	if m.URL != "" {
		return Asset{URL: m.URL, SHA256: m.SHA256, Kind: "bin", Filename: "yoyo.staging"}, true
	}
	return Asset{}, false
}
