import { useEffect, useMemo, useRef, useState } from "react";
import { ArrowUp, AtSign, Paperclip, Square, X } from "lucide-react";
import { Textarea } from "../components/ui/input";
import { Tooltip } from "../components/ui/tooltip";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import { filterSlash, slashCatalog, slashQuery } from "../lib/slash";
import type { Attachment, ContextUsage, FileHit, SkillInfo } from "../lib/protocol";
import { formatTokens, lookupCatalogModel, composerModelIds } from "../lib/models-dev";
import { MODELS_DEV_SNAPSHOT } from "../lib/models-dev.snapshot";
import { contextBreakdown, type CtxSliceId } from "../lib/context-usage";
import type { Copy } from "../lib/copy";

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
  skills?: SkillInfo[];
  files?: FileHit[];
  onSearchFiles?: (q: string) => void;
  onSend: (opts?: { steer?: boolean; attachments?: Attachment[] }) => void;
  onStop: () => void;
  onSlash?: (cmd: string, rest: string) => void;
  onModel?: (model: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
  compact?: boolean;
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
  const [fileQ, setFileQ] = useState("");
  const ref = useRef<HTMLTextAreaElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const copy = useCopy();
  const canSend = !props.disabled && !!value.trim();
  const currentModel = (props.model || "").trim();
  const modelOptions = composerModelIds(MODELS_DEV_SNAPSHOT, props.provider, currentModel, props.models);
  const slashOpen = hint === "slash";
  const meta = lookupCatalogModel(MODELS_DEV_SNAPSHOT, props.provider || "openai", currentModel);
  const fallbackWindow = meta?.model.contextWindow || 0;

  const slashPrefix = slashQuery(value);
  const slashItems = slashPrefix ? filterSlash(slashPrefix, slashCatalog(copy)) : [];
  const mentionItems = useMemo(() => {
    const last = (value.split("\n").pop() || "");
    const m = last.match(/(?:^|\s)(@(?:file:|folder:|skill:)?[\w./\\-]*)$/);
    const token = m?.[1] || "";
    if (token.startsWith("@file:") || token.startsWith("@folder:")) {
      return (props.files || []).map((f) => ({
        token: `${token.startsWith("@folder:") ? "@folder:" : "@file:"}${f.path}`,
        hint: f.kind,
      }));
    }
    if (token.startsWith("@skill:")) {
      const q = token.slice("@skill:".length).toLowerCase();
      return (props.skills || [])
        .filter((s) => !q || s.name.toLowerCase().includes(q))
        .map((s) => ({ token: `@skill:${s.name}`, hint: s.description }));
    }
    const mentions = [
      { token: "@file:", hint: copy.composer.mentionFile },
      { token: "@folder:", hint: copy.composer.mentionFolder },
      { token: "@skill:", hint: copy.composer.mentionSkill },
      { token: "@harness", hint: copy.composer.mentionHarness },
    ];
    return mentions.filter((x) => x.token.startsWith(token) || token === "@");
  }, [value, props.files, props.skills, copy]);

  const popup = hint === "slash" ? slashItems.map((s) => ({ token: s.cmd, hint: s.hint })) : mentionItems;
  const showPopup = !!hint && popup.length > 0;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 140) + "px";
  }, [value]);

  function insertAtTrigger() {
    const needsSpace = value.length > 0 && !/\s$/.test(value);
    onChange(value + (needsSpace ? " @" : "@"));
    setHint("mention");
    requestAnimationFrame(() => ref.current?.focus());
  }

  function insertMention(token: string) {
    const next = value.replace(/@[^\s]*$/, "") + token + (token.endsWith(":") ? "" : " ");
    onChange(next);
    setHint(null);
    requestAnimationFrame(() => ref.current?.focus());
  }

  function applySlash(cmd: string) {
    onChange("");
    setHint(null);
    props.onSlash?.(cmd, "");
  }

  async function addFiles(list: FileList | Attachment[]) {
    if (Array.isArray(list)) {
      setAtts((prev) => [...prev, ...list]);
      return;
    }
    const next: Attachment[] = [];
    for (const f of Array.from(list)) {
      const buf = await f.arrayBuffer();
      const bytes = new Uint8Array(buf.slice(0, 96 * 1024));
      let bin = "";
      for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
      next.push({ name: f.name, mime: f.type, data_b64: btoa(bin) });
    }
    setAtts((prev) => [...prev, ...next]);
  }

  function send() {
    if (!canSend) return;
    props.onSend({ attachments: atts });
    setAtts([]);
  }

  return (
    <div className={cn("no-drag shrink-0", props.compact ? "px-3 pb-2 pt-1" : "px-5 pb-4 pt-1 sm:px-8 lg:px-10")}>
      <div className="mx-auto w-full min-w-0">
        <div
          className={cn(
            "relative z-10 overflow-visible border bg-input-bar shadow-[var(--shadow-composer)] transition-[border-color,box-shadow] duration-200",
            slashOpen && showPopup ? "rounded-b-[20px] rounded-t-none" : "rounded-[20px]",
            focused ? "border-accent/20" : "border-border",
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
            <div
              role="listbox"
              aria-label={copy.composer.mentions}
              className="absolute bottom-full left-0 right-0 z-10 mb-1.5 max-h-56 overflow-auto rounded-2xl border border-border bg-input-bar shadow-[var(--shadow-composer)]"
            >
              {popup.slice(0, 12).map((m, i) => (
                <PopupRow key={m.token + i} item={m} active={i === hi} onPick={() => insertMention(m.token)} />
              ))}
            </div>
          ) : null}
          {atts.length ? (
            <div className="flex flex-wrap gap-1 px-3 pt-3">
              {atts.map((a, i) => (
                <button
                  type="button"
                  key={`${a.name}-${i}`}
                  className="inline-flex items-center gap-1 rounded-full bg-lift px-2.5 py-0.5 text-[11px] text-muted hover:text-foreground"
                  onClick={() => setAtts((prev) => prev.filter((_, j) => j !== i))}
                >
                  {a.name || a.path || "file"}
                  <X className="size-3" />
                </button>
              ))}
            </div>
          ) : null}
          <Textarea
            ref={ref}
            rows={1}
            value={value}
            disabled={props.disabled}
            placeholder={props.disabled ? (props.disabledReason || copy.composer.disabled) : copy.composer.placeholder}
            className="prose-select min-h-6 px-4 pt-3 pb-1 text-[13.5px] leading-[1.6] placeholder:text-muted/45"
            aria-label={copy.composer.message}
            onFocus={() => setFocused(true)}
            onBlur={() => setFocused(false)}
            onPaste={(e) => {
              const files = e.clipboardData?.files;
              if (files && files.length) {
                e.preventDefault();
                void addFiles(files);
              }
            }}
            onChange={(e) => {
              const v = e.target.value;
              onChange(v);
              const last = v.split("\n").pop() || "";
              if (slashQuery(v)) {
                setHint("slash");
                setHi(0);
                return;
              }
              const show = last === "@" || /(?:^|\s)@[\w./:\\-]*$/.test(last);
              setHint(show ? "mention" : null);
              if (show) {
                const m = last.match(/@(?:file|folder):([\w./\\-]*)$/);
                const q = m?.[1] || "";
                if (q !== fileQ) {
                  setFileQ(q);
                  props.onSearchFiles?.(q);
                }
              }
              setHi(0);
            }}
            onKeyDown={(e) => {
              if (hint && popup.length && (e.key === "ArrowDown" || e.key === "ArrowUp")) {
                e.preventDefault();
                setHi((n) => (e.key === "ArrowDown" ? (n + 1) % popup.length : (n + popup.length - 1) % popup.length));
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
          <div className="flex flex-nowrap items-center gap-x-1.5 px-2 pb-2 pt-1 sm:px-2.5">
            <Tooltip content={copy.composer.mention}>
              <button
                type="button"
                className={toolClass(false)}
                aria-label={copy.composer.mention}
                disabled={props.disabled}
                onClick={insertAtTrigger}
              >
                <AtSign className="size-[17px]" />
              </button>
            </Tooltip>
            <Tooltip content={copy.composer.attach}>
              <button
                type="button"
                className={toolClass(false)}
                aria-label={copy.composer.attach}
                disabled={props.disabled}
                onClick={async () => {
                  if (props.onPickFiles) {
                    const picked = await props.onPickFiles();
                    if (picked.length) setAtts((prev) => [...prev, ...picked]);
                    return;
                  }
                  fileRef.current?.click();
                }}
              >
                <Paperclip className="size-[17px]" />
              </button>
            </Tooltip>
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
            <span className="mx-0.5 h-4 w-px bg-border/70" aria-hidden />
            <button
              type="button"
              className={cn(
                "h-7 rounded-lg px-1.5 text-[12px] font-medium transition-colors",
                plan ? "bg-accent/10 text-accent" : "text-foreground hover:bg-lift",
              )}
              onClick={() => setPlan(!plan)}
              aria-pressed={plan}
              disabled={props.disabled}
            >
              {plan ? copy.composer.plan : copy.composer.agent}
            </button>
            <span className="ml-auto" />
            {modelOptions.length > 0 && props.onModel ? (
              <Select value={currentModel || modelOptions[0]} onValueChange={(v) => props.onModel?.(v)} disabled={props.disabled}>
                <SelectTrigger className="h-7 min-w-0 max-w-[148px] rounded-full border-transparent bg-transparent px-1.5 text-[11px] text-muted hover:bg-lift" aria-label={copy.composer.model}>
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
              <span className="hidden truncate px-1.5 text-[11px] text-muted sm:inline">{props.model}</span>
            ) : null}
            {props.running ? (
              <span className="hidden max-w-[9rem] truncate text-[11px] text-muted lg:inline">{copy.composer.queueHint}</span>
            ) : null}
            <button
              type="button"
              className={cn(
                "grid size-8 shrink-0 place-items-center rounded-full transition-[background-color,color,box-shadow] duration-200",
                props.running || canSend
                  ? "bg-accent text-accent-fg shadow-[0_6px_18px_-6px_color-mix(in_srgb,var(--accent)_70%,transparent)]"
                  : "bg-[color-mix(in_srgb,var(--muted)_18%,transparent)] text-muted",
              )}
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
              {props.running ? <Square className="size-3 fill-current" /> : <ArrowUp className="size-4" />}
            </button>
          </div>
        </div>
        <ContextMeter
          ctx={props.ctx}
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

function toolClass(active: boolean) {
  return cn(
    "flex size-7 items-center justify-center rounded-lg transition-[transform,colors] duration-150 hover:scale-[1.06] active:scale-[0.92] disabled:pointer-events-none disabled:opacity-30",
    active ? "bg-accent/10 text-accent" : "text-foreground hover:bg-lift",
  );
}

function CommandWeld(props: {
  label: string;
  items: { token: string; hint: string }[];
  hi: number;
  focused: boolean;
  onPick: (token: string) => void;
}) {
  return (
    <div
      role="listbox"
      aria-label={props.label}
      className={cn(
        "absolute -inset-x-px bottom-[calc(100%-1px)] z-10 max-h-56 overflow-auto rounded-t-[20px] border border-b-0 bg-input-bar",
        props.focused ? "border-accent/20" : "border-border",
      )}
    >
      {props.items.slice(0, 12).map((m, i) => (
        <PopupRow key={m.token + i} item={m} active={i === props.hi} onPick={() => props.onPick(m.token)} />
      ))}
    </div>
  );
}

function PopupRow(props: { item: { token: string; hint: string }; active: boolean; onPick: () => void }) {
  return (
    <button
      type="button"
      role="option"
      aria-selected={props.active}
      className={cn("flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left hover:bg-lift", props.active && "bg-lift")}
      onMouseDown={(e) => {
        e.preventDefault();
        props.onPick();
      }}
    >
      <span className="font-mono text-xs text-foreground">{props.item.token}</span>
      <span className="text-[11px] text-muted">{props.item.hint}</span>
    </button>
  );
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
  fallbackWindow: number;
  queued?: number;
  running?: boolean;
  compact?: boolean;
  copy: Copy;
}) {
  const copy = props.copy;
  const br = contextBreakdown(props.ctx, props.fallbackWindow);
  const queued = props.queued || 0;
  if (br.capacity <= 0) {
    if (!props.running && queued <= 0) return null;
    return (
      <div className="mt-2 flex min-w-0 items-center gap-2 px-0.5 text-[11px] text-muted">
        {props.running ? <span>{copy.composer.queueHint}</span> : null}
        {queued > 0 ? <span>{copy.composer.queuedCount.replace("{n}", String(queued))}</span> : null}
      </div>
    );
  }

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
        className="mt-2 min-w-0 px-0.5"
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
