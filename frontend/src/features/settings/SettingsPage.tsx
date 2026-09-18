import { useEffect, useRef } from "react";
import { useCopy } from "../../lib/i18n";
import { useMedia } from "../../lib/media";
import type { SettingsTab } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { ScrollFade } from "../../components/ui/scroll-fade";
import { SettingsSidebar } from "./SettingsSidebar";
import {
  AppearanceSettings,
  DangerSettings,
  GeneralSettings,
  LogsSettings,
  McpSettings,
  type SettingsHost,
  PluginsSettings,
  PolicySettings,
  ProviderSettings,
  ShortcutsSettings,
  UpdatesSettings,
} from "./pages";
import { AutomationSettings, ConnectorSettings, IsolationSettings, MemorySettings } from "./PersonalPages";

export function SettingsPage({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const tab = useUI((s) => s.settingsTab);
  const section = useUI((s) => s.settingsSection);
  const nav = useUI((s) => s.settingsNav);
  const setSettingsTab = useUI((s) => s.setSettingsTab);
  const compact = useMedia("(max-width: 860px)");
  const scroller = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!section) return;
    const root = scroller.current;
    if (!root) return;
    const el = root.querySelector(`[data-setting-section-highlight-target="${section}"]`) as HTMLElement | null;
    if (!el) return;
    el.scrollIntoView({ behavior: "smooth", block: "center" });
    el.classList.add("setting-section-breathe");
    const t = window.setTimeout(() => el.classList.remove("setting-section-breathe"), 4100);
    return () => window.clearTimeout(t);
  }, [section, nav, tab]);

  function onTab(next: SettingsTab) {
    setSettingsTab(next);
  }

  return (
    <div className="flex h-full min-h-0">
      <SettingsSidebar tab={tab} onTab={onTab} compact={compact} />
      <div className="w-px self-stretch bg-border/80" aria-hidden />
      <ScrollFade className="min-w-0 flex-1">
        <div ref={scroller} className="mx-auto max-w-[680px] px-8 py-8">
          {tab === "general" ? <GeneralSettings host={host} /> : null}
          {tab === "appearance" ? <AppearanceSettings host={host} /> : null}
          {tab === "provider" ? <ProviderSettings host={host} /> : null}
          {tab === "policy" ? <PolicySettings host={host} /> : null}
          {tab === "isolation" ? <IsolationSettings host={host} /> : null}
          {tab === "connectors" ? <ConnectorSettings /> : null}
          {tab === "memory" ? <MemorySettings /> : null}
          {tab === "automations" ? <AutomationSettings /> : null}
          {tab === "mcp" ? <McpSettings host={host} /> : null}
          {tab === "plugins" ? <PluginsSettings host={host} /> : null}
          {tab === "shortcuts" ? <ShortcutsSettings host={host} /> : null}
          {tab === "updates" ? <UpdatesSettings host={host} /> : null}
          {tab === "logs" ? <LogsSettings host={host} /> : null}
          {tab === "danger" ? <DangerSettings host={host} /> : null}
          <p className="sr-only">{copy.settings.hint}</p>
        </div>
      </ScrollFade>
    </div>
  );
}
