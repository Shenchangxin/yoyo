// 学习模块向模型要一次补全的入口，以及从回答里取出 JSON。不持有密钥，也不知道用的是哪家模型。
// 判分、起草大纲、出题和写课本都走这里。调用期间不持有库锁。

package learn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrNoModel 表示宿主没有接上模型。需要模型的功能返回它，不编造结果。
var ErrNoModel = errors.New("learning needs a chat model; add one in Settings")

// CompleteRequest 是一次不带工具的补全。Purpose 是 judge、outline、questions 或 book，宿主可以据此选路线。
// SessionID 非空时宿主用那段对话的模型，否则用默认模型。MaxTokens 小于等于 0 时由宿主取默认。
type CompleteRequest struct {
	Purpose     string
	SessionID   string
	System      string
	User        string
	MaxTokens   int
	Temperature float64
}

// CompleteFunc 跑一次补全并返回正文。ctx 取消时返回错误。
type CompleteFunc func(ctx context.Context, req CompleteRequest) (string, error)

// completeJSON 要一次补全并把回答解成 out。第一次解不开时把错误告诉模型再要一次，仍解不开就返回错误。
// req 原样交给 Complete。out 是指向结构或 map 的指针。
func (s *Store) completeJSON(ctx context.Context, req CompleteRequest, out any) error {
	if s == nil || s.Complete == nil {
		return ErrNoModel
	}
	if ctx == nil {
		ctx = context.Background()
	}
	text, err := s.Complete(ctx, req)
	if err != nil {
		return err
	}
	first := decodeJSON(text, out)
	if first == nil {
		return nil
	}
	retry := req
	retry.User = req.User + "\n\nYour previous reply could not be parsed as JSON (" + first.Error() + "). Reply again with one valid JSON value only, no prose and no code fence."
	text, err = s.Complete(ctx, retry)
	if err != nil {
		return err
	}
	if err := decodeJSON(text, out); err != nil {
		return fmt.Errorf("model reply was not valid JSON: %w", err)
	}
	return nil
}

// decodeJSON 从 text 里取出第一个完整的 JSON 对象或数组并解到 out。代码围栏和前后的说明文字会被跳过。
func decodeJSON(text string, out any) error {
	raw := extractJSON(text)
	if raw == "" {
		return errors.New("no JSON found")
	}
	return json.Unmarshal([]byte(raw), out)
}

// extractJSON 返回 text 里第一个括号配平的 JSON 对象或数组。字符串里的括号不计。找不到时返回空串。
func extractJSON(text string) string {
	start := strings.IndexAny(text, "{[")
	for start >= 0 {
		if end := balancedEnd(text, start); end > start {
			candidate := text[start : end+1]
			if json.Valid([]byte(candidate)) {
				return candidate
			}
		}
		next := strings.IndexAny(text[start+1:], "{[")
		if next < 0 {
			return ""
		}
		start += next + 1
	}
	return ""
}

// balancedEnd 返回从 start 开始的那个括号对应的收尾位置。没有配平时返回 -1。
func balancedEnd(text string, start int) int {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		c := text[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// languageHint 给模型一句用什么语言写的指示。sample 里有汉字就要求简体中文，否则跟随 sample 的语言。
func languageHint(sample string) string {
	for _, r := range sample {
		if r >= 0x4E00 && r <= 0x9FFF {
			return "Write every natural-language field in Simplified Chinese."
		}
	}
	return "Write every natural-language field in the same language as the learner's text."
}
