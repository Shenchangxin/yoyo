// sources.go 轮询 GitHub 议题和飞书会话。
// 没有凭证时状态是 unconfigured，不会假装已经同步。

package superagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const sourceEvery = 30 * time.Second

// SourceConfig 是两个来源的凭证和接口根。空的 GitHubAPI 用 https://api.github.com。空的 FeishuAPI 用 https://open.feishu.cn。
type SourceConfig struct {
	GitHubToken  string
	GitHubRepo   string
	GitHubAPI    string
	FeishuAppID  string
	FeishuSecret string
	FeishuChat   string
	FeishuAPI    string
}

// DefaultSources 从环境变量读凭证。GITHUB_TOKEN、GITHUB_REPO、FEISHU_APP_ID、FEISHU_APP_SECRET、FEISHU_CHAT_ID。
func DefaultSources() SourceConfig {
	return SourceConfig{
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		GitHubRepo:   os.Getenv("GITHUB_REPO"),
		GitHubAPI:    "https://api.github.com",
		FeishuAppID:  os.Getenv("FEISHU_APP_ID"),
		FeishuSecret: os.Getenv("FEISHU_APP_SECRET"),
		FeishuChat:   os.Getenv("FEISHU_CHAT_ID"),
		FeishuAPI:    "https://open.feishu.cn",
	}
}

// poll 到点就拉两个来源。now 用来和上次成功或失败的时间比较，不到 sourceEvery 就跳过。
func (rt *Runtime) poll(now time.Time) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	cfg := rt.sources
	lastGitHub, lastFeishu := rt.lastGitHub, rt.lastFeishu
	rt.mu.Unlock()
	if cfg.GitHubToken == "" || cfg.GitHubRepo == "" {
		rt.setSource("github", "unconfigured", "需要 GITHUB_TOKEN 和 GITHUB_REPO")
	} else if lastGitHub.IsZero() || now.Sub(lastGitHub) >= sourceEvery {
		rt.pollGitHub(cfg)
		rt.mu.Lock()
		rt.lastGitHub = time.Now()
		rt.mu.Unlock()
	}
	if cfg.FeishuAppID == "" || cfg.FeishuSecret == "" {
		rt.setSource("feishu", "unconfigured", "需要 FEISHU_APP_ID 和 FEISHU_APP_SECRET")
	} else if cfg.FeishuChat == "" {
		rt.setSource("feishu", "unconfigured", "需要 FEISHU_CHAT_ID")
	} else if lastFeishu.IsZero() || now.Sub(lastFeishu) >= sourceEvery {
		rt.pollFeishu(cfg)
		rt.mu.Lock()
		rt.lastFeishu = time.Now()
		rt.mu.Unlock()
	}
}

// pollGitHub 拉取 cfg 里那个仓库的开放议题，并收成触发。cfg 的令牌或仓库为空时不请求。
func (rt *Runtime) pollGitHub(cfg SourceConfig) {
	repo := strings.Trim(strings.TrimSpace(cfg.GitHubRepo), "/")
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		rt.setSource("github", "error", "GITHUB_REPO 应为 owner/name")
		return
	}
	root := strings.TrimRight(cfg.GitHubAPI, "/")
	endpoint := root + "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/issues?state=open&per_page=20"
	body, err := rt.request(http.MethodGet, endpoint, cfg.GitHubToken, nil, "application/vnd.github+json")
	if err != nil {
		rt.setSource("github", "error", clip(err.Error(), 180))
		return
	}
	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &issues); err != nil {
		rt.setSource("github", "error", clip(string(body), 180))
		return
	}
	n := 0
	for _, issue := range issues {
		if issue.Number <= 0 {
			continue
		}
		text := strings.TrimSpace(issue.Title + "\n" + issue.Body)
		thread := "github:" + repo
		rt.mu.Lock()
		rt.eng.Receive(Trigger{
			ID:        fmt.Sprintf("github:%s#%d", repo, issue.Number),
			Source:    "github",
			Threads:   []string{thread},
			Primary:   thread,
			Human:     true,
			Addressed: strings.Contains(text, "@"),
			Sender:    issue.User.Login,
			Text:      clip(text, 4000),
		})
		rt.mu.Unlock()
		n++
	}
	rt.setSource("github", "ok", fmt.Sprintf("开放议题 %d", n))
}

// pollFeishu 用 cfg 换租户令牌，再读指定会话的消息。凭证或会话为空时不请求。
func (rt *Runtime) pollFeishu(cfg SourceConfig) {
	root := strings.TrimRight(cfg.FeishuAPI, "/")
	tokenBody, err := rt.request(http.MethodPost, root+"/open-apis/auth/v3/tenant_access_token/internal", "", map[string]string{
		"app_id": cfg.FeishuAppID, "app_secret": cfg.FeishuSecret,
	}, "application/json")
	if err != nil {
		rt.setSource("feishu", "error", clip(err.Error(), 180))
		return
	}
	var tok struct {
		Code  int    `json:"code"`
		Token string `json:"tenant_access_token"`
		Msg   string `json:"msg"`
	}
	if err := json.Unmarshal(tokenBody, &tok); err != nil || tok.Code != 0 || tok.Token == "" {
		detail := tok.Msg
		if detail == "" {
			detail = clip(string(tokenBody), 180)
		}
		rt.setSource("feishu", "error", detail)
		return
	}
	query := url.Values{}
	query.Set("container_id_type", "chat")
	query.Set("container_id", cfg.FeishuChat)
	body, err := rt.request(http.MethodGet, root+"/open-apis/im/v1/messages?"+query.Encode(), tok.Token, nil, "application/json")
	if err != nil {
		rt.setSource("feishu", "error", clip(err.Error(), 180))
		return
	}
	var page struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				MessageID string `json:"message_id"`
				Body      struct {
					Content string `json:"content"`
				} `json:"body"`
				Sender struct {
					ID         string `json:"id"`
					SenderType string `json:"sender_type"`
				} `json:"sender"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &page); err != nil || page.Code != 0 {
		detail := page.Msg
		if detail == "" {
			detail = clip(string(body), 180)
		}
		rt.setSource("feishu", "error", detail)
		return
	}
	n := 0
	thread := "feishu:" + cfg.FeishuChat
	for _, item := range page.Data.Items {
		if item.MessageID == "" || item.Sender.SenderType == "app" {
			continue
		}
		text := feishuText(item.Body.Content)
		rt.mu.Lock()
		rt.eng.Receive(Trigger{
			ID:        "feishu:" + item.MessageID,
			Source:    "feishu",
			Threads:   []string{thread},
			Primary:   thread,
			Human:     true,
			Addressed: strings.Contains(text, "@"),
			Sender:    item.Sender.ID,
			Text:      clip(text, 4000),
		})
		rt.mu.Unlock()
		n++
	}
	rt.setSource("feishu", "ok", fmt.Sprintf("消息 %d", n))
}

// feishuText 从 content 里取出文本。content 是 JSON 且带 text 时用那段，否则原样返回。
func feishuText(content string) string {
	var body struct {
		Text string `json:"text"`
	}
	if json.Unmarshal([]byte(content), &body) == nil && body.Text != "" {
		return body.Text
	}
	return content
}

// request 发送 method 到 endpoint。token 非空时带 Bearer。payload 非空时编码成 JSON。accept 写入 Accept。
func (rt *Runtime) request(method, endpoint, token string, payload any, accept string) ([]byte, error) {
	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "yoyo")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := rt.client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d %s", res.StatusCode, clip(string(body), 160))
	}
	return body, nil
}

// setSource 记下 name 的 state 和 detail。name 为空时什么都不做。
func (rt *Runtime) setSource(name, state, detail string) {
	if rt == nil || name == "" {
		return
	}
	rt.mu.Lock()
	if rt.status == nil {
		rt.status = map[string]SourceState{}
	}
	rt.status[name] = SourceState{Name: name, State: state, Detail: detail}
	rt.mu.Unlock()
}
