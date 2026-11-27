// 同一工具、同一组规范化参数的连续调用提醒。读文件和计划更新既不计数也不清零。
// 提醒是另插的一条用户消息。工具结果本身不改写，这条链也不结束回合。Run 在每批工具之后调用 observe。

package runtime

// 短提醒在计数第一次到达 3 时发出，长提醒在 5 和 8 时发出。8 之后同一条链不再出提醒。参数预览最多 500 个 rune。
const ()

// 短提醒和长提醒各有英文和中文。voiceNudge 按操作者语言挑选，本文件不改这两段文案。
const repeatGentleEN = "You are repeating the exact same tool call with identical arguments. Carefully analyze the previous result before calling again: if the task is not complete, try a different approach or different arguments instead of repeating the call."
const repeatGentleZH = "你正在用完全相同的参数重复调用同一个工具。先看上一次的结果再决定：任务还没完成就换做法或换参数，不要再发这一次调用。"

const repeatDetailPrefixEN = "Repeated tool call detected:"
const repeatDetailPrefixZH = "检测到重复的工具调用："
