import { motion } from "motion/react";
import { useCopy } from "../../lib/i18n";
import { DURATION, EASE } from "../../lib/motion";
import type { SettingsTab } from "../../lib/protocol";
import { cn } from "../../lib/utils";
import { SETTINGS_TABS } from "./registry";
import {
  AlertTriangle,
  Boxes,
  Download,
  Keyboard,
  KeyRound,
  Palette,
  Plug,
  ScrollText,
  Shield,
  SlidersHorizontal,
  Lock,
  Link2,
  Brain,
  Clock,
} from "lucide-react";

const ICONS = {
  sliders: SlidersHorizontal,
  palette: Palette,
  key: KeyRound,
  shield: Shield,
  plug: Plug,
  boxes: Boxes,
  keyboard: Keyboard,
  download: Download,
  scroll: ScrollText,
  alert: AlertTriangle,
  lock: Lock,
  link: Link2,
  brain: Brain,
  clock: Clock,
} as const;

export function SettingsSidebar(props: {
  tab: SettingsTab;
  onTab: (tab: SettingsTab) => void;
  compact?: boolean;
}) {
  const copy = useCopy();
  return (
    <nav
      className={cn("shrink-0 overflow-auto py-3", props.compact ? "w-14 px-1.5" : "w-[200px] px-2")}
      aria-label={copy.settings.title}
    >
      {SETTINGS_TABS.map((t) => {
        const Icon = ICONS[t.icon as keyof typeof ICONS] || SlidersHorizontal;
        const label = copy.settings.tabs[t.key];
        const active = props.tab === t.key;
        return (
          <button
            type="button"
            key={t.key}
            aria-current={active ? "page" : undefined}
            title={label}
            className={cn(
              "relative mb-0.5 flex w-full items-center gap-2 rounded-lg px-2.5 py-[7px] text-left text-[13px] font-medium transition-colors duration-180",
              active ? "text-foreground" : "text-muted hover:text-foreground",
              props.compact && "justify-center px-0",
            )}
            onClick={() => props.onTab(t.key)}
          >
            {active ? (
              <motion.span
                layoutId="settings-nav-pill"
                className="absolute inset-0 rounded-lg bg-lift"
                transition={{ duration: DURATION, ease: EASE }}
              />
            ) : null}
            <Icon className="relative z-10 size-4 shrink-0" aria-hidden />
            {props.compact ? <span className="sr-only">{label}</span> : <span className="relative z-10">{label}</span>}
          </button>
        );
      })}
    </nav>
  );
}
