/**
 * 拥有：学习表面左侧的四条泳道。
 * 不拥有：泳道里的内容。点下去只改当前泳道。
 */

import { BookOpen, Library, ListChecks, Sun } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import type { LearnLane } from "../../lib/learn-scope";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";

const LANES: { id: LearnLane; icon: typeof Sun }[] = [
  { id: "today", icon: Sun },
  { id: "study", icon: ListChecks },
  { id: "library", icon: Library },
  { id: "practice", icon: BookOpen },
];

/** LearnRailNav 画出今日、学习、文库和练习。当前泳道用抬起的底色。 */
export function LearnRailNav() {
  const copy = useCopy();
  const L = copy.learn;
  const lane = useUI((s) => s.learnLane);
  const setLane = useUI((s) => s.setLearnLane);
  const labels: Record<LearnLane, string> = {
    today: L.today,
    study: L.study,
    library: L.library,
    practice: L.practice,
  };
  return (
    <nav aria-label={copy.rail.pages} data-testid="learn-rail" className="px-1.5 pb-1">
      {LANES.map((item) => {
        const Icon = item.icon;
        const on = lane === item.id;
        return (
          <button
            key={item.id}
            type="button"
            aria-current={on ? "page" : undefined}
            className={cn(
              "mb-px flex min-h-8 w-full items-center gap-2 rounded-lg px-2.5 py-[6px] text-left text-[13px] transition-[background-color,color] duration-[var(--duration-fast)] ease-[var(--ease-out)]",
              on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
            )}
            onClick={() => setLane(item.id)}
          >
            <Icon className="size-3.5 shrink-0 opacity-70" aria-hidden />
            <span className="truncate">{labels[item.id]}</span>
          </button>
        );
      })}
    </nav>
  );
}
