// 学习模块用模型的两个出口：一次不带工具的补全，和文本嵌入。
// 密钥和路线在这里取，internal/learn 拿不到。嵌入花费记在学习库，不记进对话用量。

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// learnCompleteTimeout 按用途给出一次补全的时限。写课本的正文最长，判分最短。
func learnCompleteTimeout(purpose string) time.Duration {
	switch purpose {
	case "book":
		return 4 * time.Minute
	case "judge":
		return 45 * time.Second
	default:
		return 2 * time.Minute
	}
}

// learnComplete 跑一次不带工具的补全。req.SessionID 非空时用那段对话的模型和路线，否则用默认聊天模型。
// 没有可用的聊天连接时返回 learn.ErrNoModel。ctx 取消或超过时限时返回错误。
func (appHost *learnHost) learnComplete(ctx context.Context, req learn.CompleteRequest) (string, error) {
	if appHost.app == nil {
		return "", learn.ErrNoModel
	}
	meta := SessionMeta{}
	if strings.TrimSpace(req.SessionID) != "" {
		if m, err := appHost.app.GetSession(req.SessionID); err == nil {
			meta = m
		}
	}
	client, err := appHost.app.ClientFor(meta)
	if err != nil {
		return "", fmt.Errorf("%w (%v)", learn.ErrNoModel, err)
	}
	model := appHost.app.Config.Model
	if meta.Model != "" {
		model = meta.Model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1200
	}
	if limit := maxOutputTokensFor(model); limit > 0 && maxTokens > limit {
		maxTokens = limit
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, learnCompleteTimeout(req.Purpose))
	defer cancel()
	msgs := make([]runtime.Message, 0, 2)
	if strings.TrimSpace(req.System) != "" {
		msgs = append(msgs, runtime.Message{Role: runtime.RoleSystem, Content: req.System})
	}
	msgs = append(msgs, runtime.Message{Role: runtime.RoleUser, Content: req.User})
	out, err := client.Chat(ctx, runtime.ChatRequest{Model: model, Messages: msgs, MaxTokens: maxTokens, Temperature: req.Temperature})
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(out.Content)
	if text == "" {
		return "", fmt.Errorf("the model returned an empty reply")
	}
	return text, nil
}

// embedTexts 向配置的嵌入端点请求向量。model 为空时返回错误，重建仍可以只留词法索引。
// texts 按最多 16 条分批。花费不写入对话用量。
func (appHost *learnHost) embedTexts(model string, texts []string) ([][]float32, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("no embed model")
	}
	if appHost.app == nil || strings.TrimSpace(appHost.app.Config.BaseURL) == "" {
		return nil, fmt.Errorf("embeddings not configured")
	}
	key := ""
	if appHost.app.Vault != nil {
		key, _ = appHost.app.Vault.Lease("openai")
		if key == "" {
			key, _ = appHost.app.Vault.Get("api_key")
		}
	}
	if key == "" {
		return nil, fmt.Errorf("embeddings not configured")
	}
	out := make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += 16 {
		end := i + 16
		if end > len(texts) {
			end = len(texts)
		}
		part, err := appHost.app.embedBatch(model, key, texts[i:end])
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	if appHost.app.Learn != nil {
		appHost.app.Learn.NoteEmbed()
	}
	return out, nil
}

// embedBatch 向 BaseURL 下的 /embeddings 发一批文本。超时 45 秒。返回的条数和 texts 不一致时报错。
// model 和 key 原样写进请求。
func (appHost *learnHost) embedBatch(model, key string, texts []string) ([][]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": model, "input": texts})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(appHost.app.Config.BaseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings http %d", res.StatusCode)
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embeddings count")
	}
	out := make([][]float32, len(parsed.Data))
	for i, row := range parsed.Data {
		vec := make([]float32, len(row.Embedding))
		for j, x := range row.Embedding {
			vec[j] = float32(x)
		}
		out[i] = vec
	}
	return out, nil
}
