// 从 GitHub 的 git tree 拉取技能目录。单文件超过 1 MiB、整包超过 16 MiB 或文件数达到 400 即失败。

package skillpack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	maxPackFiles = 400
	maxPackBytes = 16 << 20
	maxFileBytes = 1 << 20
)

// Get 拉取一个 URL 的正文。测试可以替换它。
var Get = defaultGet

// PackFile 是包里的一个相对路径和它的字节。
type PackFile struct {
	Rel  string
	Data []byte
}

type ghTree struct {
	SHA  string `json:"sha"`
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"tree"`
}

// fetchGitHub 用 api.github.com 的递归 tree 列出文件，再从 raw.githubusercontent.com 下载。repo 必须是 owner/name。ref 空则用 main，skillsRel 空则用 skills。跳过不通过 validRel 的路径。单文件超过 1 MiB 返回 "<rel> larger than 1MiB"。总字节超过 16 MiB 返回 "pack larger than 16MiB"。文件数达到 400 返回 "pack has more than 400 files"。没有 SKILL.md 返回 "pack has no SKILL.md"。成功时第二个返回值是树的 SHA。
func fetchGitHub(repo, ref, skillsRel string) ([]PackFile, string, error) {
	repo = strings.Trim(strings.ReplaceAll(repo, "\\", "/"), "/")
	if repo == "" || strings.Count(repo, "/") != 1 {
		return nil, "", fmt.Errorf("invalid github repo")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = "main"
	}
	skillsRel = strings.Trim(strings.ReplaceAll(skillsRel, "\\", "/"), "/")
	if skillsRel == "" {
		skillsRel = "skills"
	}
	treeURL := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", repo, url.PathEscape(ref))
	raw, err := Get(treeURL)
	if err != nil {
		return nil, "", err
	}
	var tree ghTree
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, "", fmt.Errorf("github tree: %w", err)
	}
	prefix := skillsRel + "/"
	var files []PackFile
	total := 0
	for _, n := range tree.Tree {
		if n.Type != "blob" {
			continue
		}
		p := strings.ReplaceAll(n.Path, "\\", "/")
		if p != skillsRel && !strings.HasPrefix(p, prefix) {
			continue
		}
		rel := strings.TrimPrefix(p, prefix)
		if p == skillsRel {
			continue
		}
		if !validRel(rel) {
			continue
		}
		if n.Size > maxFileBytes {
			return nil, "", fmt.Errorf("%s larger than 1MiB", rel)
		}
		data, err := Get(rawGitHubURL(repo, ref, p))
		if err != nil {
			return nil, "", fmt.Errorf("%s: %w", rel, err)
		}
		if len(data) > maxFileBytes {
			return nil, "", fmt.Errorf("%s larger than 1MiB", rel)
		}
		if err := scanFile(rel, data); err != nil {
			return nil, "", fmt.Errorf("%s: %w", rel, err)
		}
		total += len(data)
		if total > maxPackBytes {
			return nil, "", fmt.Errorf("pack larger than 16MiB")
		}
		if len(files) >= maxPackFiles {
			return nil, "", fmt.Errorf("pack has more than %d files", maxPackFiles)
		}
		files = append(files, PackFile{Rel: rel, Data: data})
	}
	if !packHasSkillMD(files) {
		return nil, "", fmt.Errorf("pack has no SKILL.md")
	}
	return files, tree.SHA, nil
}

// rawGitHubURL 拼出 raw.githubusercontent.com 上的文件地址。
func rawGitHubURL(repo, ref, path string) string {
	return "https://raw.githubusercontent.com/" + repo + "/" + ref + "/" + strings.TrimPrefix(path, "/")
}

// defaultGet 用 45 秒超时 GET。User-Agent 是 Yoyo-Workstation。YOYO_GITHUB_TOKEN 或 GITHUB_TOKEN 非空时带 Bearer。api.github.com 再加 Accept application/vnd.github+json 和 X-GitHub-Api-Version 2022-11-28。状态码大于等于 300 返回 "pack fetch: " 加状态。正文最多读到 16 MiB 再多 1 字节。
func defaultGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Yoyo-Workstation")
	if tok := githubAuthToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if strings.Contains(rawURL, "api.github.com") {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pack fetch: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxPackBytes+1))
}

// githubAuthToken 先读 YOYO_GITHUB_TOKEN，再读 GITHUB_TOKEN。都空则返回空串。
func githubAuthToken() string {
	for _, k := range []string{"YOYO_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// packHasSkillMD 在任一文件的基名是 skill.md（不区分大小写）时返回 true。
func packHasSkillMD(files []PackFile) bool {
	for _, f := range files {
		if strings.EqualFold(pathBase(f.Rel), "skill.md") {
			return true
		}
	}
	return false
}

// pathBase 返回相对路径的最后一段。
func pathBase(rel string) string {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[i+1:]
	}
	return rel
}
