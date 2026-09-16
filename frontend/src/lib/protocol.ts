export type Lab = "agent" | "harbor" | "evolve" | "harness";
export type Surface = Lab | "settings";
export type SettingsTab =
  | "general"
  | "appearance"
  | "provider"
  | "policy"
  | "mcp"
  | "plugins"
  | "shortcuts"
  | "updates"
  | "logs"
  | "danger";

export type Thread = {
  id: string;
  title: string;
  workspace: string;
  harness: string;
  createdAt: string;
  archived?: boolean;
  pinned?: boolean;
  model?: string;
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
  window?: number;
  prefixTokens?: number;
  schemaTokens?: number;
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
  workspaceReady: boolean;
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
  closeToTray: boolean;
  updateUrl: string;
  keymap?: Record<string, string>;
  locale?: string;
  alwaysOnTop?: boolean;
  startAtLogin?: boolean;
  notificationsEnabled?: boolean;
  notifyWhenUnfocusedOnly?: boolean;
  uiScale?: number;
  updateChannel?: string;
  theme?: string;
};

export type Notice = {
  id: string;
  title: string;
  body: string;
  ts: string;
  sessionId?: string;
};

export type Attachment = {
  path?: string;
  name?: string;
  mime?: string;
  data_b64?: string;
};

export type SkillInfo = {
  name: string;
  description: string;
};

export type FileHit = {
  path: string;
  kind: string;
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
