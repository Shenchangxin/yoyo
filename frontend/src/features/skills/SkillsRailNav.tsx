/**
 * 拥有：技能表面左侧的已装、市场和技能包。
 * 不拥有：这三页的内容。点下去只改页签。
 */

import { Boxes, Package, Store } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { useUI, type SkillsTab } from "../../lib/store";
import { cn } from "../../lib/utils";

const TABS: { id: SkillsTab; icon: typeof Package }[] = [
  { id: "installed", icon: Package },
  { id: "market", icon: Store },
  { id: "packs", icon: Boxes },
];

/** SkillsRailNav 画出技能的三个页签。当前页用抬起的底色。 */
export function SkillsRailNav() {
  const copy = useCopy();
  const tab = useUI((s) => s.skillsTab);
  const setTab = useUI((s) => s.setSkillsTab);
  const labels: Record<SkillsTab, string> = {
    installed: copy.skills.installed,
    market: copy.skills.market,
    packs: copy.skills.packs,
  };
  return (
    <nav aria-label={copy.skills.title} data-testid="skills-rail" className="px-1.5 pb-1">
      {TABS.map((item) => {
        const Icon = item.icon;
        const on = tab === item.id;
        return (
          <button
            key={item.id}
            type="button"
            aria-current={on ? "page" : undefined}
            className={cn(
              "mb-px flex min-h-8 w-full items-center gap-2 rounded-lg px-2.5 py-[6px] text-left text-[13px] transition-[background-color,color] duration-[var(--duration-fast)] ease-[var(--ease-out)]",
              on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
            )}
            onClick={() => setTab(item.id)}
          >
            <Icon className="size-3.5 shrink-0 opacity-70" aria-hidden />
            <span className="truncate">{labels[item.id]}</span>
          </button>
        );
      })}
    </nav>
  );
}
