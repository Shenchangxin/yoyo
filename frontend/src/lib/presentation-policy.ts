/**
 * 拥有：转录四种详略档位如何决定折叠时机、过程分组和推理预览。
 * 不拥有：把条目排成视觉行。那是 transcript-layout 的事。
 */

/** TranscriptViewMode 转录详略：紧凑、标准、详细或啰嗦。 */
export type TranscriptViewMode = "compact" | "standard" | "detailed" | "verbose";
/** CollapseTiming 已完成工作何时折叠：回合结束时，或下一条输入之后。 */
export type CollapseTiming = "completion" | "next-input";

/** ChatPresentationPolicy 某一详略档位下的展示能力，外加折叠时机。 */
export type ChatPresentationPolicy = {
  mode: TranscriptViewMode;
  collapseTiming: CollapseTiming;
  foldCompletedTurns: boolean;
  stepGrouping: "collapsed" | "history" | "none";
  liveProcessDetail: boolean;
  settledReasoningPreview: boolean;
};

/** 各档位的固定能力。折叠时机由调用方另填。 */
const POLICIES: Record<TranscriptViewMode, Omit<ChatPresentationPolicy, "collapseTiming">> = {
  compact: {
    mode: "compact",
    foldCompletedTurns: true,
    stepGrouping: "collapsed",
    liveProcessDetail: false,
    settledReasoningPreview: false,
  },
  standard: {
    mode: "standard",
    foldCompletedTurns: true,
    stepGrouping: "collapsed",
    liveProcessDetail: true,
    settledReasoningPreview: true,
  },
  detailed: {
    mode: "detailed",
    foldCompletedTurns: true,
    stepGrouping: "history",
    liveProcessDetail: true,
    settledReasoningPreview: true,
  },
  verbose: {
    mode: "verbose",
    foldCompletedTurns: false,
    stepGrouping: "none",
    liveProcessDetail: false,
    settledReasoningPreview: true,
  },
};

/** presentationPolicyFor 按详略档位取出策略，并带上折叠时机。未知档位退回标准。mode 用作下标。collapseTiming 写入字段 collapseTiming。 */
export function presentationPolicyFor(
  mode: TranscriptViewMode,
  collapseTiming: CollapseTiming = "completion",
): ChatPresentationPolicy {
  return { ...POLICIES[mode] || POLICIES.standard, collapseTiming };
}

/** coerceTranscriptView 把存储的字符串收成档位。expanded 视为详细，其余未知值视为标准。v 为 expanded 时返回 detailed；有的分支原样返回。 */
export function coerceTranscriptView(v: string | null | undefined): TranscriptViewMode {
  if (v === "compact" || v === "standard" || v === "detailed" || v === "verbose") return v;
  if (v === "expanded") return "detailed";
  return "standard";
}

/** coerceCollapseTiming 把 next-input、next_input、next-message 收成下一条输入后折叠，其余视为回合结束时折叠。v 与 next-input、next_input、next-message 比较是否相等。 */
export function coerceCollapseTiming(v: string | null | undefined): CollapseTiming {
  if (v === "next-input" || v === "next_input" || v === "next-message") return "next-input";
  return "completion";
}

/**
 * processGrouped 过程轨是否收成一组。none 不分组，collapsed 始终分组。history 在进行中保持展开；时机若是下一条输入，下一条用户消息还没来时也保持展开。
 * policy 的 stepGrouping 为 none 时返回假，的 stepGrouping 为 collapsed 时返回真，其余分支还看别的条件。
 * live 为真时返回假。
 * nextUser 为假时走另一支。
 */
export function processGrouped(policy: ChatPresentationPolicy, live: boolean, nextUser: boolean): boolean {
  if (policy.stepGrouping === "none") return false;
  if (policy.stepGrouping === "collapsed") return true;
  if (live) return false;
  if (policy.collapseTiming === "next-input" && !nextUser) return false;
  return true;
}

/** turnWorkFolded 已完成的工作块是否折叠。进行中不折叠。时机若是下一条输入，要等下一条用户消息出现。policy 的 collapseTiming 为 next-input 时返回 nextUser，其余分支还看别的条件。live 作为后备值。nextUser 原样返回。 */
export function turnWorkFolded(policy: ChatPresentationPolicy, live: boolean, nextUser: boolean): boolean {
  if (!policy.foldCompletedTurns || live) return false;
  if (policy.collapseTiming === "next-input") return nextUser;
  return true;
}
