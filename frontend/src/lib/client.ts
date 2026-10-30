/*
 * 桌面和手机共用的请求门。
 * 原生窗口走 Wails 生成的 Service，或 Call.ByName。
 * 浏览器门（onBrowserDoor）为真时 wailsService 返回空，请求走 /api。
 * 桌面 webview 不提供这些 HTTP 路由。绑定模块缺失时不能靠退回 /api。
 * Vite 必须看见 boundService 的 glob，生产构建才会把生成的 Service 打进包。
 * 对运行时路径使用 @vite-ignore 会在安装包里 404（资源服务器上没有 /bindings/*.js），
 * 于是每次调用都落到 /api/*，而桌面 webview 并不提供这些路由。
 * 手机页是同一份包，必须向电脑上的 /api 发请求。生成的 Service 只跟桌面窗口说话。
 */

import { asArray, asBool, bool, boolOr, errMessage, num, pick, str } from "./normalize";
import { getLocale } from "./i18n";
import { browserSurface } from "./chrome";
import { snapUiScale } from "./scale";
import type { AppConfig, Approval, Attachment, AuthMode, ChatProvider, ContextInventory, ContextObject, ContextUsage, FileHit, Health, Hunk, PackStatus, RunStatus, SessionTrace, SkillInfo, SpillBlob, Thread, ThreadChannel, TraceArtifact, TraceEvent, TraceStats } from "./protocol";
import { defaultMaterials, type SkinInfo, type SkinResolved, type TokenMap } from "./skin/schema";
import { outlineOfRaw, type OutlineTurn } from "./turn-outline";

/** ApiError 是 HTTP 失败，身上带着状态码和响应体。 */
export class ApiError extends Error {
  status: number;
  body: any;
  /** constructor 记下状态码、响应体和消息。status 记到 this.status。body 记到 this.body。message 传给 super。 */
  constructor(status: number, body: any, message: string) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

// Vite 必须在构建时看见这个 glob，生成的 Service 才会打进生产包。
// 对运行时路径使用 `@vite-ignore` 时，安装包的资源服务上没有 /bindings/*.js，会 404，
// 随后每次调用落到 `/api/*`，而桌面 webview 不提供这些 HTTP 路由。绑定模块缺失时不能靠退回 /api。
const boundService = import.meta.glob("../../bindings/**/internal/desktop/service.ts");

/** unwrapService 从生成模块里取出带有会话列表或健康检查的 Service。mod 读取 Service、default，为空时不读，作为后备值。 */
function unwrapService(mod: any): any | null {
  const svc = mod?.Service ?? mod?.default ?? mod;
  if (svc && typeof (svc.ListSessions || svc.Health || svc.listSessions) === "function") return svc;
  return null;
}

/** nativeDesktop 窗口带桌面标记或 Wails 环境时，视为原生桌面。 */
function nativeDesktop(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as any;
  return !!(w.__YOYO_DESKTOP__ || w._wails?.environment?.OS);
}

/** onBrowserDoor 手机页是同一套包，必须打到电脑上的 /api。生成的 Service 只跟桌面窗口说话。文档标了 browser，或会话存储里有 yoyo_browser，就算浏览器门。 */
function onBrowserDoor(): boolean {
  if (typeof document !== "undefined" && document.documentElement.dataset.browser === "1") return true;
  try {
    return !!sessionStorage.getItem("yoyo_browser");
  } catch {
    return false;
  }
}

/** browserToken 取出浏览器门放在会话存储里的令牌。读不到就空。 */
function browserToken(): string {
  try {
    return sessionStorage.getItem("yoyo_browser") || "";
  } catch {
    return "";
  }
}

/** browserFetchInit 补上 JSON 类型、会话令牌和携带凭据。init 读取 headers，为空时不读，展开传递。 */
function browserFetchInit(init?: RequestInit): RequestInit {
  const headers = new Headers(init?.headers);
  if (!headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const token = browserToken();
  if (token && !headers.has("X-Yoyo-Session")) headers.set("X-Yoyo-Session", token);
  return { ...init, credentials: "include", headers };
}

/** namedDesktopService 用代理把每次调用交给 Wails 的 Call.ByName，名字是桌面 Service 的全名。 */
function namedDesktopService(): any {
  /** name 拼出桌面 Service 某一个方法的全名。method 拼进字符串。 */
  const name = (method: string) => `github.com/Shenchangxin/yoyo/internal/desktop.Service.${method}`;
  return new Proxy(
    {},
    {
      /** get 根据参数得到结果。_target 函数体内没有读取。prop 与 then 比较是否相等，用 typeof 看类型，传给 name。 */
      get(_target, prop) {
        if (typeof prop !== "string" || prop === "then") return undefined;
        return (...args: any[]) =>
          import("@wailsio/runtime").then((mod: any) => {
            const byName = mod.Call?.ByName;
            if (typeof byName !== "function") throw new Error("Wails Call.ByName missing");
            return byName(name(prop), ...args);
          });
      },
    },
  );
}

/**
 * wailsService 解析这一次调用该用的宿主绑定。
 * 原生窗口优先走 Wails 生成的 Service；生成模块没有可用导出时改走 Call.ByName。
 * 浏览器门 onBrowserDoor 为真时返回空，调用方的请求走 /api。
 * 端到端开关 VITE_E2E 打开时同样返回空。
 * 桌面 webview 不提供这些 HTTP 路由，绑定模块缺失时不能靠退回 /api。
 */
async function wailsService(): Promise<any | null> {
  // 端到端和浏览器门都不走生成绑定，调用方改打 /api。
  if (import.meta.env.VITE_E2E) return null;
  if (onBrowserDoor()) return null;
  const loaders = Object.values(boundService);
  if (loaders.length) {
    try {
      const svc = unwrapService(await loaders[0]!());
      if (svc) return svc;
    } catch {
      /* 运行时没有生成出来的模块 */
    }
  }
  if (nativeDesktop()) return namedDesktopService();
  // 生成模块缺失，且当前不是桌面窗口时返回空。桌面 webview 没有这些 HTTP 路由，这里不能改去打 /api。
  return null;
}

/** svcMethod 在绑定对象上按候选名找一个可调用的方法。s 为假时返回 null，否则返回 null；按下标读取，传给 bind。names 用来遍历。 */
function svcMethod(s: any, ...names: string[]): ((...args: any[]) => any) | null {
  if (!s) return null;
  for (const n of names) {
    if (typeof s[n] === "function") return s[n].bind(s);
  }
  return null;
}

/** timedSignal 做出到时中止的信号。宿主有 AbortSignal.timeout 就用它。ms 传给 timeout，传给 setTimeout。 */
function timedSignal(ms: number): AbortSignal {
  const timeout = (AbortSignal as { timeout?: (n: number) => AbortSignal }).timeout;
  if (typeof timeout === "function") return timeout(ms);
  const ctrl = new AbortController();
  setTimeout(() => ctrl.abort(), ms);
  return ctrl.signal;
}

/** http 按浏览器门的头去拉 JSON。失败抛 ApiError。解析不了的响应体保留原文。path 拼进字符串，传给 fetch，写进 地址。init 传给 browserFetchInit。 */
async function http<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    const next = browserFetchInit(init);
    res = await fetch(path, {
      ...next,
      signal: next.signal ?? timedSignal(8000),
    });
  } catch (e) {
    throw new Error(`${path}: ${errMessage(e)}`);
  }
  const text = await res.text();
  let body: any = text;
  try {
    body = text ? JSON.parse(text) : {};
  } catch {
    /* 保留原文 */
  }
  if (!res.ok) {
    const raw = typeof body === "object" && body?.error ? body.error : text || res.statusText;
    throw new ApiError(res.status, body, `${path}: ${raw}`);
  }
  return body as T;
}

/** parseAuthMode 收成 ask、auto_edit 或 full，对不上就用 default。v 转成字符串，转成小写，做替换。 */
export function parseAuthMode(v: any): AuthMode {
  const s = str(v).toLowerCase().replace(/[-\s]/g, "_");
  if (s === "ask" || s === "ask_every_time" || s === "strict") return "ask";
  if (s === "auto_edit" || s === "autoedit") return "auto_edit";
  if (s === "full" || s === "full_access") return "full";
  if (!s) return "default";
  return "default";
}

/** channelOf 保留 video 和 learn，其余收成 agent。raw 与 video、learn 比较是否相等。 */
function channelOf(raw: string): ThreadChannel {
  if (raw === "video") return "video";
  if (raw === "learn") return "learn";
  return "agent";
}

/** threadOf 把宿主的会话收成线程。隔离时工具根用 worktree。v 用 pick 读取 origin_workspace、worktree、isolate、workspace、id、title 等字段。 */
export function threadOf(v: any): Thread {
  const workspace = str(pick(v, "origin_workspace", "OriginWorkspace", "workspace", "Workspace"));
  const worktree = str(pick(v, "worktree", "Worktree"));
  const isolate = bool(pick(v, "isolate", "Isolate"));
  const toolRoot = isolate && worktree ? worktree : str(pick(v, "workspace", "Workspace"), workspace);
  return {
    id: str(pick(v, "id", "ID")),
    title: str(pick(v, "title", "Title"), "thread"),
    workspace,
    toolRoot,
    worktree,
    originWorkspace: workspace,
    isolate,
    harness: str(pick(v, "harness", "Harness")),
    createdAt: str(pick(v, "created_at", "CreatedAt")),
    updatedAt: str(pick(v, "updated_at", "UpdatedAt")),
    preview: str(pick(v, "preview", "Preview")),
    archived: bool(pick(v, "archived", "Archived")),
    pinned: bool(pick(v, "pinned", "Pinned")),
    model: str(pick(v, "model", "Model")),
    connectionId: str(pick(v, "connection_id", "ConnectionID", "connectionId")),
    authMode: parseAuthMode(pick(v, "auth_mode", "AuthMode", "authMode")),
    pinnedSkills: asArray(pick(v, "pinned_skills", "PinnedSkills")).map(String).filter(Boolean),
    loadedSkills: asArray(pick(v, "loaded_skills", "LoadedSkills")).map(String).filter(Boolean),
    channel: channelOf(str(pick(v, "channel", "Channel"))),
    interrupted: bool(pick(v, "interrupted", "Interrupted")),
    queued: num(pick(v, "queued", "Queued")),
    parentId: str(pick(v, "parent_id", "ParentID", "parentId")),
    planText: str(pick(v, "plan_text", "PlanText", "planText")),
    profileId: str(pick(v, "profile_id", "ProfileID", "profileId")),
    pageId: str(pick(v, "page_id", "PageID", "pageId")),
    libraryId: str(pick(v, "library_id", "LibraryID", "libraryId")),
    pathId: str(pick(v, "path_id", "PathID", "pathId")),
    materialId: str(pick(v, "material_id", "MaterialID", "materialId")),
    researchBrief: str(pick(v, "research_brief", "ResearchBrief", "researchBrief")),
    researchShot: str(pick(v, "research_shot", "ResearchShot", "researchShot")),
  };
}

/** healthOf 把宿主的健康载荷收成 Health。v 用 pick 读取 usage、ok、harness、model、version、isolated 等字段。 */
export function healthOf(v: any): Health {
  const usage = pick(v, "usage") || {};
  return {
    ok: bool(pick(v, "ok")),
    harness: str(pick(v, "harness")),
    model: str(pick(v, "model")),
    version: str(pick(v, "version"), "0.1.0"),
    isolated: bool(pick(v, "isolated")),
    isolationKind: str(pick(v, "isolation_kind", "isolationKind") || pick(pick(pick(v, "isolation"), "os") || {}, "kind", "Kind")),
    budgetUsd: num(pick(v, "budget_usd", "budgetUsd")),
    usageUsd: num(pick(usage, "usd")),
    workspaceReady: bool(pick(v, "workspace_ready", "workspaceReady")),
    videoWorkspace: str(pick(v, "video_workspace", "VideoWorkspace", "videoWorkspace")),
    videoWorkspaceReady: bool(pick(v, "video_workspace_ready", "videoWorkspaceReady")),
    chatProviders: asArray(pick(v, "chat_providers", "ChatProviders", "chatProviders")).map(chatProviderOf),
    paused: bool(pick(v, "paused", "Paused")),
  };
}

/** chatProviderOf 把一条聊天供应商收成 ChatProvider，并保证当前模型在列表里。v 用 pick 读取 models、model、id、name、vendor、endpoint 等字段。 */
export function chatProviderOf(v: any): ChatProvider {
  const models = asArray(pick(v, "models", "Models")).map(String).filter(Boolean);
  const model = str(pick(v, "model", "Model"));
  if (model && !models.includes(model)) models.unshift(model);
  return {
    id: str(pick(v, "id", "ID")),
    name: str(pick(v, "name", "Name")),
    vendor: str(pick(v, "vendor", "Vendor")),
    endpoint: str(pick(v, "endpoint", "Endpoint", "base_url", "BaseURL", "baseUrl")),
    model,
    models,
    hasKey: bool(pick(v, "has_key", "HasKey", "hasKey")),
    isDefault: bool(pick(v, "is_default", "IsDefault", "isDefault")),
    active: pick(v, "active", "Active") == null ? true : bool(pick(v, "active", "Active")),
  };
}

/** configOf 把宿主的配置收成 AppConfig。v 用 pick 读取 provider、model、base_url、workspace、video_workspace、auto_allow 等字段。 */
export function configOf(v: any): AppConfig {
  return {
    provider: str(pick(v, "provider"), "openai"),
    model: str(pick(v, "model")),
    baseUrl: str(pick(v, "base_url", "BaseURL", "baseUrl")),
    workspace: str(pick(v, "workspace", "Workspace")),
    videoWorkspace: str(pick(v, "video_workspace", "VideoWorkspace", "videoWorkspace")),
    autoAllow: bool(pick(v, "auto_allow", "AutoAllow", "autoAllow")),
    maxBudgetUsd: num(pick(v, "max_budget_usd", "MaxBudgetUSD", "maxBudgetUsd")),
    usdPerMtok: num(pick(v, "usd_per_mtok", "USDPerMTok", "usdPerMtok")),
    models: asArray(pick(v, "models", "Models")).map(String).filter(Boolean),
    closeToTray: bool(pick(v, "close_to_tray", "CloseToTray", "closeToTray")),
    companionEnabled: boolOr(pick(v, "companion_enabled", "CompanionEnabled", "companionEnabled"), true),
    updateUrl: str(pick(v, "update_url", "UpdateURL", "updateUrl")),
    updateAutoCheck: boolOr(pick(v, "update_auto_check", "UpdateAutoCheck", "updateAutoCheck"), true),
    keymap: (pick(v, "keymap", "Keymap") || {}) as Record<string, string>,
    locale: str(pick(v, "locale", "Locale")),
    alwaysOnTop: bool(pick(v, "always_on_top", "AlwaysOnTop", "alwaysOnTop")),
    startAtLogin: bool(pick(v, "start_at_login", "StartAtLogin", "startAtLogin")),
    notificationsEnabled: boolOr(pick(v, "notifications_enabled", "NotificationsEnabled", "notificationsEnabled"), true),
    notifyWhenUnfocusedOnly: bool(pick(v, "notify_when_unfocused_only", "NotifyWhenUnfocusedOnly", "notifyWhenUnfocusedOnly")),
    uiScale: snapUiScale(num(pick(v, "ui_scale", "UIScale", "uiScale"), 1) || 1),
    showThinking: bool(pick(v, "show_thinking", "ShowThinking", "showThinking")),
    transcriptView: str(pick(v, "transcript_view", "TranscriptView", "transcriptView")),
    collapseTiming: str(pick(v, "collapse_timing", "CollapseTiming", "collapseTiming")),
    updateChannel: str(pick(v, "update_channel", "UpdateChannel", "updateChannel"), "latest"),
    theme: str(pick(v, "theme", "Theme"), "system"),
    paletteDark: str(pick(v, "palette_dark", "PaletteDark", "paletteDark"), "ink"),
    paletteLight: str(pick(v, "palette_light", "PaletteLight", "paletteLight"), "neutral"),
    skin: str(pick(v, "skin", "Skin")),
    gateMode: str(pick(v, "gate_mode", "GateMode", "gateMode"), "manual"),
    crashResume: boolOr(pick(v, "crash_resume", "CrashResume", "crashResume"), true),
    searchUrl: str(pick(v, "search_url", "SearchURL", "searchUrl")),
    searchKey: str(pick(v, "search_key", "SearchKey", "searchKey")),
    paused: bool(pick(v, "paused", "Paused")),
  };
}

/** approvalOf 把一条待审批收成 Approval。v 用 pick 读取 request、id、session_id字段，作为后备值。 */
export function approvalOf(v: any): Approval {
  const req = pick(v, "request", "Request") || v;
  return {
    id: str(pick(v, "id", "ID")),
    action: str(pick(req, "action", "Action")),
    command: str(pick(req, "command", "Command")),
    path: str(pick(req, "path", "Path")),
    level: str(pick(req, "level", "Level")),
    sessionId: str(pick(req, "session_id", "SessionID", "sessionId") ?? pick(v, "session_id", "SessionID", "sessionId")),
  };
}

/** hasOwnNum 这些键里是否至少有一个非空值。v 为假时走另一支，按下标读取，用 typeof 看类型。keys 调用 some。 */
function hasOwnNum(v: any, ...keys: string[]): boolean {
  if (!v || typeof v !== "object") return false;
  return keys.some((k) => v[k] != null && v[k] !== "");
}

/** contextOf 把宿主的用量收成 ContextUsage。对话或身份词元字段出现时记下分片已知。v 用 pick 读取 tokens、budget、window、prefix_tokens、dynamic_tokens、schema_tokens 等字段，传给 hasOwnNum。 */
export function contextOf(v: any): ContextUsage {
  const slicesKnown = hasOwnNum(v, "chat_tokens", "ChatTokens", "identity_tokens", "IdentityTokens");
  return {
    tokens: num(pick(v, "tokens", "Tokens")),
    budget: num(pick(v, "budget", "Budget")),
    window: num(pick(v, "window", "Window")),
    prefixTokens: num(pick(v, "prefix_tokens", "PrefixTokens")),
    dynamicTokens: num(pick(v, "dynamic_tokens", "DynamicTokens")),
    schemaTokens: num(pick(v, "schema_tokens", "SchemaTokens")),
    identityTokens: num(pick(v, "identity_tokens", "IdentityTokens")),
    rulesTokens: num(pick(v, "rules_tokens", "RulesTokens")),
    catalogTokens: num(pick(v, "catalog_tokens", "CatalogTokens")),
    skillBodyTokens: num(pick(v, "skill_body_tokens", "SkillBodyTokens")),
    memoryTokens: num(pick(v, "memory_tokens", "MemoryTokens")),
    todayTokens: num(pick(v, "today_tokens", "TodayTokens")),
    notesTokens: num(pick(v, "notes_tokens", "NotesTokens")),
    chatTokens: slicesKnown ? num(pick(v, "chat_tokens", "ChatTokens")) : undefined,
    slicesKnown,
    providerPrompt: num(pick(v, "provider_prompt", "ProviderPrompt")),
    cachedTokens: num(pick(v, "cached_tokens", "CachedTokens")),
    cacheReported: !!pick(v, "cache_reported", "CacheReported"),
    cacheStable: !!pick(v, "cache_stable", "CacheStable"),
    prefixHash: str(pick(v, "prefix_hash", "PrefixHash")),
    dynamicAt: str(pick(v, "dynamic_at", "DynamicAt")),
    trigger: str(pick(v, "trigger", "Trigger")),
    hydrated: num(pick(v, "hydrated", "Hydrated")),
    note: str(pick(v, "note", "Note")),
    layers: asArray(pick(v, "layers", "Layers")).map(String),
    elided: num(pick(v, "elided", "Elided")),
  };
}

/** hunkOf 把一条差异块收成 Hunk。v 用 pick 读取 id、file、header、body字段。 */
export function hunkOf(v: any): Hunk {
  return {
    id: str(pick(v, "id", "ID")),
    file: str(pick(v, "file", "File")),
    header: str(pick(v, "header", "Header")),
    body: str(pick(v, "body", "Body")),
  };
}

/** health 走 Health 或 /api/health。 */
export async function health(): Promise<Health> {
  const s = await wailsService();
  return healthOf(s?.Health ? await s.Health() : await http("/api/health"));
}

/** getConfig 走 GetConfig 或 /api/config。 */
export async function getConfig(): Promise<AppConfig> {
  const s = await wailsService();
  return configOf(s?.GetConfig ? await s.GetConfig() : await http("/api/config"));
}

/**
 * setConfig 提交配置，更新通道固定为 latest。有 SetConfig 时走它，并在有 locale 时再走 SetLocale；否则 POST /api/config。
 * cfg 读取 provider、model、baseUrl、workspace、videoWorkspace、autoAllow 等。
 */
export async function setConfig(cfg: AppConfig): Promise<void> {
  const body = {
    provider: cfg.provider,
    model: cfg.model,
    base_url: cfg.baseUrl,
    workspace: cfg.workspace,
    video_workspace: cfg.videoWorkspace || "",
    auto_allow: cfg.autoAllow,
    max_budget_usd: cfg.maxBudgetUsd,
    usd_per_mtok: cfg.usdPerMtok,
    models: cfg.models,
    close_to_tray: cfg.closeToTray,
    companion_enabled: cfg.companionEnabled !== false,
    CompanionEnabled: cfg.companionEnabled !== false,
    update_url: cfg.updateUrl,
    update_auto_check: cfg.updateAutoCheck !== false,
    locale: cfg.locale || "",
    Locale: cfg.locale || "",
    keymap: cfg.keymap || {},
    always_on_top: !!cfg.alwaysOnTop,
    start_at_login: !!cfg.startAtLogin,
    notifications_enabled: cfg.notificationsEnabled !== false,
    notify_when_unfocused_only: !!cfg.notifyWhenUnfocusedOnly,
    ui_scale: snapUiScale(cfg.uiScale || 1),
    show_thinking: !!cfg.showThinking,
    transcript_view: cfg.transcriptView || "",
    collapse_timing: cfg.collapseTiming || "",
    update_channel: "latest",
    theme: cfg.theme || "system",
    palette_dark: cfg.paletteDark || "ink",
    palette_light: cfg.paletteLight || "neutral",
    skin: cfg.skin || "",
    gate_mode: cfg.gateMode || "manual",
    crash_resume: cfg.crashResume !== false,
    search_url: cfg.searchUrl || "",
    search_key: cfg.searchKey || "",
  };
  const s = await wailsService();
  if (s?.SetConfig) {
    await s.SetConfig(body);
    const loc = svcMethod(s, "SetLocale", "setLocale");
    if (loc && cfg.locale) await loc(cfg.locale);
    return;
  }
  await http("/api/config", { method: "POST", body: JSON.stringify(body) });
}

/** tokenMapOrUndef 非空对象才当成色板，否则空。v 为假时走另一支，用 typeof 看类型，传给 Array.isArray，传给 Object.keys。 */
function tokenMapOrUndef(v: any): TokenMap | undefined {
  if (!v || typeof v !== "object" || Array.isArray(v)) return undefined;
  const keys = Object.keys(v);
  if (!keys.length) return undefined;
  return v as TokenMap;
}

/** skinInfoOf 把宿主的皮肤摘要收成 SkinInfo。v 用 pick 读取 id、name、author、description、builtin、preserveEvidence 等字段。 */
function skinInfoOf(v: any): SkinInfo {
  return {
    id: str(pick(v, "id", "ID")),
    name: str(pick(v, "name", "Name")),
    author: str(pick(v, "author", "Author")),
    description: str(pick(v, "description", "Description")),
    builtin: bool(pick(v, "builtin", "Builtin")),
    preserveEvidence: boolOr(pick(v, "preserveEvidence", "PreserveEvidence"), true),
    capabilities: asArray(pick(v, "capabilities", "Capabilities")).map(String),
    hasWallpaper: bool(pick(v, "hasWallpaper", "HasWallpaper")),
    hasFonts: bool(pick(v, "hasFonts", "HasFonts")),
    dark: str(pick(v, "dark", "Dark")),
    light: str(pick(v, "light", "Light")),
    wallpaperUrl: str(pick(v, "wallpaperUrl", "WallpaperURL", "wallpaper_url")),
    previewUrl: str(pick(v, "previewUrl", "PreviewURL", "preview_url")),
    previewDark: tokenMapOrUndef(pick(v, "previewDark", "PreviewDark")),
    previewLight: tokenMapOrUndef(pick(v, "previewLight", "PreviewLight")),
    materials: (() => {
      const mats = pick(v, "materials", "Materials") || {};
      const fit = str(pick(mats, "wallpaperFit", "WallpaperFit"), "cover");
      return {
        grain: num(pick(mats, "grain", "Grain"), 0.04),
        glassBlur: str(pick(mats, "glassBlur", "GlassBlur")),
        wallpaperFit: (fit === "contain" || fit === "tile" ? fit : "cover") as "cover" | "contain" | "tile",
        wallpaperDim: num(pick(mats, "wallpaperDim", "WallpaperDim"), 0.45),
        radiusScale: num(pick(mats, "radiusScale", "RadiusScale"), 1) || 1,
      };
    })(),
  };
}

/** skinResolvedOf 把解析后的皮肤收成 SkinResolved。v 用 pick 读取 tokens、materials、fonts、windowRgb、contrast、id 等字段。 */
function skinResolvedOf(v: any): SkinResolved {
  const tokens = (pick(v, "tokens", "Tokens") || {}) as TokenMap;
  const mats = pick(v, "materials", "Materials") || {};
  const fonts = asArray(pick(v, "fonts", "Fonts")).map((f: any) => ({
    family: str(pick(f, "family", "Family")),
    url: str(pick(f, "url", "URL")),
    role: str(pick(f, "role", "Role")),
  }));
  const rgb = asArray(pick(v, "windowRgb", "WindowRGB", "window_rgb")).map(Number);
  const contrast = pick(v, "contrast", "Contrast") || {};
  const fit = str(pick(mats, "wallpaperFit", "WallpaperFit"), "cover");
  return {
    id: str(pick(v, "id", "ID")),
    name: str(pick(v, "name", "Name")),
    builtin: bool(pick(v, "builtin", "Builtin")),
    mode: str(pick(v, "mode", "Mode"), "dark") === "light" ? "light" : "dark",
    preserveEvidence: boolOr(pick(v, "preserveEvidence", "PreserveEvidence"), true),
    tokens: tokens && typeof tokens === "object" ? tokens : {},
    windowRgb: rgb.length >= 3 ? rgb.slice(0, 3) : [28, 28, 26],
    wallpaperUrl: str(pick(v, "wallpaperUrl", "WallpaperURL", "wallpaper_url")),
    wallpaperFile: str(pick(v, "wallpaperFile", "WallpaperFile", "wallpaper_file")),
    fonts,
    materials: {
      grain: num(pick(mats, "grain", "Grain"), 0.04),
      glassBlur: str(pick(mats, "glassBlur", "GlassBlur")),
      wallpaperFit: fit === "contain" || fit === "tile" ? fit : "cover",
      wallpaperDim: num(pick(mats, "wallpaperDim", "WallpaperDim"), 0.45),
      radiusScale: num(pick(mats, "radiusScale", "RadiusScale"), 1) || 1,
    },
    contrast: {
      foreground: num(pick(contrast, "foreground", "Foreground")),
      muted: num(pick(contrast, "muted", "Muted")),
      pass: boolOr(pick(contrast, "pass", "Pass"), true),
      reason: str(pick(contrast, "reason", "Reason")),
    },
  };
}

/** listSkins 走 ListSkins 或 /api/skins。 */
export async function listSkins(): Promise<SkinInfo[]> {
  const s = await wailsService();
  const fn = svcMethod(s, "ListSkins", "listSkins");
  const raw = fn ? await fn() : await http("/api/skins");
  return asArray(raw).map(skinInfoOf);
}

/** getSkin 走 GetSkin 或 /api/skins?id=。 */
export async function getSkin(id: string): Promise<{ info: SkinInfo; dark: TokenMap; light: TokenMap }> {
  const s = await wailsService();
  const fn = svcMethod(s, "GetSkin", "getSkin");
  const raw = fn ? await fn(id) : await http(`/api/skins?id=${encodeURIComponent(id)}`);
  return {
    info: skinInfoOf(pick(raw, "info", "Info") || raw),
    dark: (pick(raw, "dark", "Dark") || {}) as TokenMap,
    light: (pick(raw, "light", "Light") || {}) as TokenMap,
  };
}

/** importSkinB64 走 ImportSkin，或 POST /api/skins 且 op 为 import。bytesB64 传给 ImportSkin，放进 op 为 import 的请求体。 */
export async function importSkinB64(bytesB64: string): Promise<SkinInfo> {
  const s = await wailsService();
  const fn = svcMethod(s, "ImportSkin", "importSkin");
  const raw = fn
    ? await fn(bytesB64)
    : await http("/api/skins", { method: "POST", body: JSON.stringify({ op: "import", bytes_b64: bytesB64 }) });
  return skinInfoOf(raw);
}

/** importSkinPath 走 ImportSkinPath，或 POST /api/skins 且 op 为 import。path 传给 ImportSkinPath，放进 op 为 import 的请求体。 */
export async function importSkinPath(path: string): Promise<SkinInfo> {
  const s = await wailsService();
  const fn = svcMethod(s, "ImportSkinPath", "importSkinPath");
  const raw = fn
    ? await fn(path)
    : await http("/api/skins", { method: "POST", body: JSON.stringify({ op: "import", path }) });
  return skinInfoOf(raw);
}

/** exportSkin 走 ExportSkin，或 POST /api/skins 且 op 为 export。id 拼进字符串，传给 ExportSkin，放进 op 为 export 的请求体。 */
export async function exportSkin(id: string): Promise<{ filename: string; bytesB64: string }> {
  const s = await wailsService();
  const fn = svcMethod(s, "ExportSkin", "exportSkin");
  const raw = fn
    ? await fn(id)
    : await http("/api/skins", { method: "POST", body: JSON.stringify({ op: "export", id }) });
  return {
    filename: str(pick(raw, "filename", "Filename"), `${id}.yoyoskin`),
    bytesB64: str(pick(raw, "bytes_b64", "bytesB64", "BytesB64")),
  };
}

/** deleteSkin 走 DeleteSkin，或 POST /api/skins 且 op 为 delete。id 传给 DeleteSkin，放进 op 为 delete 的请求体。 */
export async function deleteSkin(id: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "DeleteSkin", "deleteSkin");
  if (fn) {
    await fn(id);
    return;
  }
  await http("/api/skins", { method: "POST", body: JSON.stringify({ op: "delete", id }) });
}

/** activateSkin 走 ActivateSkin，或 POST /api/skins 且 op 为 activate。id 传给 ActivateSkin，放进 op 为 activate 的请求体。 */
export async function activateSkin(id: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "ActivateSkin", "activateSkin");
  if (fn) {
    await fn(id);
    return;
  }
  await http("/api/skins", { method: "POST", body: JSON.stringify({ op: "activate", id }) });
}

/** resolveSkin 走 ResolveSkin，或 /api/skins?id= 且带 resolve=1。mode 拼进字符串，传给 ResolveSkin，写进 /api/skins?id={…}&resolve=1&mode={mode}。 */
export async function resolveSkin(id: string, mode: "dark" | "light"): Promise<SkinResolved> {
  const s = await wailsService();
  const fn = svcMethod(s, "ResolveSkin", "resolveSkin");
  const raw = fn
    ? await fn(id, mode)
    : await http(`/api/skins?id=${encodeURIComponent(id)}&resolve=1&mode=${mode}`);
  return skinResolvedOf(raw);
}

/** SkinSaveIn 是保存皮肤时交给宿主的字段。 */
export type SkinSaveIn = {
  id?: string;
  name: string;
  author?: string;
  description?: string;
  preserveEvidence?: boolean;
  dark: TokenMap;
  light: TokenMap;
  materials?: ReturnType<typeof defaultMaterials>;
  wallpaper_b64?: string;
  wallpaper_name?: string;
  font_sans_b64?: string;
  font_mono_b64?: string;
  preview_b64?: string;
};

/** saveSkin 走 SaveSkin，或 POST /api/skins 且 op 为 save。in_ 传给 SaveSkin，放进 op 为 save 的请求体，展开传递。 */
export async function saveSkin(in_: SkinSaveIn): Promise<SkinInfo> {
  const s = await wailsService();
  const fn = svcMethod(s, "SaveSkin", "saveSkin");
  const raw = fn
    ? await fn(in_)
    : await http("/api/skins", { method: "POST", body: JSON.stringify({ op: "save", ...in_ }) });
  return skinInfoOf(raw);
}

/** raiseSession 走 RaiseSession；没有该方法则改走 raiseWindow。id 传给 RaiseSession。 */
export async function raiseSession(id: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "RaiseSession", "raiseSession");
  if (fn) {
    await fn(id);
    return;
  }
  await raiseWindow();
}

/** raiseWindow 走 RaiseWindow。 */
export async function raiseWindow(): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "RaiseWindow", "raiseWindow");
  if (fn) await fn();
}

/** showCompanion 走 ShowCompanion。 */
export async function showCompanion(): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "ShowCompanion", "showCompanion");
  if (fn) await fn();
}

/** dismissCompanion 先走 HideCompanion，没有再走 DismissCompanion。 */
export async function dismissCompanion(): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "HideCompanion", "hideCompanion") || svcMethod(s, "DismissCompanion", "dismissCompanion");
  if (fn) await fn();
}

/** setCompanionCaption 走 SetCompanionCaption。on 传给 SetCompanionCaption。 */
export async function setCompanionCaption(on: boolean): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "SetCompanionCaption", "setCompanionCaption");
  if (fn) await fn(on);
}

/** setCompanionCaptionSize 走 SetCompanionCaptionSize。w 四舍五入，传给 SetCompanionCaptionSize。h 四舍五入，传给 SetCompanionCaptionSize。 */
export async function setCompanionCaptionSize(w: number, h: number): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "SetCompanionCaptionSize", "setCompanionCaptionSize");
  if (fn) await fn(Math.round(w), Math.round(h));
}

/** placeCompanion 走 PlaceCompanion。x 四舍五入，传给 PlaceCompanion。y 四舍五入，传给 PlaceCompanion。 */
export async function placeCompanion(x: number, y: number): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "PlaceCompanion", "placeCompanion");
  if (fn) await fn(Math.round(x), Math.round(y));
}

/** setAPIKey 只提交密钥，走 SetAPIKey 或 POST /api/key。value 传给 SetAPIKey，放进请求体。 */
export async function setAPIKey(value: string): Promise<void> {
  const s = await wailsService();
  if (s?.SetAPIKey) return s.SetAPIKey(value);
  await http("/api/key", { method: "POST", body: JSON.stringify({ value }) });
}

/**
 * createSession 优先走 CreateSessionOn；否则走 CreateSession，agent 时直接返回，video 且有 SetSessionChannel 时再走它；对不上则 POST /api/sessions。
 * workspace 传给 CreateSessionOn，传给 CreateSession，放进请求体。
 * channel 与 video、agent 比较是否相等，传给 CreateSessionOn，放进请求体。
 */
export async function createSession(workspace: string, channel: ThreadChannel = "agent"): Promise<Thread> {
  const s = await wailsService();
  if (s?.CreateSessionOn) {
    return threadOf(await s.CreateSessionOn(workspace, channel));
  }
  if (s?.CreateSession) {
    const t = threadOf(await s.CreateSession(workspace));
    if (channel !== "agent" && s.SetSessionChannel) {
      return threadOf(await s.SetSessionChannel(t.id, channel));
    }
    if (channel === "agent") return t;
  }
  const raw = await http("/api/sessions", { method: "POST", body: JSON.stringify({ workspace, channel }) });
  return threadOf(raw);
}

/** listSessions 走 ListSessions 或 /api/sessions。 */
export async function listSessions(): Promise<Thread[]> {
  const s = await wailsService();
  const raw = s?.ListSessions ? await s.ListSessions() : await http("/api/sessions");
  return asArray(raw).map(threadOf).filter((t) => t.id);
}

/**
 * searchSessions 查询为空时改走 listSessions；否则走 SearchSessions 或 /api/sessions?q=。
 * query 去掉首尾空白后，编码，为假时走另一支，拼进字符串，传给 SearchSessions，写进 /api/sessions?q={q}{…}。
 * includeArchived 为真时才继续，传给 SearchSessions，写进 /api/sessions?q={…}{includeArchived}。
 */
export async function searchSessions(query: string, includeArchived = false): Promise<Thread[]> {
  const s = await wailsService();
  const q = query.trim();
  if (!q) return listSessions();
  const raw = s?.SearchSessions
    ? await s.SearchSessions(q, includeArchived)
    : await http(`/api/sessions?q=${encodeURIComponent(q)}${includeArchived ? "&archived=1" : ""}`);
  return asArray(raw).map(threadOf).filter((t) => t.id);
}

/** trajectory 走 Trajectory 或 /api/sessions/{id}/trajectory。 */
export async function trajectory(id: string): Promise<any[]> {
  const s = await wailsService();
  if (s?.Trajectory) return asArray(await s.Trajectory(id));
  return asArray(await http(`/api/sessions/${id}/trajectory`));
}

/** TrajectoryPage 是一页轨迹，带事件和前后序号。 */
export type TrajectoryPage = {
  events: any[];
  headSeq: number;
  tailSeq: number;
  older: boolean;
  hubSeq: number;
};

/** trajectoryPageOf 把宿主的一页轨迹收成 TrajectoryPage。v 用 pick 读取 events、head_seq、tail_seq、older、hub_seq字段，为真时才继续，用 typeof 看类型，作为后备值，传给 Array.isArray。 */
function trajectoryPageOf(v: any): TrajectoryPage {
  if (v && typeof v === "object" && !Array.isArray(v) && v.result && typeof v.result === "object" && !Array.isArray(v.result)) {
    const inner = v.result;
    if (pick(inner, "events", "Events") != null || pick(inner, "head_seq", "HeadSeq") != null || pick(inner, "hub_seq", "HubSeq") != null) {
      v = inner;
    }
  }
  return {
    events: asArray(pick(v, "events", "Events") ?? v),
    headSeq: num(pick(v, "head_seq", "HeadSeq")),
    tailSeq: num(pick(v, "tail_seq", "TailSeq")),
    older: asBool(pick(v, "older", "Older")),
    hubSeq: num(pick(v, "hub_seq", "HubSeq")),
  };
}

/** trajectoryPage 走 TrajectoryPage 或 /api/sessions/{id}/trajectory/page。opts 读取 before、turns，为空时不读。 */
export async function trajectoryPage(id: string, opts?: { before?: number; turns?: number }): Promise<TrajectoryPage> {
  const s = await wailsService();
  const fn = svcMethod(s, "TrajectoryPage", "trajectoryPage");
  if (fn) return trajectoryPageOf(await fn(id, opts?.before || 0, opts?.turns || 0));
  const q = new URLSearchParams();
  if (opts?.before) q.set("before", String(opts.before));
  if (opts?.turns) q.set("turns", String(opts.turns));
  const qs = q.toString();
  return trajectoryPageOf(await http(`/api/sessions/${id}/trajectory/page${qs ? `?${qs}` : ""}`));
}

/** unwrapResult 宿主把结果包在 result 里时拆出来。v 读取 result，为真时才继续，用 typeof 看类型，传给 Array.isArray。 */
function unwrapResult(v: any): any {
  if (v && typeof v === "object" && !Array.isArray(v) && v.result != null) return v.result;
  return v;
}

/** trajectoryOutline 走 TrajectoryOutline 或 /api/sessions/{id}/trajectory/outline。 */
export async function trajectoryOutline(id: string): Promise<OutlineTurn[]> {
  const s = await wailsService();
  const fn = svcMethod(s, "TrajectoryOutline", "trajectoryOutline");
  const raw = fn ? await fn(id) : await http(`/api/sessions/${id}/trajectory/outline`);
  return asArray(unwrapResult(raw)).map(outlineOfRaw).filter((t) => t.seq > 0 || t.id || t.key);
}

/**
 * trajectoryAround 走 TrajectoryAround 或 /api/sessions/{id}/trajectory/around。
 * seq 转成字符串，为真时才继续，传给 TrajectoryAround，交给 q.set。
 * turns 为假时用 0，转成字符串，传给 TrajectoryAround，交给 q.set。
 */
export async function trajectoryAround(id: string, seq: number, turns?: number): Promise<TrajectoryPage> {
  const s = await wailsService();
  const fn = svcMethod(s, "TrajectoryAround", "trajectoryAround");
  if (fn) return trajectoryPageOf(await fn(id, seq, turns || 0));
  const q = new URLSearchParams();
  if (seq) q.set("seq", String(seq));
  if (turns) q.set("turns", String(turns));
  const qs = q.toString();
  return trajectoryPageOf(await http(`/api/sessions/${id}/trajectory/around${qs ? `?${qs}` : ""}`));
}

/** LiveBatch 是按全局序号从中心环拉的一页。gap 是空洞，要重新加载。more 表示先按顺序应用 events，再拉下一页。 */
export type LiveBatch = { events: any[]; head: number; gap: boolean; more: boolean };

/** liveSync 走 LiveSync 或 /api/sessions/{id}/live/sync。after 转成字符串，编码，拼进字符串，传给 LiveSync，写进 /api/sessions/{…}/live/sync?after={after}。 */
export async function liveSync(id: string, after: number): Promise<LiveBatch> {
  const s = await wailsService();
  const fn = svcMethod(s, "LiveSync", "liveSync");
  const raw = fn
    ? await fn(id, after)
    : await http(`/api/sessions/${id}/live/sync?after=${encodeURIComponent(String(after))}`);
  return {
    events: asArray(raw?.events ?? raw?.Events),
    head: Number(raw?.head ?? raw?.Head) || 0,
    gap: raw?.gap === true || raw?.Gap === true,
    more: raw?.more === true || raw?.More === true,
  };
}

/** setRendererMemory 走 SetRendererMemory，失败也被吞掉；没有该方法则不发请求。level 传给 SetRendererMemory。 */
export async function setRendererMemory(level: "low" | "normal"): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "SetRendererMemory", "setRendererMemory");
  if (fn) {
    await fn(level).catch(() => {});
    return;
  }
}

/** sessionTraceOf 把宿主的会话轨迹收成 SessionTrace。v 用 pick 读取 stats、events、artifacts、session_id、title、workspace 等字段，为真时才继续，用 typeof 看类型。 */
export function sessionTraceOf(v: any): SessionTrace {
  if (v && typeof v === "object" && v.result && typeof v.result === "object" && !Array.isArray(v.result)) {
    v = v.result;
  }
  const statsRaw = pick(v, "stats", "Stats") || {};
  const stats: TraceStats = {
    events: num(pick(statsRaw, "events", "Events")),
    users: num(pick(statsRaw, "users", "Users")),
    assistants: num(pick(statsRaw, "assistants", "Assistants")),
    toolCalls: num(pick(statsRaw, "tool_calls", "ToolCalls")),
    toolResults: num(pick(statsRaw, "tool_results", "ToolResults")),
    errors: num(pick(statsRaw, "errors", "Errors")),
    compactions: num(pick(statsRaw, "compactions", "Compactions")),
    deltas: num(pick(statsRaw, "deltas", "Deltas")),
    tokens: num(pick(statsRaw, "tokens", "Tokens")),
    durationMs: num(pick(statsRaw, "duration_ms", "DurationMs")),
    spillBytes: num(pick(statsRaw, "spill_bytes", "SpillBytes")),
  };
  const events: TraceEvent[] = asArray(pick(v, "events", "Events")).map((ev: any, i: number) => ({
    index: num(pick(ev, "index", "Index"), i),
    ts: str(pick(ev, "ts", "TS")),
    type: str(pick(ev, "type", "Type")),
    source: str(pick(ev, "source", "Source")),
    lane: str(pick(ev, "lane", "Lane")) || "dialog",
    round: str(pick(ev, "round", "Round")),
    name: str(pick(ev, "name", "Name")),
    id: str(pick(ev, "id", "ID")),
    summary: str(pick(ev, "summary", "Summary")),
    detail: str(pick(ev, "detail", "Detail")),
    bytes: num(pick(ev, "bytes", "Bytes")),
    elapsedMs: num(pick(ev, "elapsed_ms", "ElapsedMs")),
    spillId: str(pick(ev, "spill_id", "SpillID", "spillId")),
    tokens: num(pick(ev, "tokens", "Tokens")),
    promptTokens: num(pick(ev, "prompt_tokens", "PromptTokens")),
    completionTokens: num(pick(ev, "completion_tokens", "CompletionTokens")),
    cachedTokens: num(pick(ev, "cached_tokens", "CachedTokens")),
    ttftMs: num(pick(ev, "ttft_ms", "TtftMs")),
    cacheReported: bool(pick(ev, "cache_reported", "CacheReported")),
    error: bool(pick(ev, "error", "Error")),
  }));
  const artifacts: TraceArtifact[] = asArray(pick(v, "artifacts", "Artifacts")).map((art: any) => ({
    kind: str(pick(art, "kind", "Kind")),
    id: str(pick(art, "id", "ID")),
    label: str(pick(art, "label", "Label")) || str(pick(art, "id", "ID")),
    bytes: num(pick(art, "bytes", "Bytes")),
    preview: str(pick(art, "preview", "Preview")),
  }));
  return {
    sessionId: str(pick(v, "session_id", "SessionID", "sessionId")),
    title: str(pick(v, "title", "Title")),
    workspace: str(pick(v, "workspace", "Workspace")),
    harness: str(pick(v, "harness", "Harness")),
    model: str(pick(v, "model", "Model")),
    createdAt: str(pick(v, "created_at", "CreatedAt")),
    startedAt: str(pick(v, "started_at", "StartedAt")),
    endedAt: str(pick(v, "ended_at", "EndedAt")),
    stats,
    events,
    artifacts,
  };
}

/** spillBlobOf 把溢出正文收成 SpillBlob。v 用 pick 读取 id、bytes、text、truncated字段，为真时才继续，用 typeof 看类型。 */
export function spillBlobOf(v: any): SpillBlob {
  if (v && typeof v === "object" && v.result && typeof v.result === "object" && !Array.isArray(v.result)) {
    v = v.result;
  }
  return {
    id: str(pick(v, "id", "ID")),
    bytes: num(pick(v, "bytes", "Bytes")),
    text: str(pick(v, "text", "Text")),
    truncated: bool(pick(v, "truncated", "Truncated")),
  };
}

/** sessionTrace 走 SessionTrace 或 /api/sessions/{id}/trace。 */
export async function sessionTrace(id: string): Promise<SessionTrace> {
  const s = await wailsService();
  const raw = s?.SessionTrace ? await s.SessionTrace(id) : await http(`/api/sessions/${id}/trace`);
  return sessionTraceOf(raw);
}

/**
 * spillBlob 走 SpillBlob 或 /api/sessions/{id}/spill/{blob}。
 * sessionID 拼进字符串，传给 SpillBlob，写进 /api/sessions/{sessionID}/spill/{…}。
 * blobID 编码，拼进字符串，传给 SpillBlob，写进 /api/sessions/{…}/spill/{blobID}。
 */
export async function spillBlob(sessionID: string, blobID: string): Promise<SpillBlob> {
  const s = await wailsService();
  const raw = s?.SpillBlob
    ? await s.SpillBlob(sessionID, blobID)
    : await http(`/api/sessions/${sessionID}/spill/${encodeURIComponent(blobID)}`);
  return spillBlobOf(raw);
}

/**
 * retry 优先走 RetrySession；否则用 StartSendOpts 带上 __resume__；再否则 POST /api/sessions/{id}/retry，状态 404 时改走 send。
 * sessionID 拼进字符串，传给 RetrySession，传给 StartSendOpts，传给 send，写进 /api/sessions/{sessionID}/retry。
 */
export async function retry(sessionID: string): Promise<void> {
  const s = await wailsService();
  const bound = svcMethod(s, "RetrySession", "retrySession");
  if (bound) {
    await bound(sessionID);
    return;
  }
  if (s?.StartSendOpts) {
    await s.StartSendOpts(sessionID, "", false, [{ name: "__resume__", path: "__resume__" }]);
    return;
  }
  try {
    await http(`/api/sessions/${sessionID}/retry`, { method: "POST" });
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      await send(sessionID, "", { attachments: [{ name: "__resume__", path: "__resume__" }] });
      return;
    }
    throw e;
  }
}

const REQUEST_ID_NAME = "__request_id__";

/** withRequestId 把请求 id 收进附件，并去掉旧的同名附件。atts 为假时改用后备值，做过滤。requestId 为假时用空字符串，转成字符串，去掉首尾空白后，为假时走另一支，写入字段 path。 */
function withRequestId(atts: Attachment[], requestId?: string): Attachment[] {
  const id = String(requestId || "").trim();
  const kept = (atts || []).filter((a) => a.name !== REQUEST_ID_NAME);
  if (!id) return kept;
  return [...kept, { name: REQUEST_ID_NAME, path: id }];
}

/**
 * send 优先走 StartSendOpts，否则 StartSend，再否则 POST /api/sessions/{id}/messages；报错含 queued 则当作已排队。
 * sessionID 拼进字符串，传给 StartSendOpts，传给 StartSend，写进 /api/sessions/{sessionID}/messages。
 * text 传给 StartSendOpts，传给 StartSend，放进请求体。
 * opts 读取 attachments、requestId、plan，为空时不读。
 */
export async function send(sessionID: string, text: string, opts?: { plan?: boolean; attachments?: Attachment[]; requestId?: string }): Promise<{ queued?: boolean }> {
  const s = await wailsService();
  const atts = withRequestId(opts?.attachments || [], opts?.requestId);
  try {
    if (s?.StartSendOpts) {
      const raw = await s.StartSendOpts(sessionID, text, !!opts?.plan, atts);
      return { queued: isQueuedResult(raw) };
    }
    if (s?.StartSend) {
      const raw = await s.StartSend(sessionID, text, !!opts?.plan);
      return { queued: isQueuedResult(raw) };
    }
    const raw = await http<any>(`/api/sessions/${sessionID}/messages`, {
      method: "POST",
      body: JSON.stringify({ text, async: true, plan: !!opts?.plan, request_id: opts?.requestId || "", attachments: atts }),
    });
    return { queued: !!raw?.queued || isQueuedResult(raw) };
  } catch (e) {
    const m = errMessage(e);
    if (/queued/i.test(m)) return { queued: true };
    throw e;
  }
}

/** isQueuedResult 返回值或文本表示这次发送进了队列。raw 为 null 时返回假；判断是否为空，用 typeof 看类型，传给 test。 */
function isQueuedResult(raw: any): boolean {
  if (raw == null) return false;
  if (typeof raw === "string") return /queued/i.test(raw);
  if (typeof raw === "object") return !!(raw.queued || raw.Queued);
  return false;
}

/** queueList 走 QueueList；服务对象在但没有该方法则返回空数组，没有服务才走 /api/sessions/{id}/queue。sessionID 拼进字符串，传给 QueueList，写进 /api/sessions/{sessionID}/queue。 */
export async function queueList(sessionID: string): Promise<{ id?: string; text?: string; plan?: boolean }[]> {
  const s = await wailsService();
  if (s?.QueueList) return asArray(await s.QueueList(sessionID));
  if (s) return [];
  return asArray(await http(`/api/sessions/${sessionID}/queue`));
}

/**
 * queueCancel 走 QueueCancel；服务对象在但没有该方法则返回假，没有服务才 DELETE /api/sessions/{id}/queue。
 * sessionID 拼进字符串，传给 QueueCancel，写进 /api/sessions/{sessionID}/queue?id={…}。
 * itemID 编码，拼进字符串，传给 QueueCancel，写进 /api/sessions/{…}/queue?id={itemID}。
 */
export async function queueCancel(sessionID: string, itemID: string): Promise<boolean> {
  const s = await wailsService();
  if (s?.QueueCancel) return asBool(await s.QueueCancel(sessionID, itemID));
  if (s) return false;
  const r = await http<any>(`/api/sessions/${sessionID}/queue?id=${encodeURIComponent(itemID)}`, { method: "DELETE" });
  return asBool(pick(r, "ok", "Ok") ?? r);
}

/**
 * queueReorder 走 QueueReorder；服务对象在但没有该方法则返回假，没有服务才 POST /api/sessions/{id}/queue。
 * sessionID 拼进字符串，传给 QueueReorder，写进 /api/sessions/{sessionID}/queue。
 * itemID 传给 QueueReorder，放进请求体。
 * delta 传给 QueueReorder，放进请求体。
 */
export async function queueReorder(sessionID: string, itemID: string, delta: number): Promise<boolean> {
  const s = await wailsService();
  if (s?.QueueReorder) return asBool(await s.QueueReorder(sessionID, itemID, delta));
  if (s) return false;
  const r = await http<any>(`/api/sessions/${sessionID}/queue`, {
    method: "POST",
    body: JSON.stringify({ id: itemID, delta }),
  });
  return asBool(pick(r, "ok", "Ok") ?? r);
}

/**
 * applySessionWorktree 走 ApplySessionWorktree 或 POST /api/sessions/{id}/worktree。
 * sessionID 拼进字符串，传给 ApplySessionWorktree，写进 /api/sessions/{sessionID}/worktree。
 */
export async function applySessionWorktree(sessionID: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ApplySessionWorktree
    ? await s.ApplySessionWorktree(sessionID)
    : await http(`/api/sessions/${sessionID}/worktree`, { method: "POST" });
  return threadOf(raw);
}

/** running 走 Running 或 /api/sessions/{id}/running。sessionID 拼进字符串，传给 Running，写进 /api/sessions/{sessionID}/running。 */
export async function running(sessionID: string): Promise<boolean> {
  const s = await wailsService();
  if (s?.Running) {
    const v = await s.Running(sessionID);
    return asBool(v);
  }
  const r = await http<any>(`/api/sessions/${sessionID}/running`);
  return asBool(r);
}

/** runningIDs 走 RunningIDs；服务对象在但没有该方法则返回空数组，没有服务才走 /api/running。 */
export async function runningIDs(): Promise<string[]> {
  const s = await wailsService();
  if (s?.RunningIDs) {
    const v = await s.RunningIDs();
    return asArray(v).map(String).filter(Boolean);
  }
  if (s) return [];
  const r = await http<any>("/api/running");
  return asArray(pick(r, "ids", "IDs") ?? r).map(String).filter(Boolean);
}

/** runningStatus 走 RunningStatus；服务对象在但没有该方法则返回空列表，没有服务才读 /api/running 的 runs。 */
export async function runningStatus(): Promise<RunStatus[]> {
  const s = await wailsService();
  const raw = s?.RunningStatus ? await s.RunningStatus() : s ? [] : pick(await http<any>("/api/running"), "runs", "Runs") ?? [];
  return asArray(raw).map((v) => ({
    id: str(pick(v, "id", "ID")),
    startedAt: str(pick(v, "started_at", "StartedAt")),
    lastTool: str(pick(v, "last_tool", "LastTool")),
  })).filter((x) => x.id);
}

/** contextUsage 走 ContextUsage 或 /api/sessions/{id}/context。sessionID 拼进字符串，传给 ContextUsage，写进 /api/sessions/{sessionID}/context。 */
export async function contextUsage(sessionID: string): Promise<ContextUsage> {
  const s = await wailsService();
  const raw = s?.ContextUsage ? await s.ContextUsage(sessionID) : await http(`/api/sessions/${sessionID}/context`);
  return contextOf(raw);
}

/** interrupt 走 Interrupt，并最多等 1500 毫秒；没有该方法则 POST /api/sessions/{id}/interrupt。sessionID 拼进字符串，传给 Interrupt，写进 /api/sessions/{sessionID}/interrupt。 */
export async function interrupt(sessionID: string): Promise<void> {
  const s = await wailsService();
  if (s?.Interrupt) {
    await Promise.race([
      s.Interrupt(sessionID),
      new Promise<void>((resolve) => window.setTimeout(resolve, 1500)),
    ]);
    return;
  }
  await http(`/api/sessions/${sessionID}/interrupt`, { method: "POST" });
}

/** approvals 走 PendingApprovals 或 /api/approvals。 */
export async function approvals(): Promise<Approval[]> {
  const s = await wailsService();
  const raw = s?.PendingApprovals ? await s.PendingApprovals() : await http("/api/approvals");
  return asArray(raw).map(approvalOf).filter((a) => a.id);
}

/**
 * resolveApproval 优先走 ResolveApprovalAnswer；否则走 ResolveApproval，有 answer 且 length 不少于 3 才把 answer 传入；再否则 POST /api/approvals。
 * id 传给 ResolveApprovalAnswer，传给 ResolveApproval，放进请求体。
 * decision 传给 ResolveApprovalAnswer，传给 ResolveApproval，放进请求体。
 */
export async function resolveApproval(id: string, decision: string, answer = ""): Promise<void> {
  const s = await wailsService();
  if (s?.ResolveApprovalAnswer) return s.ResolveApprovalAnswer(id, decision, answer);
  if (s?.ResolveApproval) {
    if (answer && s.ResolveApproval.length >= 3) return s.ResolveApproval(id, decision, answer);
    return s.ResolveApproval(id, decision);
  }
  await http("/api/approvals", { method: "POST", body: JSON.stringify({ id, decision, answer }) });
}

/** harness 走 Harness 或 /api/harness。 */
export async function harness(): Promise<any> {
  const s = await wailsService();
  if (s?.Harness) return s.Harness();
  return http("/api/harness");
}

/** checkout l3 时走 CheckoutL3，否则走 Checkout；都没有则 POST /api/harness/checkout。hash 传给 CheckoutL3，传给 Checkout，放进请求体。 */
export async function checkout(hash: string, l3 = false): Promise<void> {
  const s = await wailsService();
  if (l3 && s?.CheckoutL3) return s.CheckoutL3(hash);
  if (!l3 && s?.Checkout) return s.Checkout(hash);
  await http("/api/harness/checkout", { method: "POST", body: JSON.stringify({ hash, l3 }) });
}

/** rollback 走 Rollback 或 POST /api/harness/rollback。 */
export async function rollback(): Promise<void> {
  const s = await wailsService();
  if (s?.Rollback) return s.Rollback();
  await http("/api/harness/rollback", { method: "POST" });
}

/** revealHarness 走 RevealHarness 或 POST /api/harness/reveal。hash 传给 RevealHarness，放进请求体。 */
export async function revealHarness(hash = ""): Promise<void> {
  const s = await wailsService();
  if (s?.RevealHarness) return s.RevealHarness(hash);
  await http("/api/harness/reveal", { method: "POST", body: JSON.stringify({ hash }) });
}

/**
 * diff 走 DiffDetail 或 /api/harness/diff。
 * a 编码，拼进字符串，传给 DiffDetail，写进 /api/harness/diff?a={a}&b={…}。
 * b 编码，拼进字符串，传给 DiffDetail，写进 /api/harness/diff?a={…}&b={b}。
 */
export async function diff(a: string, b: string): Promise<any> {
  const s = await wailsService();
  if (s?.DiffDetail) return s.DiffDetail(a, b);
  return http(`/api/harness/diff?a=${encodeURIComponent(a)}&b=${encodeURIComponent(b)}`);
}

/** runEval 有 snapshot 时走 RunEvalAt，没有该方法则 rpcJSON 调 eval.run；没有 snapshot 时走 RunEval 或 POST /api/eval。opts 读取 snapshot，为空时不读。 */
export async function runEval(opts?: { snapshot?: string }): Promise<any> {
  const snapshot = String(opts?.snapshot || "").trim();
  const s = await wailsService();
  if (snapshot) {
    const fn = svcMethod(s, "RunEvalAt", "runEvalAt");
    if (fn) return fn(snapshot);
    return rpcJSON("eval.run", { snapshot }, 10 * 60 * 1000);
  }
  if (s?.RunEval) return s.RunEval();
  return http("/api/eval", { method: "POST" });
}

/** bestOfN 走 BestOfN 或 POST /api/eval/best。n 传给 BestOfN，放进请求体。 */
export async function bestOfN(n: number): Promise<any> {
  const s = await wailsService();
  if (s?.BestOfN) return s.BestOfN(n);
  return http("/api/eval/best", { method: "POST", body: JSON.stringify({ n }) });
}

/**
 * evolve 优先走 EvolveLab，其次 EvolveRun；没有 rounds、sealed、promote 时才走 EvolveK，并且 k 为 3 才走 Evolve；否则 POST /api/evolve。
 * opts 读取 rounds、sealed、promote、behavior、index、baselines 等，为空时不读。
 */
export async function evolve(k = 3, opts?: {
  rounds?: number;
  sealed?: boolean;
  promote?: boolean;
  behavior?: boolean;
  index?: boolean;
  baselines?: number;
  maxUsd?: number;
}): Promise<any> {
  const body = {
    k,
    rounds: opts?.rounds ?? 0,
    sealed: !!opts?.sealed,
    promote: !!opts?.promote,
    behavior: !!opts?.behavior,
    index: !!opts?.index,
    baselines: opts?.baselines ?? 0,
    max_usd: opts?.maxUsd ?? 0,
  };
  const s = await wailsService();
  if (s?.EvolveLab) {
    return s.EvolveLab(body.k, body.rounds, body.sealed, body.promote, body.behavior, body.index, body.baselines, body.max_usd);
  }
  if (s?.EvolveRun) return s.EvolveRun(body.k, body.rounds, body.sealed, body.promote);
  if (s?.EvolveK && !body.rounds && !body.sealed && !body.promote) return s.EvolveK(k);
  if (s?.Evolve && k === 3 && !body.rounds && !body.sealed && !body.promote) return s.Evolve();
  return http("/api/evolve", { method: "POST", body: JSON.stringify(body) });
}

/** evolveOverlay 走 OverlaySession；没有该方法则 rpcJSON 调 evolve.overlay。opts 读取 session、task、reuseTurn、maxUsd、overlayK、overlayMode。 */
export async function evolveOverlay(opts: {
  session: string;
  task?: string;
  reuseTurn?: boolean;
  maxUsd?: number;
  overlayK?: number;
  overlayMode?: string;
}): Promise<any> {
  const session = String(opts.session || "").trim();
  const task = opts.task || "";
  const reuseTurn = opts.reuseTurn !== false;
  const maxUsd = opts.maxUsd ?? 0;
  const overlayK = Math.max(1, Math.min(4, opts.overlayK || 1));
  const overlayMode = opts.overlayMode === "seq" ? "seq" : "agg";
  const s = await wailsService();
  const fn = svcMethod(s, "OverlaySession", "overlaySession");
  if (fn) return fn(session, task, reuseTurn, maxUsd, overlayK, overlayMode);
  return rpcJSON("evolve.overlay", {
    session,
    task,
    reuse_turn: reuseTurn,
    max_usd: maxUsd,
    overlay_k: overlayK,
    overlay_mode: overlayMode,
  }, Math.min(60, 15 * overlayK) * 60 * 1000);
}

/** overlayLibrary 走 OverlayLibrary；没有该方法则 rpcJSON 调 evolve.library。session 为假时用空字符串，转成字符串，去掉首尾空白后，传给 OverlayLibrary，写入字段 session。 */
export async function overlayLibrary(session: string): Promise<any> {
  const id = String(session || "").trim();
  const s = await wailsService();
  const fn = svcMethod(s, "OverlayLibrary", "overlayLibrary");
  if (fn) return fn(id);
  return rpcJSON("evolve.library", { session: id });
}

/** archive 走 Archive 或 /api/archive。 */
export async function archive(): Promise<any[]> {
  const s = await wailsService();
  if (s?.Archive) return asArray(await s.Archive());
  return asArray(await http("/api/archive"));
}

/** plugins 走 Plugins 或 /api/plugins。 */
export async function plugins(): Promise<any> {
  const s = await wailsService();
  if (s?.Plugins) return s.Plugins();
  return http("/api/plugins");
}

/** unloadFiber 走 UnloadFiber。name 传给 UnloadFiber。 */
export async function unloadFiber(name: string): Promise<void> {
  const s = await wailsService();
  if (s?.UnloadFiber) return s.UnloadFiber(name);
}

/** forkSession 有 from 且有 ForkSessionFrom 时走它，没有 from 且有 ForkSession 时走它，否则 POST /api/sessions/{id}/fork。 */
export async function forkSession(id: string, from = ""): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ForkSessionFrom && from
    ? await s.ForkSessionFrom(id, from)
    : s?.ForkSession && !from
      ? await s.ForkSession(id)
      : await http(`/api/sessions/${id}/fork`, { method: "POST", body: from ? JSON.stringify({ from }) : undefined });
  return threadOf(raw);
}

/** dumpSession 走 DumpSession 或 /api/sessions/{id}/dump。 */
export async function dumpSession(id: string): Promise<{ id: string; path: string }> {
  const s = await wailsService();
  const raw = s?.DumpSession
    ? await s.DumpSession(id)
    : await http(`/api/sessions/${id}/dump`);
  const paths = pick(raw, "paths", "Paths") || {};
  return {
    id: str(pick(raw, "id", "ID"), id),
    path: str(pick(paths, "dump", "Dump"), str(pick(raw, "path", "Path"))),
  };
}

/** renameSession 走 RenameSession 或 POST /api/sessions/{id}/title。 */
export async function renameSession(id: string, title: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "RenameSession", "renameSession");
  if (fn) {
    await fn(id, title);
    return;
  }
  await http(`/api/sessions/${id}/title`, { method: "POST", body: JSON.stringify({ title }) });
}

/** playbook 走 Playbook 或 /api/playbook。 */
export async function playbook(): Promise<any> {
  const s = await wailsService();
  if (s?.Playbook) return s.Playbook();
  return http("/api/playbook");
}

/** ratePlaybook 走 RatePlaybook 或 POST /api/playbook。id 传给 RatePlaybook，放进请求体。helpful 传给 RatePlaybook，放进请求体。 */
export async function ratePlaybook(id: string, helpful: boolean): Promise<any> {
  const s = await wailsService();
  if (s?.RatePlaybook) return s.RatePlaybook(id, helpful);
  return http("/api/playbook", { method: "POST", body: JSON.stringify({ id, helpful }) });
}

/** workspaceHunks 走 WorkspaceHunks 或 /api/workspace/hunks。 */
export async function workspaceHunks(workspace: string): Promise<{ diff: string; hunks: Hunk[] }> {
  const s = await wailsService();
  const raw = s?.WorkspaceHunks
    ? await s.WorkspaceHunks(workspace)
    : await http(`/api/workspace/hunks?workspace=${encodeURIComponent(workspace || "")}`);
  return {
    diff: str(pick(raw, "diff", "Diff")),
    hunks: asArray(pick(raw, "hunks", "Hunks")).map(hunkOf).filter((h) => h.id),
  };
}

/** FilePreview 是工作区文件预览的正文、语言和种类。 */
export type FilePreview = {
  path: string;
  text: string;
  lang: string;
  html: boolean;
  binary: boolean;
  truncated: boolean;
  bytes: number;
  kind: string;
  mime: string;
  office: boolean;
};

/**
 * previewWorkspaceFile 走 PreviewWorkspaceFile 或 /api/workspace/file。
 * path 为假时用空字符串，编码，拼进字符串，传给 PreviewWorkspaceFile，写进 /api/workspace/file?workspace={…}&path={path}。
 */
export async function previewWorkspaceFile(workspace: string, path: string): Promise<FilePreview> {
  const s = await wailsService();
  const raw = s?.PreviewWorkspaceFile
    ? await s.PreviewWorkspaceFile(workspace, path)
    : await http(`/api/workspace/file?workspace=${encodeURIComponent(workspace || "")}&path=${encodeURIComponent(path || "")}`);
  return {
    path: str(pick(raw, "path", "Path")),
    text: str(pick(raw, "text", "Text")),
    lang: str(pick(raw, "lang", "Lang")),
    html: bool(pick(raw, "html", "HTML")),
    binary: bool(pick(raw, "binary", "Binary")),
    truncated: bool(pick(raw, "truncated", "Truncated")),
    bytes: num(pick(raw, "bytes", "Bytes")),
    kind: str(pick(raw, "kind", "Kind")),
    mime: str(pick(raw, "mime", "MIME", "Mime")),
    office: bool(pick(raw, "office", "Office")),
  };
}

/** FileBlob 是工作区二进制文件的种类和 base64。 */
export type FileBlob = {
  path: string;
  kind: string;
  mime: string;
  bytes: number;
  truncated: boolean;
  base64: string;
};

/**
 * readWorkspaceBlob 走 ReadWorkspaceBlob 或 /api/workspace/blob。
 * path 为假时用空字符串，编码，拼进字符串，传给 ReadWorkspaceBlob，写进 /api/workspace/blob?workspace={…}&path={path}。
 */
export async function readWorkspaceBlob(workspace: string, path: string): Promise<FileBlob> {
  const s = await wailsService();
  const raw = s?.ReadWorkspaceBlob
    ? await s.ReadWorkspaceBlob(workspace, path)
    : await http(`/api/workspace/blob?workspace=${encodeURIComponent(workspace || "")}&path=${encodeURIComponent(path || "")}`, {
        signal: AbortSignal.timeout(30_000),
      });
  return {
    path: str(pick(raw, "path", "Path")),
    kind: str(pick(raw, "kind", "Kind")),
    mime: str(pick(raw, "mime", "MIME", "Mime")),
    bytes: num(pick(raw, "bytes", "Bytes")),
    truncated: bool(pick(raw, "truncated", "Truncated")),
    base64: str(pick(raw, "base64", "Base64")),
  };
}

/** applyHunks 走 ApplyHunks 或 POST /api/workspace/hunks。ids 传给 ApplyHunks，放进请求体。 */
export async function applyHunks(workspace: string, ids: string[]): Promise<void> {
  const s = await wailsService();
  if (s?.ApplyHunks) return s.ApplyHunks(workspace, ids);
  await http("/api/workspace/hunks", { method: "POST", body: JSON.stringify({ workspace, ids }) });
}

/** reverseHunks 走 ReverseHunks，或 POST /api/workspace/hunks 并带上 reverse。ids 传给 ReverseHunks，放进请求体。snapshot 传给 ReverseHunks，放进请求体。 */
export async function reverseHunks(workspace: string, ids: string[], snapshot: string): Promise<void> {
  const s = await wailsService();
  if (s?.ReverseHunks) return s.ReverseHunks(workspace, ids, snapshot);
  await http("/api/workspace/hunks", { method: "POST", body: JSON.stringify({ workspace, ids, snapshot, reverse: true }) });
}

/** runEvalSafety 走 RunEvalSafety 或 POST /api/eval/safety。 */
export async function runEvalSafety(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalSafety) return s.RunEvalSafety();
  return http("/api/eval/safety", { method: "POST" });
}

/** runEvalTB 走 RunEvalTB 或 POST /api/eval/tb。 */
export async function runEvalTB(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalTB) return s.RunEvalTB();
  return http("/api/eval/tb", { method: "POST" });
}

/** runEvalSealed 走 RunEvalSealed 或 POST /api/eval/sealed。 */
export async function runEvalSealed(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalSealed) return s.RunEvalSealed();
  return http("/api/eval/sealed", { method: "POST" });
}

/** runEvalTransfer 走 RunEvalTransfer 或 POST /api/eval/transfer。 */
export async function runEvalTransfer(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalTransfer) return s.RunEvalTransfer();
  return http("/api/eval/transfer", { method: "POST" });
}

/** runEvalIndex 走 RunEvalIndex 或 POST /api/eval/index。 */
export async function runEvalIndex(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalIndex) return s.RunEvalIndex();
  return http("/api/eval/index", { method: "POST" });
}

/** runEvalBehavior 走 RunEvalBehavior 或 POST /api/eval/behavior。 */
export async function runEvalBehavior(): Promise<any> {
  const s = await wailsService();
  if (s?.RunEvalBehavior) return s.RunEvalBehavior();
  return http("/api/eval/behavior", { method: "POST" });
}

/** lastEval 走 LastEval 或 /api/eval/last。 */
export async function lastEval(): Promise<any> {
  const s = await wailsService();
  if (s?.LastEval) return s.LastEval();
  return http("/api/eval/last");
}

/** lastEvolve 走 LastEvolve 或 /api/evolve/last。 */
export async function lastEvolve(): Promise<any> {
  const s = await wailsService();
  if (s?.LastEvolve) return s.LastEvolve();
  return http("/api/evolve/last");
}

/** bestOfModels 走 BestOfModels 或 POST /api/eval/best。models 传给 BestOfModels，放进请求体。 */
export async function bestOfModels(models: string[]): Promise<any> {
  const s = await wailsService();
  if (s?.BestOfModels) return s.BestOfModels(models);
  return http("/api/eval/best", { method: "POST", body: JSON.stringify({ models }) });
}

/** applyUpdate 走 ApplyUpdate 或 POST /api/update。 */
export async function applyUpdate(): Promise<void> {
  const s = await wailsService();
  if (s?.ApplyUpdate) return s.ApplyUpdate();
  await http("/api/update", { method: "POST", body: JSON.stringify({}) });
}

/** checkoutSafe 先调用 checkout；确认信息里出现 L3 时，同意后再用 l3 调用 checkout。hash 传给 checkout。 */
export async function checkoutSafe(hash: string): Promise<void> {
  try {
    await checkout(hash, false);
  } catch (e: any) {
    const l3 = e?.body?.l3 || String(e.message || e).includes("L3 confirmation");
    if (!l3) throw e;
    const surfaces = (e?.body?.surfaces || ["loop_preset/policy_pack"]).join(", ");
    const zh = getLocale() === "zh-CN";
    const ok = window.confirm(
      zh ? `L3 门禁：将改动 ${surfaces}。确认 checkout？` : `L3 gate: changing ${surfaces}. Confirm checkout?`,
    );
    if (!ok) throw e;
    await checkout(hash, true);
  }
}

/** deleteSession 走 DeleteSession 或 DELETE /api/sessions/{id}。 */
export async function deleteSession(id: string): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "DeleteSession", "deleteSession");
  if (fn) {
    await fn(id);
    return;
  }
  await http(`/api/sessions/${id}`, { method: "DELETE" });
}

/** archiveSession 走 ArchiveSession 或 POST /api/sessions/{id}/archive。archived 传给 ArchiveSession，放进请求体。 */
export async function archiveSession(id: string, archived: boolean): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ArchiveSession
    ? await s.ArchiveSession(id, archived)
    : await http(`/api/sessions/${id}/archive`, { method: "POST", body: JSON.stringify({ archived }) });
  return threadOf(raw);
}

/** pinSession 走 PinSession 或 POST /api/sessions/{id}/pin。pinned 传给 PinSession，放进请求体。 */
export async function pinSession(id: string, pinned: boolean): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.PinSession
    ? await s.PinSession(id, pinned)
    : await http(`/api/sessions/${id}/pin`, { method: "POST", body: JSON.stringify({ pinned }) });
  return threadOf(raw);
}

/** exportSession 走 ExportSession 或 /api/sessions/{id}/export。 */
export async function exportSession(id: string): Promise<string> {
  const s = await wailsService();
  if (s?.ExportSession) return String(await s.ExportSession(id) || "");
  const raw = await http<any>(`/api/sessions/${id}/export`);
  return str(pick(raw, "markdown"));
}

/** setSessionModel 走 SetSessionModel 或 POST /api/sessions/{id}/model。 */
export async function setSessionModel(id: string, model: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionModel
    ? await s.SetSessionModel(id, model)
    : await http(`/api/sessions/${id}/model`, { method: "POST", body: JSON.stringify({ model }) });
  return threadOf(raw);
}

/** setSessionAuthMode 走 SetSessionAuthMode 或 POST /api/sessions/{id}/auth。mode 传给 SetSessionAuthMode，放进请求体。 */
export async function setSessionAuthMode(id: string, mode: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionAuthMode
    ? await s.SetSessionAuthMode(id, mode)
    : await http(`/api/sessions/${id}/auth`, { method: "POST", body: JSON.stringify({ mode }) });
  return threadOf(raw);
}

/** setSessionWorkspace 走 SetSessionWorkspace 或 POST /api/sessions/{id}/workspace。 */
export async function setSessionWorkspace(id: string, workspace: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionWorkspace
    ? await s.SetSessionWorkspace(id, workspace)
    : await http(`/api/sessions/${id}/workspace`, { method: "POST", body: JSON.stringify({ workspace }) });
  return threadOf(raw);
}

/** setSessionIsolate 走 SetSessionIsolate 或 POST /api/sessions/{id}/isolate。 */
export async function setSessionIsolate(id: string, isolate: boolean): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionIsolate
    ? await s.SetSessionIsolate(id, isolate)
    : await http(`/api/sessions/${id}/isolate`, { method: "POST", body: JSON.stringify({ isolate }) });
  return threadOf(raw);
}

/** setSessionPinnedSkills 走 SetSessionPinnedSkills 或 POST /api/sessions/{id}/skills。names 传给 SetSessionPinnedSkills，放进请求体。 */
export async function setSessionPinnedSkills(id: string, names: string[]): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionPinnedSkills
    ? await s.SetSessionPinnedSkills(id, names)
    : await http(`/api/sessions/${id}/skills`, { method: "POST", body: JSON.stringify({ names }) });
  return threadOf(raw);
}

/** contextInventoryOf 把上下文清单收成 ContextInventory。v 用 pick 读取 shape、objects、session、plan、plan_path字段，作为后备值。 */
export function contextInventoryOf(v: any): ContextInventory {
  const shape = contextOf(pick(v, "shape", "Shape") || v);
  const objects: ContextObject[] = asArray(pick(v, "objects", "Objects")).map((o) => ({
    id: str(pick(o, "id", "ID")),
    kind: str(pick(o, "kind", "Kind")),
    title: str(pick(o, "title", "Title")),
    detail: str(pick(o, "detail", "Detail")),
    tokens: num(pick(o, "tokens", "Tokens")),
    path: str(pick(o, "path", "Path")),
    stale: bool(pick(o, "stale", "Stale")),
    pinned: bool(pick(o, "pinned", "Pinned")),
  }));
  return {
    session: str(pick(v, "session", "Session")),
    shape,
    objects,
    plan: str(pick(v, "plan", "Plan")),
    planPath: str(pick(v, "plan_path", "PlanPath", "planPath")),
  };
}

/** contextInventory 走 ContextInventory 或 /api/sessions/{id}/inventory。sessionID 拼进字符串，传给 ContextInventory，写进 /api/sessions/{sessionID}/inventory。 */
export async function contextInventory(sessionID: string): Promise<ContextInventory> {
  const s = await wailsService();
  const raw = s?.ContextInventory ? await s.ContextInventory(sessionID) : await http(`/api/sessions/${sessionID}/inventory`);
  return contextInventoryOf(raw);
}

/** setSessionPlan 走 SetSessionPlan 或 POST /api/sessions/{id}/plan。text 传给 SetSessionPlan，放进请求体。 */
export async function setSessionPlan(id: string, text: string): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.SetSessionPlan
    ? await s.SetSessionPlan(id, text)
    : await http(`/api/sessions/${id}/plan`, { method: "POST", body: JSON.stringify({ text }) });
  return threadOf(raw);
}

/** restoreSessionFiles 走 RestoreSessionFiles 或 POST /api/sessions/{id}/restore。from 传给 RestoreSessionFiles，放进请求体。 */
export async function restoreSessionFiles(id: string, from = ""): Promise<string> {
  const s = await wailsService();
  if (s?.RestoreSessionFiles) return String(await s.RestoreSessionFiles(id, from) || "");
  const raw = await http<any>(`/api/sessions/${id}/restore`, { method: "POST", body: JSON.stringify({ from }) });
  return str(pick(raw, "note"));
}

/** contextPin 走 ContextPin 或 POST /api/sessions/{id}/pin。kind 传给 ContextPin，放进请求体。key 传给 ContextPin，放进请求体。pinned 传给 ContextPin，放进请求体。 */
export async function contextPin(id: string, kind: string, key: string, pinned: boolean): Promise<Thread> {
  const s = await wailsService();
  const raw = s?.ContextPin
    ? await s.ContextPin(id, kind, key, pinned)
    : await http(`/api/sessions/${id}/pin`, { method: "POST", body: JSON.stringify({ kind, key, pinned }) });
  return threadOf(raw);
}

/** compactSession 有 CompactSessionFocus 就走它；否则仅在没有 focus 时走 CompactSession；再否则 POST /api/sessions/{id}/compact。 */
export async function compactSession(id: string, focus?: string): Promise<string> {
  const s = await wailsService();
  if (s?.CompactSessionFocus) return String(await s.CompactSessionFocus(id, focus || "") || "");
  if (s?.CompactSession && !focus) return String(await s.CompactSession(id) || "");
  const raw = await http<any>(`/api/sessions/${id}/compact`, { method: "POST", body: JSON.stringify({ focus: focus || "" }) });
  return str(pick(raw, "note"));
}

/** steer 走 Steer 或 POST /api/sessions/{id}/steer。text 传给 Steer，放进请求体。 */
export async function steer(id: string, text: string): Promise<void> {
  const s = await wailsService();
  if (s?.Steer) return s.Steer(id, text);
  await http(`/api/sessions/${id}/steer`, { method: "POST", body: JSON.stringify({ text }) });
}

/** workspaceTree 走 WorkspaceTree 或 /api/workspace/tree。 */
export async function workspaceTree(workspace: string): Promise<FileHit[]> {
  const s = await wailsService();
  const raw = s?.WorkspaceTree
    ? await s.WorkspaceTree(workspace)
    : await http(`/api/workspace/tree?workspace=${encodeURIComponent(workspace || "")}`);
  return asArray(raw).map((v) => ({ path: str(pick(v, "path", "Path")), kind: str(pick(v, "kind", "Kind"), "file") })).filter((h) => h.path);
}

/**
 * searchFiles 走 SearchFiles 或 /api/fs/search。
 * workspace 为假时用空字符串，编码，拼进字符串，传给 SearchFiles，写进 /api/fs/search?workspace={workspace}&q={…}。
 * query 编码，拼进字符串，传给 SearchFiles，写进 /api/fs/search?workspace={…}&q={query}。
 */
export async function searchFiles(workspace: string, query: string): Promise<FileHit[]> {
  const s = await wailsService();
  const raw = s?.SearchFiles
    ? await s.SearchFiles(workspace, query)
    : await http(`/api/fs/search?workspace=${encodeURIComponent(workspace || "")}&q=${encodeURIComponent(query)}`);
  return asArray(raw).map((v) => ({ path: str(pick(v, "path", "Path")), kind: str(pick(v, "kind", "Kind"), "file") })).filter((h) => h.path);
}

/** listSkills 优先走 ListSkillsFor，否则 ListSkills，再否则 /api/skills。workspace 为假时用空字符串，编码，拼进字符串，传给 ListSkillsFor，写进 /api/skills{workspace}。 */
export async function listSkills(workspace?: string): Promise<SkillInfo[]> {
  const s = await wailsService();
  const raw = s?.ListSkillsFor
    ? await s.ListSkillsFor(workspace || "")
    : s?.ListSkills
      ? await s.ListSkills()
      : await http(`/api/skills${workspace ? `?workspace=${encodeURIComponent(workspace)}` : ""}`);
  return asArray(raw).map((v) => ({
    name: str(pick(v, "name", "Name")),
    description: str(pick(v, "description", "Description")),
    body: str(pick(v, "body", "Body")),
    dir: str(pick(v, "dir", "Dir")),
    source: str(pick(v, "source", "Source")),
    icon: str(pick(v, "icon", "Icon")),
    displayName: str(pick(v, "display_name", "DisplayName", "displayName")),
    files: str(pick(v, "files", "Files")),
    slug: str(pick(v, "slug", "Slug")),
    incomplete: str(pick(v, "incomplete", "Incomplete")) === "1" || bool(pick(v, "incomplete", "Incomplete")),
    pack: str(pick(v, "pack", "Pack")),
    installId: str(pick(v, "install_id", "InstallID", "installId")),
  })).filter((x) => x.name);
}

/**
 * getSkill 走 GetSkill 或 /api/skills。
 * workspace 为假时用空字符串，编码，拼进字符串，传给 GetSkill，写进 /api/skills?workspace={workspace}&name={…}。
 * name 编码，拼进字符串，传给 GetSkill，写进 /api/skills?workspace={…}&name={name}。
 */
export async function getSkill(workspace: string, name: string): Promise<SkillInfo | null> {
  const s = await wailsService();
  const raw = s?.GetSkill
    ? await s.GetSkill(workspace || "", name)
    : await http(`/api/skills?workspace=${encodeURIComponent(workspace || "")}&name=${encodeURIComponent(name)}`);
  const info = {
    name: str(pick(raw, "name", "Name")),
    description: str(pick(raw, "description", "Description")),
    body: str(pick(raw, "body", "Body")),
    dir: str(pick(raw, "dir", "Dir")),
    source: str(pick(raw, "source", "Source")),
    icon: str(pick(raw, "icon", "Icon")),
    displayName: str(pick(raw, "display_name", "DisplayName", "displayName")),
    files: str(pick(raw, "files", "Files")),
    slug: str(pick(raw, "slug", "Slug")),
    incomplete: str(pick(raw, "incomplete", "Incomplete")) === "1" || bool(pick(raw, "incomplete", "Incomplete")),
    pack: str(pick(raw, "pack", "Pack")),
    installId: str(pick(raw, "install_id", "InstallID", "installId")),
  };
  return info.name ? info : null;
}

/** SkillMarketHub 是技能市场的一个来源。 */
export type SkillMarketHub = {
  id: string;
  name: string;
  homepage: string;
  attribution: string;
  browse: boolean;
  search: boolean;
};

/** SkillMarketItem 是市场上的一个技能条目。 */
export type SkillMarketItem = {
  slug: string;
  name: string;
  purpose: string;
  prerequisites: string;
  category: string;
  featured: boolean;
  icon?: string;
  displayName?: string;
  version?: string;
  hasScripts?: boolean;
  hub?: string;
  id?: string;
  owner?: string;
  pageUrl?: string;
  repo?: string;
  downloads?: number;
  stars?: number;
};

/** SkillMarketCatalog 是一次技能市场查询的结果。 */
export type SkillMarketCatalog = {
  source: string;
  fetchedAt: string;
  items: SkillMarketItem[];
  hub?: string;
  hubs?: SkillMarketHub[];
  query?: string;
  searchOnly?: boolean;
  warning?: string;
  attribution?: string;
  total?: number;
  truncated?: boolean;
};

/**
 * skillMarket 走 SkillMarket 或 /api/skills/market。
 * refresh 为真时才继续，传给 SkillMarket。
 * hub 为假时用空字符串，交给 qs.set，传给 SkillMarket。
 * query 为假时用空字符串，交给 qs.set，传给 SkillMarket。
 */
export async function skillMarket(refresh = false, hub = "", query = ""): Promise<SkillMarketCatalog> {
  const s = await wailsService();
  const qs = new URLSearchParams();
  if (refresh) qs.set("refresh", "1");
  if (hub) qs.set("hub", hub);
  if (query) qs.set("q", query);
  const raw = s?.SkillMarket
    ? await s.SkillMarket(refresh, hub || "", query || "")
    : await http(`/api/skills/market${qs.toString() ? `?${qs}` : ""}`);
  return {
    source: str(pick(raw, "source", "Source"), "infometa/workbuddyskills"),
    fetchedAt: str(pick(raw, "fetched_at", "FetchedAt")),
    hub: str(pick(raw, "hub", "Hub")),
    query: str(pick(raw, "query", "Query")),
    searchOnly: bool(pick(raw, "search_only", "SearchOnly", "searchOnly")),
    warning: str(pick(raw, "warning", "Warning")),
    attribution: str(pick(raw, "attribution", "Attribution")),
    total: num(pick(raw, "total", "Total")),
    truncated: bool(pick(raw, "truncated", "Truncated")),
    hubs: asArray(pick(raw, "hubs", "Hubs")).map((v) => ({
      id: str(pick(v, "id", "ID")),
      name: str(pick(v, "name", "Name")),
      homepage: str(pick(v, "homepage", "Homepage")),
      attribution: str(pick(v, "attribution", "Attribution")),
      browse: bool(pick(v, "browse", "Browse")),
      search: bool(pick(v, "search", "Search")),
    })).filter((h) => h.id),
    items: asArray(pick(raw, "items", "Items")).map((v) => ({
      slug: str(pick(v, "slug", "Slug")),
      name: str(pick(v, "name", "Name")),
      purpose: str(pick(v, "purpose", "Purpose")),
      prerequisites: str(pick(v, "prerequisites", "Prerequisites")),
      category: str(pick(v, "category", "Category")),
      featured: bool(pick(v, "featured", "Featured")),
      icon: str(pick(v, "icon", "Icon")),
      displayName: str(pick(v, "display_name", "DisplayName", "displayName")),
      version: str(pick(v, "version", "Version")),
      hasScripts: bool(pick(v, "has_scripts", "HasScripts", "hasScripts")),
      hub: str(pick(v, "hub", "Hub")),
      id: str(pick(v, "id", "ID")),
      owner: str(pick(v, "owner", "Owner")),
      pageUrl: str(pick(v, "page_url", "PageURL", "pageUrl")),
      repo: str(pick(v, "repo", "Repo")),
      downloads: num(pick(v, "downloads", "Downloads")),
      stars: num(pick(v, "stars", "Stars")),
    })).filter((x) => x.slug),
  };
}

/** modelsCatalog 走 ModelCatalog 或 /api/models/catalog。refresh 为真时才继续，传给 ModelCatalog，写进 /api/models/catalog{refresh}。 */
export async function modelsCatalog(refresh = false): Promise<unknown> {
  const s = await wailsService();
  const fn = svcMethod(s, "ModelCatalog");
  if (fn) return fn(refresh);
  return http(`/api/models/catalog${refresh ? "?refresh=1" : ""}`);
}

/** SkillInstallResult 是安装市场技能的结果。 */
export type SkillInstallResult = {
  ok: boolean;
  warning?: string;
  files?: string[];
};

/** installMarketSkill 走 InstallMarketSkill 或 POST /api/skills/market。slug 传给 InstallMarketSkill，放进请求体。 */
export async function installMarketSkill(slug: string): Promise<SkillInstallResult> {
  const s = await wailsService();
  const raw = s?.InstallMarketSkill
    ? await s.InstallMarketSkill(slug)
    : await http("/api/skills/market", { method: "POST", body: JSON.stringify({ slug }) });
  return {
    ok: bool(pick(raw, "ok", "OK")) || raw == null,
    warning: str(pick(raw, "warning", "Warning")),
    files: asArray(pick(raw, "files", "Files")).map((v) => str(v)).filter(Boolean),
  };
}

/** uninstallMarketSkill 走 UninstallMarketSkill，或 POST /api/skills/market 且 op 为 uninstall。slug 传给 UninstallMarketSkill，放进 op 为 uninstall 的请求体。 */
export async function uninstallMarketSkill(slug: string): Promise<void> {
  const s = await wailsService();
  if (s?.UninstallMarketSkill) {
    await s.UninstallMarketSkill(slug);
    return;
  }
  await http("/api/skills/market", { method: "POST", body: JSON.stringify({ slug, op: "uninstall" }) });
}

/** asPackStatus 把一条载荷收成包状态，对不上就空。v 用 pick 读取 id、origin、enable_workspace、name、description、license 等字段。 */
function asPackStatus(v: any): PackStatus | null {
  const id = str(pick(v, "id", "ID"));
  if (!id) return null;
  const originRaw = pick(v, "origin", "Origin") || {};
  const wsRaw = pick(v, "enable_workspace", "EnableWorkspace", "enableWorkspace");
  let enableWorkspace: boolean | null | undefined;
  if (wsRaw === undefined || wsRaw === null) enableWorkspace = null;
  else enableWorkspace = bool(wsRaw);
  return {
    id,
    name: str(pick(v, "name", "Name"), id),
    description: str(pick(v, "description", "Description")),
    license: str(pick(v, "license", "License")),
    version: str(pick(v, "version", "Version")),
    commit: str(pick(v, "commit", "Commit")),
    bootstrapSkill: str(pick(v, "bootstrap_skill", "BootstrapSkill", "bootstrapSkill")),
    methodology: bool(pick(v, "methodology", "Methodology")),
    installed: bool(pick(v, "installed", "Installed")),
    enabled: bool(pick(v, "enabled", "Enabled")),
    enableGlobal: bool(pick(v, "enable_global", "EnableGlobal", "enableGlobal")),
    enableWorkspace,
    root: str(pick(v, "root", "Root")),
    skillsDir: str(pick(v, "skills_dir", "SkillsDir", "skillsDir")),
    skillCount: num(pick(v, "skill_count", "SkillCount", "skillCount")),
    skills: asArray(pick(v, "skills", "Skills")).map((x) => str(x)).filter(Boolean),
    origin: {
      kind: str(pick(originRaw, "kind", "Kind")),
      repo: str(pick(originRaw, "repo", "Repo")),
      ref: str(pick(originRaw, "ref", "Ref")),
      skillsRel: str(pick(originRaw, "skills_rel", "SkillsRel", "skillsRel")),
      path: str(pick(originRaw, "path", "Path")),
    },
    installedAt: str(pick(v, "installed_at", "InstalledAt", "installedAt")),
    known: bool(pick(v, "known", "Known")),
  };
}

/** listPacks 走 ListPacks 或 /api/packs。workspace 为假时用空字符串，编码，拼进字符串，传给 ListPacks，写进 /api/packs{workspace}。 */
export async function listPacks(workspace?: string): Promise<PackStatus[]> {
  const s = await wailsService();
  const raw = s?.ListPacks
    ? await s.ListPacks(workspace || "")
    : await http(`/api/packs${workspace ? `?workspace=${encodeURIComponent(workspace)}` : ""}`);
  return asArray(raw).map(asPackStatus).filter((x): x is PackStatus => !!x);
}

/** installPack 走 InstallPack，或 POST /api/packs 且 op 为 install。id 传给 InstallPack，放进 op 为 install 的请求体。path 传给 InstallPack，放进 op 为 install 的请求体。 */
export async function installPack(id: string, path = ""): Promise<PackStatus | Record<string, unknown>> {
  const s = await wailsService();
  const raw = s?.InstallPack
    ? await s.InstallPack(id, path)
    : await http("/api/packs", { method: "POST", body: JSON.stringify({ id, path, op: "install" }) });
  return asPackStatus(raw) || (raw as Record<string, unknown>);
}

/** uninstallPack 走 UninstallPack，或 POST /api/packs 且 op 为 uninstall。id 传给 UninstallPack，放进 op 为 uninstall 的请求体。 */
export async function uninstallPack(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.UninstallPack) {
    await s.UninstallPack(id);
    return;
  }
  await http("/api/packs", { method: "POST", body: JSON.stringify({ id, op: "uninstall" }) });
}

/**
 * enablePack 走 EnablePack，或 POST /api/packs 且 op 为 enable。
 * id 传给 EnablePack，放进 op 为 enable 的请求体。
 * scope 传给 EnablePack，放进 op 为 enable 的请求体。
 * enabled 传给 EnablePack，放进 op 为 enable 的请求体。
 */
export async function enablePack(id: string, scope: "workspace" | "global" | "inherit", enabled = true): Promise<void> {
  const s = await wailsService();
  if (s?.EnablePack) {
    await s.EnablePack(id, scope, enabled);
    return;
  }
  await http("/api/packs", { method: "POST", body: JSON.stringify({ id, op: "enable", scope, enabled }) });
}

/** updatePack 走 UpdatePack，或 POST /api/packs 且 op 为 update。id 传给 UpdatePack，放进 op 为 update 的请求体。 */
export async function updatePack(id: string): Promise<PackStatus | Record<string, unknown>> {
  const s = await wailsService();
  const raw = s?.UpdatePack
    ? await s.UpdatePack(id)
    : await http("/api/packs", { method: "POST", body: JSON.stringify({ id, op: "update" }) });
  return asPackStatus(raw) || (raw as Record<string, unknown>);
}

/** openPath 走 OpenPath，或 POST /api/open 且 kind 为 path。 */
export async function openPath(path: string): Promise<void> {
  const s = await wailsService();
  if (s?.OpenPath) return s.OpenPath(path);
  await http("/api/open", { method: "POST", body: JSON.stringify({ path, kind: "path" }) });
}

/** openURL 走 OpenURL，或 POST /api/open 且 kind 为 url。raw 传给 OpenURL，放进请求体。 */
export async function openURL(raw: string): Promise<void> {
  const s = await wailsService();
  if (s?.OpenURL) return s.OpenURL(raw);
  await http("/api/open", { method: "POST", body: JSON.stringify({ path: raw, kind: "url" }) });
}

/** openInEditor 走 OpenInEditor，或 POST /api/open 且 kind 为 editor。path 传给 OpenInEditor，放进请求体。 */
export async function openInEditor(path: string): Promise<void> {
  const s = await wailsService();
  if (s?.OpenInEditor) return s.OpenInEditor(path);
  await http("/api/open", { method: "POST", body: JSON.stringify({ path, kind: "editor" }) });
}

/** openWorkspaceTerminal 走 OpenWorkspaceTerminal，或 POST /api/open 且 kind 为 terminal。dir 传给 OpenWorkspaceTerminal，放进请求体。 */
export async function openWorkspaceTerminal(dir: string): Promise<void> {
  const s = await wailsService();
  if (s?.OpenWorkspaceTerminal) return s.OpenWorkspaceTerminal(dir);
  await http("/api/open", { method: "POST", body: JSON.stringify({ path: dir, kind: "terminal" }) });
}

/** popOutThread 走 PopOutThread。id 传给 PopOutThread。 */
export async function popOutThread(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.PopOutThread) return s.PopOutThread(id);
}

/** pickFolder 走 PickFolder；没有该方法时，browserSurface 为真则用 prompt 问路径，否则返回空串。 */
export async function pickFolder(): Promise<string> {
  const s = await wailsService();
  if (s?.PickFolder) return str(await s.PickFolder());
  if (!browserSurface()) return "";
  const typed = window.prompt(getLocale() === "zh-CN" ? "填写这台电脑上的文件夹路径" : "Folder path on this computer");
  return (typed || "").trim();
}

/** pickFiles 走 PickFiles；没有该方法则返回空数组。 */
export async function pickFiles(): Promise<string[]> {
  const s = await wailsService();
  if (s?.PickFiles) return asArray(await s.PickFiles()).map(String).filter(Boolean);
  return [];
}

/** startMCP 走 StartMCP 或 POST /api/mcp/start。name 传给 StartMCP，放进请求体。command 传给 StartMCP，放进请求体。args 传给 StartMCP，放进请求体。 */
export async function startMCP(name: string, command: string, args: string[]): Promise<void> {
  const s = await wailsService();
  if (s?.StartMCP) return s.StartMCP(name, command, args);
  await http("/api/mcp/start", { method: "POST", body: JSON.stringify({ name, command, args }) });
}

/** startMCPHTTP 走 StartMCPHTTP 或 POST /api/mcp/http。name 传给 StartMCPHTTP，放进请求体。endpoint 传给 StartMCPHTTP，放进请求体。 */
export async function startMCPHTTP(name: string, endpoint: string): Promise<void> {
  const s = await wailsService();
  if (s?.StartMCPHTTP) return s.StartMCPHTTP(name, endpoint);
  await http("/api/mcp/http", { method: "POST", body: JSON.stringify({ name, endpoint }) });
}

/** inboxList 走 InboxList 或 /api/inbox。 */
export async function inboxList(): Promise<any[]> {
  const s = await wailsService();
  if (s?.InboxList) return asArray(await s.InboxList());
  return asArray(await http("/api/inbox"));
}

/** inboxRead 走 InboxMarkRead 或 POST /api/inbox。id 传给 InboxMarkRead，放进请求体。 */
export async function inboxRead(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.InboxMarkRead) return s.InboxMarkRead(id);
  await http("/api/inbox", { method: "POST", body: JSON.stringify({ id }) });
}

/** inboxDismiss 走 InboxDismiss，或 POST /api/inbox 且 op 为 dismiss。id 传给 InboxDismiss，放进 op 为 dismiss 的请求体。 */
export async function inboxDismiss(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.InboxDismiss) { await s.InboxDismiss(id); return; }
  await http("/api/inbox", { method: "POST", body: JSON.stringify({ id, op: "dismiss" }) });
}

/** isolationReport 走 IsolationReport 或 /api/isolation。 */
export async function isolationReport(): Promise<any> {
  const s = await wailsService();
  if (s?.IsolationReport) return s.IsolationReport();
  return http("/api/isolation");
}

/** connectors 走 ConnectorsList，有 ConnectorCatalog 时一并读取；否则走 /api/connectors。 */
export async function connectors(): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorsList) {
    return { accounts: asArray(await s.ConnectorsList()), catalog: asArray(s.ConnectorCatalog ? await s.ConnectorCatalog() : []) };
  }
  return http("/api/connectors");
}

/** connectorConnect 走 ConnectorConnect 或 POST /api/connectors。row 传给 ConnectorConnect，传给 JSON.stringify。 */
export async function connectorConnect(row: any): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorConnect) return s.ConnectorConnect(row);
  return http("/api/connectors", { method: "POST", body: JSON.stringify(row) });
}

/**
 * connectorAuthURL 走 ConnectorAuthURL 或 POST /api/connectors/oauth。
 * provider 传给 ConnectorAuthURL，放进请求体。
 * clientId 传给 ConnectorAuthURL，放进请求体。
 * redirect 传给 ConnectorAuthURL，放进请求体。
 */
export async function connectorAuthURL(provider: string, clientId: string, redirect: string): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorAuthURL) return s.ConnectorAuthURL(provider, clientId, redirect);
  return http("/api/connectors/oauth", { method: "POST", body: JSON.stringify({ provider, client_id: clientId, redirect }) });
}

/**
 * connectorComplete 走 ConnectorComplete 或 POST /api/connectors/oauth。
 * provider 传给 ConnectorComplete，放进请求体。
 * code 传给 ConnectorComplete，放进请求体。
 * clientId 传给 ConnectorComplete，放进请求体。
 * secret 传给 ConnectorComplete，放进请求体。
 * redirect 传给 ConnectorComplete，放进请求体。
 */
export async function connectorComplete(provider: string, code: string, clientId = "", secret = "", redirect = "http://127.0.0.1:3080/oauth"): Promise<any> {
  const s = await wailsService();
  if (s?.ConnectorComplete) return s.ConnectorComplete(provider, code, clientId, secret, redirect);
  return http("/api/connectors/oauth", {
    method: "POST",
    body: JSON.stringify({ provider, code, client_id: clientId, secret, redirect }),
  });
}

/** connectorDisconnect 走 ConnectorDisconnect；服务对象在但没有该方法则返回假，没有服务才 DELETE /api/connectors。id 编码，拼进字符串，传给 ConnectorDisconnect，写进 /api/connectors?id={id}。 */
export async function connectorDisconnect(id: string): Promise<boolean> {
  const s = await wailsService();
  if (s?.ConnectorDisconnect) return asBool(await s.ConnectorDisconnect(id));
  if (s) return false;
  const r = await http<any>(`/api/connectors?id=${encodeURIComponent(id)}`, { method: "DELETE" });
  return asBool(pick(r, "ok", "Ok") ?? true);
}

/** triggerWebhook 走 TriggerWebhook 或 POST /api/hooks/{id}。 */
export async function triggerWebhook(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.TriggerWebhook) { await s.TriggerWebhook(id); return; }
  await http(`/api/hooks/${encodeURIComponent(id)}`, { method: "POST" });
}

/** memoryList 走 MemoryList 或 /api/memory。q 传给 MemoryList，写进 /api/memory?{q}，写入字段 q。kind 传给 MemoryList，写入字段 kind。 */
export async function memoryList(q = "", kind = ""): Promise<any[]> {
  const s = await wailsService();
  if (s?.MemoryList) return asArray(await s.MemoryList(q, kind));
  const qs = new URLSearchParams({ q, kind });
  return asArray(await http(`/api/memory?${qs}`));
}

/** memoryPromote 走 MemoryPromote，或 POST /api/memory 且 op 为 promote。id 传给 MemoryPromote，放进 op 为 promote 的请求体。 */
export async function memoryPromote(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.MemoryPromote) { await s.MemoryPromote(id); return; }
  await http("/api/memory", { method: "POST", body: JSON.stringify({ op: "promote", id }) });
}

/** memoryForget 走 MemoryForget，或 POST /api/memory 且 op 为 forget。id 传给 MemoryForget，放进 op 为 forget 的请求体。 */
export async function memoryForget(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.MemoryForget) { await s.MemoryForget(id); return; }
  await http("/api/memory", { method: "POST", body: JSON.stringify({ op: "forget", id }) });
}

/** memoryWrite 走 MemoryWrite 或 POST /api/memory。kind 传给 MemoryWrite，放进请求体。text 传给 MemoryWrite，放进请求体。project 传给 MemoryWrite，放进请求体。 */
export async function memoryWrite(kind: string, text: string, project = ""): Promise<any> {
  const s = await wailsService();
  if (s?.MemoryWrite) return s.MemoryWrite(kind, text, project);
  return http("/api/memory", { method: "POST", body: JSON.stringify({ kind, text, project }) });
}

/** personalSnapshot 走 PersonalSnapshot 或 /api/personal。 */
export async function personalSnapshot(): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalSnapshot) return s.PersonalSnapshot();
  return http("/api/personal");
}

/**
 * personalAnswer 走 PersonalAnswer，或 POST /api/personal 且 op 为 answer。
 * id 传给 PersonalAnswer，放进 op 为 answer 的请求体。
 * text 传给 PersonalAnswer，放进 op 为 answer 的请求体。
 */
export async function personalAnswer(id: string, text: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalAnswer) return s.PersonalAnswer(id, text);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "answer", id, text }) });
}

/**
 * personalDecide 走 PersonalDecide，或 POST /api/personal 且 op 为 decide。
 * id 传给 PersonalDecide，放进 op 为 decide 的请求体。
 * hash 传给 PersonalDecide，放进 op 为 decide 的请求体。
 * approve 传给 PersonalDecide，放进 op 为 decide 的请求体。
 */
export async function personalDecide(id: string, hash: string, approve: boolean): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalDecide) return s.PersonalDecide(id, hash, approve);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "decide", id, hash, approve }) });
}

/** personalIdea 走 PersonalIdea，或 POST /api/personal 且 op 为 idea。id 传给 PersonalIdea，放进 op 为 idea 的请求体。action 传给 PersonalIdea，放进 op 为 idea 的请求体。 */
export async function personalIdea(id: string, action: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalIdea) return s.PersonalIdea(id, action);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "idea", id, action }) });
}

/**
 * personalGoal 走 PersonalGoal，或 POST /api/personal 且 op 为 goal。
 * title 传给 PersonalGoal，放进 op 为 goal 的请求体。
 * description 传给 PersonalGoal，放进 op 为 goal 的请求体。
 * milestones 传给 PersonalGoal，放进 op 为 goal 的请求体。
 */
export async function personalGoal(title: string, description: string, milestones: string[] = []): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalGoal) return s.PersonalGoal(title, description, milestones);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "goal", title, description, milestones }) });
}

/**
 * personalWatch 走 PersonalWatch，或 POST /api/personal 且 op 为 watch。
 * title 传给 PersonalWatch，放进 op 为 watch 的请求体。
 * url 传给 PersonalWatch，放进 op 为 watch 的请求体。
 * condition 传给 PersonalWatch，放进 op 为 watch 的请求体。
 * value 传给 PersonalWatch，放进 op 为 watch 的请求体。
 * interval 传给 PersonalWatch，放进 op 为 watch 的请求体。
 */
export async function personalWatch(title: string, url: string, condition: string, value: string, interval = 15): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalWatch) return s.PersonalWatch(title, url, condition, value, interval);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "watch", title, url, condition, value, interval_minutes: interval }) });
}

/** personalCancel 走 PersonalCancel，或 POST /api/personal 且 op 为 cancel。id 传给 PersonalCancel，放进 op 为 cancel 的请求体。 */
export async function personalCancel(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalCancel) return s.PersonalCancel(id);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "cancel", id }) });
}

/** personalPause 走 PersonalPause，或 POST /api/personal 且 op 为 pause。id 传给 PersonalPause，放进 op 为 pause 的请求体。 */
export async function personalPause(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalPause) return s.PersonalPause(id);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "pause", id }) });
}

/** personalResume 走 PersonalResume，或 POST /api/personal 且 op 为 resume。id 传给 PersonalResume，放进 op 为 resume 的请求体。 */
export async function personalResume(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalResume) return s.PersonalResume(id);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "resume", id }) });
}

/** personalRetry 走 PersonalRetry，或 POST /api/personal 且 op 为 retry。id 传给 PersonalRetry，放进 op 为 retry 的请求体。 */
export async function personalRetry(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalRetry) return s.PersonalRetry(id);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "retry", id }) });
}

/**
 * personalGoalStatus 走 PersonalGoalStatus，或 POST /api/personal 且 op 为 goal_status。
 * id 传给 PersonalGoalStatus，放进 op 为 goal_status 的请求体。
 * status 传给 PersonalGoalStatus，放进 op 为 goal_status 的请求体。
 */
export async function personalGoalStatus(id: string, status: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalGoalStatus) return s.PersonalGoalStatus(id, status);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "goal_status", id, status }) });
}

/**
 * personalMilestone 走 PersonalMilestone，或 POST /api/personal 且 op 为 milestone。
 * goalId 传给 PersonalMilestone，放进 op 为 milestone 的请求体。
 * milestoneId 传给 PersonalMilestone，放进 op 为 milestone 的请求体。
 * done 传给 PersonalMilestone，放进 op 为 milestone 的请求体。
 */
export async function personalMilestone(goalId: string, milestoneId: string, done: boolean): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalMilestone) return s.PersonalMilestone(goalId, milestoneId, done);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "milestone", goal_id: goalId, milestone_id: milestoneId, done }) });
}

/**
 * personalMonitorStatus 走 PersonalMonitorStatus，或 POST /api/personal 且 op 为 monitor_status。
 * id 传给 PersonalMonitorStatus，放进 op 为 monitor_status 的请求体。
 * status 传给 PersonalMonitorStatus，放进 op 为 monitor_status 的请求体。
 */
export async function personalMonitorStatus(id: string, status: string): Promise<any> {
  const s = await wailsService();
  if (s?.PersonalMonitorStatus) return s.PersonalMonitorStatus(id, status);
  return http("/api/personal", { method: "POST", body: JSON.stringify({ op: "monitor_status", id, status }) });
}

/** scheduleList 走 ScheduleList 或 /api/schedule。 */
export async function scheduleList(): Promise<any[]> {
  const s = await wailsService();
  if (s?.ScheduleList) return asArray(await s.ScheduleList());
  return asArray(await http("/api/schedule"));
}

/** scheduleCreate 走 ScheduleCreate 或 POST /api/schedule。job 传给 ScheduleCreate，传给 JSON.stringify。 */
export async function scheduleCreate(job: any): Promise<any> {
  const s = await wailsService();
  if (s?.ScheduleCreate) return s.ScheduleCreate(job);
  return http("/api/schedule", { method: "POST", body: JSON.stringify(job) });
}

/** scheduleCancel 走 ScheduleCancel 或 DELETE /api/schedule。id 编码，拼进字符串，传给 ScheduleCancel，写进 /api/schedule?id={id}。 */
export async function scheduleCancel(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.ScheduleCancel) { await s.ScheduleCancel(id); return; }
  await http(`/api/schedule?id=${encodeURIComponent(id)}`, { method: "DELETE" });
}

/** scheduleRetry 走 ScheduleRetry 或 POST /api/schedule/retry。id 传给 ScheduleRetry，放进请求体。 */
export async function scheduleRetry(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.ScheduleRetry) return s.ScheduleRetry(id);
  return http("/api/schedule/retry", { method: "POST", body: JSON.stringify({ id }) });
}

/** setPaused 走 SetPaused 或 POST /api/pause。paused 传给 SetPaused，放进请求体。 */
export async function setPaused(paused: boolean): Promise<void> {
  const s = await wailsService();
  if (s?.SetPaused) { await s.SetPaused(paused); return; }
  await http("/api/pause", { method: "POST", body: JSON.stringify({ paused }) });
}

/** pagesList 走 PagesList 或 /api/pages。q 传给 PagesList，写进 /api/pages?{q}，写入字段 q。spaceId 传给 PagesList，写入字段 space_id。 */
export async function pagesList(q = "", spaceId = ""): Promise<any[]> {
  const s = await wailsService();
  if (s?.PagesList) return asArray(await s.PagesList(q, spaceId));
  const qs = new URLSearchParams({ q, space_id: spaceId });
  return asArray(await http(`/api/pages?${qs}`));
}

/** pagesGet 走 PagesGet 或 /api/pages/{id}。spaceId 编码，为真时才继续，拼进字符串，传给 PagesGet。 */
export async function pagesGet(id: string, spaceId = ""): Promise<any> {
  const s = await wailsService();
  if (s?.PagesGet) return s.PagesGet(spaceId, id);
  const qs = spaceId ? `?space_id=${encodeURIComponent(spaceId)}` : "";
  return http(`/api/pages/${encodeURIComponent(id)}${qs}`);
}

/** pagesSave 走 PagesSave 或 POST /api/pages。p 读取 expected_revision、revision，传给 PagesSave，放进请求体，展开传递。 */
export async function pagesSave(p: any): Promise<any> {
  const s = await wailsService();
  const expected = Number(p.expected_revision || p.revision || 0);
  if (s?.PagesSave) return s.PagesSave(p, expected);
  return http("/api/pages", { method: "POST", body: JSON.stringify({ ...p, expected_revision: expected }) });
}

/** pagesDelete 走 PagesDelete 或 DELETE /api/pages/{id}。spaceId 编码，为真时才继续，拼进字符串，传给 PagesDelete。 */
export async function pagesDelete(id: string, spaceId = ""): Promise<void> {
  const s = await wailsService();
  if (s?.PagesDelete) { await s.PagesDelete(spaceId, id); return; }
  const qs = spaceId ? `?space_id=${encodeURIComponent(spaceId)}` : "";
  await http(`/api/pages/${encodeURIComponent(id)}${qs}`, { method: "DELETE" });
}

/** pagesReviews 走 PagesReviews 或 /api/pages/reviews。 */
export async function pagesReviews(): Promise<any[]> {
  const s = await wailsService();
  if (s?.PagesReviews) return asArray(await s.PagesReviews());
  return asArray(await http("/api/pages/reviews"));
}

/**
 * pagesPropose 走 PagesPropose 或 POST /api/pages/propose。
 * session 传给 PagesPropose，放进请求体。
 * title 传给 PagesPropose，放进请求体。
 * content 传给 PagesPropose，放进请求体。
 * pageId 传给 PagesPropose，放进请求体。
 */
export async function pagesPropose(session: string, title: string, content: string, pageId = ""): Promise<any> {
  const s = await wailsService();
  if (s?.PagesPropose) return s.PagesPropose(session, title, content, pageId, "", "", 0);
  return http("/api/pages/propose", { method: "POST", body: JSON.stringify({ session, title, content, id: pageId }) });
}

/** pagesDecide 走 PagesDecide 或 POST /api/pages/reviews。id 传给 PagesDecide，放进请求体。hash 传给 PagesDecide，放进请求体。approve 传给 PagesDecide，放进请求体。 */
export async function pagesDecide(id: string, hash: string, approve: boolean): Promise<any> {
  const s = await wailsService();
  if (s?.PagesDecide) return s.PagesDecide(id, hash, approve);
  return http("/api/pages/reviews", { method: "POST", body: JSON.stringify({ id, hash, approve }) });
}

/** profilesList 走 ProfilesList 或 /api/profiles。 */
export async function profilesList(): Promise<any[]> {
  const s = await wailsService();
  if (s?.ProfilesList) return asArray(await s.ProfilesList());
  return asArray(await http("/api/profiles"));
}

/** profilesSave 走 ProfilesSave 或 POST /api/profiles。p 传给 ProfilesSave，传给 JSON.stringify。 */
export async function profilesSave(p: any): Promise<any> {
  const s = await wailsService();
  if (s?.ProfilesSave) return s.ProfilesSave(p);
  return http("/api/profiles", { method: "POST", body: JSON.stringify(p) });
}

/** setSessionProfile 走 SetSessionProfile 或 POST /api/sessions/{id}/profile。profileId 传给 SetSessionProfile，放进请求体。 */
export async function setSessionProfile(id: string, profileId: string): Promise<any> {
  const s = await wailsService();
  if (s?.SetSessionProfile) return threadOf(await s.SetSessionProfile(id, profileId));
  return threadOf(await http(`/api/sessions/${encodeURIComponent(id)}/profile`, { method: "POST", body: JSON.stringify({ profile_id: profileId }) }));
}

/** setSessionPage 走 SetSessionPage 或 POST /api/sessions/{id}/page。pageId 传给 SetSessionPage，放进请求体。 */
export async function setSessionPage(id: string, pageId: string): Promise<any> {
  const s = await wailsService();
  if (s?.SetSessionPage) return threadOf(await s.SetSessionPage(id, pageId));
  return threadOf(await http(`/api/sessions/${encodeURIComponent(id)}/page`, { method: "POST", body: JSON.stringify({ page_id: pageId }) }));
}

/** voiceTranscribe 走 VoiceTranscribe，或用 fetch POST /api/voice/transcribe。filename 传给 VoiceTranscribe，放进请求体。audio 传给 VoiceTranscribe，放进请求体。 */
export async function voiceTranscribe(filename: string, audio: string): Promise<string> {
  const s = await wailsService();
  if (s?.VoiceTranscribe) return String(await s.VoiceTranscribe(filename, audio) || "");
  const r = await fetch("/api/voice/transcribe", browserFetchInit({
    method: "POST",
    body: JSON.stringify({ filename, audio }),
    signal: timedSignal(120000),
  }));
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new ApiError(r.status, body, String(body.error || r.statusText));
  return String(body.text || "");
}

/** voiceSpeak 走 VoiceSpeak，或用 fetch POST /api/voice/speak。text 传给 VoiceSpeak，放进请求体。 */
export async function voiceSpeak(text: string): Promise<{ audio: string; contentType: string }> {
  const s = await wailsService();
  if (s?.VoiceSpeak) {
    const r = await s.VoiceSpeak(text);
    return { audio: String(r?.audio || ""), contentType: String(r?.content_type || r?.contentType || "audio/mpeg") };
  }
  const r = await fetch("/api/voice/speak", browserFetchInit({
    method: "POST",
    body: JSON.stringify({ text }),
    signal: timedSignal(120000),
  }));
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new ApiError(r.status, body, String(body.error || r.statusText));
  return { audio: String(body.audio || ""), contentType: String(body.content_type || "audio/mpeg") };
}

/**
 * voiceReceipt 走 VoiceReceipt 或 POST /api/voice/receipt。
 * session 传给 VoiceReceipt，放进请求体。
 * durationSec 传给 VoiceReceipt，放进请求体。
 * transcript 传给 VoiceReceipt，放进请求体。
 */
export async function voiceReceipt(session: string, durationSec: number, transcript: string): Promise<void> {
  const s = await wailsService();
  if (s?.VoiceReceipt) { s.VoiceReceipt(session, durationSec, transcript); return; }
  await http("/api/voice/receipt", { method: "POST", body: JSON.stringify({ session, duration_sec: durationSec, transcript }) });
}

/** setCompanionCall 走 SetCompanionCall。on 传给 SetCompanionCall。 */
export async function setCompanionCall(on: boolean): Promise<void> {
  const s = await wailsService();
  const fn = svcMethod(s, "SetCompanionCall", "setCompanionCall");
  if (fn) await fn(on);
}

/** placeIdol 有 PlaceIdol 就走它并返回；否则 idolCall 使用 pin，再 SetConfig 或 POST /api/config 写入 companion_enabled，然后 showCompanion。id 传给 PlaceIdol，写入字段 id。 */
export async function placeIdol(id: string): Promise<void> {
  const s = await wailsService();
  const place = svcMethod(s, "PlaceIdol", "placeIdol");
  if (place) {
    await place(id);
    return;
  }
  await idolCall("pin", { id });
  const set = svcMethod(s, "SetConfig", "setConfig");
  if (set) await set({ companion_enabled: true });
  else await http("/api/config", { method: "POST", body: JSON.stringify({ companion_enabled: true }) });
  await showCompanion();
}

/** idolCall 走 IdolCall 或 POST /api/idol。op 传给 IdolCall，放进请求体。payload 为空时改用后备值，传给 JSON.stringify，放进请求体。 */
export async function idolCall(op: string, payload: Record<string, unknown> = {}): Promise<any> {
  const body = JSON.stringify(payload ?? {});
  const s = await wailsService();
  const fn = svcMethod(s, "IdolCall", "idolCall");
  if (fn) {
    const raw = await fn(op, body);
    if (raw == null || raw === "") return {};
    if (typeof raw === "string") {
      try { return JSON.parse(raw); } catch { return { raw }; }
    }
    return raw;
  }
  return http("/api/idol", {
    method: "POST",
    body: JSON.stringify({ op, payload }),
    signal: AbortSignal.timeout(180000),
  });
}

/** connectorMention 走 ConnectorMention 或 POST /api/connectors/mention。account 传给 ConnectorMention，放进请求体。text 传给 ConnectorMention，放进请求体。 */
export async function connectorMention(account: string, text: string): Promise<void> {
  const s = await wailsService();
  if (s?.ConnectorMention) { await s.ConnectorMention(account, text); return; }
  await http("/api/connectors/mention", { method: "POST", body: JSON.stringify({ account, text }) });
}

/** projectsList 走 ProjectsList 或 /api/projects。 */
export async function projectsList(): Promise<any[]> {
  const s = await wailsService();
  if (s?.ProjectsList) return asArray(await s.ProjectsList());
  return asArray(await http("/api/projects"));
}

/** projectCreate 走 ProjectCreate 或 POST /api/projects。p 传给 ProjectCreate，传给 JSON.stringify。 */
export async function projectCreate(p: any): Promise<any> {
  const s = await wailsService();
  if (s?.ProjectCreate) return s.ProjectCreate(p);
  return http("/api/projects", { method: "POST", body: JSON.stringify(p) });
}

/** reviewQueue 走 ReviewQueue 或 /api/review/queue。 */
export async function reviewQueue(): Promise<any> {
  const s = await wailsService();
  if (s?.ReviewQueue) return s.ReviewQueue();
  return http("/api/review/queue");
}

/** BrowserView 是浏览器车道当前的地址、画面和日志。 */
export type BrowserView = {
  url: string;
  title: string;
  lane: string;
  profile: string;
  profileName: string;
  recording: boolean;
  tabCount: number;
  headed: boolean;
  live: boolean;
  text: string;
  screenshot: string;
  preview: string;
  log: { op?: string; detail?: string; url?: string; ts?: string }[];
};

/** browserView 走 BrowserView 或 /api/browser/view。 */
export async function browserView(): Promise<BrowserView> {
  const s = await wailsService();
  const raw = s?.BrowserView ? await s.BrowserView() : await http("/api/browser/view");
  return {
    url: str(pick(raw, "url", "URL")),
    title: str(pick(raw, "title", "Title")),
    lane: str(pick(raw, "lane", "Lane"), "isolated"),
    profile: str(pick(raw, "profile", "Profile")),
    profileName: str(pick(raw, "profileName", "ProfileName"), "default"),
    recording: bool(pick(raw, "recording", "Recording")),
    tabCount: num(pick(raw, "tabCount", "TabCount")),
    headed: bool(pick(raw, "headed", "Headed")),
    live: bool(pick(raw, "live", "Live")),
    text: str(pick(raw, "text", "Text")),
    screenshot: str(pick(raw, "screenshot", "Screenshot")),
    preview: str(pick(raw, "preview", "Preview")),
    log: asArray(pick(raw, "log", "Log")).map((row: any) => ({
      op: str(pick(row, "op")),
      detail: str(pick(row, "detail")),
      url: str(pick(row, "url")),
      ts: str(pick(row, "ts", "TS")),
    })),
  };
}

/**
 * browserTakeover 有路径且有 BrowserTakeoverAt 时走它，否则走 BrowserTakeover，再否则 POST /api/browser/takeover。
 * path 为假时用空字符串，去掉首尾空白后，为真时才继续，传给 BrowserTakeoverAt，放进请求体。
 */
export async function browserTakeover(path?: string): Promise<void> {
  const s = await wailsService();
  const rel = (path || "").trim();
  if (rel && s?.BrowserTakeoverAt) {
    await s.BrowserTakeoverAt(rel);
    return;
  }
  if (s?.BrowserTakeover) {
    await s.BrowserTakeover();
    return;
  }
  await http("/api/browser/takeover", { method: "POST", body: JSON.stringify({ path: rel }) });
}

/** clipboardRead 走 ClipboardRead；没有该方法则读 navigator.clipboard，失败则返回空串。 */
export async function clipboardRead(): Promise<string> {
  const s = await wailsService();
  if (s?.ClipboardRead) return String(await s.ClipboardRead() || "");
  try {
    if (navigator.clipboard?.readText) return (await navigator.clipboard.readText()) || "";
  } catch {
    /* 页面没有拿到剪贴板权限 */
  }
  return "";
}

/** captureScreenshot 有工作区且有 Screenshot 时把图写到 .yoyo/captures/shot.png 并返回该相对路径；否则返回空串。workspace 去掉首尾空白，为假时走另一支，传给 joinWorkspace。 */
export async function captureScreenshot(workspace = ""): Promise<string> {
  const rel = ".yoyo/captures/shot.png";
  if (!workspace.trim()) return "";
  const s = await wailsService();
  if (s?.Screenshot) {
    const { joinWorkspace } = await import("./workspace");
    await s.Screenshot(joinWorkspace(workspace, rel));
    return rel;
  }
  return "";
}

/** admitExpert 走 AdmitExpert 或 POST /api/expert/admit。dir 传给 AdmitExpert，放进请求体。 */
export async function admitExpert(dir: string): Promise<any> {
  const s = await wailsService();
  if (s?.AdmitExpert) return s.AdmitExpert(dir);
  return http("/api/expert/admit", { method: "POST", body: JSON.stringify({ dir }) });
}

/** distillExpert 走 DistillExpert 或 POST /api/expert/distill。name 传给 DistillExpert，放进请求体。description 传给 DistillExpert，放进请求体。body 传给 DistillExpert，放进请求体。 */
export async function distillExpert(name: string, description: string, body: string): Promise<any> {
  const s = await wailsService();
  if (s?.DistillExpert) return s.DistillExpert(name, description, body);
  return http("/api/expert/distill", { method: "POST", body: JSON.stringify({ name, description, body }) });
}

/** stopMCP 走 StopMCP 或 POST /api/mcp/stop。name 传给 StopMCP，放进请求体。 */
export async function stopMCP(name: string): Promise<void> {
  const s = await wailsService();
  if (s?.StopMCP) return s.StopMCP(name);
  await http("/api/mcp/stop", { method: "POST", body: JSON.stringify({ name }) });
}

/** about 走 About 或 /api/about。 */
export async function about(): Promise<any> {
  const s = await wailsService();
  if (s?.About) return s.About();
  return http("/api/about");
}

/** doctor 走 Doctor 或 /api/doctor。 */
export async function doctor(): Promise<any> {
  const s = await wailsService();
  if (s?.Doctor) return s.Doctor();
  return http("/api/doctor");
}

/** logs 走 Logs 或 /api/logs。limit 传给 Logs。 */
export async function logs(limit = 80): Promise<any> {
  const s = await wailsService();
  if (s?.Logs) return s.Logs(limit);
  return http("/api/logs");
}

/** journal 走 Journal 或 /api/journal。limit 传给 Journal。 */
export async function journal(limit = 80): Promise<any> {
  const s = await wailsService();
  if (s?.Journal) return s.Journal(limit);
  return http("/api/journal");
}

/** exportDiagnostics 走 ExportDiagnostics 或 POST /api/logs/export。 */
export async function exportDiagnostics(): Promise<string> {
  const s = await wailsService();
  if (s?.ExportDiagnostics) {
    const path = await s.ExportDiagnostics();
    return String(path || "");
  }
  const r = await http<{ path?: string }>("/api/logs/export", { method: "POST", body: "{}" });
  return String(r?.path || "");
}

/** diagnoseSession 走 DiagnoseSession 或 POST /api/logs/diagnose。session 传给 DiagnoseSession，放进请求体。 */
export async function diagnoseSession(session: string): Promise<string> {
  const s = await wailsService();
  if (s?.DiagnoseSession) return String((await s.DiagnoseSession(session)) || "");
  const r = await http<{ text?: string }>("/api/logs/diagnose", { method: "POST", body: JSON.stringify({ session }) });
  return String(r?.text || "");
}

/** reportFrontend events 为空时直接返回；否则走 FrontendLogs 或 POST /api/logs。 */
export async function reportFrontend(events: { ts?: string; level: string; msg: string; stack?: string; source?: string; session_id?: string }[]): Promise<void> {
  if (!events.length) return;
  const s = await wailsService();
  if (s?.FrontendLogs) {
    await s.FrontendLogs(events);
    return;
  }
  await http("/api/logs", { method: "POST", body: JSON.stringify({ events }) });
}

/** checkUpdate 走 CheckUpdate 或 /api/update。 */
export async function checkUpdate(): Promise<any> {
  const s = await wailsService();
  if (s?.CheckUpdate) return s.CheckUpdate();
  return http("/api/update");
}

/** updateStatus 走 UpdateStatus 或 /api/update/status。 */
export async function updateStatus(): Promise<any> {
  const s = await wailsService();
  if (s?.UpdateStatus) return s.UpdateStatus();
  return http("/api/update/status");
}

/** remoteStatus 走 RemoteStatus 或 /api/remote/status。 */
export async function remoteStatus(): Promise<any> {
  const s = await wailsService();
  if (s?.RemoteStatus) return s.RemoteStatus();
  return http("/api/remote/status");
}

/** remotePair 走 RemotePair 或 POST /api/remote/pair。 */
export async function remotePair(): Promise<any> {
  const s = await wailsService();
  if (s?.RemotePair) return s.RemotePair();
  return http("/api/remote/pair", { method: "POST", body: "{}" });
}

/** remoteApprove 走 RemoteApprove 或 POST /api/remote/approve。id 传给 RemoteApprove，放进请求体。 */
export async function remoteApprove(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.RemoteApprove) return s.RemoteApprove(id);
  return http("/api/remote/approve", { method: "POST", body: JSON.stringify({ id }) });
}

/** remoteDeny 走 RemoteDeny 或 POST /api/remote/deny。id 传给 RemoteDeny，放进请求体。 */
export async function remoteDeny(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.RemoteDeny) return s.RemoteDeny(id);
  return http("/api/remote/deny", { method: "POST", body: JSON.stringify({ id }) });
}

/** remoteRevoke 走 RemoteRevoke 或 POST /api/remote/revoke。id 传给 RemoteRevoke，放进请求体。 */
export async function remoteRevoke(id: string): Promise<any> {
  const s = await wailsService();
  if (s?.RemoteRevoke) return s.RemoteRevoke(id);
  return http("/api/remote/revoke", { method: "POST", body: JSON.stringify({ id }) });
}

/** remoteDisable 走 RemoteDisable 或 POST /api/remote/disable。 */
export async function remoteDisable(): Promise<any> {
  const s = await wailsService();
  if (s?.RemoteDisable) return s.RemoteDisable();
  return http("/api/remote/disable", { method: "POST", body: "{}" });
}

/** testProvider 有 id 且有 TestChatProvider 时走它，否则走 TestProvider，再否则 POST /api/provider/test。 */
export async function testProvider(id = ""): Promise<any> {
  const s = await wailsService();
  if (id && s?.TestChatProvider) return s.TestChatProvider(id);
  if (s?.TestProvider) return s.TestProvider();
  return http("/api/provider/test", { method: "POST", body: JSON.stringify({ id }) });
}

/** listChatProviders 走 ListChatProviders 或 /api/providers。 */
export async function listChatProviders(): Promise<ChatProvider[]> {
  const s = await wailsService();
  if (s?.ListChatProviders) return asArray(await s.ListChatProviders()).map(chatProviderOf);
  return asArray(await http("/api/providers")).map(chatProviderOf);
}

/** upsertChatProvider 走 UpsertChatProvider 或 POST /api/providers。provider 读取 id、name、vendor、endpoint、model、models 等。apiKey 传给 UpsertChatProvider，放进请求体。 */
export async function upsertChatProvider(
  provider: Partial<ChatProvider> & { make_default?: boolean; makeDefault?: boolean; active?: boolean },
  apiKey = "",
): Promise<ChatProvider> {
  const s = await wailsService();
  const body = {
    id: provider.id || "",
    name: provider.name || "",
    vendor: provider.vendor || "custom",
    endpoint: provider.endpoint || "",
    model: provider.model || "",
    models: provider.models || [],
    active: provider.active,
    make_default: !!(provider.make_default ?? provider.makeDefault ?? provider.isDefault),
  };
  if (s?.UpsertChatProvider) return chatProviderOf(await s.UpsertChatProvider(body, apiKey));
  return chatProviderOf(await http("/api/providers", { method: "POST", body: JSON.stringify({ ...body, api_key: apiKey }) }));
}

/** deleteChatProvider 走 DeleteChatProvider 或 DELETE /api/providers/{id}。 */
export async function deleteChatProvider(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.DeleteChatProvider) {
    await s.DeleteChatProvider(id);
    return;
  }
  await http(`/api/providers/${encodeURIComponent(id)}`, { method: "DELETE" });
}

/** setDefaultChatProvider 走 SetDefaultChatProvider 或 POST /api/providers/{id}/default。 */
export async function setDefaultChatProvider(id: string): Promise<void> {
  const s = await wailsService();
  if (s?.SetDefaultChatProvider) {
    await s.SetDefaultChatProvider(id);
    return;
  }
  await http(`/api/providers/${encodeURIComponent(id)}/default`, { method: "POST", body: "{}" });
}

/** replaceMCP 走 ReplaceMCP 或 POST /api/mcp/replace。servers 传给 ReplaceMCP，放进请求体。 */
export async function replaceMCP(servers: { name: string; command: string; args: string[]; endpoint?: string }[]): Promise<void> {
  const s = await wailsService();
  if (s?.ReplaceMCP) return s.ReplaceMCP(servers);
  await http("/api/mcp/replace", { method: "POST", body: JSON.stringify({ servers }) });
}

/** revealLogs 走 RevealLogs。 */
export async function revealLogs(): Promise<void> {
  const s = await wailsService();
  if (s?.RevealLogs) return s.RevealLogs();
}

/** revealJournal 走 RevealJournal。 */
export async function revealJournal(): Promise<void> {
  const s = await wailsService();
  if (s?.RevealJournal) return s.RevealJournal();
}

/** keyStatus 走 KeyStatus 或 /api/key/status。 */
export async function keyStatus(): Promise<any> {
  const s = await wailsService();
  if (s?.KeyStatus) return s.KeyStatus();
  return http("/api/key/status");
}

/** quit 在 VITE_E2E 时直接返回；否则发出 yoyo:do-quit，发不出就调用 window.close。 */
export async function quit(): Promise<void> {
  if (import.meta.env.VITE_E2E) return;
  try {
    const mod: any = await import("@wailsio/runtime");
    if (mod.Events?.Emit) {
      mod.Events.Emit("yoyo:do-quit");
      return;
    }
  } catch {
    /* 浏览器里没有 Wails 事件 */
  }
  window.close();
}

/** errMessage 再导出错误文本的取出。定义在 normalize。 */
export { errMessage };

/** learnCall 把学习模块的方法交给 videoCall。桌面没有单独的 HTTP。method 以 learn. 开头。params 原样转发。 */
export function learnCall(method: string, params: Record<string, any> = {}): Promise<any> {
  return videoCall(method, params);
}

/** videoCall 走 VideoCall；没有该方法则走 rpcJSON，也就是 POST /api/rpc。method 传给 VideoCall，传给 rpcJSON。params 传给 VideoCall，传给 rpcJSON。 */
export async function videoCall(method: string, params: Record<string, any> = {}): Promise<any> {
  const s = await wailsService();
  const fn = svcMethod(s, "VideoCall", "videoCall");
  if (fn) return fn(method, params);
  return rpcJSON(method, params, 120000);
}

/** rpcJSON 向 /api/rpc 发 JSON-RPC。解析失败保留原文。响应里的 error 抛成 ApiError。method 拼进字符串，放进请求体。params 放进请求体。timeoutMs 传给 timedSignal。 */
async function rpcJSON(method: string, params: Record<string, any> = {}, timeoutMs = 120000): Promise<any> {
  let res: Response;
  try {
    res = await fetch("/api/rpc", browserFetchInit({
      method: "POST",
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
      signal: timedSignal(timeoutMs),
    }));
  } catch (e) {
    throw new Error(`${method}: ${errMessage(e)}`);
  }
  const text = await res.text();
  let body: any = text;
  try {
    body = text ? JSON.parse(text) : {};
  } catch {
    /* 保留原文 */
  }
  if (!res.ok) throw new ApiError(res.status, body, `${method}: ${typeof body === "object" && body?.error ? body.error : text || res.statusText}`);
  if (body?.error) throw new ApiError(500, body, String(body.error.message || body.error));
  return body?.result ?? body;
}

/** runtime 把 PowerShell 运行时的状态和安装转到 videoCall。 */
export const runtime = {
  /** pwshStatus 走 videoCall("runtime.pwsh.status")。 */
  pwshStatus: () => videoCall("runtime.pwsh.status"),
  /** pwshInstall 走 videoCall("runtime.pwsh.install")。 */
  pwshInstall: () => videoCall("runtime.pwsh.install"),
};

/** video 把短剧、剪辑和工作室调用转到 videoCall，不在这里实现画面。 */
export const video = {
  /** status 走 videoCall("video.status")。 */
  status: () => videoCall("video.status"),
  /** installFFmpeg 走 videoCall("video.ffmpeg.install")。 */
  installFFmpeg: () => videoCall("video.ffmpeg.install"),
  /** installYTDLP 走 videoCall("video.ytdlp.install")。 */
  installYTDLP: () => videoCall("video.ytdlp.install"),
  /** modes 走 videoCall("video.modes")。 */
  modes: () => videoCall("video.modes"),
  /** templates 走 videoCall("video.templates")。 */
  templates: () => videoCall("video.templates"),
  /** providers 走 videoCall("video.providers.list")。serviceType 写入字段 service_type。 */
  providers: (serviceType = "") => videoCall("video.providers.list", { service_type: serviceType }),
  /** upsertProvider 走 videoCall("video.providers.upsert")。provider 写入字段 provider。apiKey 写入字段 api_key。 */
  upsertProvider: (provider: any, apiKey = "") => videoCall("video.providers.upsert", { provider, api_key: apiKey }),
  /** deleteProvider 走 videoCall("video.providers.delete")。id 写入字段 id。 */
  deleteProvider: (id: string) => videoCall("video.providers.delete", { id }),
  /** testProvider 走 videoCall("video.providers.test")。id 写入字段 id。 */
  testProvider: (id: string) => videoCall("video.providers.test", { id }),
  /** connections 走 videoCall("connections.list")。capability 写入字段 capability。 */
  connections: (capability = "") => videoCall("connections.list", { capability }),
  /** upsertConnection 走 videoCall("connections.upsert")。connection 写入字段 connection。apiKey 写入字段 api_key。 */
  upsertConnection: (connection: any, apiKey = "") => videoCall("connections.upsert", { connection, api_key: apiKey }),
  /** deleteConnection 走 videoCall("connections.delete")。id 写入字段 id。 */
  deleteConnection: (id: string) => videoCall("connections.delete", { id }),
  /** testConnection 走 videoCall("connections.test")。id 写入字段 id。 */
  testConnection: (id: string) => videoCall("connections.test", { id }),
  /** connectionDefaults 走 videoCall("connections.defaults")。 */
  connectionDefaults: () => videoCall("connections.defaults"),
  /** setConnectionDefault 走 videoCall("connections.setDefault")。capability 写入字段 capability。id 写入字段 id。 */
  setConnectionDefault: (capability: string, id: string) => videoCall("connections.setDefault", { capability, id }),
  /** connectionTemplates 走 videoCall("connections.templates")。 */
  connectionTemplates: () => videoCall("connections.templates"),
  /** styles 走 videoCall("video.styles")。 */
  styles: () => videoCall("video.styles"),
  /** allStyles 走 videoCall("video.styles.all")。 */
  allStyles: () => videoCall("video.styles.all"),
  /** upsertStyle 走 videoCall("video.styles.upsert")。s 传给 videoCall。 */
  upsertStyle: (s: Record<string, any>) => videoCall("video.styles.upsert", s),
  /** deleteStyle 走 videoCall("video.styles.delete")。id 写入字段 id。 */
  deleteStyle: (id: string) => videoCall("video.styles.delete", { id }),
  /** settings 走 videoCall("video.settings.get")。 */
  settings: () => videoCall("video.settings.get"),
  /** setSetting 走 videoCall("video.settings.set")。key 写入字段 key。value 写入字段 value。 */
  setSetting: (key: string, value: string) => videoCall("video.settings.set", { key, value }),
  /** jobs 走 videoCall("video.jobs.list")。episodeId 写入字段 episode_id。 */
  jobs: (episodeId = "") => videoCall("video.jobs.list", { episode_id: episodeId }),
  /** cancelJob 走 videoCall("video.jobs.cancel")。id 写入字段 id。 */
  cancelJob: (id: string) => videoCall("video.jobs.cancel", { id }),
  /** retryJob 走 videoCall("video.jobs.retry")。id 写入字段 id。 */
  retryJob: (id: string) => videoCall("video.jobs.retry", { id }),
  /** applyJob 走 videoCall("video.jobs.apply")。id 写入字段 id。 */
  applyJob: (id: string) => videoCall("video.jobs.apply", { id }),
  /** listDramas 走 videoCall("drama.list")。 */
  listDramas: () => videoCall("drama.list"),
  /** createDrama 走 videoCall("drama.create")。d 传给 videoCall。 */
  createDrama: (d: Record<string, any>) => videoCall("drama.create", d),
  /** getDrama 走 videoCall("drama.get")。id 写入字段 id。 */
  getDrama: (id: string) => videoCall("drama.get", { id }),
  /** updateDrama 走 videoCall("drama.update")。d 传给 videoCall。 */
  updateDrama: (d: Record<string, any>) => videoCall("drama.update", d),
  /** deleteDrama 走 videoCall("drama.delete")。id 写入字段 id。 */
  deleteDrama: (id: string) => videoCall("drama.delete", { id }),
  /** episodes 走 videoCall("drama.episodes")。dramaId 写入字段 drama_id。 */
  episodes: (dramaId: string) => videoCall("drama.episodes", { drama_id: dramaId }),
  /** createEpisode 走 videoCall("drama.episode.create")。dramaId 写入字段 drama_id。title 写入字段 title。content 写入字段 content。 */
  createEpisode: (dramaId: string, title: string, content: string) => videoCall("drama.episode.create", { drama_id: dramaId, title, content }),
  /** getEpisode 走 videoCall("drama.episode.get")。id 写入字段 id。 */
  getEpisode: (id: string) => videoCall("drama.episode.get", { id }),
  /** updateEpisode 走 videoCall("drama.episode.update")。ep 传给 videoCall。 */
  updateEpisode: (ep: Record<string, any>) => videoCall("drama.episode.update", ep),
  /** deleteEpisode 走 videoCall("drama.episode.delete")。id 写入字段 id。 */
  deleteEpisode: (id: string) => videoCall("drama.episode.delete", { id }),
  /** bundle 走 videoCall("drama.bundle")。episodeId 写入字段 episode_id。 */
  bundle: (episodeId: string) => videoCall("drama.bundle", { episode_id: episodeId }),
  /** bind 走 videoCall("drama.bind")。sessionId 写入字段 session_id。episodeId 写入字段 episode_id。dramaId 写入字段 drama_id。 */
  bind: (sessionId: string, episodeId = "", dramaId = "") => videoCall("drama.bind", { session_id: sessionId, episode_id: episodeId, drama_id: dramaId }),
  /** stage 走 videoCall("drama.stage.run")。sessionId 写入字段 session_id。episodeId 写入字段 episode_id。dramaId 写入字段 drama_id。 */
  stage: (sessionId: string, episodeId: string, stage: string, dramaId = "") => videoCall("drama.stage.run", { session_id: sessionId, episode_id: episodeId, stage, drama_id: dramaId }),
  /** ingestSource 走 videoCall("drama.source.ingest")。dramaId 写入字段 drama_id。text 写入字段 text。title 写入字段 title。 */
  ingestSource: (dramaId: string, text: string, title = "") => videoCall("drama.source.ingest", { drama_id: dramaId, text, title }),
  /** ingestSourceFile 走 videoCall("drama.source.ingest_file")。dramaId 写入字段 drama_id。path 写入字段 path。 */
  ingestSourceFile: (dramaId: string, path: string) => videoCall("drama.source.ingest_file", { drama_id: dramaId, path }),
  /** getSource 走 videoCall("drama.source.get")。dramaId 写入字段 drama_id。 */
  getSource: (dramaId: string) => videoCall("drama.source.get", { drama_id: dramaId }),
  /** getPlan 走 videoCall("drama.plan.get")。dramaId 写入字段 drama_id。 */
  getPlan: (dramaId: string) => videoCall("drama.plan.get", { drama_id: dramaId }),
  /** proposeEpisodes 走 videoCall("drama.plan.propose")。dramaId 写入字段 drama_id。episodes 写入字段 episodes。 */
  proposeEpisodes: (dramaId: string, episodes: any[] = []) => videoCall("drama.plan.propose", { drama_id: dramaId, episodes }),
  /** patchPlan 走 videoCall("drama.plan.patch")。dramaId 写入字段 drama_id。episodes 写入字段 episodes。 */
  patchPlan: (dramaId: string, episodes: any[]) => videoCall("drama.plan.patch", { drama_id: dramaId, episodes }),
  /** splitPlanEpisode 走 videoCall("drama.plan.split")。dramaId 写入字段 drama_id。n 写入字段 n。 */
  splitPlanEpisode: (dramaId: string, n: number) => videoCall("drama.plan.split", { drama_id: dramaId, n }),
  /** mergePlanEpisodes 走 videoCall("drama.plan.merge")。dramaId 写入字段 drama_id。n 写入字段 n。 */
  mergePlanEpisodes: (dramaId: string, n: number) => videoCall("drama.plan.merge", { drama_id: dramaId, n }),
  /** commitEpisodes 走 videoCall("drama.plan.commit")。dramaId 写入字段 drama_id。 */
  commitEpisodes: (dramaId: string) => videoCall("drama.plan.commit", { drama_id: dramaId }),
  /** queueScripts 走 videoCall("drama.plan.queue_scripts")。dramaId 写入字段 drama_id。sessionId 写入字段 session_id。 */
  queueScripts: (dramaId: string, sessionId: string) => videoCall("drama.plan.queue_scripts", { drama_id: dramaId, session_id: sessionId }),
  /** dramaJobs 走 videoCall("drama.jobs")。dramaId 写入字段 drama_id。 */
  dramaJobs: (dramaId: string) => videoCall("drama.jobs", { drama_id: dramaId }),
  /** bible 走 videoCall("drama.bible")。dramaId 写入字段 drama_id。 */
  bible: (dramaId: string) => videoCall("drama.bible", { drama_id: dramaId }),
  /** saveAsset 走 videoCall("drama.assets.save")。kind 写入字段 kind。id 写入字段 id。fields 展开传递。 */
  saveAsset: (kind: string, id: string, fields: Record<string, any>) => videoCall("drama.assets.save", { kind, id, ...fields }),
  /** createAsset 走 videoCall("drama.assets.create")。kind 写入字段 kind。episodeId 写入字段 episode_id。fields 展开传递。 */
  createAsset: (kind: string, episodeId: string, fields: Record<string, any>) => videoCall("drama.assets.create", { kind, episode_id: episodeId, ...fields }),
  /** deleteAsset 走 videoCall("drama.assets.delete")。kind 写入字段 kind。id 写入字段 id。 */
  deleteAsset: (kind: string, id: string) => videoCall("drama.assets.delete", { kind, id }),
  /** generateAsset 走 videoCall("drama.assets.generate")。kind 写入字段 kind。id 写入字段 id。episodeId 写入字段 episode_id。 */
  generateAsset: (kind: string, id: string, episodeId = "") => videoCall("drama.assets.generate", { kind, id, episode_id: episodeId }),
  /** generateMissingAssets 走 videoCall("drama.assets.generate_missing")。episodeId 写入字段 episode_id。 */
  generateMissingAssets: (episodeId: string) => videoCall("drama.assets.generate_missing", { episode_id: episodeId }),
  /** uploadAsset 走 videoCall("drama.assets.upload")。kind 写入字段 kind。id 写入字段 id。dataB64 写入字段 data_b64。 */
  uploadAsset: (kind: string, id: string, dataB64: string) => videoCall("drama.assets.upload", { kind, id, data_b64: dataB64 }),
  /** saveShots 走 videoCall("drama.shots.save")。episodeId 写入字段 episode_id。replace 写入字段 replace。 */
  saveShots: (episodeId: string, shots: any[], replace = false) => videoCall("drama.shots.save", { episode_id: episodeId, shots, replace }),
  /** updateShot 走 videoCall("drama.shots.update")。s 传给 videoCall。 */
  updateShot: (s: Record<string, any>) => videoCall("drama.shots.update", s),
  /** deleteShot 走 videoCall("drama.shots.delete")。id 写入字段 id。 */
  deleteShot: (id: string) => videoCall("drama.shots.delete", { id }),
  /** generateShot 走 videoCall("drama.shots.generate")。id 写入字段 id。 */
  generateShot: (id: string) => videoCall("drama.shots.generate", { id }),
  /** generateMissingShots 走 videoCall("drama.shots.generate_missing")。episodeId 写入字段 episode_id。 */
  generateMissingShots: (episodeId: string) => videoCall("drama.shots.generate_missing", { episode_id: episodeId }),
  /** merge 走 videoCall("drama.merge")。episodeId 写入字段 episode_id。shotIds 写入字段 shot_ids。 */
  merge: (episodeId: string, shotIds: string[]) => videoCall("drama.merge", { episode_id: episodeId, shot_ids: shotIds }),
  /** importHuobao 走 videoCall("drama.import")。dbPath 写入字段 db_path。staticDir 写入字段 static_dir。 */
  importHuobao: (dbPath: string, staticDir = "") => videoCall("drama.import", { db_path: dbPath, static_dir: staticDir }),
  /** skipRewrite 走 videoCall("drama.skip_rewrite")。episodeId 写入字段 episode_id。 */
  skipRewrite: (episodeId: string) => videoCall("drama.skip_rewrite", { episode_id: episodeId }),
  /** canvasHttp 走 videoCall("canvas.http")。method 写入字段 method。path 写入字段 path。extra 展开传递。 */
  canvasHttp: (method: string, path: string, extra: Record<string, any> = {}) => videoCall("canvas.http", { method, path, ...extra }),
  /** canvasBind 走 videoCall("canvas.bind")。sessionId 写入字段 session_id。projectId 写入字段 project_id。 */
  canvasBind: (sessionId: string, projectId: string) => videoCall("canvas.bind", { session_id: sessionId, project_id: projectId }),
  /** history 走 videoCall("video.history")。 */
  history: () => videoCall("video.history"),
  /** sessionProject 走 videoCall("video.session.project")。sessionId 写入字段 session_id。 */
  sessionProject: (sessionId: string) => videoCall("video.session.project", { session_id: sessionId }),
  /** mediaUrl 走 videoCall("media.url")。hash 写入字段 hash。 */
  mediaUrl: (hash: string) => videoCall("media.url", { hash }),
  /** fetchTrends 走 videoCall("video.trends.fetch")。platforms 写入字段 platforms。limit 写入字段 limit。 */
  fetchTrends: (platforms = "", limit = 15) => videoCall("video.trends.fetch", { platforms, limit }),
  /** listTopics 走 videoCall("video.topic.list")。status 写入字段 status。 */
  listTopics: (status = "") => videoCall("video.topic.list", { status }),
  /** getTopic 走 videoCall("video.topic.get")。id 写入字段 id。 */
  getTopic: (id: string) => videoCall("video.topic.get", { id }),
  /** saveTopic 走 videoCall("video.topic.save")。 */
  saveTopic: (topic: Record<string, any>) => videoCall("video.topic.save", topic),
  /** deleteTopic 走 videoCall("video.topic.delete")。id 写入字段 id。 */
  deleteTopic: (id: string) => videoCall("video.topic.delete", { id }),
  /** topicFromTrend 走 videoCall("video.topic.from_trend")。platform 写入字段 platform。title 写入字段 title。hot 写入字段 hot。 */
  topicFromTrend: (platform: string, title: string, hot = "") => videoCall("video.topic.from_trend", { platform, title, hot }),
  /** scheduleTopic 走 videoCall("video.topic.schedule")。id 写入字段 id。sessionId 写入字段 session_id。aspect 写入字段 aspect。 */
  scheduleTopic: (id: string, sessionId = "", aspect = "") => videoCall("video.topic.schedule", { id, session_id: sessionId, aspect }),
  /** unscheduleTopic 走 videoCall("video.topic.unschedule")。id 写入字段 id。 */
  unscheduleTopic: (id: string) => videoCall("video.topic.unschedule", { id }),
  /** listStudio 走 videoCall("video.studio.list")。 */
  listStudio: () => videoCall("video.studio.list"),
  /** createStudio 走 videoCall("video.studio.create")。p 传给 videoCall。 */
  createStudio: (p: Record<string, any> = {}) => videoCall("video.studio.create", p),
  /** getStudio 走 videoCall("video.studio.get")。id 写入字段 id。 */
  getStudio: (id: string) => videoCall("video.studio.get", { id }),
  /** updateStudio 走 videoCall("video.studio.update")。p 传给 videoCall。 */
  updateStudio: (p: Record<string, any>) => videoCall("video.studio.update", p),
  /** deleteStudio 走 videoCall("video.studio.delete")。id 写入字段 id。 */
  deleteStudio: (id: string) => videoCall("video.studio.delete", { id }),
  /** studioBind 走 videoCall("video.studio.bind")。sessionId 写入字段 session_id。projectId 写入字段 id、project_id。 */
  studioBind: (sessionId: string, projectId: string) => videoCall("video.studio.bind", { session_id: sessionId, id: projectId, project_id: projectId }),
  /** assembleStudio 走 videoCall("video.studio.assemble")。id 写入字段 id。 */
  assembleStudio: (id: string) => videoCall("video.studio.assemble", { id }),
  /** autoStudio 走 videoCall("video.studio.auto")。id 写入字段 id。 */
  autoStudio: (id: string) => videoCall("video.studio.auto", { id }),
  /** autoTopics 走 videoCall("video.studio.auto.topics")。topicIds 写入字段 topic_ids。 */
  autoTopics: (topicIds: string[]) => videoCall("video.studio.auto.topics", { topic_ids: topicIds }),
  /** listStudioBGM 走 videoCall("video.studio.bgm.list")。 */
  listStudioBGM: () => videoCall("video.studio.bgm.list"),
  /** useStudioBGM 走 videoCall("video.studio.bgm.use")。id 写入字段 id。name 写入字段 name。 */
  useStudioBGM: (id: string, name = "random") => videoCall("video.studio.bgm.use", { id, name }),
  /** clearStudioBGM 走 videoCall("video.studio.bgm.clear")。id 写入字段 id。 */
  clearStudioBGM: (id: string) => videoCall("video.studio.bgm.clear", { id }),
  /** speakStudio 走 videoCall("video.studio.speak")。id 写入字段 id。text 写入字段 text。 */
  speakStudio: (id: string, text = "") => videoCall("video.studio.speak", { id, text }),
  /** studioReframe 走 videoCall("video.studio.ops.reframe")。hash 写入字段 hash。aspect 写入字段 aspect。 */
  studioReframe: (hash: string, aspect = "9:16") => videoCall("video.studio.ops.reframe", { hash, aspect }),
  /** studioBurnin 走 videoCall("video.studio.ops.burnin")。hash 写入字段 hash。subtitleHash 写入字段 subtitle_hash。 */
  studioBurnin: (hash: string, subtitleHash: string) => videoCall("video.studio.ops.burnin", { hash, subtitle_hash: subtitleHash }),
  /** studioSlideshow 走 videoCall("video.studio.ops.slideshow")。hashes 写入字段 hashes。aspect 写入字段 aspect。duration 写入字段 duration。 */
  studioSlideshow: (hashes: string[], aspect = "9:16", duration = 3) => videoCall("video.studio.ops.slideshow", { hashes, aspect, duration }),
  /** studioWrap 走 videoCall("video.studio.ops.wrap")。hash 写入字段 hash。title 写入字段 title。 */
  studioWrap: (hash: string, title: string) => videoCall("video.studio.ops.wrap", { hash, title }),
  /** putStudioMedia 走 videoCall("video.studio.media.put")。dataB64 写入字段 data_b64。mime 写入字段 mime。kind 写入字段 kind。 */
  putStudioMedia: (dataB64: string, mime = "", kind = "") => videoCall("video.studio.media.put", { data_b64: dataB64, mime, kind }),
  /** putStudioMediaPath 走 videoCall("video.studio.media.put")。path 写入字段 path。kind 写入字段 kind。 */
  putStudioMediaPath: (path: string, kind = "video") => videoCall("video.studio.media.put", { path, kind }),
  /** publishPlatforms 走 videoCall("video.studio.publish.platforms")。 */
  publishPlatforms: () => videoCall("video.studio.publish.platforms"),
  /** listPublishes 走 videoCall("video.studio.publish.list")。projectId 写入字段 project_id。platform 写入字段 platform。 */
  listPublishes: (projectId = "", platform = "") => videoCall("video.studio.publish.list", { project_id: projectId, platform }),
  /** draftPublish 走 videoCall("video.studio.publish.draft")。p 传给 videoCall。 */
  draftPublish: (p: Record<string, any>) => videoCall("video.studio.publish.draft", p),
  /** exportPublish 走 videoCall("video.studio.publish.export")。id 写入字段 id。 */
  exportPublish: (id: string) => videoCall("video.studio.publish.export", { id }),
  /** openPublish 走 videoCall("video.studio.publish.open")。id 写入字段 id。 */
  openPublish: (id: string) => videoCall("video.studio.publish.open", { id }),
  /** openPublishAnalytics 走 videoCall("video.studio.publish.analytics")。id 写入字段 id。 */
  openPublishAnalytics: (id: string) => videoCall("video.studio.publish.analytics", { id }),
  /** markPublish 走 videoCall("video.studio.publish.mark")。id 写入字段 id。status 写入字段 status。extra 展开传递。 */
  markPublish: (id: string, status: string, extra: Record<string, any> = {}) => videoCall("video.studio.publish.mark", { id, status, ...extra }),
  /** remotionStatus 走 videoCall("video.studio.remotion.status")。 */
  remotionStatus: () => videoCall("video.studio.remotion.status"),
  /** installRemotion 走 videoCall("video.studio.remotion.install")。 */
  installRemotion: () => videoCall("video.studio.remotion.install"),
  /** personaTemplate 走 videoCall("video.studio.persona.template")。name 写入字段 name。 */
  personaTemplate: (name = "") => videoCall("video.studio.persona.template", { name }),
  /** createPersona 走 videoCall("video.studio.persona.create")。name 写入字段 name。topicId 写入字段 topic_id。 */
  createPersona: (name = "", topicId = "") => videoCall("video.studio.persona.create", { name, topic_id: topicId }),
  /** studioCatalog 走 videoCall("video.studio.catalog")。 */
  studioCatalog: () => videoCall("video.studio.catalog"),
  /** studioPreflight 走 videoCall("video.studio.preflight")。id 写入字段 id。gate 写入字段 gate。 */
  studioPreflight: (id: string, gate = "") => videoCall("video.studio.preflight", { id, gate }),
  /** approveStudio 走 videoCall("video.studio.approve")。id 写入字段 id。 */
  approveStudio: (id: string) => videoCall("video.studio.approve", { id }),
  /** skipStudioAudio 走 videoCall("video.studio.skip_audio")。id 写入字段 id。 */
  skipStudioAudio: (id: string) => videoCall("video.studio.skip_audio", { id }),
  /** applyStudioPreset 走 videoCall("video.studio.preset")。id 写入字段 id。presetId 写入字段 preset_id。 */
  applyStudioPreset: (id: string, presetId: string) => videoCall("video.studio.preset", { id, preset_id: presetId }),
  /** applyStudioTemplate 走 videoCall("video.studio.template")。id 写入字段 id。templateId 写入字段 template_id。 */
  applyStudioTemplate: (id: string, templateId: string) => videoCall("video.studio.template", { id, template_id: templateId }),
  /** setStudioIdentity 走 videoCall("video.studio.identity")。id 写入字段 id。characterId 写入字段 character_id。styleId 写入字段 style_id。 */
  setStudioIdentity: (id: string, characterId: string, styleId: string) => videoCall("video.studio.identity", { id, character_id: characterId, style_id: styleId }),
  /** generateStudioShot 走 videoCall("video.studio.generate.shot")。id 写入字段 id。shotId 写入字段 shot_id。 */
  generateStudioShot: (id: string, shotId = "") => videoCall("video.studio.generate.shot", { id, shot_id: shotId }),
  /** generateStudioMissing 走 videoCall("video.studio.generate.missing")。id 写入字段 id。 */
  generateStudioMissing: (id: string) => videoCall("video.studio.generate.missing", { id }),
  /** generateStudioCover 走 videoCall("video.studio.cover")。id 写入字段 id。 */
  generateStudioCover: (id: string) => videoCall("video.studio.cover", { id }),
  /** pickStudioCover 走 videoCall("video.studio.cover.pick")。id 写入字段 id。hash 写入字段 hash。 */
  pickStudioCover: (id: string, hash: string) => videoCall("video.studio.cover.pick", { id, hash }),
  /** writeStudioCopy 走 videoCall("video.studio.copy")。id 写入字段 id。platform 写入字段 platform。 */
  writeStudioCopy: (id: string, platform = "") => videoCall("video.studio.copy", { id, platform }),
  /** packageStudio 走 videoCall("video.studio.package")。id 写入字段 id。 */
  packageStudio: (id: string) => videoCall("video.studio.package", { id }),
  /** restoreStudio 走 videoCall("video.studio.restore")。id 写入字段 id。n 写入字段 n。 */
  restoreStudio: (id: string, n: number) => videoCall("video.studio.restore", { id, n }),
  /** studioTiming 走 videoCall("video.studio.timing")。id 写入字段 id。 */
  studioTiming: (id: string) => videoCall("video.studio.timing", { id }),
  /** ingestStudio 走 videoCall("video.studio.ingest")。id 写入字段 id。url 写入字段 url。 */
  ingestStudio: (id: string, url: string) => videoCall("video.studio.ingest", { id, url }),
  /** autoClip 走 videoCall("video.studio.autoclip")。hash 写入字段 hash。count 写入字段 count。duration 写入字段 duration。 */
  autoClip: (hash: string, count = 3, duration = "30-60") => videoCall("video.studio.autoclip", { hash, count, duration }),
  /** studioJobs 走 videoCall("video.studio.jobs")。id 写入字段 id。 */
  studioJobs: (id: string) => videoCall("video.studio.jobs", { id }),
  /** expandTopic 走 videoCall("video.topic.expand")。id 写入字段 id。 */
  expandTopic: (id: string) => videoCall("video.topic.expand", { id }),
  /** chooseTopicOption 走 videoCall("video.topic.choose")。id 写入字段 id。optionId 写入字段 option_id。 */
  chooseTopicOption: (id: string, optionId: string) => videoCall("video.topic.choose", { id, option_id: optionId }),
  /** cutList 走 videoCall("video.cut.list")。 */
  cutList: () => videoCall("video.cut.list"),
  /** cutOpen 走 videoCall("video.cut.open")。p 传给 videoCall。 */
  cutOpen: (p: { title?: string; hash?: string; resource_id?: string; session_id?: string } = {}) =>
    videoCall("video.cut.open", p),
  /** cutCreate 走 videoCall("video.cut.create")。title 写入字段 title。hash 写入字段 hash。sessionId 写入字段 session_id。 */
  cutCreate: (title = "", hash = "", sessionId = "") => videoCall("video.cut.create", { title, hash, session_id: sessionId }),
  /** remakeDraft 走 videoCall("video.remake.draft")。hash 写入字段 hash。title 写入字段 title。 */
  remakeDraft: (hash: string, title = "") => videoCall("video.remake.draft", { hash, title }),
  /** remakeReplace 走 videoCall("video.remake.replace")。id 写入字段 id。hash 写入字段 hash。 */
  remakeReplace: (id: string, hash: string) => videoCall("video.remake.replace", { id, hash }),
  /** filmOpen 走 videoCall("video.film.open")。p 传给 videoCall。 */
  filmOpen: (p: { hash?: string; title?: string; structure?: string; description?: string; url?: string }) => videoCall("video.film.open", p),
  /** filmDescribe 走 videoCall("video.film.describe")。structure 写入字段 structure。title 写入字段 title。description 写入字段 description。 */
  filmDescribe: (structure: string, title = "", description = "") => videoCall("video.film.describe", { structure, title, description }),
  /** filmGet 走 videoCall("video.film.get")。id 写入字段 id。 */
  filmGet: (id: string) => videoCall("video.film.get", { id }),
  /** filmList 走 videoCall("video.film.list")。 */
  filmList: () => videoCall("video.film.list"),
  /** filmPlan 走 videoCall("video.film.plan")。id 写入字段 id。 */
  filmPlan: (id: string) => videoCall("video.film.plan", { id }),
  /** filmBuild 走 videoCall("video.film.build")。id 写入字段 id。 */
  filmBuild: (id: string) => videoCall("video.film.build", { id }),
  /** filmSwap 走 videoCall("video.film.swap")。id 写入字段 id。patch 展开传递。 */
  filmSwap: (id: string, patch: { host?: string; product?: string; language?: string; aspect?: string; hook?: string; price?: string; cta?: string }) => videoCall("video.film.swap", { id, ...patch }),
  /** filmFork 走 videoCall("video.film.fork")。id 写入字段 id。patch 展开传递。 */
  filmFork: (id: string, patch: { host?: string; product?: string; language?: string; aspect?: string; hook?: string; price?: string; cta?: string }) => videoCall("video.film.fork", { id, ...patch }),
  /** filmBed 走 videoCall("video.film.bed")。id 写入字段 id。hash 写入字段 hash。 */
  filmBed: (id: string, hash: string) => videoCall("video.film.bed", { id, hash }),
  /** filmRewrite 走 videoCall("video.film.rewrite")。id 写入字段 id。lineId 写入字段 line_id。text 写入字段 text。 */
  filmRewrite: (id: string, lineId: string, text: string) => videoCall("video.film.rewrite", { id, line_id: lineId, text }),
  /** filmHandoff 走 videoCall("video.film.handoff")。id 写入字段 id。title 写入字段 title。 */
  filmHandoff: (id: string, title = "") => videoCall("video.film.handoff", { id, title }),
  /** cutGet 走 videoCall("video.cut.get")。id 写入字段 id。 */
  cutGet: (id: string) => videoCall("video.cut.get", { id }),
  /** cutBind 走 videoCall("video.cut.bind")。sessionId 写入字段 session_id。id 写入字段 id。 */
  cutBind: (sessionId: string, id: string) => videoCall("video.cut.bind", { session_id: sessionId, id }),
  /** cutRelease 走 videoCall("video.cut.release")。sessionId 写入字段 session_id。 */
  cutRelease: (sessionId: string) => videoCall("video.cut.release", { session_id: sessionId }),
  /** cutSession 走 videoCall("video.cut.session")。sessionId 写入字段 session_id。 */
  cutSession: (sessionId: string) => videoCall("video.cut.session", { session_id: sessionId }),
  /** cutIngest 走 videoCall("video.cut.ingest")。id 写入字段 id。hash 写入字段 hash。title 写入字段 title。 */
  cutIngest: (id: string, hash: string, title = "") => videoCall("video.cut.ingest", { id, hash, title }),
  /** cutAnalyze 走 videoCall("video.cut.analyze")。id 写入字段 id。 */
  cutAnalyze: (id: string) => videoCall("video.cut.analyze", { id }),
  /** cutPropose 走 videoCall("video.cut.propose")。id 写入字段 id。count 写入字段 count。window 写入字段 window。 */
  cutPropose: (id: string, count = 3, window = "15-30") => videoCall("video.cut.propose", { id, count, window }),
  /** cutApply 走 videoCall("video.cut.apply")。id 写入字段 id。 */
  cutApply: (id: string) => videoCall("video.cut.apply", { id }),
  /** cutExtract 走 videoCall("video.cut.extract")。id 写入字段 id。 */
  cutExtract: (id: string) => videoCall("video.cut.extract", { id }),
  /** cutEnhance 走 videoCall("video.cut.enhance")。id 写入字段 id。preset 写入字段 preset。 */
  cutEnhance: (id: string, preset = "voice") => videoCall("video.cut.enhance", { id, preset }),
  /** cutMix 走 videoCall("video.cut.mix")。id 写入字段 id。duck 写入字段 duck。 */
  cutMix: (id: string, duck = true) => videoCall("video.cut.mix", { id, duck }),
  /** cutTranscribe 走 videoCall("video.cut.transcribe")。id 写入字段 id。 */
  cutTranscribe: (id: string) => videoCall("video.cut.transcribe", { id }),
  /** cutCaptions 走 videoCall("video.cut.captions")。id 写入字段 id。text 写入字段 text。language 写入字段 language。 */
  cutCaptions: (id: string, text = "", language = "") => videoCall("video.cut.captions", { id, text, language }),
  /** cutBurnin 走 videoCall("video.cut.burnin")。id 写入字段 id。 */
  cutBurnin: (id: string) => videoCall("video.cut.burnin", { id }),
  /** cutDub 走 videoCall("video.cut.dub")。id 写入字段 id。text 写入字段 text。voice 写入字段 voice。 */
  cutDub: (id: string, text = "", voice = "") => videoCall("video.cut.dub", { id, text, voice }),
  /** cutGrade 走 videoCall("video.cut.grade")。id 写入字段 id。preset 写入字段 preset。 */
  cutGrade: (id: string, preset = "warm") => videoCall("video.cut.grade", { id, preset }),
  /** cutStem 走 videoCall("video.cut.stem")。id 写入字段 id。kind 写入字段 kind。hash 写入字段 hash。 */
  cutStem: (id: string, kind: string, hash: string) => videoCall("video.cut.stem", { id, kind, hash }),
  /** cutKeep 走 videoCall("video.cut.keep")。id 写入字段 id。clipIds 写入字段 clip_ids。 */
  cutKeep: (id: string, clipIds: string[]) => videoCall("video.cut.keep", { id, clip_ids: clipIds }),
  /** cutOmit 走 videoCall("video.cut.omit")。id 写入字段 id。index 写入字段 index。 */
  cutOmit: (id: string, index: number) => videoCall("video.cut.omit", { id, index }),
  /** route 走 videoCall("video.route")。sessionId 写入字段 session_id。dest 写入字段 dest。id 写入字段 id。title 写入字段 title。 */
  route: (sessionId: string, dest: string, id = "", title = "") => videoCall("video.route", { session_id: sessionId, dest, id, title }),
  /** providerMenu 走 videoCall("video.providers.menu")。capability 写入字段 capability。 */
  providerMenu: (capability = "") => videoCall("video.providers.menu", { capability }),
};

/** imCall 优先走桌面绑定，生成模块没有这个方法时改走 Call.ByName，浏览器门再走 /api。 */
async function imCall(method: string, args: any[], path: string, init?: RequestInit): Promise<any> {
  const s = await wailsService();
  const fn = svcMethod(s, method);
  if (fn) return fn(...args);
  if (nativeDesktop()) {
    const alt = namedDesktopService();
    return alt[method](...args);
  }
  return http(path, init);
}

/** imSnapshot 拉取消息目录、规则、账号和状态。 */
export function imSnapshot(): Promise<any> {
  return imCall("IMSnapshot", [], "/api/im");
}

/** imSaveSettings 保存共用的唤醒和监听规则。settings 放进请求体。 */
export function imSaveSettings(settings: any): Promise<any> {
  return imCall("IMSaveSettings", [settings], "/api/im/settings", {
    method: "PUT",
    body: JSON.stringify(settings),
  });
}

/** imSaveAccount 保存一个机器人。空的密钥字段表示保留原值。 */
export function imSaveAccount(account: any, secrets: Record<string, string>): Promise<any> {
  return imCall("IMSaveAccount", [account, secrets], "/api/im/accounts", {
    method: "PUT",
    body: JSON.stringify({ account, secrets }),
  });
}

/** imDeleteAccount 删除机器人并清掉它的密钥。 */
export function imDeleteAccount(id: string): Promise<any> {
  return imCall("IMDeleteAccount", [id], `/api/im/accounts/${encodeURIComponent(id)}`, { method: "DELETE" });
}

/** imSend 用一个已启用的机器人主动发文本。 */
export function imSend(accountId: string, chatId: string, text: string): Promise<any> {
  return imCall("IMSend", [accountId, chatId, text], "/api/im/send", {
    method: "POST",
    body: JSON.stringify({ account_id: accountId, chat_id: chatId, text }),
  });
}
