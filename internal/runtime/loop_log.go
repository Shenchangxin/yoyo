package runtime

import (
	"context"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

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
