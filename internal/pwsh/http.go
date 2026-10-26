// 本文件拼下载地址并创建超时 25 分钟的 HTTP 客户端。候选顺序由 candidateURLs 固定。

package pwsh

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// newHTTP 返回带 cookie jar、超时 25 分钟的客户端。建 jar 的错误被丢掉。
func newHTTP() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 25 * time.Minute, Jar: jar}
}

// repoSpec 返回站点、仓库和修订。默认是 https://www.modelscope.cn、NanoModel/ffmpeg、master。
// YOYO_PWSH_ENDPOINT、YOYO_PWSH_REPO、YOYO_PWSH_REVISION 非空时覆盖。站点和仓库会去掉末尾斜杠。
func repoSpec() (endpoint, repo, rev string) {
	endpoint = strings.TrimRight(defaultEndpoint, "/")
	repo = defaultRepo
	rev = defaultRevision
	if v := strings.TrimSpace(os.Getenv(EnvEndpoint)); v != "" {
		endpoint = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv(EnvRepo)); v != "" {
		repo = strings.Trim(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv(EnvRevision)); v != "" {
		rev = v
	}
	return endpoint, repo, rev
}

// referer 是站点去掉末尾斜杠后再加一个斜杠。
func referer(endpoint string) string {
	return strings.TrimRight(endpoint, "/") + "/"
}

// remotePath 只留压缩包文件名，再前面加上前缀。默认前缀是 ps7。YOYO_PWSH_PREFIX 非空时覆盖。前缀为空或 . 时只返回文件名。
func remotePath(archive string) string {
	archive = strings.TrimPrefix(strings.ReplaceAll(archive, "\\", "/"), "/")
	if i := strings.LastIndex(archive, "/"); i >= 0 {
		archive = archive[i+1:]
	}
	prefix := strings.Trim(defaultPrefix, "/")
	if v := strings.TrimSpace(os.Getenv(EnvPrefix)); v != "" {
		prefix = strings.Trim(v, "/")
	}
	if prefix == "" || prefix == "." {
		return archive
	}
	return prefix + "/" + archive
}

// resolveURL 拼出 <endpoint>/datasets/<repo>/resolve/<rev>/<file>。站点字符串含 modelscope. 且地址还没有问号时，末尾加上 ?view=false。
func resolveURL(endpoint, repo, rev, file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	u := strings.TrimRight(endpoint, "/") + "/datasets/" + repo + "/resolve/" + rev + "/" + file
	// ModelScope 数据集页把不带参数的 resolve 地址当成预览。view=false 才要 zip 字节。
	if strings.Contains(endpoint, "modelscope.") && !strings.Contains(u, "?") {
		u += "?view=false"
	}
	return u
}

// apiFileURL 拼出 <endpoint>/api/v1/datasets/<repo>/repo，查询参数 Revision 和 FilePath。站点解析失败返回空字符串。
func apiFileURL(endpoint, repo, rev, file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	u, err := url.Parse(strings.TrimRight(endpoint, "/") + "/api/v1/datasets/" + repo + "/repo")
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("Revision", rev)
	q.Set("FilePath", file)
	u.RawQuery = q.Encode()
	return u.String()
}

// candidateURLs 按这个顺序收集地址：非空的 YOYO_PWSH_URL、resolveURL、apiFileURL、githubURL，以及 https://ghfast.top/ 加上同一个 GitHub 地址。空字符串和重复项丢掉。
func (m *Manager) candidateURLs(archive string) []string {
	var out []string
	if v := strings.TrimSpace(os.Getenv(EnvURL)); v != "" {
		out = append(out, v)
	}
	endpoint, repo, rev := repoSpec()
	file := remotePath(archive)
	if u := resolveURL(endpoint, repo, rev, file); u != "" {
		out = append(out, u)
	}
	if u := apiFileURL(endpoint, repo, rev, file); u != "" {
		out = append(out, u)
	}
	if u := githubURL(archive); u != "" {
		out = append(out, u, "https://ghfast.top/"+u)
	}
	seen := map[string]bool{}
	uniq := make([]string, 0, len(out))
	for _, u := range out {
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		uniq = append(uniq, u)
	}
	return uniq
}
