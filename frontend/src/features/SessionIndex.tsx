/**
 * 拥有：主会话左侧中间的快捷目录。平时是一列刻度，悬停展开成可点的用户消息。
 * 不拥有：回合怎么来，也不负责滚到那一行。调用方传入 turns 和 onJump。
 */

import { useEffect, useRef } from "react";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import { isOutlineActive, outlineTurnKey, type OutlineTurn } from "../lib/turn-outline";

/** SessionIndex 少于两轮时不出现。刻度贴在阅读区左侧中间，展开时不撑开正文。 */
export function SessionIndex(props: {
  turns: OutlineTurn[];
  activeKey: string;
  onJump: (turn: OutlineTurn) => void;
}) {
  const copy = useCopy();
  const turns = props.turns.filter((t) => t.kind !== "steer");
  const navRef = useRef<HTMLElement>(null);
  useEffect(() => {
    navRef.current?.querySelector("[aria-current='true']")?.scrollIntoView({ block: "nearest" });
  }, [props.activeKey, turns.length]);
  if (turns.length < 2) return null;
  return (
    <div className="pointer-events-none absolute left-2 top-1/2 z-20 -translate-y-1/2">
      <nav
        ref={navRef}
        aria-label={copy.transcript.outline}
        data-testid="session-index"
        className="group pointer-events-auto flex max-h-[min(52vh,24rem)] w-7 flex-col overflow-hidden rounded-xl py-1.5 transition-[width,background-color,box-shadow,border-color] duration-300 ease-[var(--ease-out)] hover:w-56 hover:border hover:border-border/70 hover:bg-background/88 hover:shadow-[var(--shadow-popover)] hover:backdrop-blur-md"
      >
        <ol className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-1 [scrollbar-width:none]">
          {turns.map((t, i) => {
            const key = outlineTurnKey(t);
            const active = isOutlineActive(t, props.activeKey);
            const label = t.title || copy.transcript.outlineUntitled;
            return (
              <li key={`${key}:${i}`}>
                <button
                  type="button"
                  title={label}
                  aria-current={active ? "true" : undefined}
                  aria-label={copy.transcript.jumpTurn.replace("{n}", String(i + 1))}
                  data-testid="session-index-item"
                  className={cn(
                    "flex h-6 w-full cursor-pointer items-center gap-2 rounded-lg px-1 text-left",
                    active ? "text-foreground" : "text-muted hover:bg-foreground/[0.04] hover:text-foreground",
                  )}
                  onClick={() => props.onJump(t)}
                >
                  <span
                    className={cn(
                      "block w-[3px] shrink-0 rounded-full",
                      active ? "h-3.5 bg-accent" : "h-1.5 bg-foreground/25 group-hover:h-2",
                    )}
                    aria-hidden
                  />
                  <span className="w-4 shrink-0 text-right text-ui-caption tabular-nums text-muted/70 opacity-0 transition-opacity duration-200 group-hover:opacity-100">{i + 1}</span>
                  <span className="min-w-0 flex-1 truncate text-ui-footnote opacity-0 transition-opacity duration-200 group-hover:opacity-100">{label}</span>
                </button>
              </li>
            );
          })}
        </ol>
      </nav>
    </div>
  );
}
