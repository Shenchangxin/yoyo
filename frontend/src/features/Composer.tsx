/**
 * 拥有：输入框、提及、斜杠和附件，以及创作桌把视频送进剪辑。
 * 不拥有：消息怎么发出去，那由调用方传入 onSend。选文件、导入成片和读取配置档会调用 api。
 */

import { useEffect, useMemo, useRef, useState, type ReactNode, type RefObject } from "react";
import { BookOpen, Check, ChevronDown, Folder, GitBranch, Plus, X } from "lucide-react";
import { ProfileChip } from "./ProfileChip";
import { Tooltip } from "../components/ui/tooltip";
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from "../components/ui/select";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../components/ui/dropdown-menu";
import { THREAD_COL, THREAD_GUTTER, THREAD_GUTTER_COMPACT } from "../lib/thread";
import { cn } from "../lib/utils";
import { displayTitle, displayWorkspace } from "../lib/display-title";
import { pathReady } from "../lib/workspace";
import { toast } from "sonner";
import { useCopy } from "../lib/i18n";
import { desktopWindow } from "../lib/chrome";
import type { Copy } from "../lib/copy";
import { useUI } from "../lib/store";
import { filterSlash, slashCatalog, slashQuery } from "../lib/slash";
import type { Approval, Attachment, AuthMode, ChatProvider, ContextUsage, FileHit, Profile, SkillInfo } from "../lib/protocol";
import type { TurnStatus } from "../lib/turn-status";
import { profileOf } from "../lib/profile-label";
import * as api from "../lib/client";
import type { TaskPlan } from "../lib/plan";
import { lookupCatalogModel, composerModelIds, composerModelGroups, decodeChatModel, encodeChatModel, resolveComposerModelValue, resolveContextWindow, UNKNOWN_MODEL_WINDOW } from "../lib/models-dev";
import { useModelCatalog } from "../lib/model-catalog";
import { estimateTokens } from "../lib/context-usage";
import { ChatBorder } from "./session-chrome/ChatBorder";
import { VideoModeSwitch } from "./video/VideoModeSwitch";
import { DramaProjectChip } from "./video/DramaProjectChip";
import { CanvasProjectChip } from "./video/CanvasProjectChip";
import { StudioDeskChip } from "./video/StudioDeskChip";
import {
  childPaths,
  composeDraft,
  formatTriggerPath,
  isCompleteMention,
  keepMentionOpen,
  mentionCrumbs,
  mentionQueryOf,
  mentionTokenAt,
  mentionableSkills,
  mergeMentionPins,
  overlayActiveIndex,
  peelCompletedMentions,
  pinFromToken,
  settleBareAtPaths,
  suggestSkills,
  type MentionKind,
  type MentionPin,
  type MentionSessionRef,
} from "../lib/mentions";
import "./session-chrome/composer.css";
import { QueueDock } from "./session-chrome/QueueDock";
import { TodoPanel } from "./session-chrome/TodoPanel";
import { ContextMeter } from "./session-chrome/ContextMeter";
import { StatsPills } from "./session-chrome/StatsPills";
import { ApprovalTakeover } from "./session-chrome/ApprovalTakeover";
import { DraftEditor, type DraftEditorHandle, type DraftSnapshot } from "./session-chrome/DraftEditor";
import { CommandOverlay, type OverlayItem } from "./session-chrome/CommandOverlay";
import { AttachmentRail, DropOverlay } from "./session-chrome/Attachments";
import { isVideoPath, nativeFilePath } from "./video/ingest-clip";
import { useCutDesk } from "./video/cut-desk";
import { useCreateMode } from "./video/create-mode";
import { SendSling } from "../components/SendSling";
import { SendGlyph } from "./session-chrome/PromptBar";
import { StopGlyph } from "./session-chrome/icons";
import { PermissionSelect } from "./session-chrome/PermissionSelect";
import { WorkspaceChip } from "./session-chrome/WorkspaceChip";
import { WorkspacePicker } from "./session-chrome/WorkspacePicker";
import { TurnStatusBar } from "./transcript/TurnStatusBar";
import type { SessionStats } from "../lib/session-stats";
import { PresencePerch } from "./presence";

/**
 * Composer 在短剧桌且正在跑时不能发送，除非是改道。没有正文、芯片和附件时，只有 allowEmptySend 才能发。需要工作区但还没有路径时整框禁用。
 * props 读取 draftKey、compact、onBrowseWorkspace、workspace、disabled、hero 等，调用 onSearchFiles，调用 onSlash，调用 onIngestVideo，调用 onPickFiles，调用 onScreenshot，调用 onClipboard，调用 onSend，调用 onWorkspace，调用 onBrowseWorkspace，调用 onAuthMode，调用 onIsolate，调用 onApplyWorktree，调用 onModel，调用 onStop。
 */
export function Composer(props: {
  draftKey: string;
  running: boolean;
  disabled: boolean;
  disabledReason?: string;
  model?: string;
  models?: string[];
  provider?: string;
  connectionId?: string;
  chatProviders?: ChatProvider[];
  ctx?: ContextUsage;
  queued?: number;
  queueItems?: { id?: string; text?: string; plan?: boolean }[];
  onQueueCancel?: (id: string) => void;
  onQueueReorder?: (id: string, delta: number) => void;
  onApplyWorktree?: () => void;
  skills?: SkillInfo[];
  files?: FileHit[];
  sessions?: MentionSessionRef[];
  onSearchFiles?: (q: string) => void;
  onSend: (opts?: { steer?: boolean; attachments?: Attachment[]; text?: string }) => void;
  onStop: () => void;
  onSlash?: (cmd: string, rest: string) => void;
  onModel?: (model: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
  onIngestVideo?: (source: { path?: string; file?: File }) => Promise<void>;
  onClipboard?: () => Promise<string>;
  onScreenshot?: () => Promise<string>;
  compact?: boolean;
  flush?: boolean;
  hero?: boolean;
  authMode?: AuthMode | string;
  onAuthMode?: (mode: AuthMode) => void;
  workspace?: string;
  workspaces?: string[];
  isolate?: boolean;
  onWorkspace?: (path: string) => void;
  onBrowseWorkspace?: () => void;
  onIsolate?: (isolate: boolean) => void;
  taskPlan?: TaskPlan | null;
  ask?: Approval | null;
  approvals?: Approval[];
  onResolveAsk?: (id: string, decision: string, answer?: string) => void;
  methodologyActive?: boolean;
  autoSkills?: { name: string; loaded?: boolean }[];
  onBuildPlan?: () => void;
  sessionId?: string;
  profileId?: string;
  onProfile?: (id: string) => void;
  onSavePage?: () => void;
  onFollow?: (in_: string) => void;
  onCall?: () => void;
  turnStatus?: TurnStatus | null;
  sessionStats?: SessionStats | null;
  leading?: ReactNode;
  toolbarLeading?: ReactNode;
  toolbarTrailing?: ReactNode;
  placeholder?: string;
  sendLabel?: string;
  allowEmptySend?: boolean;
  inputTestId?: string;
  perch?: boolean;
  perchSize?: number;
}) {
  const value = useUI((s) => s.drafts[props.draftKey] || "");
  const plan = useUI((s) => s.plan);
  const setDraft = useUI((s) => s.setDraft);
  const setPlan = useUI((s) => s.setPlan);
  /** 把正文写回这条草稿。v 传给 setDraft。 */
  const onChange = (v: string) => setDraft(props.draftKey, v);
  const [hint, setHint] = useState<"plus" | null>(null);
  const [hiToken, setHiToken] = useState("");
  const overlayQueryRef = useRef("");
  const [focused, setFocused] = useState(false);
  const [atts, setAtts] = useState<Attachment[]>([]);
  const [chips, setChips] = useState<MentionPin[]>([]);
  const [dismissedSuggest, setDismissedSuggest] = useState<string[]>([]);
  const [dismissedAuto, setDismissedAuto] = useState<string[]>([]);
  const [fileQ, setFileQ] = useState<string | null>(null);
  const [drilled, setDrilled] = useState(false);
  const [caret, setCaret] = useState(0);
  const [before, setBefore] = useState("");
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [drop, setDrop] = useState(false);
  const [attaching, setAttaching] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const editorRef = useRef<DraftEditorHandle>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const chipRef = useRef<HTMLButtonElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const copy = useCopy();
  const surface = useUI((s) => s.surface);
  const videoMode = useUI((s) => s.videoMode);
  const studioDesk = useUI((s) => s.studioDesk);
  const videoPane = useUI((s) => s.videoPane);
  const videoSurface = surface === "video";
  const dramaDesk = videoPane === "drama";
  const videoing = videoSurface && !props.compact && !dramaDesk;
  const createDesk = videoPane === "create";
  const cutBound = useCutDesk((s) => !!s.project?.source_hash);
  const createHome = createDesk && !cutBound;
  const createMode = useCreateMode((s) => s.mode);
  const talkDesk = createDesk;
  const copilotDesk = studioDesk;
  const needWorkspace = !copilotDesk && !talkDesk && !!props.onBrowseWorkspace && !pathReady(props.workspace);
  const disabled = props.disabled || needWorkspace;
  const workspaceTrigger = needWorkspace && !props.disabled;
  const showWorkspaceChip = !!props.hero && !copilotDesk && !talkDesk && !dramaDesk && !!props.onBrowseWorkspace;
  const uniqueWorkspaces = [...new Set([
    ...(pathReady(props.workspace) ? [props.workspace as string] : []),
    ...(props.workspaces || []).filter((p) => pathReady(p)),
  ])];
  const placeholder = props.placeholder
    || (props.disabled
      ? (props.disabledReason || copy.composer.disabled)
      : needWorkspace
        ? copy.composer.placeholderWorkspace
      : studioDesk
        ? copy.composer.placeholderStudioDesk
      : videoPane === "create"
        ? (cutBound ? copy.composer.placeholderConductorCut : copy.composer.placeholderConductor)
      : videoSurface
        ? (videoMode === "canvas" ? copy.composer.placeholderCanvas : videoMode === "creative" ? copy.composer.placeholderCreative : copy.composer.placeholderDrama)
        : props.hero
          ? copy.composer.placeholderHero
          : copy.composer.placeholder);
  const hasPayload = !!value.trim() || chips.length > 0 || atts.length > 0;
  const canSend = !disabled && !attaching && (hasPayload || !!props.allowEmptySend) && !(dramaDesk && props.running);
  const orbStops = !!props.running && !hasPayload;
  const canSteerQueue = !disabled && !hasPayload && props.running && (props.queueItems?.length || 0) > 0;
  const modeMenu = !copilotDesk && !talkDesk && !dramaDesk && (!videoing || !!props.onIsolate);
  const currentModel = (props.model || "").trim();
  const catalog = useModelCatalog((s) => s.catalog);
  const groups = composerModelGroups(props.chatProviders, currentModel, props.connectionId);
  const grouped = groups.some((g) => g.models.length > 0);
  const modelOptions = grouped ? [] : composerModelIds(catalog, props.provider, currentModel, props.models);
  const groupedValue = grouped ? resolveComposerModelValue(props.chatProviders, currentModel, props.connectionId) : "";
  const selectedGroup = grouped ? groups.find((g) => g.providerId === decodeChatModel(groupedValue).providerId) : undefined;
  const catalogProvider = selectedGroup?.vendor || props.provider || "openai";
  const meta = lookupCatalogModel(catalog, catalogProvider, currentModel);
  const fallbackWindow = resolveContextWindow(catalog, catalogProvider, currentModel) || UNKNOWN_MODEL_WINDOW;
  const modelLabel = meta?.model.name || currentModel;
  const caretText = before;
  const slashPrefix = slashQuery(caretText);
  const mentionHit = mentionTokenAt(caretText, caretText.length);
  const mentionParsed = mentionQueryOf(mentionHit?.token || "");
  const mentionOpen = !!mentionHit && !(mentionParsed.kind === "harness" && isCompleteMention(mentionHit.token));
  const overlayKind: "plus" | "slash" | "mention" | null = slashPrefix
    ? "slash"
    : mentionOpen
      ? "mention"
      : hint === "plus"
        ? "plus"
        : null;
  const slashItems = slashPrefix ? filterSlash(slashPrefix, slashCatalog(copy)) : [];
  const mentionItems = useMemo(() => {
    const token = mentionHit?.token || "";
    const parsed = mentionQueryOf(token);
    if (parsed.kind === "harness") return [];
    if (parsed.kind === "skill") {
      return mentionableSkills(props.skills || [], parsed.query).map((s) => ({
        token: `@skill:${s.name}`,
        label: s.displayName || s.name,
        alias: s.displayName && s.displayName.toLowerCase() !== s.name.toLowerCase() ? s.name : undefined,
        hint: s.description || "",
        kind: "skill" as MentionKind,
        section: copy.composer.sectionSkills,
      }));
    }
    const q = parsed.query;
    const dirListing = q.endsWith("/");
    const dir = dirListing ? q.replace(/\/+$/, "") : "";
    let files = props.files || [];
    if (parsed.kind === "folder") files = files.filter((f) => f.kind === "dir");
    if (parsed.kind === "file") files = files.filter((f) => f.kind !== "dir");
    if (dirListing && dir) files = childPaths(files, dir);
    const withLocation = !(drilled && dirListing);
    const fileRows: OverlayItem[] = files.map((f) => {
      const directory = f.kind === "dir";
      const slash = f.path.lastIndexOf("/");
      const name = slash < 0 ? f.path : f.path.slice(slash + 1);
      const parent = slash < 0 ? "" : f.path.slice(0, slash);
      return {
        token: directory ? `@folder:${f.path}` : `@file:${f.path}`,
        label: `${name}${directory ? "/" : ""}`,
        hint: withLocation && parent ? parent : "",
        kind: directory ? "folder" : "file",
        section: copy.composer.sectionFiles,
        drill: directory,
        drillPath: directory ? f.path : undefined,
      };
    });
    if (parsed.kind === "file" || parsed.kind === "folder" || parsed.quoted) return fileRows;
    const needle = q.trim().toLowerCase();
    const sessionRows: OverlayItem[] = (props.sessions || [])
      .filter((s) => s.id !== props.sessionId)
      .filter((s) => !needle || displayTitle(s.title, s.id).toLowerCase().includes(needle))
      .map((s) => {
        const child = !!s.parentId && s.parentId === props.sessionId;
        const otherWs = s.workspace && s.workspace !== props.workspace ? displayWorkspace(s.workspace, s.workspace) : "";
        return {
          token: `@session:${s.id}`,
          label: displayTitle(s.title, copy.rail.untitled),
          hint: otherWs,
          kind: "session" as MentionKind,
          section: child ? copy.composer.sectionSubagents : copy.composer.sectionSessions,
        };
      })
      .sort((a, b) => Number(a.section === copy.composer.sectionSessions) - Number(b.section === copy.composer.sectionSessions))
      .slice(0, 8);
    if (parsed.kind === "session") return sessionRows;
    return [...fileRows, ...sessionRows];
  }, [mentionHit?.token, drilled, props.files, props.sessions, props.sessionId, props.workspace, copy, props.skills]);

  const plusItems: OverlayItem[] = createDesk
    ? [{ token: "__import_clip__", label: copy.composer.importClip, hint: copy.composer.importClipHint, kind: "attach", section: copy.composer.sectionAdd }]
    : [
    { token: "__file__", label: copy.composer.attach, hint: "", kind: "attach", section: copy.composer.sectionAdd },
    ...(props.onClipboard ? [{ token: "__clipboard__", label: copy.composer.clipboard, hint: "", kind: "clipboard" as const, section: copy.composer.sectionAdd }] : []),
    ...(props.onScreenshot && desktopWindow() ? [{ token: "__screenshot__", label: copy.composer.screenshot, hint: "", kind: "screenshot" as const, section: copy.composer.sectionAdd }] : []),
    ...slashCatalog(copy).map((s) => ({
      token: s.cmd,
      label: s.cmd,
      hint: s.hint,
      kind: "command" as const,
      section: copy.composer.commands,
    })),
  ];
  const slashPopup: OverlayItem[] = [
    ...slashItems.map((s) => ({ token: s.cmd, label: s.cmd, hint: s.hint, kind: "command" as const, section: copy.composer.commands })),
    ...mentionableSkills(props.skills || [], (slashPrefix || "").slice(1)).map((s) => ({
      token: `@skill:${s.name}`,
      label: s.displayName || s.name,
      alias: s.displayName && s.displayName.toLowerCase() !== s.name.toLowerCase() ? s.name : undefined,
      hint: s.description || "",
      kind: "skill" as const,
      section: copy.composer.sectionSkills,
    })),
  ];
  const popup: OverlayItem[] = overlayKind === "plus"
    ? plusItems
    : overlayKind === "slash"
      ? slashPopup
      : overlayKind === "mention"
        ? mentionItems
        : [];
  const hi = overlayActiveIndex(popup, hiToken);
  const mentionTrail = overlayKind === "mention"
    ? mentionCrumbs(mentionParsed.query, mentionParsed.quoted, drilled, copy.composer.crumbWorkspace)
    : undefined;
  const pinnedSkillNames = useMemo(() => {
    const names = new Set<string>();
    for (const c of chips) {
      if (c.kind === "skill") names.add(c.token.replace(/^@skill:/, ""));
    }
    for (const m of value.matchAll(/@skill:(\S+)/g)) names.add(m[1]);
    return names;
  }, [chips, value]);
  const suggestedSkills = useMemo(() => {
    if (overlayKind || disabled) return [];
    const prose = value.replace(/@(?:file|folder|skill|session):\S+|@harness\b/g, " ").trim();
    if (prose.length < 2) return [];
    return suggestSkills(props.skills || [], prose).filter((s) => !pinnedSkillNames.has(s.name) && !dismissedSuggest.includes(s.name));
  }, [overlayKind, disabled, props.skills, value, pinnedSkillNames, dismissedSuggest]);
  const autoSkillChips = useMemo(
    () => (props.autoSkills || []).filter((s) => s.name && !dismissedAuto.includes(s.name) && !pinnedSkillNames.has(s.name)),
    [props.autoSkills, dismissedAuto, pinnedSkillNames],
  );
  const showPopup = overlayKind === "mention" || (!!overlayKind && popup.length > 0);
  const toolApproval = (props.approvals || []).find((a) => a.action !== "ask_user" && (!props.sessionId || !a.sessionId || a.sessionId === props.sessionId));
  const takeover = (props.ask && props.onResolveAsk) || (toolApproval && props.onResolveAsk);

  useEffect(() => {
    setHint(null);
    setHiToken("");
    setBefore("");
    setAtts([]);
    setChips([]);
    setDismissedSuggest([]);
    setDismissedAuto([]);
    setFileQ(null);
    setDrilled(false);
    overlayQueryRef.current = "";
  }, [props.draftKey]);

  useEffect(() => {
    const q = overlayKind === "slash" ? slashPrefix || "" : overlayKind === "mention" ? (mentionHit?.token || "") : overlayKind || "";
    if (q === overlayQueryRef.current) return;
    overlayQueryRef.current = q;
    setHiToken("");
  }, [overlayKind, slashPrefix, mentionHit?.token]);

  useEffect(() => {
    if (hint !== "plus") return;
    /** 点在卡片外面就关掉加号浮层。卡片里面的点击留着。e 读取 target。 */
    const onDoc = (e: PointerEvent) => {
      const t = e.target as Node;
      if (cardRef.current?.contains(t)) return;
      setHint(null);
    };
    document.addEventListener("pointerdown", onDoc, true);
    return () => document.removeEventListener("pointerdown", onDoc, true);
  }, [hint]);

  useEffect(() => {
    if (!props.onProfile) return;
    void api.profilesList().then((rows) => setProfiles((rows || []).map(profileOf))).catch(() => setProfiles([]));
  }, [props.onProfile]);

  useEffect(() => {
    const ed = editorRef.current;
    if (!ed) return;
    const snap = ed.snapshot();
    if (snap.text === value) return;
    ed.setPlain(value);
    const next = ed.snapshot();
    setChips(next.chips);
    setCaret(next.caret);
    setBefore(next.before);
    if (!value.trim()) return;
    ed.focus();
    cardRef.current?.scrollIntoView({ block: "center", behavior: "smooth" });
  }, [value]);

  /** 斜杠出现时关掉加号。完整的 harness 提及不再下钻。文件和文件夹查询变了才去搜。snap 读取 caret、before。 */
  function applyCaret(snap: DraftSnapshot) {
    setCaret(snap.caret);
    setBefore(snap.before);
    const slash = slashQuery(snap.before);
    if (slash) {
      if (hint === "plus") setHint(null);
      return;
    }
    const tokenHit = mentionTokenAt(snap.before, snap.before.length);
    if (tokenHit) {
      const parsed = mentionQueryOf(tokenHit.token);
      if (parsed.kind === "harness" && isCompleteMention(tokenHit.token)) {
        setDrilled(false);
        return;
      }
      if (hint === "plus") setHint(null);
      if (parsed.kind !== "skill" && parsed.kind !== "harness" && parsed.kind !== "session") {
        const q = parsed.query;
        if (q !== fileQ) {
          setFileQ(q);
          props.onSearchFiles?.(q);
        }
      }
      return;
    }
    setDrilled(false);
  }

  /** 把芯片和正文写回草稿，再按光标更新浮层。snap 读取 chips、text，传给 applyCaret。 */
  function applySnapshot(snap: DraftSnapshot) {
    setChips(snap.chips);
    onChange(snap.text);
    applyCaret(snap);
  }

  /**
   * 还要继续选时只替换触发词。完整提及收成芯片。其余当成普通文字再加一个空格。
   * token 与 @file:、@folder: 比较是否相等，加上  ，看结尾，作为分支结果，传给 keepMentionOpen，传给 isCompleteMention，传给 pinFromToken。
   * label 传给 pinFromToken。
   */
  function insertMention(token: string, label?: string) {
    if (keepMentionOpen(token)) {
      editorRef.current?.replaceTrigger(token.endsWith(":") ? token : token);
      if (token === "@file:" || token === "@folder:") {
        setFileQ("");
        props.onSearchFiles?.("");
      }
      editorRef.current?.focus();
      return;
    }
    if (isCompleteMention(token)) {
      editorRef.current?.replaceMention(pinFromToken(token, label));
      setHint(null);
      setHiToken("");
      setDrilled(false);
      editorRef.current?.focus();
      return;
    }
    editorRef.current?.insertText(token + " ");
    setHint(null);
    setDrilled(false);
    editorRef.current?.focus();
  }

  /** 没有路径就返回。否则把触发词换成这个目录，并按目录加斜杠去搜。item 读取 drillPath、token。 */
  function drillMention(item: OverlayItem) {
    const path = item.drillPath || pinFromToken(item.token).detail;
    if (!path) return;
    const parsed = mentionQueryOf(mentionHit?.token || "@");
    setDrilled(true);
    editorRef.current?.replaceTrigger(formatTriggerPath(path, parsed.quoted, true));
    setFileQ(`${path}/`);
    props.onSearchFiles?.(`${path}/`);
    setHiToken("");
    editorRef.current?.focus();
  }

  /** 只有单独的 @ 或 @" 时不算下钻。text 与 @、@\" 比较是否不等，传给 editorRef.current.replaceTrigger，传给 mentionQueryOf。 */
  function applyTrigger(text: string) {
    setDrilled(text !== "@" && text !== "@\"");
    editorRef.current?.replaceTrigger(text);
    const parsed = mentionQueryOf(text);
    setFileQ(parsed.query);
    props.onSearchFiles?.(parsed.query);
    setHiToken("");
    editorRef.current?.focus();
  }

  /** 清空输入，把命令交给 onSlash，剩余参数是空字符串。cmd 传给 props.onSlash。 */
  function applySlash(cmd: string) {
    editorRef.current?.setPlain("");
    onChange("");
    setHint(null);
    props.onSlash?.(cmd, "");
  }

  /** __import_clip__ 或创作桌走导入。__file__ 走选文件。剪贴板和截图各走自己的回调。@ 开头插入提及，其余当斜杠命令。item 读取 token、label。 */
  function pickOverlay(item: OverlayItem) {
    if (item.token === "__file__" || item.token === "__import_clip__") {
      setHint(null);
      if (item.token === "__import_clip__" || createDesk) void importClip();
      else void pickAttachments();
      return;
    }
    if (item.token === "__clipboard__") {
      setHint(null);
      void pasteOsClipboard();
      return;
    }
    if (item.token === "__screenshot__") {
      setHint(null);
      void grabScreenshot();
      return;
    }
    if (item.token.startsWith("@")) {
      if (overlayKind === "slash") editorRef.current?.replaceSlash();
      insertMention(item.token, item.label);
      return;
    }
    if (overlayKind === "mention") {
      insertMention(item.token, item.label);
      return;
    }
    applySlash(item.token);
  }

  /** 禁用时不打开。再点一次关掉加号浮层。 */
  function togglePlus() {
    if (disabled) return;
    setHiToken("");
    setHint((h) => (h === "plus" ? null : "plus"));
    editorRef.current?.focus();
  }

  /** 桌面窗口用系统选文件，只收视频路径。选了文件但没有视频时提示。错误信息正好是 clip too large 时用过大文案。否则点隐藏的文件框。 */
  async function importClip() {
    if (disabled) return;
    if (props.onIngestVideo && desktopWindow()) {
      const paths = await api.pickFiles();
      const path = paths.find((item) => isVideoPath(item));
      if (!path) {
        if (paths.length) toast.error(copy.composer.importClipHint);
        return;
      }
      try {
        await props.onIngestVideo({ path });
      } catch (e) {
        toast.error(e instanceof Error && e.message === "clip too large" ? copy.composer.clipTooLarge : api.errMessage(e));
      }
      return;
    }
    fileRef.current?.click();
  }

  /** 不是创作桌且是桌面窗口时用 onPickFiles。否则点隐藏的文件框。 */
  async function pickAttachments() {
    if (disabled) return;
    if (!createDesk && props.onPickFiles && desktopWindow()) {
      const picked = await props.onPickFiles();
      if (picked.length) setAtts((prev) => [...prev, ...picked]);
      return;
    }
    fileRef.current?.click();
  }

  /** 第一项不是浏览器 File 时当成已经做好的附件。创作桌有导入回调时，视频走 onIngestVideo，有本地路径就传路径。list 取第一段，读取 length，作为分支结果，传给 isDomFile，传给 Array.from，展开传递。 */
  async function addFiles(list: FileList | File[] | Attachment[]) {
    if (Array.isArray(list) && list.length && !isDomFile(list[0])) {
      setAtts((prev) => [...prev, ...(list as Attachment[])]);
      return;
    }
    const files = Array.isArray(list) ? (list as File[]) : Array.from(list);
    if (createDesk && props.onIngestVideo) {
      const videos = files.filter(isVideoFile);
      const rest = files.filter((f) => !isVideoFile(f));
      if (videos.length) {
        setAttaching(true);
        try {
          for (const f of videos) {
            const path = nativeFilePath(f);
            if (path) await props.onIngestVideo({ path });
            else await props.onIngestVideo({ file: f });
          }
        } catch (e) {
          const tooBig = e instanceof Error && e.message === "clip too large";
          toast.error(tooBig ? copy.composer.clipTooLarge : api.errMessage(e));
        } finally {
          setAttaching(false);
        }
      }
      if (!rest.length) return;
      const next: Attachment[] = [];
      for (const f of rest) next.push(await fileToAttachment(f));
      setAtts((prev) => [...prev, ...next]);
      return;
    }
    const next: Attachment[] = [];
    const needsIngest = createDesk && files.some(isVideoFile);
    if (needsIngest) setAttaching(true);
    try {
      for (const f of files) {
        next.push(createDesk && isVideoFile(f) ? await ingestCreateClip(f, props.sessionId) : await fileToAttachment(f));
      }
    } catch (e) {
      toast.error(api.errMessage(e));
      return;
    } finally {
      if (needsIngest) setAttaching(false);
    }
    setAtts((prev) => [...prev, ...next]);
  }

  /** 没有回调或已禁用就返回。名字取路径最后一段，缺了用 screenshot.png。 */
  async function grabScreenshot() {
    if (!props.onScreenshot || disabled) return;
    const p = await props.onScreenshot();
    if (!p) return;
    setAtts((prev) => [...prev, { path: p, name: p.replace(/^.*[\\/]/, "") || "screenshot.png", mime: "image/png" }]);
  }

  /** 现有正文不是空白结尾时先补一个换行。 */
  async function pasteOsClipboard() {
    if (!props.onClipboard || disabled) return;
    const t = await props.onClipboard();
    if (!t) return;
    editorRef.current?.insertText((value && !/\s$/.test(value) ? "\n" : "") + t);
  }

  /** 短剧桌正在跑且不是改道时不发。正文、附件都空且不允许空发时不发。发出后清空附件、芯片和编辑器。steer 为假时走另一支，写入字段 steer。 */
  function send(steer?: boolean) {
    if (attaching || (dramaDesk && props.running && !steer)) return;
    const snap = editorRef.current?.snapshot() || { text: value, chips, caret, before: value };
    const settled = settleBareAtPaths(snap.text, props.files || []);
    const peeled = peelCompletedMentions(settled);
    const pins = mergeMentionPins(snap.chips, peeled.chips);
    const stamp = footageStamp(atts);
    const body = [composeDraft(pins, peeled.text), stamp].filter(Boolean).join("\n\n");
    if (!body && !atts.length && !props.allowEmptySend) return;
    props.onSend({ steer, attachments: atts, text: body });
    setAtts([]);
    setChips([]);
    onChange("");
    editorRef.current?.setPlain("");
  }

  /**
   * Ctrl 或 Meta 加 Shift 加 S 截图，加 V 贴系统剪贴板。浮层开着时上下键循环，Tab 能下钻就下钻，提及浮层的 Tab 只拦住不选中。Enter 选中当前项。以 / 开头且完全匹配第一条斜杠时直接执行。输入法组合中不处理这些键。
   * e 读取 ctrlKey、metaKey、shiftKey、key、isComposing。
   * snap 读取 before、text。
   * _before 函数体内没有读取。
   */
  function onEditorKey(e: KeyboardEvent, snap: DraftSnapshot, _before: string): boolean {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === "s") {
      void grabScreenshot();
      return true;
    }
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === "v") {
      void pasteOsClipboard();
      return true;
    }
    const composing = e.isComposing;
    if (overlayKind && popup.length && !composing && (e.key === "ArrowDown" || e.key === "ArrowUp")) {
      const cur = overlayActiveIndex(popup, hiToken);
      const next = e.key === "ArrowDown" ? (cur + 1) % popup.length : (cur + popup.length - 1) % popup.length;
      setHiToken(popup[next]?.token || "");
      return true;
    }
    if (overlayKind && popup.length && e.key === "Tab" && !e.shiftKey && !composing) {
      const item = popup[hi] || popup[0];
      if (item.drill) {
        drillMention(item);
        return true;
      }
      if (overlayKind !== "mention") {
        pickOverlay(item);
        return true;
      }
      return true;
    }
    if (overlayKind && popup.length && e.key === "Enter" && !e.shiftKey && !composing) {
      pickOverlay(popup[hi] || popup[0]);
      return true;
    }
    if (e.key === "Escape") {
      setHint(null);
      setDrilled(false);
      return true;
    }
    if (e.key === "Enter" && !e.shiftKey && !composing) {
      const slash = slashQuery(snap.before);
      if (slash && slashItems[0] && slash === slashItems[0].cmd) {
        applySlash(slashItems[0].cmd);
        return true;
      }
      if (snap.text.trim().startsWith("/")) {
        const [cmd, ...rest] = snap.text.trim().slice(1).split(/\s+/);
        editorRef.current?.setPlain("");
        onChange("");
        props.onSlash?.("/" + cmd, rest.join(" "));
        return true;
      }
      if (canSteerQueue && (e.metaKey || e.ctrlKey)) {
        props.onSend({ steer: true, text: "" });
        return true;
      }
      if (!canSend) return true;
      send(props.running && (e.metaKey || e.ctrlKey));
      return true;
    }
    return false;
  }

  /** 打开或关掉工作区选择。 */
  function toggleWorkspacePicker() {
    setPickerOpen((open) => !open);
  }

  const pad = props.flush ? "" : props.compact || copilotDesk ? THREAD_GUTTER_COMPACT : THREAD_GUTTER;
  const col = props.flush ? "w-full min-w-0" : THREAD_COL;
  const primaryStops = orbStops;
  const shownLabel = primaryStops ? copy.composer.stop : props.running && hasPayload ? copy.composer.sendQueue : (props.sendLabel || copy.composer.send);
  const placeholderText = canSteerQueue ? copy.composer.steerQueue : placeholder;
  const pendingTool = toolApproval && toolApproval.action !== "ask_user" ? toolApproval : null;
  const perch = !!props.perch;

  return (
    <div className={cn("relative no-drag w-full shrink-0", col, pad)} data-testid={props.compact ? undefined : "composer-column"}>
      <div
        className="sc-root"
        data-hero={props.hero ? "true" : "false"}
        data-compact={props.compact ? "true" : "false"}
        data-focused={focused ? "true" : "false"}
        data-perch={perch ? "true" : "false"}
        data-create-desk={createDesk ? "true" : "false"}
        data-drama-desk={dramaDesk ? "true" : "false"}
      >
        {takeover && props.onResolveAsk ? (
          <>
            {showWorkspaceChip ? (
              <HeroWorkspaceRow
                chipRef={chipRef}
                workspace={props.workspace}
                items={uniqueWorkspaces}
                pickerOpen={pickerOpen}
                copy={copy}
                onToggle={toggleWorkspacePicker}
                onPick={(path) => props.onWorkspace?.(path)}
                onAdd={() => { void props.onBrowseWorkspace?.(); }}
                onClose={() => setPickerOpen(false)}
              />
            ) : null}
            {perch ? <PresencePerch size={props.perchSize} /> : null}
            <ApprovalTakeover ask={props.ask} approval={pendingTool} onResolve={props.onResolveAsk} />
          </>
        ) : (
          <>
            {showWorkspaceChip ? (
              <HeroWorkspaceRow
                chipRef={chipRef}
                workspace={props.workspace}
                items={uniqueWorkspaces}
                pickerOpen={pickerOpen}
                copy={copy}
                onToggle={toggleWorkspacePicker}
                onPick={(path) => props.onWorkspace?.(path)}
                onAdd={() => { void props.onBrowseWorkspace?.(); }}
                onClose={() => setPickerOpen(false)}
              />
            ) : null}
            <div className="sc-upper">
              <TurnStatusBar status={props.turnStatus} compact={props.compact} />
              <TodoPanel plan={props.taskPlan} copy={copy} />
              <QueueDock
                items={props.queueItems || []}
                onCancel={props.onQueueCancel}
                onReorder={props.onQueueReorder}
                copy={copy}
              />
            </div>
            {perch ? <PresencePerch size={props.perchSize} /> : null}
            <ChatBorder className="sc-card-frame">
            <div
              ref={cardRef}
              className="sc-card"
              data-composer-card
              data-workspace-trigger={workspaceTrigger ? "true" : undefined}
              onPointerDown={(e) => {
                if (!workspaceTrigger) return;
                e.stopPropagation();
              }}
              onClick={(e) => {
                const t = e.target as HTMLElement;
                if (t.closest("button, [role='combobox'], a, input, [contenteditable='true']")) return;
                if (workspaceTrigger) {
                  toggleWorkspacePicker();
                  return;
                }
                if (!disabled) editorRef.current?.focus();
              }}
              onFocusCapture={() => setFocused(true)}
              onBlurCapture={(e) => {
                if (!e.currentTarget.contains(e.relatedTarget as Node)) setFocused(false);
              }}
              onDragOver={(e) => {
                if (disabled) return;
                e.preventDefault();
                setDrop(true);
              }}
              onDragLeave={() => setDrop(false)}
              onDrop={(e) => {
                if (disabled) return;
                e.preventDefault();
                setDrop(false);
                if (e.dataTransfer?.files?.length) void addFiles(e.dataTransfer.files);
              }}
            >
              <DropOverlay active={drop} label={createDesk ? copy.video.conductor.drop : undefined} />
              {showPopup && (overlayKind === "slash" || overlayKind === "plus") ? (
                <CommandOverlay
                  label={overlayKind === "plus" ? copy.composer.addFilesOrCommands : copy.composer.mentionSuggestions}
                  items={popup}
                  hi={hi}
                  onPick={pickOverlay}
                  onHover={(i) => setHiToken(popup[i]?.token || "")}
                />
              ) : null}
              {showPopup && overlayKind === "mention" ? (
                <CommandOverlay
                  label={copy.composer.mentionSuggestions}
                  items={popup}
                  hi={hi}
                  crumbs={mentionTrail}
                  crumbsLabel={copy.composer.mentionCrumbs}
                  drillHint={copy.composer.mentionDrill}
                  drillKey={copy.composer.mentionDrillKey}
                  drillAria={copy.composer.mentionDrill}
                  empty={fileQ === null || (props.files || []).length === 0 ? copy.composer.mentionType : copy.composer.mentionEmpty}
                  onPick={pickOverlay}
                  onDrill={drillMention}
                  onCrumb={(crumb) => applyTrigger(crumb.query)}
                  onHover={(i) => setHiToken(popup[i]?.token || "")}
                />
              ) : null}
              {suggestedSkills.length || autoSkillChips.length ? (
                <div className="sc-rail">
                  {autoSkillChips.map((s) => (
                    <Tooltip key={`auto-${s.name}`} content={copy.composer.autoLoadedSkillHint}>
                      <button
                        type="button"
                        data-testid="auto-skill-chip"
                        className="inline-flex max-w-[14rem] items-center gap-1 rounded-md bg-lift px-1.5 py-0.5 text-[11px] text-foreground"
                        onClick={() => setDismissedAuto((prev) => prev.includes(s.name) ? prev : [...prev, s.name])}
                      >
                        <BookOpen className="size-3 shrink-0 opacity-70" />
                        <span className="truncate">{copy.composer.autoLoadedSkill.replace("{name}", s.name)}</span>
                        <X className="size-3 shrink-0 opacity-70" />
                      </button>
                    </Tooltip>
                  ))}
                  {suggestedSkills.map((s) => (
                    <Tooltip key={`suggest-${s.name}`} content={s.description || copy.composer.suggestSkillHint}>
                      <button
                        type="button"
                        data-testid="suggest-skill-chip"
                        className="inline-flex max-w-[14rem] items-center gap-1 rounded-md border border-border/70 px-1.5 py-0.5 text-[11px] text-muted hover:bg-lift hover:text-foreground"
                        onClick={() => insertMention(`@skill:${s.name}`)}
                      >
                        <BookOpen className="size-3 shrink-0 opacity-70" />
                        <span className="truncate">{copy.composer.suggestSkill} · {s.displayName || s.name}</span>
                        <X
                          className="size-3 shrink-0 opacity-70"
                          onClick={(e) => {
                            e.stopPropagation();
                            setDismissedSuggest((prev) => prev.includes(s.name) ? prev : [...prev, s.name]);
                          }}
                        />
                      </button>
                    </Tooltip>
                  ))}
                </div>
              ) : null}
              {props.leading ? <div className="sc-accessory">{props.leading}</div> : null}
              {attaching ? <p className="sc-notice" role="status">{copy.composer.attachingClip}</p> : null}
              <AttachmentRail items={atts} onRemove={(i) => setAtts((prev) => prev.filter((_, j) => j !== i))} />
              <DraftEditor
                ref={editorRef}
                draftKey={props.draftKey}
                disabled={disabled}
                placeholder={placeholderText}
                ariaLabel={props.placeholder || copy.composer.message}
                testId={props.inputTestId}
                workspaceTrigger={workspaceTrigger}
                pickerOpen={pickerOpen}
                onSnapshot={applySnapshot}
                onCaret={applyCaret}
                onKeyDown={onEditorKey}
                onPasteFiles={(files) => { void addFiles(files); }}
              />
              <div
                className="sc-row"
                onMouseDown={(e) => {
                  const el = e.target as HTMLElement;
                  if (el.closest(".ant-select, .yc-select, .canvas-composer-model-picker, .drama-style-trigger")) return;
                  if (el.closest("button, [role='combobox']")) e.preventDefault();
                }}
              >
                <div className="sc-tools">
                  {props.toolbarLeading ? <div className="sc-toolbar-leading">{props.toolbarLeading}</div> : null}
                  {copilotDesk || talkDesk || dramaDesk ? null : videoing ? (
                    <>
                      <VideoModeSwitch disabled={disabled} />
                      {videoMode === "drama" ? <DramaProjectChip disabled={disabled} /> : null}
                      {videoMode === "canvas" ? <CanvasProjectChip disabled={disabled} /> : null}
                      {videoMode === "creative" ? <StudioDeskChip disabled={disabled} /> : null}
                    </>
                  ) : null}
                  {!copilotDesk && !talkDesk && !dramaDesk && props.onProfile && profiles.length ? (
                    <ProfileChip
                      profiles={profiles}
                      profileId={props.profileId}
                      disabled={disabled}
                      onProfile={props.onProfile}
                    />
                  ) : null}
                  {props.compact || (createHome && createMode !== "clip") ? null : (
                  <Tooltip content={createDesk ? copy.composer.importClip : copy.composer.addFilesOrCommands}>
                    <button
                      type="button"
                      className="sc-add"
                      aria-label={createDesk ? copy.composer.importClip : copy.composer.addFilesOrCommands}
                      aria-haspopup={createDesk ? undefined : "listbox"}
                      aria-expanded={createDesk ? undefined : hint === "plus"}
                      data-testid={studioDesk ? "studio-copilot-attach" : "composer-add"}
                      disabled={disabled}
                      onClick={createDesk ? () => void importClip() : togglePlus}
                    >
                      <Plus className="size-3.5" />
                    </button>
                  </Tooltip>
                  )}
                  <input
                    ref={fileRef}
                    type="file"
                    multiple
                    accept={createDesk ? "video/*,audio/*,image/*" : undefined}
                    className="hidden"
                    onChange={(e) => {
                      if (e.target.files) void addFiles(e.target.files);
                      e.target.value = "";
                    }}
                  />
                  <div className="sc-modes">
                    {props.onAuthMode && !talkDesk && !dramaDesk && !copilotDesk ? (
                      <PermissionSelect
                        value={props.authMode}
                        disabled={disabled}
                        testId={studioDesk ? "studio-copilot-auth" : "composer-auth"}
                        onChange={(mode) => props.onAuthMode?.(mode)}
                      />
                    ) : null}
                    {modeMenu ? (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button
                            type="button"
                            className="sc-chip-btn"
                            data-on={plan && !videoing ? "true" : "false"}
                            aria-label={videoing ? copy.composer.location : copy.composer.mode}
                            disabled={disabled}
                          >
                            <span className="truncate">{videoing ? copy.composer.location : plan ? copy.composer.plan : copy.composer.agent}</span>
                            {props.isolate ? <GitBranch className="size-3 shrink-0 opacity-70" aria-hidden /> : null}
                            <ChevronDown className="size-3 shrink-0 opacity-70" />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="start" side="top" className="w-60">
                          {!videoing ? (
                            <>
                              <DropdownMenuLabel>{copy.composer.mode}</DropdownMenuLabel>
                              <DropdownMenuItem className="items-start" onSelect={() => setPlan(false)}>
                                <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                                  <span>{copy.composer.agent}</span>
                                  <span className="text-[11px] text-muted">{copy.composer.agentHint}</span>
                                </span>
                                {!plan ? <Check className="mt-0.5 size-3.5 shrink-0" /> : null}
                              </DropdownMenuItem>
                              <DropdownMenuItem className="items-start" onSelect={() => setPlan(true)}>
                                <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                                  <span>{copy.composer.plan}</span>
                                  <span className="text-[11px] text-muted">{copy.composer.planHint}</span>
                                </span>
                                {plan ? <Check className="mt-0.5 size-3.5 shrink-0" /> : null}
                              </DropdownMenuItem>
                            </>
                          ) : null}
                          {!studioDesk && props.onIsolate ? (
                            <>
                              {!videoing ? <DropdownMenuSeparator /> : null}
                              <DropdownMenuLabel>{copy.composer.location}</DropdownMenuLabel>
                              <DropdownMenuItem disabled={disabled || props.running} onSelect={() => props.onIsolate?.(false)}>
                                <Folder className="size-3.5" />
                                <span className="min-w-0 flex-1 truncate">{copy.composer.isolateOff}</span>
                                {!props.isolate ? <Check className="size-3.5 shrink-0" /> : null}
                              </DropdownMenuItem>
                              <DropdownMenuItem disabled={disabled || props.running} onSelect={() => props.onIsolate?.(true)}>
                                <GitBranch className="size-3.5" />
                                <span className="min-w-0 flex-1 truncate">{copy.composer.isolateOn}</span>
                                {props.isolate ? <Check className="size-3.5 shrink-0" /> : null}
                              </DropdownMenuItem>
                              {props.isolate && props.onApplyWorktree ? (
                                <DropdownMenuItem disabled={disabled || props.running} onSelect={() => props.onApplyWorktree?.()}>
                                  <GitBranch className="size-3.5" />
                                  <span className="min-w-0 flex-1 truncate">{copy.composer.applyWorktree}</span>
                                </DropdownMenuItem>
                              ) : null}
                            </>
                          ) : null}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    ) : null}
                  </div>
                </div>
                <div className="sc-trailing">
                  {props.toolbarTrailing}
                  {createHome ? null : grouped && props.onModel ? (
                    <Select value={groupedValue} onValueChange={(v) => props.onModel?.(v)} disabled={disabled}>
                      <SelectTrigger className="sc-chip-btn max-w-[16rem] border-0 shadow-none max-[799px]:max-w-[7.5rem]" aria-label={copy.composer.model} title={selectedGroup ? `${selectedGroup.providerName} · ${currentModel}` : currentModel}>
                        <span className="truncate">{selectedGroup ? `${selectedGroup.providerName} · ${modelLabel}` : modelLabel}</span>
                      </SelectTrigger>
                      <SelectContent className="min-w-[min(18rem,calc(100vw-1.5rem))]">
                        {groups.map((g) => (
                          <SelectGroup key={g.providerId}>
                            <SelectLabel>{g.providerName}{g.isDefault ? ` · ${copy.settings.defaultBadge}` : ""}</SelectLabel>
                            {g.models.map((m) => {
                              const label = lookupCatalogModel(catalog, g.vendor, m)?.model.name || m;
                              return <SelectItem key={`${g.providerId}::${m}`} value={encodeChatModel(g.providerId, m)}>{label}</SelectItem>;
                            })}
                          </SelectGroup>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : modelOptions.length > 0 && props.onModel ? (
                    <Select value={currentModel || modelOptions[0]} onValueChange={(v) => props.onModel?.(v)} disabled={disabled}>
                      <SelectTrigger className="sc-chip-btn max-w-[8.5rem] border-0 shadow-none" aria-label={copy.composer.model} title={currentModel}>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {modelOptions.map((m) => {
                          const label = lookupCatalogModel(catalog, props.provider || "", m)?.model.name || m;
                          return <SelectItem key={m} value={m}>{label}</SelectItem>;
                        })}
                      </SelectContent>
                    </Select>
                  ) : currentModel && !props.leading && !props.toolbarLeading ? (
                    <span className="hidden max-w-[8.5rem] truncate px-1.5 text-[12px] text-muted sm:inline" title={currentModel}>{modelLabel}</span>
                  ) : null}
                  {props.running && hasPayload ? (
                    <Tooltip content={copy.composer.stop}>
                      <button type="button" className="sc-primary" data-stop="true" onClick={() => props.onStop()} aria-label={copy.composer.stop} data-testid="composer-stop">
                        <StopGlyph />
                      </button>
                    </Tooltip>
                  ) : null}
                  {primaryStops ? (
                    <Tooltip content={shownLabel}>
                      <button
                        type="button"
                        className="sc-primary"
                        data-stop="true"
                        data-testid="composer-send"
                        aria-label={shownLabel}
                        onClick={() => props.onStop()}
                      >
                        <SendGlyph busy morphDuration={240} squash={0.12} tilt={8} />
                      </button>
                    </Tooltip>
                  ) : (
                    <Tooltip content={shownLabel}>
                      <SendSling
                        testId="composer-send"
                        disabled={!canSend}
                        ariaLabel={shownLabel}
                        title={props.running && hasPayload ? copy.composer.queueHint : undefined}
                        onSend={() => send()}
                      />
                    </Tooltip>
                  )}
                </div>
              </div>
            </div>
            </ChatBorder>
            {props.compact ? null : (
            <div className="sc-dock" data-composer-dock>
              <StatsPills
                stats={props.sessionStats}
                ctx={props.ctx}
                turnStatus={props.turnStatus}
                copy={copy}
              />
              <ContextMeter
                ctx={props.ctx}
                draftTokens={estimateTokens(composeDraft(chips, value))}
                fallbackWindow={fallbackWindow}
                copy={copy}
              />
            </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

/** 首页上的工作区芯片和选择层。路径还没就绪时不显示当前名。props 读取 chipRef、workspace、pickerOpen、copy、onToggle、items 等。 */
function HeroWorkspaceRow(props: {
  chipRef: RefObject<HTMLButtonElement | null>;
  workspace?: string;
  items: string[];
  pickerOpen: boolean;
  copy: Copy;
  onToggle: () => void;
  onPick: (path: string) => void;
  onAdd: () => void;
  onClose: () => void;
}) {
  return (
    <div className="sc-hero-ws">
      <WorkspaceChip
        ref={props.chipRef}
        label={pathReady(props.workspace) ? displayWorkspace(props.workspace, props.workspace) : undefined}
        menuOpen={props.pickerOpen}
        chooseLabel={props.copy.composer.chooseWorkspace}
        onClick={props.onToggle}
      />
      <WorkspacePicker
        open={props.pickerOpen}
        anchorRef={props.chipRef}
        items={props.items}
        selected={props.workspace}
        copy={props.copy}
        onPick={props.onPick}
        onAdd={props.onAdd}
        onClose={props.onClose}
      />
    </div>
  );
}

/** 运行环境没有 File 时返回假。v 用 instanceof 看是不是 File。 */
function isDomFile(v: unknown): v is File {
  return typeof File !== "undefined" && v instanceof File;
}

/** 图片 2MB，文本类 8MB，其余 96KB。f 读取 type、name。 */
function attachmentByteCap(f: File): number {
  if (f.type.startsWith("image/")) return 2 * 1024 * 1024;
  const name = f.name || "";
  if (f.type.startsWith("text/") || /\.(md|markdown|txt|json|csv|tsv|html?|xml|ya?ml|rst|tex)$/i.test(name)) {
    return 8 * 1024 * 1024;
  }
  return 96 * 1024;
}

/** 超过上限就截断再做成 base64。没有名字时图片叫 image.png，其余叫 file。f 读取 name、type，调用 arrayBuffer，传给 attachmentByteCap。 */
async function fileToAttachment(f: File): Promise<Attachment> {
  const buf = await f.arrayBuffer();
  const cap = attachmentByteCap(f);
  const slice = buf.byteLength > cap ? buf.slice(0, cap) : buf;
  const bytes = new Uint8Array(slice);
  let bin = "";
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
  return { name: f.name || (f.type.startsWith("image/") ? "image.png" : "file"), mime: f.type, data_b64: btoa(bin) };
}

/** 类型以 video/ 开头，或扩展名是 mp4、mov、mkv、webm、avi、m4v、mpeg、mpg。f 读取 type、name。 */
function isVideoFile(f: File): boolean {
  if (f.type.startsWith("video/")) return true;
  return /\.(mp4|mov|mkv|webm|avi|m4v|mpeg|mpg)$/i.test(f.name || "");
}

/** 只认 yoyo-cut:id:hash。对不上返回 null。path 为假时用空字符串，转成字符串，传给 exec。 */
function parseCutRef(path?: string): { id: string; hash: string } | null {
  const m = /^yoyo-cut:([^:]+):(.+)$/.exec(String(path || ""));
  return m ? { id: m[1], hash: m[2] } : null;
}

/** 只给带 yoyo-cut 引用的附件拼一行入库说明。没有这种附件就返回空字符串。atts 调用 flatMap。 */
function footageStamp(atts: Attachment[]): string {
  const lines = atts.flatMap((a) => {
    const ref = parseCutRef(a.path);
    if (!ref) return [];
    return [`源片已入库。cut_id=${ref.id} cas=${ref.hash}。用 cut_analyze / cut_propose / cut_apply。不要检查代码仓库。`];
  });
  return lines.join("\n");
}

/** 读成 data URL 后去掉逗号前缀。没有哈希或没有工程 id 就抛错。附件路径写成 yoyo-cut:id:hash。f 读取 type、name，传给 r.readAsDataURL。sessionId 为假时用空字符串，传给 api.video.cutCreate。 */
async function ingestCreateClip(f: File, sessionId?: string): Promise<Attachment> {
  const data = await new Promise<string>((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result || ""));
    r.onerror = () => reject(r.error);
    r.readAsDataURL(f);
  });
  const b64 = data.includes(",") ? data.slice(data.indexOf(",") + 1) : data;
  const ref = await api.video.putStudioMedia(b64, f.type || "video/mp4", "video");
  const hash = String(ref?.cas_hash || ref?.hash || "");
  if (!hash) throw new Error("upload did not return a CAS hash");
  const p = await api.video.cutCreate(f.name.replace(/\.[^.]+$/, ""), hash, sessionId || "") as { id?: string };
  const id = String(p?.id || "");
  if (!id) throw new Error("cut project was not created");
  return { name: f.name, mime: f.type || "video/mp4", path: `yoyo-cut:${id}:${hash}` };
}
