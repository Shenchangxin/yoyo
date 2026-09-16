import { useEffect, useMemo, useRef, useState } from "react";
import { ArrowUp, AtSign, Paperclip, Square } from "lucide-react";
import { Button } from "../components/ui/button";
import { Textarea } from "../components/ui/input";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { useUI } from "../lib/store";
import { filterSlash, slashQuery } from "../lib/slash";
import type { Attachment, FileHit, SkillInfo } from "../lib/protocol";

const MENTIONS = [
  { token: "@file:", hint: "pin a file (bounded)" },
  { token: "@folder:", hint: "pin a directory listing" },
  { token: "@skill:", hint: "inject a skill body" },
  { token: "@harness", hint: "inject active harness summary" },
];

export function Composer(props: {
  draftKey: string;
  running: boolean;
  disabled: boolean;
  disabledReason?: string;
  model?: string;
  models?: string[];
  queued?: number;
  skills?: SkillInfo[];
  files?: FileHit[];
  onSearchFiles?: (q: string) => void;
  onSend: (opts?: { steer?: boolean; attachments?: Attachment[] }) => void;
  onStop: () => void;
  onSlash?: (cmd: string, rest: string) => void;
  onModel?: (model: string) => void;
  onPickFiles?: () => Promise<Attachment[]>;
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

  const slashPrefix = slashQuery(value);
  const slashItems = slashPrefix ? filterSlash(slashPrefix) : [];
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
    return MENTIONS.filter((x) => x.token.startsWith(token) || token === "@");
  }, [value, props.files, props.skills]);

  const popup = hint === "slash" ? slashItems.map((s) => ({ token: s.cmd, hint: s.hint })) : mentionItems;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  }, [value]);

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

  return (
    <div className="no-drag shrink-0 px-4 pb-4 pt-2">
      <div className="mx-auto w-full max-w-3xl">
        <div
          className={cn(
            "relative rounded-[var(--radius-composer)] border bg-panel transition-[border-color,box-shadow] duration-150",
            focused ? "border-accent shadow-[0_0_0_3px_color-mix(in_srgb,var(--accent)_22%,transparent)]" : "border-border",
            props.disabled && "opacity-70",
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
          {atts.length ? (
            <div className="flex flex-wrap gap-1 px-3 pt-2">
              {atts.map((a, i) => (
                <button
                  type="button"
                  key={`${a.name}-${i}`}
                  className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted"
                  onClick={() => setAtts((prev) => prev.filter((_, j) => j !== i))}
                >
                  {a.name || a.path || "file"} ×
                </button>
              ))}
            </div>
          ) : null}
          <Textarea
            ref={ref}
            rows={2}
            value={value}
            disabled={props.disabled}
            placeholder={props.disabled ? (props.disabledReason || copy.composer.disabled) : copy.composer.placeholder}
            className="prose-select min-h-[52px] px-4 pt-3 pb-1"
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
                props.onSend({ attachments: atts });
                setAtts([]);
              }
            }}
          />
          {hint && popup.length ? (
            <div role="listbox" aria-label={hint === "slash" ? copy.composer.commands : copy.composer.mentions} className="absolute bottom-14 left-4 z-10 w-80 overflow-hidden rounded-xl border border-border bg-sidebar">
              {popup.slice(0, 12).map((m, i) => (
                <button
                  type="button"
                  key={m.token + i}
                  role="option"
                  aria-selected={i === hi}
                  className={cn("flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left hover:bg-lift", i === hi && "bg-lift")}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    if (hint === "slash") applySlash(m.token);
                    else insertMention(m.token);
                  }}
                >
                  <span className="font-mono text-xs text-foreground">{m.token}</span>
                  <span className="text-[11px] text-muted">{m.hint}</span>
                </button>
              ))}
            </div>
          ) : null}
          <div className="flex items-center gap-2 px-2 pb-2">
            <Button variant="ghost" size="icon" aria-label={copy.composer.mention} disabled={props.disabled} onClick={() => { setHint("mention"); ref.current?.focus(); }}>
              <AtSign />
            </Button>
            <Button
              variant="ghost"
              size="icon"
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
              <Paperclip />
            </Button>
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
            <button
              type="button"
              className={cn(
                "rounded-full px-3 py-1 text-xs",
                plan ? "bg-accent/15 text-accent" : "bg-lift text-muted hover:text-foreground",
              )}
              onClick={() => setPlan(!plan)}
              aria-pressed={plan}
            >
              {plan ? copy.composer.plan : copy.composer.agent}
            </button>
            {props.models && props.models.length && props.onModel ? (
              <select
                className="max-w-[140px] truncate rounded-full bg-lift px-2 py-1 text-[11px] text-muted"
                aria-label={copy.composer.model}
                value={props.model || props.models[0]}
                onChange={(e) => props.onModel?.(e.target.value)}
              >
                {(props.model && !props.models.includes(props.model) ? [props.model, ...props.models] : props.models).map((m) => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
            ) : props.model ? (
              <span className="hidden truncate text-[11px] text-muted sm:inline">{props.model}</span>
            ) : null}
            <span className="hidden text-[11px] text-muted sm:inline">
              {props.disabled
                ? (props.disabledReason || copy.composer.workspaceRequired)
                : props.running
                  ? copy.composer.queueHint
                  : plan
                    ? copy.composer.planHint
                    : copy.composer.enter}
              {props.queued ? ` · ${props.queued}` : ""}
            </span>
            <span className="ml-auto" />
            {props.running ? (
              <>
                <Button variant="lift" size="sm" onClick={() => props.onSend({ steer: true, attachments: atts })} disabled={!value.trim()}>
                  {copy.composer.steer}
                </Button>
                <Button variant="danger" size="send" onClick={props.onStop} aria-label={copy.composer.stop}>
                  <Square className="size-3 fill-current" />
                </Button>
              </>
            ) : null}
            <Button
              size="send"
              disabled={!canSend}
              onClick={() => {
                props.onSend({ attachments: atts });
                setAtts([]);
              }}
              aria-label={props.running ? copy.composer.queue : copy.composer.send}
              className={cn(!canSend && "bg-lift text-muted")}
            >
              <ArrowUp className="size-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
