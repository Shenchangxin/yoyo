/**
 * 拥有：设置页的外壳，按当前页签挂上对应面板。旧页签落到现行布局。
 * 不拥有：各面板里的字段。本页不调用 api，面板会。
 */
import { useEffect, useLayoutEffect, useRef } from "react";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useMedia } from "../../lib/media";
import type { SettingsTab } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { SettingsSidebar } from "./SettingsSidebar";
import {
  AdvancedSettings,
  AppearanceSettings,
  GeneralSettings,
  LabsSection,
  type SettingsHost,
  PolicySettings,
  ProviderSettings,
  GenerationSettings,
} from "./pages";
import { FiberSection, McpSection } from "./ExtensionsPanel";
import { ConnectorsSettings } from "./ConnectorsSettings";
import { MessagingSettings } from "./MessagingSettings";
import { HandheldSettings } from "./HandheldPanel";
import { IdolSettings } from "./IdolSettings";
import { ProactiveSettings } from "./ProactivePanel";
import { GenerationCatalog } from "./GenerationCatalog";
import { isSettingsTab, resolveSettingsTarget } from "./registry";

/** SettingsPage 按当前页签挂上面板。host 写到 GeneralSettings 的 host。navInRail 为假时走另一支。 */
export function SettingsPage({ host, navInRail }: { host: SettingsHost; navInRail?: boolean }) {
  const copy = useCopy();
  const tab = useUI((s) => s.settingsTab);
  const section = useUI((s) => s.settingsSection);
  const nav = useUI((s) => s.settingsNav);
  const setSettingsTab = useUI((s) => s.setSettingsTab);
  const openSettings = useUI((s) => s.openSettings);
  const compact = useMedia("(max-width: 900px)");
  const scroller = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isSettingsTab(tab)) return;
    const next = resolveSettingsTarget(tab, section);
    openSettings(next.tab, next.section);
  }, [tab, section, openSettings]);

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
        const rootRect = root.getBoundingClientRect();
        const elRect = el.getBoundingClientRect();
        const gap = 16;
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

  /** 切到另一个页签。next 传给 setSettingsTab。 */
  function onTab(next: SettingsTab) {
    setSettingsTab(next);
  }

  return (
    <div className={cn("flex h-full min-h-0 w-full min-w-0 overflow-hidden", compact && !navInRail && "flex-col")}>
      {navInRail ? null : (
        <SettingsSidebar
          tab={tab}
          onTab={onTab}
          onSection={(t, id) => openSettings(t, id)}
          compact={compact}
        />
      )}
      <div
        className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background"
        data-settings-content
      >
        <div
          ref={scroller}
          className="h-full min-h-0 min-w-0 flex-1 overflow-y-auto overflow-x-hidden overscroll-contain [overflow-anchor:none]"
          style={{ scrollPaddingTop: "1.25rem" }}
        >
          <div className="box-border mx-auto min-h-full w-full max-w-[760px] px-4 py-5 sm:px-6">
            <h1 className="sr-only">{copy.settings.tabs[tab]}</h1>
            {tab === "general" ? <GeneralSettings host={host} /> : null}
            {tab === "appearance" ? <AppearanceSettings host={host} /> : null}
            {tab === "proactive" ? <ProactiveSettings host={host} /> : null}
            {tab === "provider" ? <ProviderSettings host={host} /> : null}
            {tab === "generation" ? (
              <GenerationCatalog>
                <GenerationSettings host={host} />
              </GenerationCatalog>
            ) : null}
            {tab === "policy" ? <PolicySettings host={host} /> : null}
            {tab === "connectors" ? (
              <>
                <ConnectorsSettings />
                <MessagingSettings />
                <McpSection host={host} />
                <HandheldSettings />
              </>
            ) : null}
            {tab === "advanced" ? (
              <>
                <AdvancedSettings host={host} />
                <FiberSection host={host} />
                <LabsSection host={host} />
              </>
            ) : null}
            {tab === "idol" ? <IdolSettings /> : null}
            <p className="sr-only">{copy.settings.hint}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
