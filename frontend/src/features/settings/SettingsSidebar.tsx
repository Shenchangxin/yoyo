/**
 * 拥有：设置侧栏和轨道里的页签导航：搜索、分组、页签。
 * 不拥有：表单。只切换当前页签，不调用 api。
 */
import { useMemo, useState } from "react";
import { motion } from "motion/react";
import {
  Bell,
  Blocks,
  Clapperboard,
  Cpu,
  Heart,
  Info,
  Keyboard,
  MessageCircle,
  Palette,
  Plug,
  Repeat,
  Search,
  Shield,
  SlidersHorizontal,
  Smartphone,
  Wrench,
  X,
} from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import type { SettingsTab } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { navTabs, SETTINGS_GROUPS, SETTINGS_SECTIONS, tabsInGroup } from "./registry";

const ICONS = {
  bell: Bell,
  sliders: SlidersHorizontal,
  palette: Palette,
  keyboard: Keyboard,
  cpu: Cpu,
  film: Clapperboard,
  repeat: Repeat,
  heart: Heart,
  shield: Shield,
  blocks: Blocks,
  message: MessageCircle,
  plug: Plug,
  phone: Smartphone,
  wrench: Wrench,
  info: Info,
} as const;

/** 按名字取出页签图标。key 用作下标。 */
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

/** SettingsSidebar 列出分组和页签。props 读取 compact、variant、tab，调用 onTab，调用 onSection。 */
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
        {navTabs().map((t) => {
          const Icon = iconFor(t.icon);
          const on = props.tab === t.key;
          return (
            <button
              key={t.key}
              type="button"
              aria-current={on ? "page" : undefined}
              onClick={() => props.onTab(t.key)}
              className={cn(
                "flex h-8 shrink-0 items-center gap-1.5 rounded-full px-3 text-ui-body font-medium transition-colors",
                on ? "bg-accent/10 text-accent" : "text-muted hover:text-foreground",
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
        rail ? "w-full" : "glass-chrome w-[232px]",
      )}
      aria-label={copy.settings.title}
    >
      <div className="px-3 pb-2 pt-3">
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
            className="no-drag h-8 w-full rounded-lg border border-transparent bg-foreground/[0.055] pl-8 pr-7 text-ui-body text-foreground outline-none transition-[background-color,border-color,box-shadow] duration-[var(--duration-fast)] placeholder:text-muted hover:bg-foreground/[0.075] focus-visible:border-accent/45 focus-visible:bg-background focus-visible:ring-2 focus-visible:ring-accent/20 [&::-webkit-search-cancel-button]:hidden"
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

      <div className={cn("min-h-0 flex-1 overflow-auto px-2", rail ? "pb-16" : "pb-4")}>
        {q ? (
          results.length ? (
            <ul className="pt-1">
              {results.map((s) => (
                <li key={s.id}>
                  <button
                    type="button"
                    onClick={() => props.onSection(s.tab, s.id)}
                    className="flex w-full flex-col items-start gap-0.5 rounded-md px-2.5 py-1.5 text-left transition-colors hover:bg-lift/70"
                  >
                    <span className="text-ui-body font-medium text-foreground">
                      {copy.settings.sections[s.titleKey]}
                    </span>
                    <span className="text-ui-caption text-muted">{copy.settings.tabs[s.tab]}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="px-2.5 py-6 text-center text-ui-footnote text-muted">{copy.settings.searchEmpty}</p>
          )
        ) : (
          SETTINGS_GROUPS.map((group) => (
            <div key={group} className="pt-4 first:pt-1">
              <div className="px-2.5 pb-1 text-ui-caption font-medium text-muted/80">
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

/** 画一个页签按钮：一枚线条图标加文字，选中时底下滑过来一块浅底。props 读取 icon、active、label、onClick。 */
function TabButton(props: {
  tab: SettingsTab;
  icon: string;
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  const Icon = iconFor(props.icon);
  const reduced = useMotionReduced();
  return (
    <button
      type="button"
      aria-current={props.active ? "page" : undefined}
      onClick={props.onClick}
      className={cn(
        "group relative mb-px flex h-8 w-full items-center gap-2.5 rounded-md px-2.5 text-left text-ui-body transition-colors",
        props.active ? "font-medium text-accent" : "text-foreground/75 hover:bg-lift/60 hover:text-foreground",
      )}
    >
      {props.active ? (
        <motion.span
          layoutId="settings-nav-pill"
          className="absolute inset-0 rounded-md bg-accent/10"
          transition={motionTransition(reduced, DURATION)}
        />
      ) : null}
      <Icon
        className={cn(
          "relative z-10 size-4 shrink-0 transition-colors",
          props.active ? "text-accent" : "text-muted group-hover:text-foreground",
        )}
        aria-hidden
        strokeWidth={1.75}
      />
      <span className="relative z-10 truncate">{props.label}</span>
    </button>
  );
}
