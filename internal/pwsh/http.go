package pwsh

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

func newHTTP() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 25 * time.Minute, Jar: jar}
}

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

func referer(endpoint string) string {
	return strings.TrimRight(endpoint, "/") + "/"
}

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

func resolveURL(endpoint, repo, rev, file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	u := strings.TrimRight(endpoint, "/") + "/datasets/" + repo + "/resolve/" + rev + "/" + file
	// ModelScope's dataset page treats a bare resolve URL as a preview;
	// view=false forces the zip bytes.
	if strings.Contains(endpoint, "modelscope.") && !strings.Contains(u, "?") {
		u += "?view=false"
	}
	return u
}

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
