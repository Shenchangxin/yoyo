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
import { formatTokens, lookupCatalogModel, mergeModelIds, modelsForProvider } from "../lib/models-dev";
import { MODELS_DEV_SNAPSHOT } from "../lib/models-dev.snapshot";

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
  const catalogModels = modelsForProvider(MODELS_DEV_SNAPSHOT, props.provider || "openai").map((m) => m.id);
  const models = mergeModelIds(props.model, props.models, catalogModels);
  const currentModel = (props.model || "").trim();
  const modelOptions = currentModel && !models.includes(currentModel) ? [currentModel, ...models] : models;
  const slashOpen = hint === "slash";
  const meta = lookupCatalogModel(MODELS_DEV_SNAPSHOT, props.provider || "openai", currentModel);
  const budget = (props.ctx?.budget && props.ctx.budget > 0) ? props.ctx.budget : (meta?.model.contextWindow || 0);
  const used = props.ctx?.tokens || 0;
  const pct = budget > 0 ? Math.min(100, Math.round((used / budget) * 100)) : 0;

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
    <div className={cn("no-drag shrink-0", props.compact ? "px-2 pb-2 pt-1" : "px-4 pb-4 pt-1")}>
      <div className={cn("mx-auto w-full", !props.compact && "max-w-2xl")}>
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
                  {modelOptions.map((m) => (
                    <SelectItem key={m} value={m}>{m}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : currentModel ? (
              <span className="hidden truncate px-1.5 text-[11px] text-muted sm:inline">{props.model}</span>
            ) : null}
            {props.running ? (
              <button
                type="button"
                className="h-7 rounded-lg px-1.5 text-[12px] font-medium text-muted hover:bg-lift hover:text-foreground"
                onClick={() => { props.onSend({ steer: true, attachments: atts }); setAtts([]); }}
                disabled={!value.trim()}
              >
                {copy.composer.steer}
              </button>
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
        {budget > 0 ? (
          <div className="mt-1.5 flex items-center gap-2 px-1.5 text-[11px] tabular-nums text-muted" title={props.ctx?.note || copy.composer.context}>
            <span className="h-1 w-16 overflow-hidden rounded-full bg-lift">
              <span className="block h-full rounded-full bg-accent" style={{ width: pct + "%" }} />
            </span>
            <span>{formatTokens(used)} / {formatTokens(budget)}</span>
            {meta?.model.name ? <span className="min-w-0 truncate">{meta.model.name}</span> : null}
          </div>
        ) : null}
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
