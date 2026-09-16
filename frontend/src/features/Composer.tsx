import { useEffect, useRef, useState } from "react";
import { ArrowUp, AtSign, Square } from "lucide-react";
import { Button } from "../components/ui/button";
import { Textarea } from "../components/ui/input";
import { cn } from "../lib/utils";
import { copy } from "../lib/copy";
import { useUI } from "../lib/store";

const MENTIONS = [
  { token: "@file:", hint: "pin a file (bounded)" },
  { token: "@folder:", hint: "pin a directory listing" },
  { token: "@harness", hint: "inject active harness summary" },
];

export function Composer(props: {
  draftKey: string;
  running: boolean;
  disabled: boolean;
  disabledReason?: string;
  model?: string;
  onSend: () => void;
  onStop: () => void;
}) {
  const value = useUI((s) => s.drafts[props.draftKey] || "");
  const plan = useUI((s) => s.plan);
  const setDraft = useUI((s) => s.setDraft);
  const setPlan = useUI((s) => s.setPlan);
  const onChange = (v: string) => setDraft(props.draftKey, v);
  const [hint, setHint] = useState(false);
  const [hi, setHi] = useState(0);
  const [focused, setFocused] = useState(false);
  const ref = useRef<HTMLTextAreaElement>(null);
  const canSend = !props.disabled && !props.running && !!value.trim();

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  }, [value]);

  function insertMention(token: string) {
    const next = value.replace(/@\w*$/, "") + token;
    onChange(next);
    setHint(false);
    requestAnimationFrame(() => ref.current?.focus());
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
        >
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
            onChange={(e) => {
              const v = e.target.value;
              onChange(v);
              const last = v.split("\n").pop() || "";
              const show = last === "@" || /(?:^|\s)@\w*$/.test(last);
              setHint(show);
              setHi(0);
            }}
            onKeyDown={(e) => {
              if (hint && (e.key === "ArrowDown" || e.key === "ArrowUp")) {
                e.preventDefault();
                setHi((n) => (e.key === "ArrowDown" ? (n + 1) % MENTIONS.length : (n + MENTIONS.length - 1) % MENTIONS.length));
                return;
              }
              if (hint && (e.key === "Tab" || e.key === "Enter") && !e.shiftKey) {
                e.preventDefault();
                insertMention(MENTIONS[hi].token);
                return;
              }
              if (e.key === "Escape") {
                setHint(false);
                return;
              }
              if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
                e.preventDefault();
                if (canSend) props.onSend();
              }
            }}
          />
          {hint ? (
            <div role="listbox" aria-label={copy.composer.mentions} className="absolute bottom-14 left-4 z-10 w-72 overflow-hidden rounded-xl border border-border bg-sidebar">
              {MENTIONS.map((m, i) => (
                <button
                  type="button"
                  key={m.token}
                  role="option"
                  aria-selected={i === hi}
                  className={cn("flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left hover:bg-lift", i === hi && "bg-lift")}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    insertMention(m.token);
                  }}
                >
                  <span className="font-mono text-xs text-foreground">{m.token}</span>
                  <span className="text-[11px] text-muted">{m.hint}</span>
                </button>
              ))}
            </div>
          ) : null}
          <div className="flex items-center gap-2 px-2 pb-2">
            <Button variant="ghost" size="icon" aria-label={copy.composer.mention} disabled={props.disabled} onClick={() => { setHint(true); ref.current?.focus(); }}>
              <AtSign />
            </Button>
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
            <span className="hidden text-[11px] text-muted sm:inline">
              {props.disabled ? (props.disabledReason || copy.composer.workspaceRequired) : plan ? copy.composer.planHint : copy.composer.enter}
            </span>
            {props.model ? <span className="ml-auto hidden truncate text-[11px] text-muted sm:inline">{props.model}</span> : <span className="ml-auto" />}
            {props.running ? (
              <Button variant="danger" size="send" onClick={props.onStop} aria-label={copy.composer.stop}>
                <Square className="size-3 fill-current" />
              </Button>
            ) : (
              <Button
                size="send"
                disabled={!canSend}
                onClick={props.onSend}
                aria-label={copy.composer.send}
                className={cn(!canSend && "bg-lift text-muted")}
              >
                <ArrowUp className="size-4" />
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
