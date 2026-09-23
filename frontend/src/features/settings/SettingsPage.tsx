import { useEffect, useRef } from "react";
import { useCopy } from "../../lib/i18n";
import { useMedia } from "../../lib/media";
import type { SettingsTab } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { ScrollFade } from "../../components/ui/scroll-fade";
import { SettingsSidebar } from "./SettingsSidebar";
import {
  AdvancedSettings,
  AppearanceSettings,
  GeneralSettings,
  type SettingsHost,
  PolicySettings,
  ProviderSettings,
  GenerationSettings,
  ShortcutsSettings,
} from "./pages";
import { ExtensionsSettings } from "./ExtensionsPanel";
import { PersonalSettings } from "./PersonalPanel";

export function SettingsPage({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const tab = useUI((s) => s.settingsTab);
  const section = useUI((s) => s.settingsSection);
  const nav = useUI((s) => s.settingsNav);
  const setSettingsTab = useUI((s) => s.setSettingsTab);
  const openSettings = useUI((s) => s.openSettings);
  const compact = useMedia("(max-width: 900px)");
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
      <SettingsSidebar
        tab={tab}
        onTab={onTab}
        onSection={(t, id) => openSettings(t, id)}
        compact={compact}
      />
      <ScrollFade className="min-w-0 flex-1 bg-background">
        <div ref={scroller} className="mx-auto max-w-[42rem] px-8 py-10">
          {tab === "general" ? <GeneralSettings host={host} /> : null}
          {tab === "appearance" ? <AppearanceSettings host={host} /> : null}
          {tab === "shortcuts" ? <ShortcutsSettings host={host} /> : null}
          {tab === "provider" ? <ProviderSettings host={host} /> : null}
          {tab === "generation" ? <GenerationSettings /> : null}
          {tab === "policy" ? <PolicySettings host={host} /> : null}
          {tab === "extensions" ? <ExtensionsSettings host={host} /> : null}
          {tab === "personal" ? <PersonalSettings /> : null}
          {tab === "advanced" ? <AdvancedSettings host={host} /> : null}
          <p className="sr-only">{copy.settings.hint}</p>
        </div>
      </ScrollFade>
    </div>
  );
}
