import { useEffect, useMemo, useRef, useState } from "react";
import { ListTree, PanelLeftClose } from "lucide-react";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { useMedia } from "../lib/media";
import { useUI } from "../lib/store";
import {
  MIN_OUTLINE_TURNS,
  OUTLINE_SEARCH_AT,
  isOutlineActive,
  outlineTurnKey,
  type OutlineStatus,
  type OutlineTurn,
} from "../lib/turn-outline";

export function TurnOutline(props: {
  turns: OutlineTurn[];
  activeKey: string;
  onJump: (turn: OutlineTurn) => void;
  status?: OutlineStatus;
}) {
  const copy = useCopy();
  const collapsed = useUI((s) => s.outlineCollapsed);
  const setCollapsed = useUI((s) => s.setOutlineCollapsed);
  const autoTicks = useMedia("(max-width: 719px)");
  const ticks = collapsed || autoTicks;
  const [query, setQuery] = useState("");
  const navRef = useRef<HTMLElement>(null);
  const rows = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return props.turns.map((t, i) => ({ t, i }));
    return props.turns
      .map((t, i) => ({ t, i }))
      .filter(({ t, i }) => t.title.toLowerCase().includes(q) || String(i + 1).includes(q));
  }, [props.turns, query]);

  useEffect(() => {
    const el = navRef.current?.querySelector("[aria-current='true']");
    el?.scrollIntoView({ block: "nearest" });
  }, [props.activeKey, ticks]);

  if (props.turns.length < MIN_OUTLINE_TURNS) return null;

  return (
    <aside
      className={cn(
        "flex h-full min-h-0 shrink-0 flex-col border-r border-border/40",
        ticks ? "w-6" : "w-[11.25rem]",
      )}
      data-testid="turn-outline"
    >
      {autoTicks ? null : (
        <div className={cn("flex h-8 shrink-0 items-center", ticks ? "justify-center" : "justify-between pl-2 pr-1")}>
          {ticks ? null : (
            <span className="text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/70">{copy.transcript.outline}</span>
          )}
          <button
            type="button"
            className="inline-flex size-6 cursor-pointer items-center justify-center rounded-md text-muted/70 hover:bg-lift/50 hover:text-muted"
            aria-expanded={!collapsed}
            aria-label={collapsed ? copy.transcript.outlineShow : copy.transcript.outlineHide}
            onClick={() => setCollapsed((v) => !v)}
          >
            {collapsed ? <ListTree className="size-3.5" aria-hidden /> : <PanelLeftClose className="size-3.5" aria-hidden />}
          </button>
        </div>
      )}
      {!ticks && props.turns.length >= OUTLINE_SEARCH_AT ? (
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={copy.transcript.outlineSearch}
          autoComplete="off"
          spellCheck={false}
          className="mx-1.5 mb-1 h-7 rounded-md border border-border/60 bg-transparent px-2 text-[12px] text-foreground placeholder:text-muted/60"
        />
      ) : null}
      <nav
        ref={navRef}
        aria-label={copy.transcript.outline}
        className={cn("min-h-0 flex-1 overflow-y-auto pb-3", ticks ? "px-0" : "px-1")}
      >
        {ticks ? (
          <ol className="flex flex-col items-center gap-[3px] pt-1">
            {props.turns.map((t, i) => {
              const key = outlineTurnKey(t);
              const active = isOutlineActive(t, props.activeKey);
              return (
                <li key={`${key}:${i}`}>
                  <button
                    type="button"
                    title={t.title || copy.transcript.outlineUntitled}
                    aria-label={copy.transcript.jumpTurn.replace("{n}", String(i + 1))}
                    aria-current={active ? "true" : undefined}
                    data-testid="turn-outline-item"
                    className="flex h-2.5 w-5 cursor-pointer items-center justify-center"
                    onClick={() => props.onJump(t)}
                  >
                    <span
                      className={cn(
                        "block w-[2px] rounded-full",
                        active ? "h-2.5 bg-accent" : "h-[7px] bg-foreground/18 hover:bg-foreground/40",
                        t.live && "bg-accent/70",
                        props.status === "working" && i === props.turns.length - 1 && "animate-pulse bg-accent",
                      )}
                    />
                  </button>
                </li>
              );
            })}
          </ol>
        ) : (
          <ol className="flex flex-col">
            {rows.map(({ t, i }) => {
              const key = outlineTurnKey(t);
              const active = isOutlineActive(t, props.activeKey);
              const label = t.title || copy.transcript.outlineUntitled;
              return (
                <li key={`${key}:${i}`}>
                  <button
                    type="button"
                    title={label}
                    aria-current={active ? "true" : undefined}
                    data-testid="turn-outline-item"
                    className={cn(
                      "relative flex w-full cursor-pointer items-start gap-1.5 rounded-md px-2 py-[5px] text-left text-[12px] leading-4",
                      active ? "text-foreground" : "text-muted hover:bg-lift/40 hover:text-foreground/85",
                    )}
                    onClick={() => props.onJump(t)}
                  >
                    {active ? (
                      <span className="absolute left-0 top-1.5 bottom-1.5 w-[2px] rounded-full bg-accent" aria-hidden />
                    ) : null}
                    <span className="w-4 shrink-0 pt-px text-right text-[10px] tabular-nums text-muted/55">{i + 1}</span>
                    <span className="min-w-0 flex-1 truncate">{label}</span>
                    {t.live || (props.status === "working" && i === props.turns.length - 1) ? (
                      <span className="mt-0.5 size-1 shrink-0 rounded-full bg-accent" aria-hidden />
                    ) : null}
                  </button>
                </li>
              );
            })}
          </ol>
        )}
      </nav>
    </aside>
  );
}
