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

// Get fetches a URL. Tests may override.
var Get = defaultGet

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

func rawGitHubURL(repo, ref, path string) string {
	return "https://raw.githubusercontent.com/" + repo + "/" + ref + "/" + strings.TrimPrefix(path, "/")
}

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

func githubAuthToken() string {
	for _, k := range []string{"YOYO_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func packHasSkillMD(files []PackFile) bool {
	for _, f := range files {
		if strings.EqualFold(pathBase(f.Rel), "skill.md") {
			return true
		}
	}
	return false
}

func pathBase(rel string) string {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[i+1:]
	}
	return rel
}
