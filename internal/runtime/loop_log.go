// 把一轮模型调用和一轮工具结果写进诊断日志。Run 在 completeChat 之后调用。
// 不决定停止，也不改消息。

package runtime

import (
	"context"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// logModelRound 记录回合序号、用量、耗时、工具名和 truncated。err 非 nil 时再记一条 error。
// ctx 传给 diaglog.InfoContext。
// round 写入日志的 round。
// msg 读取用量、耗时、Truncated，以及非空的工具名。
func logModelRound(ctx context.Context, round int, msg Message, err error) {
	names := make([]string, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		if tc.Name != "" {
			names = append(names, tc.Name)
		}
	}
	args := []any{
		"component", "runtime",
		"round", round,
		"prompt_tokens", msg.PromptTokens,
		"cached_tokens", msg.CachedTokens,
		"completion_tokens", msg.CompletionTokens,
		"elapsed_ms", msg.ElapsedMs,
		"ttft_ms", msg.FirstTokenMs,
		"tools", strings.Join(names, ","),
		"truncated", msg.Truncated,
	}
	if err != nil {
		args = append(args, "error", err.Error())
	}
	diaglog.InfoContext(ctx, "model round", args...)
}

// logToolRound 记录本批工具名。正文去掉空白后以前缀 ERROR 开头的结果计入 tool_error，不要求冒号。
// ctx 传给 diaglog.InfoContext。
// round 写入日志的 round。
// results 收集非空 Name，正文去掉空白后以 ERROR 开头的计入 tool_error。
func logToolRound(ctx context.Context, round int, results []Message) {
	errs := 0
	names := make([]string, 0, len(results))
	for _, m := range results {
		if m.Name != "" {
			names = append(names, m.Name)
		}
		body := strings.TrimSpace(m.Content)
		if strings.HasPrefix(body, "ERROR") {
			errs++
		}
	}
	diaglog.InfoContext(ctx, "tool round",
		"component", "runtime",
		"round", round,
		"tools", strings.Join(names, ","),
		"tool_error", errs,
	)
}
