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
	"strconv"
	"strings"
	"time"
)

const sourceEvery = 30 * time.Second

// githubThread 是一个议题或拉取请求自己的线程。同一仓库的两件工作不共用线程。
func githubThread(repo string, number int, pull bool) string {
	if pull {
		return fmt.Sprintf("github:%s!%d", repo, number)
	}
	return fmt.Sprintf("github:%s#%d", repo, number)
}

// feishuThread 是一条话题。root 是话题根，没有时用 message。两者都空时退回整个会话。
func feishuThread(chat, root, message string) string {
	key := strings.TrimSpace(root)
	if key == "" {
		key = strings.TrimSpace(message)
	}
	if key == "" {
		return "feishu:" + chat
	}
	return "feishu:" + chat + ":" + key
}

// mentions 判断 text 有没有点到 account。account 为空时，任意 @ 都算点名。
func mentions(text, account string) bool {
	account = strings.TrimPrefix(strings.TrimSpace(account), "@")
	if account == "" {
		return strings.Contains(text, "@")
	}
	return strings.Contains(strings.ToLower(text), "@"+strings.ToLower(account))
}

// SourceConfig 是两个来源的凭证和接口根。空的 GitHubAPI 用 https://api.github.com。空的 FeishuAPI 用 https://open.feishu.cn。
// GitHubAccount 和 FeishuMention 是被点名的名字，不含 @。空着时，正文里任意 @ 都算点名。
type SourceConfig struct {
	GitHubToken   string
	GitHubRepo    string
	GitHubAPI     string
	GitHubAccount string
	FeishuAppID   string
	FeishuSecret  string
	FeishuChat    string
	FeishuAPI     string
	FeishuMention string
}

// DefaultSources 从环境变量读凭证。GITHUB_TOKEN、GITHUB_REPO、GITHUB_ACCOUNT、FEISHU_APP_ID、FEISHU_APP_SECRET、FEISHU_CHAT_ID、FEISHU_MENTION。
func DefaultSources() SourceConfig {
	return SourceConfig{
		GitHubToken:   os.Getenv("GITHUB_TOKEN"),
		GitHubRepo:    os.Getenv("GITHUB_REPO"),
		GitHubAPI:     "https://api.github.com",
		GitHubAccount: os.Getenv("GITHUB_ACCOUNT"),
		FeishuAppID:   os.Getenv("FEISHU_APP_ID"),
		FeishuSecret:  os.Getenv("FEISHU_APP_SECRET"),
		FeishuChat:    os.Getenv("FEISHU_CHAT_ID"),
		FeishuAPI:     "https://open.feishu.cn",
		FeishuMention: os.Getenv("FEISHU_MENTION"),
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
	rt.mu.Lock()
	intake := rt.intake
	lastMail := rt.lastMail
	rt.mu.Unlock()
	if intake != nil && (lastMail.IsZero() || now.Sub(lastMail) >= sourceEvery) {
		for _, tr := range intake() {
			if tr.ID == "" || strings.TrimSpace(tr.Text) == "" {
				continue
			}
			rt.eng.Receive(tr)
		}
		rt.mu.Lock()
		rt.lastMail = time.Now()
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
		Number      int             `json:"number"`
		Title       string          `json:"title"`
		Body        string          `json:"body"`
		PullRequest json.RawMessage `json:"pull_request"`
		User        struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &issues); err != nil {
		rt.setSource("github", "error", clip(string(body), 180))
		return
	}
	issuesN, prsN := 0, 0
	for _, issue := range issues {
		if issue.Number <= 0 {
			continue
		}
		text := strings.TrimSpace(issue.Title + "\n" + issue.Body)
		pull := len(issue.PullRequest) > 0 && string(issue.PullRequest) != "null"
		thread := githubThread(repo, issue.Number, pull)
		id := fmt.Sprintf("github:%s#%d", repo, issue.Number)
		if pull {
			id = fmt.Sprintf("github:%s!pr#%d", repo, issue.Number)
			prsN++
		} else {
			issuesN++
		}
		rt.mu.Lock()
		rt.eng.Receive(Trigger{
			ID:        id,
			Source:    "github",
			Channel:   "github",
			Ref:       strconv.Itoa(issue.Number),
			Threads:   []string{thread},
			Primary:   thread,
			Human:     personActor(issue.User.Login, issue.User.Type),
			Addressed: mentions(text, cfg.GitHubAccount),
			Sender:    issue.User.Login,
			Text:      clip(text, 4000),
		})
		rt.mu.Unlock()
	}
	rt.pollGitHubComments(cfg, root, parts[0], parts[1], repo, issues)
	rt.pollKeptThreads(cfg, root, parts[0], parts[1], repo, issues)
	rt.pollReviewThreads(cfg, root, parts[0], parts[1], repo)
	rt.pollCheckThreads(cfg, root, parts[0], parts[1], repo)
	rt.bindOwnedPulls(cfg, root, parts[0], parts[1], repo, issues)
	rt.pollFollowedPulls(cfg, root, parts[0], parts[1], repo)
	rt.mu.Lock()
	rt.commentSince = time.Now().Add(-time.Minute)
	rt.mu.Unlock()
	if thread := rt.readTrouble(); thread != "" {
		rt.setSource("github", "error", thread+" 连续三次没读到")
		return
	}
	rt.setSource("github", "ok", fmt.Sprintf("议题 %d，拉取请求 %d", issuesN, prsN))
}

// bindOwnedPulls 让工单跟随自己短名对应的拉取请求。一次最多看四条。
func (rt *Runtime) bindOwnedPulls(cfg SourceConfig, root, owner, name, repo string, issues []struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	PullRequest json.RawMessage `json:"pull_request"`
	User        struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}) {
	if rt == nil || rt.eng == nil {
		return
	}
	seen := 0
	for _, issue := range issues {
		pull := len(issue.PullRequest) > 0 && string(issue.PullRequest) != "null"
		if !pull || issue.Number <= 0 {
			continue
		}
		if seen >= 4 {
			break
		}
		seen++
		endpoint := root + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/pulls/" + strconv.Itoa(issue.Number)
		body, err := rt.request(http.MethodGet, endpoint, cfg.GitHubToken, nil, "application/vnd.github+json")
		thread := githubThread(repo, issue.Number, true)
		if err != nil {
			rt.noteReadMiss(thread)
			continue
		}
		rt.clearReadMiss(thread)
		var got struct {
			Head struct {
				Ref string `json:"ref"`
			} `json:"head"`
		}
		if json.Unmarshal(body, &got) != nil {
			continue
		}
		short := got.Head.Ref
		if i := strings.LastIndex(short, "/"); i >= 0 {
			short = short[i+1:]
		}
		rt.eng.ClaimBranch(thread, short)
	}
}

// noteReadMiss 记下这条线程又没读到。满三次后由 readTrouble 报出来。
func (rt *Runtime) noteReadMiss(thread string) {
	if rt == nil || thread == "" {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.readMiss == nil {
		rt.readMiss = map[string]int{}
	}
	rt.readMiss[thread]++
}

// clearReadMiss 清掉这条线程的失败次数。
func (rt *Runtime) clearReadMiss(thread string) {
	if rt == nil || thread == "" {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.readMiss, thread)
}

// readTrouble 返回连续三次没读到的线程。没有时返回空串。
func (rt *Runtime) readTrouble() string {
	if rt == nil {
		return ""
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for thread, n := range rt.readMiss {
		if n >= 3 {
			return thread
		}
	}
	return ""
}

// pollFollowedPulls 查看工单正在跟随、却不在这一页开放列表里的拉取请求。关闭或合并时结束跟随。
func (rt *Runtime) pollFollowedPulls(cfg SourceConfig, root, owner, name, repo string) {
	if rt == nil || rt.eng == nil {
		return
	}
	var threads []string
	for _, thread := range rt.eng.FollowThreads() {
		prefix := "github:" + repo + "!"
		if strings.HasPrefix(thread, prefix) {
			threads = append(threads, thread)
		}
	}
	if len(threads) > 4 {
		threads = threads[:4]
	}
	for _, thread := range threads {
		number := strings.TrimPrefix(thread, "github:"+repo+"!")
		if number == "" {
			continue
		}
		endpoint := root + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/pulls/" + url.PathEscape(number)
		body, err := rt.request(http.MethodGet, endpoint, cfg.GitHubToken, nil, "application/vnd.github+json")
		if err != nil {
			rt.noteReadMiss(thread)
			continue
		}
		rt.clearReadMiss(thread)
		var pull struct {
			State  string `json:"state"`
			Merged bool   `json:"merged"`
		}
		if json.Unmarshal(body, &pull) != nil || pull.State != "closed" {
			continue
		}
		outcome := "closed"
		if pull.Merged {
			outcome = "merged"
		}
		rt.noteClose(thread, outcome)
		rt.eng.ThreadClosed(thread, outcome, time.Now().UTC().Format(time.RFC3339))
	}
}

// pollGitHubComments 补拉最多八个议题或拉取请求的评论。评论进那一件工作自己的线程。单条请求失败不影响议题本身。
func (rt *Runtime) pollGitHubComments(cfg SourceConfig, root, owner, name, repo string, issues []struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	PullRequest json.RawMessage `json:"pull_request"`
	User        struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}) {
	fetched := 0
	for _, issue := range issues {
		if issue.Number <= 0 {
			continue
		}
		if fetched >= 8 {
			break
		}
		fetched++
		pull := len(issue.PullRequest) > 0 && string(issue.PullRequest) != "null"
		thread := githubThread(repo, issue.Number, pull)
		rt.fetchComments(cfg, root, owner, name, repo, issue.Number, thread)
	}
}

// pollKeptThreads 继续读已经不在开放列表里、但两周内仍有订阅的议题和拉取请求。一次最多补四条。
func (rt *Runtime) pollKeptThreads(cfg SourceConfig, root, owner, name, repo string, issues []struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	PullRequest json.RawMessage `json:"pull_request"`
	User        struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}) {
	if rt == nil || rt.eng == nil {
		return
	}
	seen := map[int]bool{}
	for _, issue := range issues {
		if issue.Number > 0 {
			seen[issue.Number] = true
		}
	}
	extra := 0
	for _, thread := range rt.eng.PollableThreads(time.Now()) {
		number, ok := githubNumber(thread, repo)
		if !ok || seen[number] {
			continue
		}
		if extra >= 4 {
			break
		}
		extra++
		seen[number] = true
		rt.fetchComments(cfg, root, owner, name, repo, number, thread)
	}
}

// githubNumber 从 thread 取出这个仓库的议题或拉取请求编号。对不上时 ok 为假。
func githubNumber(thread, repo string) (int, bool) {
	rest := strings.TrimPrefix(thread, "github:"+repo)
	if rest == thread || rest == "" {
		return 0, false
	}
	if rest[0] != '#' && rest[0] != '!' {
		return 0, false
	}
	number, err := strconv.Atoi(rest[1:])
	return number, err == nil && number > 0
}

// personActor 在登录名是机器人时返回假。GitHub 的 Bot 类型和 [bot] 后缀都不算人。
func personActor(login, kind string) bool {
	login = strings.ToLower(strings.TrimSpace(login))
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "bot" || strings.HasSuffix(login, "[bot]") {
		return false
	}
	return login != ""
}

// sinceFor 返回 thread 自己的读游标，没有时退回全局评论游标。游标往前重叠五秒。
func (rt *Runtime) sinceFor(thread string) time.Time {
	if rt == nil {
		return time.Time{}
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	raw := ""
	if rt.cursors != nil {
		raw = rt.cursors[thread]
	}
	if raw == "" {
		return rt.commentSince
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return rt.commentSince
	}
	return at.Add(-5 * time.Second)
}

// markCursor 把 thread 的读游标写成 at。
func (rt *Runtime) markCursor(thread string, at time.Time) {
	if rt == nil || thread == "" || at.IsZero() {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.cursors == nil {
		rt.cursors = map[string]string{}
	}
	rt.cursors[thread] = at.UTC().Format(time.RFC3339)
}

// pollReviewThreads 只读还在看的拉取请求上的审阅评论。每条审阅有自己的线程，同时投进所属拉取请求。
func (rt *Runtime) pollReviewThreads(cfg SourceConfig, root, owner, name, repo string) {
	if rt == nil || rt.eng == nil {
		return
	}
	rt.mu.Lock()
	keys := rt.eng.PollableThreads(time.Now())
	rt.mu.Unlock()
	n := 0
	for _, thread := range keys {
		if !strings.Contains(thread, "!") {
			continue
		}
		number, ok := githubNumber(thread, repo)
		if !ok {
			continue
		}
		if n >= 4 {
			break
		}
		n++
		rt.fetchReviews(cfg, root, owner, name, repo, number, thread)
	}
}

// fetchReviews 读取 number 的审阅评论。失败只记在拉取请求线程上。
func (rt *Runtime) fetchReviews(cfg SourceConfig, root, owner, name, repo string, number int, prThread string) {
	endpoint := root + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/pulls/" + strconv.Itoa(number) + "/comments?per_page=10"
	body, err := rt.request(http.MethodGet, endpoint, cfg.GitHubToken, nil, "application/vnd.github+json")
	if err != nil {
		rt.noteReadMiss(prThread)
		return
	}
	rt.clearReadMiss(prThread)
	var comments []struct {
		ID        int    `json:"id"`
		Body      string `json:"body"`
		InReplyTo int    `json:"in_reply_to_id"`
		User      struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"user"`
	}
	if json.Unmarshal(body, &comments) != nil {
		return
	}
	for _, comment := range comments {
		if comment.ID <= 0 || strings.TrimSpace(comment.Body) == "" {
			continue
		}
		rootID := comment.InReplyTo
		if rootID <= 0 {
			rootID = comment.ID
		}
		review := fmt.Sprintf("github:%s!%d/review:%d", repo, number, rootID)
		rt.mu.Lock()
		rt.eng.Receive(Trigger{
			ID:        fmt.Sprintf("github:%s!review%d", repo, comment.ID),
			Source:    "github",
			Channel:   "github",
			Ref:       strconv.Itoa(number),
			Threads:   []string{review, prThread},
			Primary:   review,
			Human:     personActor(comment.User.Login, comment.User.Type),
			Addressed: mentions(comment.Body, cfg.GitHubAccount),
			Sender:    comment.User.Login,
			Text:      clip(comment.Body, 4000),
		})
		rt.mu.Unlock()
		rt.markCursor(review, time.Now())
	}
}

// pollCheckThreads 只读还在看的拉取请求上已完成的检查。默认分支上没人看的检查不进评估。
func (rt *Runtime) pollCheckThreads(cfg SourceConfig, root, owner, name, repo string) {
	if rt == nil || rt.eng == nil {
		return
	}
	rt.mu.Lock()
	keys := rt.eng.PollableThreads(time.Now())
	rt.mu.Unlock()
	n := 0
	for _, thread := range keys {
		if !strings.Contains(thread, "!") {
			continue
		}
		number, ok := githubNumber(thread, repo)
		if !ok {
			continue
		}
		if n >= 4 {
			break
		}
		n++
		rt.fetchChecks(cfg, root, owner, name, repo, number, thread)
	}
}

// fetchChecks 读取拉取请求头部的已完成检查。每一项检查有自己的线程。
func (rt *Runtime) fetchChecks(cfg SourceConfig, root, owner, name, repo string, number int, prThread string) {
	pullURL := root + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/pulls/" + strconv.Itoa(number)
	body, err := rt.request(http.MethodGet, pullURL, cfg.GitHubToken, nil, "application/vnd.github+json")
	if err != nil {
		rt.noteReadMiss(prThread)
		return
	}
	var pull struct {
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if json.Unmarshal(body, &pull) != nil || pull.Head.SHA == "" {
		return
	}
	checkURL := root + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/commits/" + url.PathEscape(pull.Head.SHA) + "/check-runs?per_page=10"
	body, err = rt.request(http.MethodGet, checkURL, cfg.GitHubToken, nil, "application/vnd.github+json")
	if err != nil {
		rt.noteReadMiss(prThread)
		return
	}
	rt.clearReadMiss(prThread)
	var page struct {
		CheckRuns []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
		} `json:"check_runs"`
	}
	if json.Unmarshal(body, &page) != nil {
		return
	}
	for _, run := range page.CheckRuns {
		if run.ID <= 0 || run.Status != "completed" || strings.TrimSpace(run.Name) == "" {
			continue
		}
		ci := fmt.Sprintf("github:%s:ci:%s", repo, run.Name)
		rt.mu.Lock()
		rt.eng.Receive(Trigger{
			ID:        fmt.Sprintf("github:%s:ci:%d", repo, run.ID),
			Source:    "github",
			Channel:   "github",
			Ref:       strconv.Itoa(number),
			Threads:   []string{ci, prThread},
			Primary:   ci,
			Human:     false,
			Addressed: false,
			Sender:    run.Name,
			Text:      clip(run.Name+" "+run.Conclusion, 4000),
		})
		rt.mu.Unlock()
		rt.markCursor(ci, time.Now())
	}
}

// fetchComments 读取 number 的评论并收成触发。请求失败只记在 thread 上。
func (rt *Runtime) fetchComments(cfg SourceConfig, root, owner, name, repo string, number int, thread string) {
	if rt == nil || rt.eng == nil || number <= 0 {
		return
	}
	endpoint := root + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/issues/" + strconv.Itoa(number) + "/comments?per_page=10"
	if since := rt.sinceFor(thread); !since.IsZero() {
		endpoint += "&since=" + url.QueryEscape(since.UTC().Format(time.RFC3339))
	}
	body, err := rt.request(http.MethodGet, endpoint, cfg.GitHubToken, nil, "application/vnd.github+json")
	if err != nil {
		rt.noteReadMiss(thread)
		return
	}
	rt.clearReadMiss(thread)
	var comments []struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"user"`
	}
	if json.Unmarshal(body, &comments) != nil {
		return
	}
	for _, comment := range comments {
		if comment.ID <= 0 || strings.TrimSpace(comment.Body) == "" {
			continue
		}
		rt.mu.Lock()
		rt.eng.Receive(Trigger{
			ID:        fmt.Sprintf("github:%s#comment%d", repo, comment.ID),
			Source:    "github",
			Channel:   "github",
			Ref:       strconv.Itoa(number),
			Threads:   []string{thread},
			Primary:   thread,
			Human:     personActor(comment.User.Login, comment.User.Type),
			Addressed: mentions(comment.Body, cfg.GitHubAccount),
			Sender:    comment.User.Login,
			Text:      clip(comment.Body, 4000),
		})
		rt.mu.Unlock()
	}
	rt.markCursor(thread, time.Now())
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
				RootID    string `json:"root_id"`
				ParentID  string `json:"parent_id"`
				ThreadID  string `json:"thread_id"`
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
	for _, item := range page.Data.Items {
		if item.MessageID == "" || item.Sender.SenderType == "app" {
			continue
		}
		text := feishuText(item.Body.Content)
		topic := item.ThreadID
		if topic == "" {
			topic = item.RootID
		}
		if topic == "" {
			topic = item.ParentID
		}
		thread := feishuThread(cfg.FeishuChat, topic, item.MessageID)
		if !feishuReadable(item.Body.Content) {
			if strings.TrimSpace(item.Body.Content) != "" {
				rt.noteReadMiss(thread)
			}
			continue
		}
		rt.clearReadMiss(thread)
		rt.markCursor(thread, time.Now())
		rt.mu.Lock()
		rt.eng.Receive(Trigger{
			ID:        "feishu:" + item.MessageID,
			Source:    "feishu",
			Channel:   "feishu",
			Ref:       item.MessageID,
			Threads:   []string{thread},
			Primary:   thread,
			Human:     true,
			Addressed: mentions(text, cfg.FeishuMention),
			Sender:    item.Sender.ID,
			Text:      clip(text, 4000),
		})
		rt.mu.Unlock()
		n++
	}
	if thread := rt.readTrouble(); strings.HasPrefix(thread, "feishu:") {
		rt.setSource("feishu", "error", thread+" 连续三次没读到")
		return
	}
	rt.setSource("feishu", "ok", fmt.Sprintf("消息 %d", n))
}

// feishuReadable 在 content 能当成文字时为真。带 text 的 JSON 用那段。没有 text 的卡片或图片记成这一话题没读到。
func feishuReadable(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" || !strings.HasPrefix(content, "{") {
		return content != ""
	}
	var body struct {
		Text string `json:"text"`
	}
	if json.Unmarshal([]byte(content), &body) != nil {
		return true
	}
	return strings.TrimSpace(body.Text) != ""
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

// flushOutbox 把还没发出的信排进 reply 副作用。还没登记处理函数时直接发。
func (rt *Runtime) flushOutbox() {
	if rt == nil || rt.eng == nil {
		return
	}
	pending := rt.eng.PendingOutbox()
	if rt.effects != nil && rt.effects.Has("reply") {
		for _, msg := range pending {
			rt.effects.Enqueue("reply", msg.ID, msg.ID)
		}
		rt.effects.Pump(time.Now())
		return
	}
	for _, msg := range pending {
		_ = rt.sendOutbound(msg.ID)
	}
}

// sendOutbound 发送 id 这一条。前面还有同一工单没发出的，返回 ErrLater。
func (rt *Runtime) sendOutbound(id string) error {
	if rt == nil || rt.eng == nil || id == "" {
		return nil
	}
	pending := rt.eng.PendingOutbox()
	var msg Outbound
	found := false
	for _, item := range pending {
		if item.ID == id {
			msg = item
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	for _, item := range pending {
		if item.ID == id {
			break
		}
		if item.TicketID != "" && item.TicketID == msg.TicketID && item.Source == msg.Source {
			return ErrLater
		}
	}
	rt.mu.Lock()
	cfg := rt.sources
	rt.mu.Unlock()
	switch msg.Source {
	case "github":
		if cfg.GitHubToken == "" || strings.TrimSpace(cfg.GitHubRepo) == "" || msg.Ref == "" {
			return ErrLater
		}
		parts := strings.Split(strings.Trim(cfg.GitHubRepo, "/"), "/")
		if len(parts) != 2 {
			rt.eng.MarkOutbound(msg.ID, "error")
			return errCode("bad_repo")
		}
		root := strings.TrimRight(cfg.GitHubAPI, "/")
		endpoint := root + "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/issues/" + url.PathEscape(msg.Ref) + "/comments"
		_, err := rt.request(http.MethodPost, endpoint, cfg.GitHubToken, map[string]string{"body": msg.Text}, "application/vnd.github+json")
		if err != nil {
			rt.eng.MarkOutbound(msg.ID, "error")
			return err
		}
		rt.eng.MarkOutbound(msg.ID, "sent")
		return nil
	case "feishu":
		if cfg.FeishuAppID == "" || cfg.FeishuSecret == "" || cfg.FeishuChat == "" {
			return ErrLater
		}
		token, err := rt.feishuToken(cfg)
		if err != nil {
			rt.eng.MarkOutbound(msg.ID, "error")
			return err
		}
		root := strings.TrimRight(cfg.FeishuAPI, "/")
		content, _ := json.Marshal(map[string]string{"text": msg.Text})
		endpoint := root + "/open-apis/im/v1/messages?receive_id_type=chat_id"
		payload := map[string]string{
			"receive_id": cfg.FeishuChat, "msg_type": "text", "content": string(content),
		}
		if ref := strings.TrimSpace(msg.Ref); ref != "" {
			endpoint = root + "/open-apis/im/v1/messages/" + url.PathEscape(ref) + "/reply"
			payload = map[string]string{"msg_type": "text", "content": string(content)}
		}
		_, err = rt.request(http.MethodPost, endpoint, token, payload, "application/json")
		if err != nil {
			rt.eng.MarkOutbound(msg.ID, "error")
			return err
		}
		rt.eng.MarkOutbound(msg.ID, "sent")
		return nil
	default:
		rt.eng.MarkOutbound(msg.ID, "sent")
		return nil
	}
}

// feishuToken 用 cfg 换一张租户令牌。凭证为空或接口拒绝时返回错误。
func (rt *Runtime) feishuToken(cfg SourceConfig) (string, error) {
	root := strings.TrimRight(cfg.FeishuAPI, "/")
	body, err := rt.request(http.MethodPost, root+"/open-apis/auth/v3/tenant_access_token/internal", "", map[string]string{
		"app_id": cfg.FeishuAppID, "app_secret": cfg.FeishuSecret,
	}, "application/json")
	if err != nil {
		return "", err
	}
	var tok struct {
		Code  int    `json:"code"`
		Token string `json:"tenant_access_token"`
		Msg   string `json:"msg"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.Code != 0 || tok.Token == "" {
		if tok.Msg != "" {
			return "", fmt.Errorf("%s", tok.Msg)
		}
		return "", fmt.Errorf("feishu token")
	}
	return tok.Token, nil
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
