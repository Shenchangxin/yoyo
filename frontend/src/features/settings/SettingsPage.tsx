import { useLayoutEffect, useRef } from "react";
import { useCopy } from "../../lib/i18n";
import { useMedia } from "../../lib/media";
import type { SettingsTab } from "../../lib/protocol";
import { useUI } from "../../lib/store";
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

  useLayoutEffect(() => {
    const root = scroller.current;
    if (!root) return;
    if (!section) {
      root.scrollTop = 0;
      return;
    }
    let cancelled = false;
    let breatheTimer = 0;
    const frame = window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        if (cancelled) return;
        const el = root.querySelector(`[data-setting-section-highlight-target="${section}"]`);
        if (!(el instanceof HTMLElement)) return;
        const sticky = root.querySelector("[data-settings-page-header]");
        const headerH = sticky instanceof HTMLElement ? sticky.getBoundingClientRect().height : 0;
        const rootRect = root.getBoundingClientRect();
        const elRect = el.getBoundingClientRect();
        const gap = headerH + 16;
        const inView = elRect.top >= rootRect.top + gap - 12 && elRect.top < rootRect.bottom - 48;
        if (!inView) {
          const top = root.scrollTop + (elRect.top - rootRect.top) - gap;
          root.scrollTop = Math.max(0, top);
        }
        el.classList.add("setting-section-breathe");
        breatheTimer = window.setTimeout(() => el.classList.remove("setting-section-breathe"), 4100);
      });
    });
    return () => {
      cancelled = true;
      window.cancelAnimationFrame(frame);
      if (breatheTimer) window.clearTimeout(breatheTimer);
    };
  }, [section, nav, tab]);

  function onTab(next: SettingsTab) {
    setSettingsTab(next);
  }

  return (
    <div className="flex h-full min-h-0 overflow-hidden">
      <SettingsSidebar
        tab={tab}
        onTab={onTab}
        onSection={(t, id) => openSettings(t, id)}
        compact={compact}
      />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden" data-skin-pane>
        <div
          ref={scroller}
          className="min-h-0 min-w-0 flex-1 overflow-y-auto overflow-x-hidden overscroll-contain [overflow-anchor:none]"
          style={{ scrollPaddingTop: "5.5rem" }}
        >
          <div className="mx-auto max-w-[42rem] px-8 pb-10 pt-8">
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
        </div>
      </div>
    </div>
  );
}
