package update

import (
	"net/url"
	"os"
	"runtime"
	"strings"
)

const (
	// International ModelScope (modelscope.ai). The China site (modelscope.cn)
	// is a separate catalog — tokens and datasets do not carry over.
	DefaultEndpoint = "https://www.modelscope.ai"
	DefaultRepo     = "yoyo-agent/yoyo"
	DefaultRevision = "master"
	UpdaterUA       = "Yoyo-Updater"
	UpdaterReferer  = "https://www.modelscope.ai/"
)

const (
	EnvEndpoint = "YOYO_MODELSCOPE_ENDPOINT"
	EnvRepo     = "YOYO_MODELSCOPE_REPO"
	EnvRevision = "YOYO_MODELSCOPE_REVISION"
)

func Endpoint() string {
	if v := strings.TrimSpace(os.Getenv(EnvEndpoint)); v != "" {
		return normalizeEndpoint(v)
	}
	return DefaultEndpoint
}

// Referer is the Origin-like header ModelScope's CDN expects on resolve downloads.
func Referer() string {
	return strings.TrimRight(Endpoint(), "/") + "/"
}

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

func RepoID() string {
	if v := strings.TrimSpace(os.Getenv(EnvRepo)); v != "" {
		return strings.Trim(v, "/")
	}
	return DefaultRepo
}

func Revision() string {
	if v := strings.TrimSpace(os.Getenv(EnvRevision)); v != "" {
		return v
	}
	return DefaultRevision
}

func NormalizeChannel(ch string) Channel {
	_ = ch
	return DefaultChannel
}

// FeedURL is the public latest JSON on ModelScope. Override with a full
// update_url in config; this is what an empty URL resolves to. Check still
// tries nightly.json / stable.json if latest.json is missing or empty.
func FeedURL(ch Channel) string {
	_ = ch
	return ResolveURL("channels/" + string(DefaultChannel) + ".json")
}

func ResolveURL(file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	return Endpoint() + "/datasets/" + RepoID() + "/resolve/" + Revision() + "/" + file
}

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

func SigURL(manifestURL string) string {
	return strings.TrimRight(manifestURL, "/") + ".sig"
}

func DesktopOS() bool {
	return os.Getenv("YOYO_DESKTOP") == "1" || os.Getenv("YOYO_WORKER") == "1"
}

// DesktopPlatform is the installer matrix key (darwin is always universal).
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

func CLIPlatform() string {
	return runtime.GOOS + "-" + runtime.GOARCH + "-cli"
}

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
