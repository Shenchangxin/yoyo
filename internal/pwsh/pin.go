package pwsh

const (
	// Version is the PowerShell 7 LTS dialect the agent shell is pinned to.
	Version = "7.6.7"

	maxZipBytes          = 200 << 20
	maxUncompressedBytes = 512 << 20
	userAgent            = "Mozilla/5.0 (compatible; Yoyo-Workstation)"

	EnvBin      = "YOYO_PWSH"
	EnvEndpoint = "YOYO_PWSH_ENDPOINT"
	EnvRepo     = "YOYO_PWSH_REPO"
	EnvRevision = "YOYO_PWSH_REVISION"
	EnvPrefix   = "YOYO_PWSH_PREFIX"
	EnvURL      = "YOYO_PWSH_URL"
)

const (
	defaultEndpoint = "https://www.modelscope.cn"
	defaultRepo     = "NanoModel/ffmpeg"
	defaultRevision = "master"
	defaultPrefix   = "ps7"
)

var officialSHA256 = map[string]string{
	"PowerShell-7.6.7-win-x64.zip":   "d2f952ebdbf24386cb8923553aa58e41a846db35a6155ea04c58142a4b16dbb7",
	"PowerShell-7.6.7-win-arm64.zip": "119bd9221254ef3637523cb1990797618f5fd68848b57de40f12b561d743149e",
}

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

func githubURL(archive string) string {
	return "https://github.com/PowerShell/PowerShell/releases/download/v" + Version + "/" + archive
}
