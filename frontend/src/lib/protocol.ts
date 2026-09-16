export type Lab = "agent" | "harbor" | "evolve" | "harness" | "control";

export type Thread = {
  id: string;
  title: string;
  workspace: string;
  harness: string;
  createdAt: string;
};

export type ItemType =
  | "user"
  | "assistant"
  | "reasoning"
  | "tool_call"
  | "tool_result"
  | "context_injection"
  | "approval"
  | "error"
  | "turn_end"
  | "system"
  | "compaction"
  | "subagent"
  | "eval"
  | "evolve";

export type Item = {
  key: string;
  type: ItemType;
  sessionId: string;
  source: string;
  ts: string;
  text: string;
  name: string;
  delta: boolean;
  payload: Record<string, any>;
};

export type Approval = {
  id: string;
  action: string;
  command: string;
  path: string;
  level: string;
  sessionId: string;
};

export type ContextUsage = {
  tokens: number;
  budget: number;
  note: string;
  layers: string[];
  elided: number;
};

export type Health = {
  ok: boolean;
  harness: string;
  model: string;
  version: string;
  isolated: boolean;
  budgetUsd: number;
  usageUsd: number;
};

export type AppConfig = {
  provider: string;
  model: string;
  baseUrl: string;
  workspace: string;
  autoAllow: boolean;
  maxBudgetUsd: number;
  usdPerMtok: number;
  models: string[];
};

export type Hunk = {
  id: string;
  file: string;
  header: string;
  body: string;
};

export type Metrics = {
  heldInPass: number;
  heldInTotal: number;
  heldOutPass: number;
  heldOutTotal: number;
  safetyFail: number;
};
