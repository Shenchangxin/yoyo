// 工具结果写入轨迹前的预览。全文先放进 spill，返回的才是截过的预览。循环在派发工具后调用。
// 不负责 recall_context 的分页。

package runtime

// defaultIngestBudget 是工具结果预览的默认 rune 上限。Loop 的 ToolResultBudget 大于 0 时用那个值。
const defaultIngestBudget = 8_000
