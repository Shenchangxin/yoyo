/**
 * 拥有：设置页的外壳：页标题和一句说明，按当前页签挂上对应面板。旧页签和搬了家的区块落到现行布局。
 * 不拥有：各面板里的字段。本页不调用 api，面板会。
 */
import { useEffect, useLayoutEffect, useRef, type ReactNode } from "react";
import { motion } from "motion/react";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useMedia } from "../../lib/media";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import type { SettingsTab } from "../../lib/protocol";
import { useUI } from "../../lib/store";
import { SettingsSidebar } from "./SettingsSidebar";
import {
  AboutSettings,
  AppearanceSettings,
  BudgetSection,
  DiagnosticsSettings,
  GeneralSettings,
  LabsSection,
  type SettingsHost,
  PolicySettings,
  ProviderSettings,
  GenerationSettings,
  ShortcutsSettings,
} from "./pages";
import { FiberSection, McpSection } from "./ExtensionsPanel";
import { ConnectorsSettings } from "./ConnectorsSettings";
import { MessagingSettings } from "./MessagingSettings";
import { HandheldSettings } from "./HandheldPanel";
import { IdolSettings } from "./IdolSettings";
import { AutomationsSettings } from "./PersonalPanel";
import { ProactiveSettings } from "./ProactivePanel";
import { GenerationCatalog } from "./GenerationCatalog";
import { isSettingsTab, resolveSettingsTarget } from "./registry";

/** panelFor 是一页里排哪些面板。一页只管一件事；搬了家的区块在 registry 里记着新的页签。tab 决定取哪一页。host 交给各面板。 */
function panelFor(tab: SettingsTab, host: SettingsHost): ReactNode {
  switch (tab) {
    case "general":
      return <GeneralSettings host={host} />;
    case "appearance":
      return <AppearanceSettings host={host} />;
    case "shortcuts":
      return <ShortcutsSettings host={host} />;
    case "provider":
      return (
        <>
          <ProviderSettings host={host} />
          <BudgetSection host={host} />
        </>
      );
    case "generation":
      return (
        <GenerationCatalog>
          <GenerationSettings host={host} />
        </GenerationCatalog>
      );
    case "proactive":
      return <ProactiveSettings host={host} />;
    case "personal":
      return <AutomationsSettings />;
    case "policy":
      return <PolicySettings host={host} />;
    case "connectors":
      return <ConnectorsSettings />;
    case "messaging":
      return <MessagingSettings />;
    case "extensions":
      return (
        <>
          <McpSection host={host} />
          <FiberSection host={host} />
        </>
      );
    case "handheld":
      return <HandheldSettings />;
    case "advanced":
      return (
        <>
          <DiagnosticsSettings host={host} />
          <LabsSection host={host} />
        </>
      );
    case "about":
      return <AboutSettings host={host} />;
    case "idol":
      return <IdolSettings />;
    default:
      return null;
  }
}

/** SettingsPage 按当前页签挂上面板。host 交给各面板。navInRail 为假时侧栏画在本页左边。 */
export function SettingsPage({ host, navInRail }: { host: SettingsHost; navInRail?: boolean }) {
  const copy = useCopy();
  const tab = useUI((s) => s.settingsTab);
  const section = useUI((s) => s.settingsSection);
  const nav = useUI((s) => s.settingsNav);
  const setSettingsTab = useUI((s) => s.setSettingsTab);
  const openSettings = useUI((s) => s.openSettings);
  const compact = useMedia("(max-width: 900px)");
  const reduced = useMotionReduced();
  const scroller = useRef<HTMLDivElement>(null);

  // 旧页签落到现行页签。别的模块仍会带着旧页签打开一个已经搬家的区块，这时以区块现在所在的页为准。
  // 这一帧就按落点画，不先闪一下原来那页；随后把落点写回 store，地址栏和侧栏跟着对上。
  const target = resolveSettingsTarget(tab, section);
  const moved = !isSettingsTab(tab) || (!!target.section && target.tab !== tab);
  const shown: SettingsTab = moved ? target.tab : tab;
  useEffect(() => {
    if (moved) openSettings(target.tab, target.section);
  }, [moved, target.tab, target.section, openSettings]);

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
          <motion.div
            key={shown}
            initial={reduced ? false : { opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={motionTransition(reduced, DURATION)}
            className="box-border mx-auto min-h-full w-full max-w-[720px] px-5 pb-16 pt-7 sm:px-6"
          >
            <header className="mb-7 px-1">
              <h1 className="text-ui-display font-semibold tracking-[-0.015em] text-foreground">{copy.settings.tabs[shown]}</h1>
              <p className="mt-1 text-ui-body leading-[1.55] text-muted">{copy.settings.tabHints[shown]}</p>
            </header>
            {panelFor(shown, host)}
            <p className="sr-only">{copy.settings.hint}</p>
          </motion.div>
        </div>
      </div>
    </div>
  );
}
