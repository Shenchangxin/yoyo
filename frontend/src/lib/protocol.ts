export type Lab = "agent" | "harbor" | "evolve" | "harness";
export type HarnessTab = "overview" | "propose" | "prove" | "promote";
export type Surface = "agent" | "harness" | "settings" | "skills";
export type HarborKind = "suite" | "safety" | "tb" | "bon" | "models" | "sealed" | "transfer" | "index" | "behavior";
export type SettingsTab =
  | "general"
  | "appearance"
  | "shortcuts"
  | "provider"
  | "policy"
  | "extensions"
  | "personal"
  | "advanced";

export type AuthMode = "default" | "auto_edit" | "full" | "ask";

export type Thread = {
  id: string;
  title: string;
  workspace: string;
  toolRoot?: string;
  worktree?: string;
  originWorkspace?: string;
  isolate?: boolean;
  harness: string;
  createdAt: string;
  archived?: boolean;
  pinned?: boolean;
  model?: string;
  authMode?: AuthMode;
  pinnedSkills?: string[];
  loadedSkills?: string[];
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
  | "evolve"
  | "plan"
  | "file_change"
  | "ask_user";

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
  dynamicTokens?: number;
  schemaTokens?: number;
  providerPrompt?: number;
  note: string;
  layers: string[];
  elided: number;
};

export type TraceStats = {
  events: number;
  users: number;
  assistants: number;
  toolCalls: number;
  toolResults: number;
  errors: number;
  compactions: number;
  deltas: number;
  tokens: number;
  durationMs: number;
  spillBytes: number;
};

export type TraceEvent = {
  index: number;
  ts: string;
  type: string;
  source: string;
  lane: string;
  round: string;
  name: string;
  id: string;
  summary: string;
  detail: string;
  bytes: number;
  elapsedMs: number;
  spillId: string;
  tokens: number;
  error: boolean;
};

export type TraceArtifact = {
  kind: string;
  id: string;
  label: string;
  bytes: number;
  preview: string;
};

export type SessionTrace = {
  sessionId: string;
  title: string;
  workspace: string;
  harness: string;
  model: string;
  createdAt: string;
  startedAt: string;
  endedAt: string;
  stats: TraceStats;
  events: TraceEvent[];
  artifacts: TraceArtifact[];
};

export type SpillBlob = {
  id: string;
  bytes: number;
  text: string;
  truncated: boolean;
};

export type Health = {
  ok: boolean;
  harness: string;
  model: string;
  version: string;
  isolated: boolean;
  isolationKind: string;
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
  gateMode?: string;
  crashResume?: boolean;
  searchUrl?: string;
  searchKey?: string;
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
  body?: string;
  dir?: string;
  source?: string;
};

export type RunStatus = {
  id: string;
  startedAt: string;
  lastTool: string;
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
