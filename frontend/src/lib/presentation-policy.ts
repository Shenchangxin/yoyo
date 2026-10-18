export type TranscriptViewMode = "compact" | "standard" | "detailed" | "verbose";
export type CollapseTiming = "completion" | "next-input";

export type ChatPresentationPolicy = {
  mode: TranscriptViewMode;
  collapseTiming: CollapseTiming;
  foldCompletedTurns: boolean;
  stepGrouping: "collapsed" | "history" | "none";
  liveProcessDetail: boolean;
  settledReasoningPreview: boolean;
};

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

export function presentationPolicyFor(
  mode: TranscriptViewMode,
  collapseTiming: CollapseTiming = "completion",
): ChatPresentationPolicy {
  return { ...POLICIES[mode] || POLICIES.standard, collapseTiming };
}

export function coerceTranscriptView(v: string | null | undefined): TranscriptViewMode {
  if (v === "compact" || v === "standard" || v === "detailed" || v === "verbose") return v;
  if (v === "expanded") return "detailed";
  return "standard";
}

export function coerceCollapseTiming(v: string | null | undefined): CollapseTiming {
  if (v === "next-input" || v === "next_input" || v === "next-message") return "next-input";
  return "completion";
}

export function processGrouped(policy: ChatPresentationPolicy, live: boolean, nextUser: boolean): boolean {
  if (policy.stepGrouping === "none") return false;
  if (policy.stepGrouping === "collapsed") return true;
  if (live) return false;
  if (policy.collapseTiming === "next-input" && !nextUser) return false;
  return true;
}

export function turnWorkFolded(policy: ChatPresentationPolicy, live: boolean, nextUser: boolean): boolean {
  if (!policy.foldCompletedTurns || live) return false;
  if (policy.collapseTiming === "next-input") return nextUser;
  return true;
}
