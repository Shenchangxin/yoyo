/**
 * 拥有：设置侧栏和轨道里的页签导航。
 * 不拥有：表单。只切换当前页签，不调用 api。
 */
import { useMemo, useState } from "react";
import { motion } from "motion/react";
import {
  Blocks,
  Brain,
  Cpu,
  Palette,
  Search,
  Shield,
  SlidersHorizontal,
  Clapperboard,
  Heart,
  Wrench,
  X,
} from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import type { SettingsTab } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { SETTINGS_GROUPS, SETTINGS_SECTIONS, SETTINGS_TABS, tabsInGroup } from "./registry";

const ICONS = {
  sliders: SlidersHorizontal,
  palette: Palette,
  cpu: Cpu,
  film: Clapperboard,
  heart: Heart,
  shield: Shield,
  blocks: Blocks,
  brain: Brain,
  wrench: Wrench,
} as const;

/** 按名字取出页签图标。 */
function iconFor(key: string) {
  return ICONS[key as keyof typeof ICONS] || SlidersHorizontal;
}

/** SettingsRailNav 把侧栏放进轨道。 */
export function SettingsRailNav() {
  const tab = useUI((s) => s.settingsTab);
  const openSettings = useUI((s) => s.openSettings);
  const setSettingsTab = useUI((s) => s.setSettingsTab);
  return (
    <SettingsSidebar
      tab={tab}
      onTab={setSettingsTab}
      onSection={(t, id) => openSettings(t, id)}
      variant="rail"
    />
  );
}

/** SettingsSidebar 列出分组和页签。 */
export function SettingsSidebar(props: {
  tab: SettingsTab;
  onTab: (tab: SettingsTab) => void;
  onSection: (tab: SettingsTab, section: string) => void;
  compact?: boolean;
  variant?: "page" | "rail";
}) {
  const copy = useCopy();
  const [query, setQuery] = useState("");
  const q = query.trim().toLowerCase();

  const results = useMemo(() => {
    if (!q) return [];
    return SETTINGS_SECTIONS.filter((s) => {
      const haystack = [
        copy.settings.sections[s.titleKey],
        copy.settings.tabs[s.tab],
        s.id,
        s.keywords || "",
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(q);
    });
  }, [q, copy]);

  if (props.compact && props.variant !== "rail") {
    return (
      <nav className="flex h-12 shrink-0 items-center gap-1 overflow-x-auto border-b border-border px-3 [scrollbar-width:none]" aria-label={copy.settings.title}>
        {SETTINGS_TABS.map((t) => {
          const Icon = iconFor(t.icon);
          const on = props.tab === t.key;
          return (
            <button
              key={t.key}
              type="button"
              aria-current={on ? "page" : undefined}
              onClick={() => props.onTab(t.key)}
              className={cn(
                "flex h-8 shrink-0 items-center gap-1.5 rounded-full px-3 text-[13px] font-medium",
                on ? "bg-lift text-foreground" : "text-muted",
              )}
            >
              <Icon className="size-3.5" aria-hidden />
              {copy.settings.tabs[t.key]}
            </button>
          );
        })}
      </nav>
    );
  }

  const rail = props.variant === "rail";
  return (
    <nav
      className={cn(
        "flex h-full min-h-0 shrink-0 flex-col overflow-hidden",
        rail ? "w-full" : "glass-chrome w-[228px]",
      )}
      aria-label={copy.settings.title}
    >
      <div className="px-3 pb-1.5 pt-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={copy.settings.search}
            aria-label={copy.settings.search}
            autoComplete="off"
            name="settings-search"
            spellCheck={false}
            autoCorrect="off"
            autoCapitalize="off"
            className="no-drag h-8 w-full rounded-[9px] border border-border bg-background pl-8 pr-7 text-[13px] text-foreground placeholder:text-muted outline-none focus-visible:border-foreground/25 focus-visible:ring-1 focus-visible:ring-foreground/15 [&::-webkit-search-cancel-button]:hidden"
          />
          {query ? (
            <button
              type="button"
              aria-label={copy.settings.discard}
              onClick={() => setQuery("")}
              className="absolute right-1.5 top-1/2 flex size-5 -translate-y-1/2 items-center justify-center rounded-md text-muted transition-colors hover:bg-lift hover:text-foreground"
            >
              <X className="size-3" aria-hidden />
            </button>
          ) : null}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-2.5 pb-4">
        {q ? (
          results.length ? (
            <ul className="pt-1.5">
              {results.map((s) => (
                <li key={s.id}>
                  <button
                    type="button"
                    onClick={() => props.onSection(s.tab, s.id)}
                    className="flex w-full flex-col items-start gap-0.5 rounded-[9px] px-2.5 py-[7px] text-left transition-colors hover:bg-lift"
                  >
                    <span className="text-[13px] font-medium text-foreground">
                      {copy.settings.sections[s.titleKey]}
                    </span>
                    <span className="text-[11px] text-muted">{copy.settings.tabs[s.tab]}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="px-2.5 py-6 text-center text-[12px] text-muted">{copy.settings.searchEmpty}</p>
          )
        ) : (
          SETTINGS_GROUPS.map((group) => (
            <div key={group} className="pt-2.5 first:pt-1">
              <div className="px-2.5 pb-1 text-[11px] font-medium text-muted">
                {copy.settings.groups[group]}
              </div>
              {tabsInGroup(group).map((t) => (
                <TabButton
                  key={t.key}
                  tab={t.key}
                  icon={t.icon}
                  label={copy.settings.tabs[t.key]}
                  active={props.tab === t.key}
                  onClick={() => props.onTab(t.key)}
                />
              ))}
            </div>
          ))
        )}
      </div>
    </nav>
  );
}

/** 画一个页签按钮。 */
function TabButton(props: {
  tab: SettingsTab;
  icon: string;
  label: string;
  active: boolean;
  compact?: boolean;
  onClick: () => void;
}) {
  const Icon = iconFor(props.icon);
  const reduced = useMotionReduced();
  return (
    <button
      type="button"
      aria-current={props.active ? "page" : undefined}
      title={props.compact ? props.label : undefined}
      onClick={props.onClick}
      className={cn(
        "relative flex w-full items-center gap-2.5 rounded-[9px] text-left text-[13px] font-medium transition-colors",
        props.compact ? "justify-center px-0 py-1.5" : "mb-px px-2 py-[5px]",
        props.active ? "text-foreground" : "text-muted hover:text-foreground",
      )}
    >
      {props.active ? (
        <motion.span
          layoutId="settings-nav-pill"
          className="absolute inset-0 rounded-[9px] bg-lift"
          transition={motionTransition(reduced, DURATION)}
        />
      ) : null}
      <span
        className={cn(
          "relative z-10 flex size-[22px] shrink-0 items-center justify-center rounded-[7px] transition-colors",
          props.active ? "bg-accent text-accent-fg" : "bg-lift text-foreground/65",
        )}
      >
        <Icon className="size-[13px]" aria-hidden strokeWidth={2.1} />
      </span>
      {props.compact ? (
        <span className="sr-only">{props.label}</span>
      ) : (
        <span className="relative z-10 truncate">{props.label}</span>
      )}
    </button>
  );
}
