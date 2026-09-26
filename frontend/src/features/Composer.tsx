import { useEffect, useMemo, useRef, useState, type ClipboardEvent, type Ref } from "react";
import { ArrowUp, BookOpen, Boxes, Camera, Check, ChevronDown, ChevronUp, Clipboard, FileText, Folder, GitBranch, Paperclip, Plus, Square, X } from "lucide-react";
import { Textarea } from "../components/ui/input";
import { IconSwap } from "../components/ui/icon-swap";
import { Tooltip } from "../components/ui/tooltip";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "../components/ui/dropdown-menu";
import { THREAD_COL, THREAD_GUTTER, THREAD_GUTTER_COMPACT } from "../lib/thread";
import { cn } from "../lib/utils";
import { displayWorkspace } from "../lib/display-title";
import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import { filterSlash, slashCatalog, slashQuery } from "../lib/slash";
import type { Attachment, AuthMode, ContextUsage, FileHit, SkillInfo } from "../lib/protocol";
import type { TaskPlan } from "../lib/plan";
import { PlanChip } from "./PlanChip";
import { formatTokens, lookupCatalogModel, composerModelIds, resolveContextWindow } from "../lib/models-dev";
import { MODELS_DEV_SNAPSHOT } from "../lib/models-dev.snapshot";
import { contextBreakdown, estimateTokens, type CtxSliceId } from "../lib/context-usage";
import type { Copy } from "../lib/copy";
import { VideoModeSwitch } from "./video/VideoModeSwitch";
import { DramaProjectChip } from "./video/DramaProjectChip";
import { CanvasProjectChip } from "./video/CanvasProjectChip";
import {
  composeDraft,
  isCompleteMention,
  keepMentionOpen,
  mentionTokenAt,
  mentionableSkills,
  peelCompletedMentions,
  pinFromToken,
  type MentionKind,
  type MentionPin,
} from "../lib/mentions";

export function Composer(props: {
  draftKey: string;
  running: boolean;
  disabled: boolean;
  disabledReason?: string;
  model?: string;
  models?: string[];
  provider?: string;
  ctx?: ContextUsage;
  queued?: number;
  queueItems?: { id?: string; text?: string; plan?: boolean }[];
  onQueueCancel?: (id: string) => void;
  onQueueReorder?: (id: string, delta: number) => void;
  onApplyWorktree?: () => void;
  skills?: SkillInfo[];
  files?: FileHit[];
  onSearchFiles?: (q: string) => void;
  onSend: (opts?: { steer?: boolean; attachments?: Attachment[]; text?: string }) => void;
  onStop: () => void;
  onSlash?: (cmd: string, rest: string) => void;
  onModel?: (model: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
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
}) {
  const value = useUI((s) => s.drafts[props.draftKey] || "");
  const plan = useUI((s) => s.plan);
  const setDraft = useUI((s) => s.setDraft);
  const setPlan = useUI((s) => s.setPlan);
  const onChange = (v: string) => setDraft(props.draftKey, v);
  const [hint, setHint] = useState<"mention" | "slash" | null>(null);
  const [hi, setHi] = useState(0);
  const [focused, setFocused] = useState(false);
  const [atts, setAtts] = useState<Attachment[]>([]);
  const [chips, setChips] = useState<MentionPin[]>([]);
  const [fileQ, setFileQ] = useState<string | null>(null);
  const [caret, setCaret] = useState(0);
  const ref = useRef<HTMLTextAreaElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const copy = useCopy();
  const surface = useUI((s) => s.surface);
  const videoMode = useUI((s) => s.videoMode);
  const videoSurface = surface === "video";
  const videoing = videoSurface && !props.compact;
  const placeholder = props.disabled
    ? (props.disabledReason || copy.composer.disabled)
    : videoSurface
      ? (videoMode === "canvas" ? copy.composer.placeholderCanvas : videoMode === "creative" ? copy.composer.placeholderCreative : copy.composer.placeholderDrama)
      : copy.composer.placeholder;
  const hasPayload = !!value.trim() || chips.length > 0 || atts.length > 0;
  const canSend = !props.disabled && hasPayload;
  const currentModel = (props.model || "").trim();
  const modelOptions = composerModelIds(MODELS_DEV_SNAPSHOT, props.provider, currentModel, props.models);
  const slashOpen = hint === "slash";
  const meta = lookupCatalogModel(MODELS_DEV_SNAPSHOT, props.provider || "openai", currentModel);
  const fallbackWindow = resolveContextWindow(MODELS_DEV_SNAPSHOT, props.provider || "", currentModel);
  const modelLabel = meta?.model.name || currentModel;

  const slashPrefix = slashQuery(value);
  const slashItems = slashPrefix ? filterSlash(slashPrefix, slashCatalog(copy)) : [];
  const mentionItems = useMemo(() => {
    const token = mentionTokenAt(value, caret)?.token || "";
    if (token.startsWith("@file:") || token.startsWith("@folder:")) {
      const folder = token.startsWith("@folder:");
      return (props.files || [])
        .filter((f) => (folder ? f.kind === "dir" : true))
        .map((f) => ({
          token: `${folder ? "@folder:" : "@file:"}${f.path}`,
          label: pinFromToken(`${folder ? "@folder:" : "@file:"}${f.path}`).label,
          hint: f.path,
          kind: (folder ? "folder" : "file") as MentionKind,
        }));
    }
    if (token.startsWith("@skill:")) {
      const q = token.slice("@skill:".length);
      return mentionableSkills(props.skills || [], q).map((s) => ({
        token: `@skill:${s.name}`,
        label: s.displayName || s.name,
        hint: s.description,
        kind: "skill" as MentionKind,
      }));
    }
    const mentions = [
      { token: "@file:", label: copy.composer.mentionKindFile, hint: copy.composer.mentionFile, kind: "file" as MentionKind },
      { token: "@folder:", label: copy.composer.mentionKindFolder, hint: copy.composer.mentionFolder, kind: "folder" as MentionKind },
      { token: "@skill:", label: copy.composer.mentionKindSkill, hint: copy.composer.mentionSkill, kind: "skill" as MentionKind },
      { token: "@harness", label: copy.composer.mentionKindHarness, hint: copy.composer.mentionHarness, kind: "harness" as MentionKind },
    ];
    if (!token || token === "@") return mentions;
    return mentions.filter((x) => x.token.startsWith(token));
  }, [value, caret, props.files, props.skills, copy]);

  const popup = hint === "slash"
    ? slashItems.map((s) => ({ token: s.cmd, label: s.cmd, hint: s.hint, kind: undefined as MentionKind | undefined }))
    : mentionItems;
  const showPopup = hint === "mention" || (!!hint && popup.length > 0);
  const popupWelded = showPopup && (slashOpen || hint === "mention");

  useEffect(() => {
    setHint(null);
    setHi(0);
    setAtts([]);
    setChips([]);
    setFileQ(null);
  }, [props.draftKey]);

  useEffect(() => {
    if (popup.length && hi >= popup.length) setHi(0);
  }, [hi, popup.length]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, props.hero ? 200 : 112) + "px";
  }, [value, chips, atts, props.hero]);

  function placeCaret(pos: number) {
    setCaret(pos);
    requestAnimationFrame(() => {
      const el = ref.current;
      if (!el) return;
      el.focus();
      el.setSelectionRange(pos, pos);
    });
  }

  function insertMention(token: string) {
    const el = ref.current;
    const pos = el?.selectionStart ?? caret;
    const hit = mentionTokenAt(value, pos);
    if (isCompleteMention(token)) {
      const start = hit?.start ?? pos;
      const end = hit?.end ?? pos;
      const next = value.slice(0, start) + value.slice(end).replace(/^\s/, "");
      const peeled = peelCompletedMentions(next, start);
      setChips((c) => [...c, pinFromToken(token), ...peeled.chips]);
      onChange(peeled.text);
      setHint(null);
      setHi(0);
      placeCaret(Math.min(start, peeled.text.length));
      return;
    }
    const suffix = keepMentionOpen(token) ? "" : " ";
    let next: string;
    let newCaret: number;
    if (hit) {
      next = value.slice(0, hit.start) + token + suffix + value.slice(hit.end);
      newCaret = hit.start + token.length + suffix.length;
    } else {
      const before = value.slice(0, pos);
      const after = value.slice(pos);
      const needsSpace = before.length > 0 && !/\s$/.test(before);
      const ins = (needsSpace ? " " : "") + token + suffix;
      next = before + ins + after;
      newCaret = pos + ins.length;
    }
    onChange(next);
    setHi(0);
    if (keepMentionOpen(token)) {
      setHint("mention");
      if (token === "@file:" || token === "@folder:") {
        setFileQ("");
        props.onSearchFiles?.("");
      }
    } else {
      setHint(null);
    }
    placeCaret(newCaret);
  }

  function applySlash(cmd: string) {
    onChange("");
    setHint(null);
    props.onSlash?.(cmd, "");
  }

  async function addFiles(list: FileList | File[] | Attachment[]) {
    if (Array.isArray(list) && list.length && !isDomFile(list[0])) {
      setAtts((prev) => [...prev, ...(list as Attachment[])]);
      return;
    }
    const files = Array.isArray(list) ? (list as File[]) : Array.from(list);
    const next: Attachment[] = [];
    for (const f of files) {
      next.push(await fileToAttachment(f));
    }
    setAtts((prev) => [...prev, ...next]);
  }

  async function grabScreenshot() {
    if (!props.onScreenshot || props.disabled) return;
    const p = await props.onScreenshot();
    if (!p) return;
    setAtts((prev) => [...prev, { path: p, name: p.replace(/^.*[\\/]/, "") || "screenshot.png", mime: "image/png" }]);
  }

  async function pasteOsClipboard() {
    if (!props.onClipboard || props.disabled) return;
    const t = await props.onClipboard();
    if (!t) return;
    const next = value ? `${value}${/\s$/.test(value) ? "" : "\n"}${t}` : t;
    const peeled = peelCompletedMentions(next, next.length);
    if (peeled.chips.length) setChips((c) => [...c, ...peeled.chips]);
    onChange(peeled.text);
    placeCaret(peeled.caret);
  }

  function send(steer?: boolean) {
    const body = composeDraft(chips, value);
    const caption = body || atts.map((a) => a.name || a.path || "file").join(", ");
    if (!caption && !atts.length) return;
    props.onSend({ steer, attachments: atts, text: caption });
    setAtts([]);
    setChips([]);
    onChange("");
  }

  const pad = props.flush ? "" : props.compact ? THREAD_GUTTER_COMPACT : THREAD_GUTTER;
  const col = props.flush ? "w-full min-w-0" : THREAD_COL;
  const ghostSelect =
    "h-6 min-w-0 max-w-[9.5rem] gap-1 rounded-lg border-transparent bg-transparent px-1.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30 [&_svg]:size-3 [&_svg]:opacity-70";

  return (
    <div className="relative no-drag w-full shrink-0" data-testid={props.compact ? undefined : "composer-column"}>
      <div className={cn(col, pad, props.compact ? "pb-2 pt-1" : props.hero ? "pb-0 pt-0" : "pb-4 pt-1")}>
        <div
          className="composer-bezel"
          data-focused={focused ? "true" : "false"}
          data-welded={popupWelded ? "true" : "false"}
          data-hero={props.hero ? "true" : "false"}
        >
        <div
          className={cn(
            "composer-core",
            props.disabled && "opacity-55",
          )}
          onDragOver={(e) => {
            if (props.disabled) return;
            e.preventDefault();
          }}
          onDrop={(e) => {
            if (props.disabled) return;
            e.preventDefault();
            if (e.dataTransfer?.files?.length) void addFiles(e.dataTransfer.files);
          }}
        >
          {showPopup && slashOpen ? (
            <CommandWeld
              label={copy.composer.commands}
              items={popup}
              hi={hi}
              focused={focused}
              onPick={(item) => applySlash(item)}
            />
          ) : null}
          {showPopup && hint === "mention" ? (
            <CommandWeld
              label={copy.composer.mentions}
              items={popup}
              hi={hi}
              focused={focused}
              empty={
                mentionTokenAt(value, caret)?.token.startsWith("@file:") || mentionTokenAt(value, caret)?.token.startsWith("@folder:")
                  ? copy.composer.mentionType
                  : copy.composer.mentionEmpty
              }
              onPick={(item) => insertMention(item)}
            />
          ) : null}
          {props.taskPlan ? (
            <PlanChip key={props.draftKey} plan={props.taskPlan} running={props.running} compact={props.compact} />
          ) : null}
          {chips.length || atts.length ? (
            <div className="flex flex-wrap items-center gap-1 px-3 pt-2.5">
              {chips.map((c, i) => (
                <Tooltip key={`${c.token}-${i}`} content={c.detail || c.token}>
                  <button
                    type="button"
                    data-testid="mention-chip"
                    aria-label={c.token}
                    className="inline-flex max-w-[12rem] items-center gap-1 rounded-md bg-lift px-1.5 py-0.5 text-[11px] text-foreground hover:bg-lift/80"
                    onClick={() => setChips((prev) => prev.filter((_, j) => j !== i))}
                  >
                    <MentionGlyph kind={c.kind} />
                    <span className="truncate">{c.label}</span>
                    <X className="size-3 shrink-0 opacity-70" />
                  </button>
                </Tooltip>
              ))}
              {atts.map((a, i) => (
                <AttachmentChip key={`${a.name}-${i}`} att={a} onRemove={() => setAtts((prev) => prev.filter((_, j) => j !== i))} />
              ))}
            </div>
          ) : null}
          <Textarea
            ref={ref}
            rows={1}
            value={value}
            disabled={props.disabled}
            placeholder={placeholder}
            className={cn(
              "prose-select min-h-[1.5rem] px-3.5 pt-3 pb-1 text-[14px] leading-[1.55] placeholder:text-muted/55",
              props.hero && "min-h-[4.5rem] px-4 pt-3.5 text-[15px] placeholder:text-muted/60",
            )}
            aria-label={copy.composer.message}
            onFocus={() => {
              setFocused(true);
              const peeled = peelCompletedMentions(value);
              if (!peeled.chips.length) return;
              setChips((c) => [...c, ...peeled.chips]);
              onChange(peeled.text);
              placeCaret(Math.min(peeled.caret, peeled.text.length));
            }}
            onBlur={() => setFocused(false)}
            onClick={(e) => setCaret(e.currentTarget.selectionStart ?? value.length)}
            onKeyUp={(e) => setCaret(e.currentTarget.selectionStart ?? value.length)}
            onPaste={(e) => {
              if (ingestPasteFiles(e, (files) => void addFiles(files))) {
                e.preventDefault();
              }
            }}
            onChange={(e) => {
              const v = e.target.value;
              const pos = e.target.selectionStart ?? v.length;
              setCaret(pos);
              if (slashQuery(v)) {
                onChange(v);
                setHint("slash");
                setHi(0);
                return;
              }
              const hit = mentionTokenAt(v, pos);
              if (hit) {
                onChange(v);
                setHint("mention");
                if (hit.token.startsWith("@file:") || hit.token.startsWith("@folder:")) {
                  const q = hit.token.replace(/^@(?:file|folder):/, "");
                  if (q !== fileQ) {
                    setFileQ(q);
                    props.onSearchFiles?.(q);
                  }
                }
                setHi(0);
                return;
              }
              setHint(null);
              const peeled = peelCompletedMentions(v, pos);
              if (peeled.chips.length) {
                setChips((c) => [...c, ...peeled.chips]);
                onChange(peeled.text);
                placeCaret(peeled.caret);
                return;
              }
              onChange(v);
              setHi(0);
            }}
            onKeyDown={(e) => {
              if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === "s") {
                e.preventDefault();
                void grabScreenshot();
                return;
              }
              if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === "v") {
                e.preventDefault();
                void pasteOsClipboard();
                return;
              }
              if (hint && popup.length && (e.key === "ArrowDown" || e.key === "ArrowUp")) {
                e.preventDefault();
                setHi((n) => (e.key === "ArrowDown" ? (n + 1) % popup.length : (n + popup.length - 1) % popup.length));
                return;
              }
              if (hint === "mention" && (e.key === "Tab" || e.key === "Enter") && !e.shiftKey) {
                e.preventDefault();
                if (popup.length) insertMention(popup[hi]?.token || popup[0].token);
                return;
              }
              if (hint && popup.length && (e.key === "Tab" || e.key === "Enter") && !e.shiftKey) {
                e.preventDefault();
                const item = popup[hi];
                if (hint === "slash") applySlash(item.token);
                else insertMention(item.token);
                return;
              }
              if (e.key === "Escape") {
                setHint(null);
                return;
              }
              if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
                e.preventDefault();
                if (!canSend) return;
                const slash = slashQuery(value);
                if (slash && slashItems[0] && value.trim() === slashItems[0].cmd) {
                  applySlash(slashItems[0].cmd);
                  return;
                }
                if (value.trim().startsWith("/")) {
                  const [cmd, ...rest] = value.trim().slice(1).split(/\s+/);
                  onChange("");
                  props.onSlash?.("/" + cmd, rest.join(" "));
                  return;
                }
                send();
              }
            }}
          />
          <div className="flex flex-nowrap items-center gap-x-0.5 px-2 pb-2 pt-0.5">
            <div className="flex min-w-0 items-center gap-x-0.5">
            {videoSurface ? (
              <>
                <VideoModeSwitch disabled={props.disabled} />
                {videoMode === "drama" ? <DramaProjectChip disabled={props.disabled} /> : null}
                {videoMode === "canvas" ? <CanvasProjectChip disabled={props.disabled} /> : null}
                <span className="mx-0.5 h-3.5 w-px bg-border/70" aria-hidden />
              </>
            ) : null}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className={toolClass()}
                  aria-label={copy.composer.add}
                  disabled={props.disabled}
                >
                  <Plus className="size-3.5" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" side="top" className="w-60">
                <DropdownMenuItem
                  disabled={props.disabled}
                  onSelect={() => {
                    void (async () => {
                      if (props.onPickFiles) {
                        const picked = await props.onPickFiles();
                        if (picked.length) setAtts((prev) => [...prev, ...picked]);
                        return;
                      }
                      fileRef.current?.click();
                    })();
                  }}
                >
                  <Paperclip className="size-3.5" />
                  {copy.composer.attach}
                </DropdownMenuItem>
                {props.onClipboard ? (
                  <DropdownMenuItem disabled={props.disabled} onSelect={() => { void pasteOsClipboard(); }}>
                    <Clipboard className="size-3.5" />
                    {copy.composer.clipboard}
                  </DropdownMenuItem>
                ) : null}
                {props.onScreenshot ? (
                  <DropdownMenuItem disabled={props.disabled} onSelect={() => { void grabScreenshot(); }}>
                    <Camera className="size-3.5" />
                    {copy.composer.screenshot}
                  </DropdownMenuItem>
                ) : null}
                {props.onIsolate ? (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      disabled={props.disabled || props.running}
                      onSelect={() => props.onIsolate?.(!props.isolate)}
                    >
                      <GitBranch className="size-3.5" />
                      <span className="min-w-0 flex-1 truncate">{props.isolate ? copy.composer.isolateOn : copy.composer.isolate}</span>
                      {props.isolate ? <Check className="size-3.5 shrink-0" /> : null}
                    </DropdownMenuItem>
                    {props.isolate && props.onApplyWorktree ? (
                      <DropdownMenuItem
                        disabled={props.disabled || props.running}
                        onSelect={() => props.onApplyWorktree?.()}
                      >
                        <GitBranch className="size-3.5" />
                        <span className="min-w-0 flex-1 truncate">{copy.composer.applyWorktree}</span>
                      </DropdownMenuItem>
                    ) : null}
                  </>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>
            <input
              ref={fileRef}
              type="file"
              multiple
              className="hidden"
              onChange={(e) => {
                if (e.target.files) void addFiles(e.target.files);
                e.target.value = "";
              }}
            />
            {props.onWorkspace ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    className={cn(ghostSelect, "inline-flex max-w-[9.5rem] items-center")}
                    aria-label={copy.composer.workspace}
                    title={props.workspace || copy.composer.workspaceHint}
                    disabled={props.disabled}
                  >
                    <Folder className="size-3 shrink-0 opacity-70" />
                    <span className="truncate">{displayWorkspace(props.workspace, copy.composer.workspace)}</span>
                    <ChevronDown className="size-3 shrink-0 opacity-70" />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" side="top" className="w-64">
                  {(props.workspaces || []).map((p) => (
                    <DropdownMenuItem key={p} onSelect={() => props.onWorkspace?.(p)}>
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate">{displayWorkspace(p, p)}</span>
                        <span className="truncate text-[11px] text-muted">{p}</span>
                      </span>
                      {p === props.workspace ? <Check className="size-3.5 shrink-0" /> : null}
                    </DropdownMenuItem>
                  ))}
                  {(props.workspaces?.length && props.onBrowseWorkspace) ? <DropdownMenuSeparator /> : null}
                  {props.onBrowseWorkspace ? (
                    <DropdownMenuItem onSelect={() => props.onBrowseWorkspace?.()}>
                      {copy.composer.browseWorkspace}
                    </DropdownMenuItem>
                  ) : null}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null}
            {videoing ? null : (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className={cn(
                    ghostSelect,
                    "inline-flex items-center",
                    plan && "bg-lift text-foreground hover:text-foreground",
                  )}
                  aria-label={copy.composer.mode}
                  disabled={props.disabled}
                >
                  {plan ? copy.composer.plan : copy.composer.agent}
                  <ChevronDown className="size-3 opacity-70" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" side="top" className="w-56">
                <DropdownMenuItem
                  className="items-start"
                  onSelect={() => setPlan(false)}
                >
                  <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <span>{copy.composer.agent}</span>
                    <span className="text-[11px] text-muted">{copy.composer.agentHint}</span>
                  </span>
                  {!plan ? <Check className="mt-0.5 size-3.5 shrink-0" /> : null}
                </DropdownMenuItem>
                <DropdownMenuItem
                  className="items-start"
                  onSelect={() => setPlan(true)}
                >
                  <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <span>{copy.composer.plan}</span>
                    <span className="text-[11px] text-muted">{copy.composer.planHint}</span>
                  </span>
                  {plan ? <Check className="mt-0.5 size-3.5 shrink-0" /> : null}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            )}
            </div>
            <span className="ml-auto" />
            {props.running ? (
              <button
                type="button"
                className="h-6 rounded-md px-1.5 text-[11px] font-medium text-muted transition-colors hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30"
                disabled={props.disabled || !hasPayload}
                onClick={() => send(true)}
                aria-label={copy.composer.steer}
              >
                {copy.composer.steer}
              </button>
            ) : null}
            {modelOptions.length > 0 && props.onModel ? (
              <Select value={currentModel || modelOptions[0]} onValueChange={(v) => props.onModel?.(v)} disabled={props.disabled}>
                <SelectTrigger className={cn(ghostSelect, "max-w-[8.5rem]")} aria-label={copy.composer.model} title={currentModel}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {modelOptions.map((m) => {
                    const label = lookupCatalogModel(MODELS_DEV_SNAPSHOT, props.provider || "", m)?.model.name || m;
                    return <SelectItem key={m} value={m}>{label}</SelectItem>;
                  })}
                </SelectContent>
              </Select>
            ) : currentModel ? (
              <span className="hidden max-w-[8.5rem] truncate px-1.5 text-[11px] text-muted sm:inline" title={currentModel}>{modelLabel}</span>
            ) : null}
            {props.onAuthMode ? (
              <Select
                value={parseComposerAuth(props.authMode)}
                onValueChange={(v) => props.onAuthMode?.(v as AuthMode)}
                disabled={props.disabled}
              >
                <SelectTrigger className={cn(ghostSelect, "max-w-[7.5rem]")} aria-label={copy.composer.authMode}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="default">{copy.composer.authDefault}</SelectItem>
                  <SelectItem value="auto_edit">{copy.composer.authAutoEdit}</SelectItem>
                  <SelectItem value="full">{copy.composer.authFull}</SelectItem>
                  <SelectItem value="ask">{copy.composer.authAsk}</SelectItem>
                </SelectContent>
              </Select>
            ) : null}
            <button
              type="button"
              className="send-orb shrink-0 disabled:cursor-default"
              data-live={props.running || canSend ? "true" : "false"}
              disabled={props.running ? false : !canSend}
              onClick={() => {
                if (props.running) {
                  props.onStop();
                  return;
                }
                send();
              }}
              aria-label={props.running ? copy.composer.stop : copy.composer.send}
            >
              <IconSwap
                on={props.running}
                className="size-3.5"
                off={<ArrowUp className="size-3.5" />}
                live={<Square className="size-2.5 fill-current" />}
              />
            </button>
          </div>
        </div>
        </div>
        <QueueDock
          items={props.queueItems || []}
          onCancel={props.onQueueCancel}
          onReorder={props.onQueueReorder}
          copy={copy}
        />
        <ContextMeter
          ctx={props.ctx}
          draftTokens={estimateTokens(composeDraft(chips, value))}
          fallbackWindow={fallbackWindow}
          queued={props.queued}
          running={props.running}
          compact={props.compact}
          copy={copy}
        />
      </div>
    </div>
  );
}

function QueueDock(props: {
  items: { id?: string; text?: string; plan?: boolean }[];
  onCancel?: (id: string) => void;
  onReorder?: (id: string, delta: number) => void;
  copy: Copy;
}) {
  if (!props.items.length) return null;
  return (
    <div className="mt-1.5 flex min-w-0 flex-col gap-1" data-testid="composer-queue">
      {props.items.map((it, i) => {
        const id = it.id || `q-${i}`;
        return (
          <div key={id} className="flex min-w-0 items-center gap-1 rounded-lg bg-lift/70 px-2 py-1 text-[12px]">
            <span className="min-w-0 flex-1 truncate text-foreground/85">{it.text || props.copy.composer.queue}</span>
            {props.onReorder ? (
              <>
                <button type="button" className="grid size-6 place-items-center rounded-md text-muted hover:bg-background hover:text-foreground" aria-label={props.copy.composer.queueUp} disabled={i === 0} onClick={() => props.onReorder?.(id, -1)}>
                  <ChevronUp className="size-3.5" />
                </button>
                <button type="button" className="grid size-6 place-items-center rounded-md text-muted hover:bg-background hover:text-foreground" aria-label={props.copy.composer.queueDown} disabled={i === props.items.length - 1} onClick={() => props.onReorder?.(id, 1)}>
                  <ChevronDown className="size-3.5" />
                </button>
              </>
            ) : null}
            {props.onCancel ? (
              <button type="button" className="grid size-6 place-items-center rounded-md text-muted hover:bg-danger/10 hover:text-danger" aria-label={props.copy.composer.queueCancel} onClick={() => props.onCancel?.(id)}>
                <X className="size-3.5" />
              </button>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function toolClass() {
  return "flex size-7 cursor-pointer items-center justify-center rounded-lg text-muted transition-[background-color,color,transform] duration-200 ease-[var(--ease-out)] hover:bg-lift hover:text-foreground active:scale-[0.96] disabled:pointer-events-none disabled:opacity-30";
}

function parseComposerAuth(mode?: string): AuthMode {
  const s = (mode || "").toLowerCase().replace(/[-\s]/g, "_");
  if (s === "ask" || s === "ask_every_time" || s === "strict") return "ask";
  if (s === "auto_edit" || s === "autoedit") return "auto_edit";
  if (s === "full" || s === "full_access") return "full";
  return "default";
}

function MentionGlyph({ kind }: { kind: MentionKind }) {
  const cls = "size-3 shrink-0 opacity-80";
  if (kind === "folder") return <Folder className={cls} />;
  if (kind === "skill") return <BookOpen className={cls} />;
  if (kind === "harness") return <Boxes className={cls} />;
  return <FileText className={cls} />;
}

function AttachmentChip({ att, onRemove }: { att: Attachment; onRemove: () => void }) {
  const name = att.name || att.path || "file";
  const preview = att.mime?.startsWith("image/") && att.data_b64
    ? `data:${att.mime};base64,${att.data_b64}`
    : "";
  if (preview) {
    return (
      <button type="button" className="relative h-11 w-11 overflow-hidden rounded-lg" onClick={onRemove} aria-label={name}>
        <img src={preview} alt="" className="h-full w-full object-cover" />
        <span className="absolute right-0.5 top-0.5 grid size-3.5 place-items-center rounded-full bg-card/90 text-muted">
          <X className="size-2.5" />
        </span>
      </button>
    );
  }
  return (
    <button
      type="button"
      className="inline-flex max-w-[11rem] items-center gap-1 rounded-md bg-lift px-1.5 py-0.5 text-[11px] text-muted hover:text-foreground"
      onClick={onRemove}
      aria-label={name}
    >
      {att.mime?.startsWith("image/") ? <Camera className="size-3 shrink-0" /> : <Paperclip className="size-3 shrink-0" />}
      <span className="truncate">{name}</span>
      <X className="size-3 shrink-0 opacity-70" />
    </button>
  );
}

function CommandWeld(props: {
  label: string;
  items: { token: string; hint: string; label?: string; kind?: MentionKind }[];
  hi: number;
  focused: boolean;
  empty?: string;
  onPick: (token: string) => void;
}) {
  const activeRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: "nearest" });
  }, [props.hi]);
  return (
    <div
      role="listbox"
      aria-label={props.label}
      className={cn(
        "absolute -inset-x-px bottom-[calc(100%-1px)] z-10 max-h-52 overflow-auto rounded-t-[var(--radius-composer)] border border-b-0 bg-input-bar",
        props.focused ? "border-foreground/20" : "border-border",
      )}
    >
      {props.items.length ? props.items.map((m, i) => (
        <PopupRow
          key={m.token + i}
          item={m}
          active={i === props.hi}
          rowRef={i === props.hi ? activeRef : undefined}
          onPick={() => props.onPick(m.token)}
        />
      )) : props.empty ? (
        <div className="px-3 py-2 text-[11px] text-muted">{props.empty}</div>
      ) : null}
    </div>
  );
}

function PopupRow(props: {
  item: { token: string; hint: string; label?: string; kind?: MentionKind };
  active: boolean;
  rowRef?: Ref<HTMLButtonElement>;
  onPick: () => void;
}) {
  return (
    <button
      ref={props.rowRef}
      type="button"
      role="option"
      aria-selected={props.active}
      aria-label={`${props.item.token} ${props.item.label || ""}`.trim()}
      className={cn(
        "flex w-full items-center gap-2 px-3 py-1.5 text-left hover:bg-lift",
        props.active && "bg-lift",
      )}
      onMouseDown={(e) => {
        e.preventDefault();
        props.onPick();
      }}
    >
      {props.item.kind ? <MentionGlyph kind={props.item.kind} /> : null}
      <span className="min-w-0 flex-1 truncate text-[12px] text-foreground">{props.item.label || props.item.token}</span>
      {props.item.hint ? <span className="max-w-[55%] truncate text-[11px] text-muted">{props.item.hint}</span> : null}
    </button>
  );
}

function isDomFile(v: unknown): v is File {
  return typeof File !== "undefined" && v instanceof File;
}

async function fileToAttachment(f: File): Promise<Attachment> {
  const buf = await f.arrayBuffer();
  const cap = f.type.startsWith("image/") ? 2 * 1024 * 1024 : 96 * 1024;
  const slice = buf.byteLength > cap ? buf.slice(0, cap) : buf;
  const bytes = new Uint8Array(slice);
  let bin = "";
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
  return { name: f.name || (f.type.startsWith("image/") ? "image.png" : "file"), mime: f.type, data_b64: btoa(bin) };
}

function ingestPasteFiles(e: ClipboardEvent<HTMLTextAreaElement>, add: (files: File[]) => void): boolean {
  const files: File[] = [];
  const items = e.clipboardData?.items;
  if (items) {
    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.kind === "file" || item.type.startsWith("image/")) {
        const f = item.getAsFile();
        if (f) files.push(f);
      }
    }
  }
  const listed = e.clipboardData?.files;
  if (listed?.length) {
    for (const f of Array.from(listed)) {
      if (!files.some((x) => x === f || (x.name === f.name && x.size === f.size))) files.push(f);
    }
  }
  if (!files.length) return false;
  add(files);
  return true;
}

const SLICE_TONE: Record<CtxSliceId, string> = {
  system: "bg-[var(--ctx-system)]",
  tools: "bg-[var(--ctx-tools)]",
  dynamic: "bg-[var(--ctx-dynamic)]",
  chat: "bg-[var(--ctx-chat)]",
  free: "bg-[var(--ctx-free)]",
};

function sliceLabel(id: CtxSliceId, copy: Copy): string {
  if (id === "system") return copy.composer.ctxSystem;
  if (id === "tools") return copy.composer.ctxTools;
  if (id === "dynamic") return copy.composer.ctxDynamic;
  if (id === "chat") return copy.composer.ctxChat;
  return copy.composer.ctxFree;
}

function ContextMeter(props: {
  ctx?: ContextUsage;
  draftTokens?: number;
  fallbackWindow: number;
  queued?: number;
  running?: boolean;
  compact?: boolean;
  copy: Copy;
}) {
  const copy = props.copy;
  const br = contextBreakdown(props.ctx, props.fallbackWindow, props.draftTokens || 0);
  const queued = props.queued || 0;
  if (br.capacity <= 0) {
    if (!props.running && queued <= 0) return null;
    return (
      <div className="mt-1.5 flex min-w-0 items-center gap-2 px-0.5 text-[11px] text-muted">
        {props.running ? <span>{copy.composer.queueHint}</span> : null}
        {queued > 0 ? <span>{copy.composer.queuedCount.replace("{n}", String(queued))}</span> : null}
      </div>
    );
  }

  // An untouched window is not information. Show the meter once tokens exist,
  // a turn is running, or something is queued behind it.
  if (br.used <= 0 && !props.running && queued <= 0) return null;

  const visible = br.slices.filter((s) => s.tokens > 0);
  const legend = props.compact ? visible.filter((s) => s.id !== "free") : visible;
  const tooltip = (
    <div className="flex min-w-[11rem] flex-col gap-1 py-0.5">
      {br.slices.filter((s) => s.tokens > 0 || s.id === "free").map((s) => (
        <div key={s.id} className="flex items-center justify-between gap-6 text-[11px] tabular-nums">
          <span className="flex items-center gap-1.5 text-muted">
            <i className={cn("size-1.5 rounded-full", SLICE_TONE[s.id])} aria-hidden />
            <span>{sliceLabel(s.id, copy)}</span>
          </span>
          <span className="text-foreground">{formatTokens(s.tokens)}</span>
        </div>
      ))}
      <div className="mt-0.5 flex items-center justify-between gap-6 border-t border-border/70 pt-1 text-[11px] tabular-nums">
        <span className="text-muted">{copy.composer.ctxWindow}</span>
        <span className="text-foreground">{formatTokens(br.capacity)}</span>
      </div>
      {br.providerPrompt > 0 ? (
        <div className="flex items-center justify-between gap-6 text-[11px] tabular-nums">
          <span className="text-muted">{copy.composer.ctxProvider}</span>
          <span className="text-foreground">{formatTokens(br.providerPrompt)}</span>
        </div>
      ) : null}
      {br.cacheReported ? (
        <div className="flex items-center justify-between gap-6 text-[11px] tabular-nums">
          <span className="text-muted">{copy.composer.ctxCache}</span>
          <span className="text-foreground">{formatTokens(br.cachedTokens)}</span>
        </div>
      ) : br.providerPrompt > 0 ? (
        <div className="flex items-center justify-between gap-6 text-[11px] tabular-nums">
          <span className="text-muted">{copy.composer.ctxCache}</span>
          <span className="text-foreground">{copy.composer.ctxCacheUnknown}</span>
        </div>
      ) : null}
      {br.elided > 0 ? (
        <div className="flex items-center justify-between gap-6 text-[11px] tabular-nums">
          <span className="text-muted">{copy.composer.ctxElided}</span>
          <span className="text-foreground">{br.elided}</span>
        </div>
      ) : null}
      {br.layers.length ? <div className="text-[10px] text-muted">{br.layers.join(" · ")}</div> : null}
      {br.note ? <div className="text-[10px] text-muted">{br.note}</div> : null}
    </div>
  );

  return (
    <Tooltip content={tooltip} side="top" className="max-w-none px-2.5 py-2">
      <div
        className="mt-1.5 min-w-0 px-0.5"
        role="group"
        aria-label={copy.composer.context}
      >
        <div
          className="flex h-1.5 w-full overflow-hidden rounded-full bg-[var(--ctx-free)]"
          role="meter"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={br.pct}
          aria-label={`${copy.composer.context} ${br.pct}%`}
        >
          {visible.map((s) => (
            <span
              key={s.id}
              className={cn("h-full min-w-[2px] transition-[flex-grow] duration-300", SLICE_TONE[s.id], s.id === "free" && "min-w-0")}
              style={{ flexGrow: Math.max(s.tokens, 1), flexBasis: 0 }}
              title={`${sliceLabel(s.id, copy)} ${formatTokens(s.tokens)}`}
            />
          ))}
        </div>
        <div className="mt-1.5 flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-0.5 text-[11px] leading-none tabular-nums text-muted">
          {!props.compact
            ? legend.map((s) => (
              <span key={s.id} className="inline-flex items-center gap-1.5">
                <i className={cn("size-1.5 rounded-full", SLICE_TONE[s.id])} aria-hidden />
                <span>{sliceLabel(s.id, copy)}</span>
                <span className="text-foreground/75">{formatTokens(s.tokens)}</span>
              </span>
            ))
            : null}
          <span className={cn("inline-flex items-center gap-1.5", !props.compact && "ml-auto")}>
            <span className={cn(br.pct >= 90 ? "text-danger" : "text-foreground/80")}>{br.pct}%</span>
            <span>{formatTokens(br.used)} / {formatTokens(br.capacity)}</span>
          </span>
          {queued > 0 ? <span>{copy.composer.queuedCount.replace("{n}", String(queued))}</span> : null}
        </div>
      </div>
    </Tooltip>
  );
}
