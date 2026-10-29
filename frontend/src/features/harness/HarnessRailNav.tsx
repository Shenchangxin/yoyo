/**
 * 拥有：Harness 表面左侧的四个页签。
 * 不拥有：页签里的实验室。点下去只改当前页。
 */

import { ArrowUpRight, Circle, ShieldCheck, Sparkles } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import type { HarnessTab } from "../../lib/protocol";
import { HARNESS_TABS } from "../../lib/surface";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";

const ICONS = {
  overview: Circle,
  propose: Sparkles,
  prove: ShieldCheck,
  promote: ArrowUpRight,
} as const;

/** HarnessRailNav 画出状态、改进、检查和采用。当前页用抬起的底色。 */
export function HarnessRailNav() {
  const copy = useCopy();
  const tab = useUI((s) => s.harnessTab);
  const setTab = useUI((s) => s.setHarnessTab);
  const labels: Record<HarnessTab, string> = {
    overview: copy.rsi.overview,
    propose: copy.rsi.propose,
    prove: copy.rsi.prove,
    promote: copy.rsi.promote,
  };
  return (
    <nav aria-label={copy.rsi.title} data-testid="harness-rail" className="px-1.5 pb-1">
      {HARNESS_TABS.map((id) => {
        const Icon = ICONS[id];
        const on = tab === id;
        return (
          <button
            key={id}
            type="button"
            aria-current={on ? "page" : undefined}
            className={cn(
              "mb-px flex min-h-8 w-full items-center gap-2 rounded-lg px-2.5 py-[6px] text-left text-[13px] transition-[background-color,color] duration-[var(--duration-fast)] ease-[var(--ease-out)]",
              on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
            )}
            onClick={() => setTab(id)}
          >
            <Icon className="size-3.5 shrink-0 opacity-70" aria-hidden />
            <span className="truncate">{labels[id]}</span>
          </button>
        );
      })}
    </nav>
  );
}
