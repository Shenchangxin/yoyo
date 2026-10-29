/**
 * 拥有：前后端共用的会话、转录、上下文、追踪和视频路由类型，以及视频路径与会话通道的判定。
 * 不拥有：网络请求、界面状态，以及这些类型的渲染。
 */

/** Lab 工作台实验室：智能体、harbor、evolve 或 harness。 */
export type Lab = "agent" | "harbor" | "evolve" | "harness";
/** HarnessTab Harness 表面的四个页签。 */
export type HarnessTab = "overview" | "propose" | "prove" | "promote";
/** Surface 壳层当前打开的表面。 */
export type Surface = "agent" | "harness" | "settings" | "skills" | "video" | "pages" | "idol";
/** VideoMode 视频工作台的模式：剧集、画布或创作。 */
export type VideoMode = "drama" | "canvas" | "creative";
/** VideoPane 视频表面里的一块面板。 */
export type VideoPane = "chat" | "create" | "studio" | "drama" | "canvas" | "game" | "assets" | "skills" | "plugins" | "tasks";
/** CanvasFocus 画布停在库还是编辑器。 */
export type CanvasFocus = "library" | "editor";

/** isYingcePane 对话、工作室、创作台和游戏以外的面板走岛内路由。pane 不是 chat、studio、create 或 game 时返回真。 */
export function isYingcePane(pane: VideoPane): boolean {
  return pane !== "chat" && pane !== "studio" && pane !== "create" && pane !== "game";
}

/** isCanvasWorldPane 画布及其资产、技能、插件、任务面板属于画布世界。pane 是 canvas、assets、skills、plugins 或 tasks 时返回真。 */
export function isCanvasWorldPane(pane: VideoPane): boolean {
  return pane === "canvas" || pane === "assets" || pane === "skills" || pane === "plugins" || pane === "tasks";
}

/**
 * yingcePathForPane 把面板收成岛内路径。画布处于编辑器焦点且带项目 id 时进入该项目。
 * pane 为 create 时返回 /canvas，为 studio 时返回 /create，为 drama 时返回 /drama，为 assets 时返回 /assets，为 skills 时返回 /skills，为 plugins 时返回 /plugins，为 tasks 时返回 /tasks，其余分支还看别的条件。
 * projectId 为真时才继续，拼进字符串。
 * canvasFocus 与 editor 比较是否相等。
 */
export function yingcePathForPane(pane: VideoPane, projectId: string, canvasFocus: CanvasFocus): string {
  if (pane === "create") return "/canvas";
  if (pane === "studio") return "/create";
  if (pane === "drama") return "/drama";
  if (pane === "assets") return "/assets";
  if (pane === "skills") return "/skills";
  if (pane === "plugins") return "/plugins";
  if (pane === "tasks") return "/tasks";
  if (pane === "canvas" && canvasFocus === "editor" && projectId) return `/canvas/${projectId}`;
  return "/canvas";
}

/** VideoRouteTarget 从地址解析出的面板。entry 是要原样交给岛内路由的路径和查询，会话接续和素材交接留在里面。 */
export type VideoRouteTarget = {
  pane: VideoPane;
  canvasFocus?: CanvasFocus;
  projectId?: string;
  entry: string;
};

/** 去掉多余的结尾斜杠。空路径视为根。path 为假时用 /，读取 length，看结尾，截取。 */
function normalizeVideoPath(path: string) {
  if (path.length > 1 && path.endsWith("/")) return path.slice(0, -1);
  return path || "/";
}

/** videoRouteFromLocation 从创作页地址解析面板。转入画布时连同查询参数交给岛内路由，否则会话接续和素材交接会丢。无法识别的路径返回空。to 为假时用空字符串，转成字符串，去掉首尾空白后，按 ? 切开，解开 pathname、query，传给 URL。 */
export function videoRouteFromLocation(to: string): VideoRouteTarget | null {
  const raw = String(to || "").trim();
  let path = "/";
  let search = "";
  try {
    const url = new URL(raw, "https://yoyo.local");
    path = normalizeVideoPath(url.pathname);
    search = url.search.replace(/^\?/, "");
  } catch {
    const [pathname, query] = raw.split("?");
    path = normalizeVideoPath(pathname || "/");
    search = query || "";
  }
  const parts = path.split("/").filter(Boolean);
  let pane: VideoPane | null = null;
  let canvasFocus: CanvasFocus | undefined;
  let projectId = "";
  if (path === "/" || path === "/create") pane = "create";
  else if (path === "/studio") pane = "studio";
  else if (parts[0] === "canvas" && parts[1]) {
    pane = "canvas";
    canvasFocus = "editor";
    projectId = decodeURIComponent(parts[1]);
  } else if (path === "/canvas") {
    pane = "canvas";
    canvasFocus = "library";
  } else if (path === "/drama") pane = "drama";
  else if (path === "/assets") pane = "assets";
  else if (path === "/skills") pane = "skills";
  else if (path === "/plugins" || path === "/plugins/eagle") pane = "plugins";
  else if (path === "/tasks") pane = "tasks";
  else return null;
  const derived = yingcePathForPane(pane, projectId, canvasFocus || "library");
  let entry = "";
  if (search || projectId || path !== derived) entry = search ? `${path}?${search}` : path;
  return { pane, canvasFocus, projectId, entry };
}

/**
 * yingceHref 入口路径仍属于当前面板时保留它，含查询。否则退回该面板的派生路径。
 * pane 与 canvas、create、plugins 比较是否相等，传给 yingcePathForPane。
 * projectId 传给 yingcePathForPane。
 * canvasFocus 传给 yingcePathForPane。
 * entry 为假时返回 derived；按 ? 切开，取第一段，为假时用 /，有的分支原样返回。
 */
export function yingceHref(pane: VideoPane, projectId: string, canvasFocus: CanvasFocus, entry = ""): string {
  const derived = yingcePathForPane(pane, projectId, canvasFocus);
  if (!entry) return derived;
  const path = entry.split("?")[0] || "/";
  if (pane === "canvas" && (path === "/canvas" || path.startsWith("/canvas/"))) return entry;
  if (pane === "create" && (path === "/" || path === "/create")) return entry;
  if (pane === "plugins" && (path === "/plugins" || path === "/plugins/eagle")) return entry;
  if (path === derived) return entry;
  return derived;
}
/** HarborKind harbor 实验室里的一种评测、迁移或索引。 */
export type HarborKind = "suite" | "safety" | "tb" | "bon" | "models" | "sealed" | "transfer" | "index" | "behavior";
/** SettingsTab 设置表面的页签。 */
export type SettingsTab =
  | "general"
  | "appearance"
  | "provider"
  | "generation"
  | "policy"
  | "extensions"
  | "personal"
  | "advanced"
  | "idol";

/** AuthMode 会话权限：默认、自动编辑、完全，或每次询问。 */
export type AuthMode = "default" | "auto_edit" | "full" | "ask";

/** ThreadChannel 会话属于智能体还是视频。 */
export type ThreadChannel = "agent" | "video";

/** VideoProject 视频侧挂到对话上的一个项目。sessionId 对应会话，episodeId 对应剧集，gate 是工作室流程当前所在的闸门。 */
export type VideoProject = {
  kind: "canvas" | "drama" | "studio" | "game";
  id: string;
  title: string;
  sessionId?: string;
  episodeId?: string;
  updatedAt?: string;
  status?: string;
  gate?: string;
};

/** Profile 操作者配置档：说明、工具允许与拒绝、空间、研究与记忆开关，以及语音连接。 */
export type Profile = {
  id: string;
  name: string;
  instructions?: string;
  allow_tools?: string[];
  deny_tools?: string[];
  space_ids?: string[];
  default_space?: string;
  research?: boolean;
  memory?: boolean;
  mcp_allow?: string[];
  speech_connection?: string;
};

/** Thread 一段对话。 */
export type Thread = {
  id: string;
  title: string;
  workspace: string;
  /** 工具读写的根。隔离工作树存在时是工作树，否则是 workspace。 */
  toolRoot?: string;
  worktree?: string;
  /** 隔离之前的工作区。 */
  originWorkspace?: string;
  /** 本会话是否把工具关进独立工作树。 */
  isolate?: boolean;
  harness: string;
  createdAt: string;
  updatedAt?: string;
  preview?: string;
  archived?: boolean;
  pinned?: boolean;
  model?: string;
  /** 供应商连接。 */
  connectionId?: string;
  authMode?: AuthMode;
  /** 钉在会话上、跨回合保留的技能。 */
  pinnedSkills?: string[];
  /** 当前已经载入的技能。 */
  loadedSkills?: string[];
  channel?: ThreadChannel;
  interrupted?: boolean;
  /** 已排入下一轮、尚未发出的条数。 */
  queued?: number;
  /** 子代理所属的父会话。 */
  parentId?: string;
  planText?: string;
  profileId?: string;
  /** 页面表面里绑定的页面。 */
  pageId?: string;
  /** 这段对话绑定的文库。 */
  libraryId?: string;
  /** 这段对话绑定的学习路径。 */
  pathId?: string;
  /** 这段对话绑定的阅读材料。 */
  materialId?: string;
  /** 研究员档的最新简报。 */
  researchBrief?: string;
  /** 研究员档的最新截图。 */
  researchShot?: string;
};

/** ContextObject 上下文清单里的一条。stale 表示内容已过期，pinned 表示钉住、不让它被挤掉。 */
export type ContextObject = {
  id: string;
  kind: string;
  title: string;
  detail?: string;
  tokens?: number;
  path?: string;
  stale?: boolean;
  pinned?: boolean;
};

/** ContextInventory 一个会话的上下文用量和对象清单。plan 与 planPath 是当前计划正文及其文件。 */
export type ContextInventory = {
  session: string;
  shape: ContextUsage;
  objects: ContextObject[];
  plan?: string;
  planPath?: string;
};

/** threadChannel 缺省或非 video 时视为智能体通道。t 的 channel 为 video 时返回 video，否则返回 agent。 */
export function threadChannel(t: { channel?: string } | null | undefined): ThreadChannel {
  return t?.channel === "video" ? "video" : "agent";
}

/** isSubagentThread 子代理不是独立对话，嵌在父转录里。有 parentId，或 id 里嵌着任务分隔，即为子代理。t 为假时返回假，有 parentId 时返回真，否则把 id 交给 parentSessionId。 */
export function isSubagentThread(t: { id?: string; parentId?: string } | null | undefined): boolean {
  if (!t) return false;
  if (t.parentId) return true;
  return !!parentSessionId(t.id || "");
}

/** parentSessionId 从 `--task--` 或旧式 `/tasks/` 里切出父会话 id。没有则返回空串。 */
export function parentSessionId(id: string): string {
  const task = id.indexOf("--task--");
  if (task >= 0) return id.slice(0, task);
  const legacy = id.indexOf("/tasks/");
  if (legacy >= 0) return id.slice(0, legacy);
  return "";
}

/** channelForSurface 视频表面用视频通道，其余表面用智能体通道。surface 为 video 时返回 video，否则返回 agent。 */
export function channelForSurface(surface: Surface): ThreadChannel {
  return surface === "video" ? "video" : "agent";
}

/** ItemType 转录里一条记录的种类。 */
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
  | "turn_start"
  | "step_start"
  | "step_end"
  | "assistant_stream"
  | "inbox"
  | "system"
  | "compaction"
  | "subagent"
  | "eval"
  | "evolve"
  | "plan"
  | "file_change"
  | "ask_user";

/** Item 转录中的一条。source 区分界面乐观写入、用户、引导和技能注入。delta 表示尚未收束的增量切片。 */
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
  /** 存储索引或实时枢纽的序号，供回合大纲跳转。 */
  seq?: number;
};

/** Approval 一条待批准的操作。level 是风险级别。 */
export type Approval = {
  id: string;
  action: string;
  command: string;
  path: string;
  level: string;
  sessionId: string;
};

/** ContextUsage 会话上下文窗口的用量。各 token 字段是分片；slicesKnown 为假时分片不可信。 */
export type ContextUsage = {
  tokens: number;
  budget: number;
  /** 模型上下文窗口。缺省时界面改用 budget。 */
  window?: number;
  prefixTokens?: number;
  dynamicTokens?: number;
  schemaTokens?: number;
  identityTokens?: number;
  rulesTokens?: number;
  catalogTokens?: number;
  skillBodyTokens?: number;
  memoryTokens?: number;
  todayTokens?: number;
  notesTokens?: number;
  chatTokens?: number;
  /** 分片数字是否由后端逐项给出。 */
  slicesKnown?: boolean;
  /** 供应商实际看到的提示词 token。 */
  providerPrompt?: number;
  cachedTokens?: number;
  /** 供应商是否报告了缓存。 */
  cacheReported?: boolean;
  /** 前缀缓存是否仍稳定。移动后为假。 */
  cacheStable?: boolean;
  /** 前缀内容的哈希，用来判断缓存有没有换。 */
  prefixHash?: string;
  /** 动态段最近一次计入的时间。 */
  dynamicAt?: string;
  /** 这次用量是被什么触发重算的。 */
  trigger?: string;
  /** 已重新载入的计数。 */
  hydrated?: number;
  /** 后端附带的一句说明。 */
  note: string;
  /** 参与这次用量的层名。 */
  layers: string[];
  /** 被省略、不再占窗口的数量。 */
  elided: number;
};

/** TraceStats 一条追踪的计数：事件、回合角色、错误、压缩、增量和卸出字节。 */
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

/** TraceEvent 追踪里的一条事件。lane 是事件所在通道，spillId 指向卸到外部的正文，ttftMs 是首 token 延迟。 */
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
  promptTokens?: number;
  completionTokens?: number;
  cachedTokens?: number;
  ttftMs?: number;
  cacheReported?: boolean;
  error: boolean;
};

/** TraceArtifact 追踪附带的一块产物预览。 */
export type TraceArtifact = {
  kind: string;
  id: string;
  label: string;
  bytes: number;
  preview: string;
};

/** SessionTrace 一个会话的追踪：时间范围、统计、事件和产物。 */
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

/** SpillBlob 卸出正文的一段。truncated 表示没有整段取回。 */
export type SpillBlob = {
  id: string;
  bytes: number;
  text: string;
  truncated: boolean;
};

/** ChatProvider 一条聊天供应商连接。hasKey 表示密钥已配置，active 表示当前正在使用。 */
export type ChatProvider = {
  id: string;
  name: string;
  vendor: string;
  endpoint: string;
  model: string;
  models: string[];
  hasKey: boolean;
  isDefault: boolean;
  active: boolean;
};

/** Health 运行时健康。isolated 与 isolationKind 描述进程隔离，budgetUsd 与 usageUsd 是金额预算和已用金额。 */
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
  videoWorkspace?: string;
  videoWorkspaceReady?: boolean;
  chatProviders?: ChatProvider[];
  paused?: boolean;
};

/** AppConfig 桌面端可持久化的配置。 */
export type AppConfig = {
  provider: string;
  model: string;
  baseUrl: string;
  workspace: string;
  videoWorkspace?: string;
  autoAllow: boolean;
  maxBudgetUsd: number;
  usdPerMtok: number;
  models: string[];
  /** 关闭窗口时收进托盘。 */
  closeToTray: boolean;
  /** 是否允许伴随小窗。 */
  companionEnabled?: boolean;
  updateUrl: string;
  updateAutoCheck?: boolean;
  keymap?: Record<string, string>;
  locale?: string;
  alwaysOnTop?: boolean;
  startAtLogin?: boolean;
  notificationsEnabled?: boolean;
  /** 只在窗口失焦时发系统通知。 */
  notifyWhenUnfocusedOnly?: boolean;
  uiScale?: number;
  showThinking?: boolean;
  transcriptView?: string;
  collapseTiming?: string;
  updateChannel?: string;
	theme?: string;
  paletteDark?: string;
  paletteLight?: string;
  skin?: string;
  /** 权限闸门：手动、自动安全或跳过。 */
  gateMode?: string;
  /** 崩溃后是否恢复会话。 */
  crashResume?: boolean;
  searchUrl?: string;
  searchKey?: string;
  paused?: boolean;
};

/** Notice 一条通知。sessionId 指出它属于哪段对话。 */
export type Notice = {
  id: string;
  title: string;
  body: string;
  ts: string;
  sessionId?: string;
};

/** Attachment 随消息带上的附件。data_b64 是内联内容。 */
export type Attachment = {
  path?: string;
  name?: string;
  mime?: string;
  data_b64?: string;
};

/** SkillInfo 一个技能。incomplete 表示安装不完整，pack 与 installId 指向它所属的技能包。 */
export type SkillInfo = {
  name: string;
  description: string;
  body?: string;
  dir?: string;
  source?: string;
  icon?: string;
  displayName?: string;
  files?: string;
  incomplete?: boolean;
  slug?: string;
  pack?: string;
  installId?: string;
};

/** PackOrigin 技能包的来源。skillsRel 是仓库内技能目录的相对路径。 */
export type PackOrigin = {
  kind: string;
  repo?: string;
  ref?: string;
  skillsRel?: string;
  path?: string;
};

/** PackStatus 已登记技能包的安装与启用状态。enableGlobal 与 enableWorkspace 分别是全局和当前工作区的开关。methodology 表示它是方法论包。known 表示目录里认识它。 */
export type PackStatus = {
  id: string;
  name: string;
  description: string;
  license?: string;
  version?: string;
  commit?: string;
  bootstrapSkill?: string;
  methodology: boolean;
  installed: boolean;
  enabled: boolean;
  enableGlobal: boolean;
  enableWorkspace?: boolean | null;
  root?: string;
  skillsDir?: string;
  skillCount: number;
  skills?: string[];
  origin: PackOrigin;
  installedAt?: string;
  known: boolean;
};

/** RunStatus 一次仍在进行的运行。lastTool 是最近调用的工具名。 */
export type RunStatus = {
  id: string;
  startedAt: string;
  lastTool: string;
};

/** FileHit 文件搜索的一条命中。kind 区分文件种类。 */
export type FileHit = {
  path: string;
  kind: string;
};

/** Hunk 差异里的一块。header 是块头，body 是块正文。 */
export type Hunk = {
  id: string;
  file: string;
  header: string;
  body: string;
};

/** Metrics 评测计数：留入集与留出集的通过数和总数，以及安全失败数。 */
export type Metrics = {
  heldInPass: number;
  heldInTotal: number;
  heldOutPass: number;
  heldOutTotal: number;
  safetyFail: number;
};
