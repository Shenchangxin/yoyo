/**
 * 拥有：通用、外观、快捷键、预算、安全与隐私、诊断、实验室和关于这几页的表单。
 * 不拥有：侧栏，以及哪一块排在哪一页（那在 SettingsPage 和 registry）。改配置会调用 api 改状态。
 */
import { useEffect, useState } from "react";
import { Check, Copy, Download, FolderOpen, Stethoscope } from "lucide-react";
import { toast } from "sonner";
import * as api from "../../lib/client";
import { writeClipboard } from "../../lib/clipboard";
import { applyLocale, getLocale, useCopy, type Locale } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { DEFAULT_KEYMAP, formatShortcut, mergeKeymap, type KeymapId } from "../../lib/keymap";
import { DARK_PALETTES, LIGHT_PALETTES, useTheme, type DarkPalette, type LightPalette, type PaletteId, type ThemePref } from "../../lib/theme";
import { SkinStudio } from "./SkinStudio";
import { builtinTokens } from "../../lib/skin/builtin";
import { applyUiScale, snapUiScale, type UiScale } from "../../lib/scale";
import { browserSurface, chrome } from "../../lib/chrome";
import { asArray, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { IconSwap } from "../../components/ui/icon-swap";
import { Input } from "../../components/ui/input";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { Switch } from "../../components/ui/switch";
import { ConfirmDialog } from "../ConfirmDialog";
import { MarkWell } from "../shell/YoyoMark";
import {
  CONTROL_LG,
  Keycaps,
  SettingCodePanel,
  SettingRow,
  SettingSection,
  SettingSegmented,
  SettingValueRow,
  SettingsSaveBar,
} from "./SettingChrome";
import { cn } from "../../lib/utils";
import type { SettingsHost } from "./host";
import { openDesk } from "../superagent/needs";

/** SettingsHost 从 host 再导出。 */
export type { SettingsHost } from "./host";
/** ProviderSettings 从服务商面板再导出。 */
export { ProviderSettings } from "./ProviderPanel";
/** GenerationSettings 从生成设置再导出。 */
export { GenerationSettings } from "./GenerationSettings";

/** GeneralSettings 改工作区、窗口与启动和语言。快捷键有自己的一页，通知在「主动与通知」。host 读取 cfg，调用 patch，调用 onBrowse。 */
export function GeneralSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const cfg = host.cfg;
  const locale = (cfg.locale === "zh-CN" ? "zh-CN" : cfg.locale === "en" ? "en" : getLocale()) as Locale;
  return (
    <>
      <SettingSection id="general-workspace" title={copy.settings.sections.generalWorkspace}>
        <SettingRow stack title={copy.settings.agentWorkspace} description={copy.settings.workspaceDesc}>
          <div className="flex w-full items-center gap-2">
            <Input
              aria-label={copy.settings.workspace}
              className="h-8 flex-1 font-mono text-ui-footnote"
              value={cfg.workspace}
              onChange={(e) => void host.patch({ workspace: e.target.value })}
            />
            <Button
              variant="lift"
              onClick={async () => {
                const p = await host.onBrowse();
                if (p) await host.patch({ workspace: p });
              }}
            >
              <FolderOpen aria-hidden />
              {copy.settings.browse}
            </Button>
          </div>
        </SettingRow>
        <SettingRow stack title={copy.settings.videoWorkspace} description={copy.settings.videoWorkspaceDesc} border={false}>
          <div className="flex w-full items-center gap-2">
            <Input
              aria-label={copy.settings.videoWorkspace}
              className="h-8 flex-1 font-mono text-ui-footnote"
              value={cfg.videoWorkspace || ""}
              onChange={(e) => void host.patch({ videoWorkspace: e.target.value })}
            />
            <Button
              variant="lift"
              onClick={async () => {
                const p = await host.onBrowse();
                if (p) await host.patch({ videoWorkspace: p });
              }}
            >
              <FolderOpen aria-hidden />
              {copy.settings.browse}
            </Button>
          </div>
        </SettingRow>
      </SettingSection>

      {browserSurface() ? null : <SettingSection id="general-desktop" title={copy.settings.sections.generalDesktop}>
        <SettingRow title={copy.settings.closeToTray} description={copy.settings.closeToTrayDesc}>
          <Switch checked={!!cfg.closeToTray} onCheckedChange={(v) => void host.patch({ closeToTray: v })} />
        </SettingRow>
        <SettingRow title={copy.settings.companion} description={copy.settings.companionDesc}>
          <Switch checked={cfg.companionEnabled !== false} onCheckedChange={(v) => void host.patch({ companionEnabled: v })} />
        </SettingRow>
        <SettingRow title={copy.settings.startAtLogin} description={copy.settings.startAtLoginDesc}>
          <Switch checked={!!cfg.startAtLogin} onCheckedChange={(v) => void host.patch({ startAtLogin: v })} />
        </SettingRow>
        <SettingRow title={copy.settings.alwaysOnTop} description={copy.settings.alwaysOnTopDesc} border={false}>
          <Switch
            checked={!!cfg.alwaysOnTop}
            onCheckedChange={(v) => {
              chrome.setAlwaysOnTop(v);
              void host.patch({ alwaysOnTop: v });
            }}
          />
        </SettingRow>
      </SettingSection>}

      <SettingSection id="general-language" title={copy.settings.sections.generalLanguage}>
        <SettingRow title={copy.settings.language} description={copy.settings.languageDesc} border={false}>
          <SettingSegmented
            id="locale"
            ariaLabel={copy.settings.language}
            value={locale}
            options={[
              { value: "en" as Locale, label: copy.settings.english },
              { value: "zh-CN" as Locale, label: copy.settings.chinese },
            ]}
            onChange={(next) => {
              applyLocale(next);
              void host.patch({ locale: next });
            }}
          />
        </SettingRow>
      </SettingSection>
    </>
  );
}

/** AppearanceSettings 改主题（明暗和色板）、壁纸和对话显示。host 读取 cfg，调用 patch，写到 SkinStudio 的 host。 */
export function AppearanceSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const { pref, setPref, resolved, darkPalette, lightPalette, setDarkPalette, setLightPalette } = useTheme();
  const scale = String(snapUiScale(host.cfg.uiScale)) as `${UiScale}`;
  const view = host.cfg.transcriptView === "compact" ? "compact" : host.cfg.transcriptView === "detailed" || host.cfg.transcriptView === "verbose" ? "detailed" : "standard";
  const modes: { id: ThemePref; label: string }[] = [
    { id: "system", label: copy.settings.system },
    { id: "dark", label: copy.settings.dark },
    { id: "light", label: copy.settings.light },
  ];
  return (
    <>
      <SettingSection
        id="appearance-theme"
        title={copy.settings.sections.appearanceTheme}
        footnote={`${copy.settings.colorModeDesc} ${copy.settings.resolved}: ${resolved}.`}
        bare
      >
        <div className="grid grid-cols-3 gap-3">
          {modes.map((m) => {
            const active = pref === m.id;
            return (
              <button
                type="button"
                key={m.id}
                aria-pressed={active}
                className={cn(
                  "group cursor-pointer rounded-md border bg-card p-2 text-left transition-colors",
                  active ? "bg-lift" : "hover:bg-lift/70",
                )}
                onClick={() => {
                  setPref(m.id);
                  void host.patch({ theme: m.id });
                }}
              >
                <ThemePreview mode={m.id} dark={darkPalette} light={lightPalette} />
                <div className="mt-2 flex items-center gap-1 text-ui-footnote font-medium text-foreground">
                  {active ? <Check className="size-3.5 text-accent" aria-hidden /> : null}
                  {m.label}
                </div>
              </button>
            );
          })}
        </div>
        <PaletteGroup
          label={copy.settings.paletteGroupDark}
          ids={DARK_PALETTES}
          selected={darkPalette}
          className="mt-5"
          onPick={(id) => {
            setDarkPalette(id);
            void host.patch({ paletteDark: id });
          }}
        />
        <PaletteGroup
          label={copy.settings.paletteGroupLight}
          ids={LIGHT_PALETTES}
          selected={lightPalette}
          className="mt-4"
          onPick={(id) => {
            setLightPalette(id);
            void host.patch({ paletteLight: id });
          }}
        />
      </SettingSection>

      <SkinStudio host={host} />

      <SettingSection id="appearance-display" title={copy.settings.sections.appearanceDisplay}>
        <SettingRow title={copy.settings.uiScale} description={copy.settings.uiScaleDesc}>
          <SettingSegmented
            id="ui-scale"
            ariaLabel={copy.settings.uiScale}
            value={scale}
            options={[
              { value: "1", label: "100%" },
              { value: "1.25", label: "125%" },
              { value: "1.5", label: "150%" },
            ]}
            onChange={(next) => {
              const v = Number(next) as UiScale;
              applyUiScale(v);
              void host.patch({ uiScale: v });
            }}
          />
        </SettingRow>
        <SettingRow title={copy.settings.showThinking} description={copy.settings.showThinkingDesc}>
          <Switch
            checked={!!host.cfg.showThinking}
            onCheckedChange={(v) => void host.patch({ showThinking: v })}
          />
        </SettingRow>
        <SettingRow title={copy.settings.workDetails} description={copy.settings.workDetailsDesc} border={false}>
          <SettingSegmented
            id="transcript-view"
            ariaLabel={copy.settings.workDetails}
            value={view}
            options={[
              { value: "compact", label: copy.settings.workCompact },
              { value: "standard", label: copy.settings.workStandard },
              { value: "detailed", label: copy.settings.workDetailed },
            ]}
            onChange={(next) => void host.patch({ transcriptView: next })}
          />
        </SettingRow>
      </SettingSection>
    </>
  );
}

/** LabsSection 放给调试和实验用的显示开关：实验室细节、原始步骤、工作细节什么时候收起。 */
export function LabsSection({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const showHarnessLab = useUI((s) => s.showHarnessLab);
  const setShowHarnessLab = useUI((s) => s.setShowHarnessLab);
  const verbose = host.cfg.transcriptView === "verbose";
  return (
    <SettingSection id="advanced-labs" title={copy.settings.sections.advancedLabs} description={copy.settings.labsDesc}>
      <SettingRow title={copy.settings.showHarnessLab} description={copy.settings.showHarnessLabDesc}>
        <Switch checked={showHarnessLab} onCheckedChange={setShowHarnessLab} aria-label={copy.settings.showHarnessLab} />
      </SettingRow>
      <SettingRow title={copy.settings.workVerbose} description={copy.settings.workVerboseDesc}>
        <Switch
          checked={verbose}
          aria-label={copy.settings.workVerbose}
          onCheckedChange={(on) => void host.patch({ transcriptView: on ? "verbose" : "detailed" })}
        />
      </SettingRow>
      <SettingRow title={copy.settings.collapseTiming} description={copy.settings.collapseTimingDesc} border={false}>
        <SettingSegmented
          id="collapse-timing"
          ariaLabel={copy.settings.collapseTiming}
          value={host.cfg.collapseTiming === "next-input" ? "next-input" : "completion"}
          options={[
            { value: "completion", label: copy.settings.collapseCompletion },
            { value: "next-input", label: copy.settings.collapseNextInput },
          ]}
          onChange={(next) => void host.patch({ collapseTiming: next })}
        />
      </SettingRow>
    </SettingSection>
  );
}

type ShortcutGroupKey = "app" | "approval" | "turns";

/** SHORTCUT_GROUPS 把键位分成三组：应用、审批、对话。默认表里以后多出来的键位落到最后一组，不会丢。 */
const SHORTCUT_GROUPS: readonly { key: ShortcutGroupKey; ids: readonly KeymapId[] }[] = [
  { key: "app", ids: ["palette", "newChat", "control", "toggleReview"] },
  { key: "approval", ids: ["once", "session", "always", "deny"] },
  { key: "turns", ids: ["prevTurn", "nextTurn"] },
];

/** ShortcutsSettings 是快捷键页：按组列出键位，点一行再按键就改掉这一条。host 读取 cfg，调用 patch。 */
export function ShortcutsSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const keymap = mergeKeymap(host.cfg.keymap);
  const [armed, setArmed] = useState<KeymapId | null>(null);
  const grouped = new Set<KeymapId>(SHORTCUT_GROUPS.flatMap((g) => g.ids));
  const rest = (Object.keys(DEFAULT_KEYMAP) as KeymapId[]).filter((id) => !grouped.has(id));
  const groups = SHORTCUT_GROUPS.map((g, i) => (i === SHORTCUT_GROUPS.length - 1 ? { ...g, ids: [...g.ids, ...rest] } : g));
  return (
    <>
      {groups.map((group, i) => (
        <SettingSection
          key={group.key}
          id={i === 0 ? "shortcuts-map" : `shortcuts-${group.key}`}
          title={copy.settings.shortcutGroups[group.key]}
          footnote={i === groups.length - 1 ? copy.settings.recordHint : undefined}
        >
          {group.ids.map((id) => (
            <SettingRow key={id} list title={copy.settings.keys[id] || id}>
              <button
                type="button"
                aria-label={id}
                className={cn(
                  "-mr-1.5 inline-flex h-8 min-w-24 items-center justify-end rounded-md px-1.5 text-ui-footnote transition-colors",
                  armed === id ? "bg-accent/10 text-accent ring-1 ring-accent/50" : "hover:bg-lift/70",
                )}
                onClick={() => setArmed(id)}
                onBlur={() => setArmed(null)}
                onKeyDown={(e) => {
                  if (armed !== id) return;
                  e.preventDefault();
                  const spec = formatShortcut(e);
                  if (!spec) return;
                  void host.patch({ keymap: { ...keymap, [id]: spec } });
                  setArmed(null);
                }}
              >
                {armed === id ? <span className="px-1.5">{copy.settings.recording}</span> : <Keycaps spec={keymap[id]} />}
              </button>
            </SettingRow>
          ))}
        </SettingSection>
      ))}
    </>
  );
}

/** BudgetSection 改一个对话的花费上限和估算单价，排在模型页。改了才浮出保存条。host 读取 cfg，调用 patch。 */
export function BudgetSection({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const savedBudget = String(host.cfg.maxBudgetUsd || 0);
  const savedRate = String(host.cfg.usdPerMtok || 0);
  const [budget, setBudget] = useState(savedBudget);
  const [rate, setRate] = useState(savedRate);
  useEffect(() => {
    setBudget(savedBudget);
    setRate(savedRate);
  }, [savedBudget, savedRate]);
  const dirty = budget !== savedBudget || rate !== savedRate;
  return (
    <>
      <SettingSection id="policy-budget" title={copy.settings.sections.policyBudget}>
        <SettingRow title={copy.settings.maxBudget} description={copy.settings.maxBudgetDesc}>
          <Input
            aria-label={copy.settings.maxBudget}
            className="h-8 w-28 text-right tabular-nums"
            type="number"
            min={0}
            value={budget}
            onChange={(e) => setBudget(e.target.value)}
          />
        </SettingRow>
        <SettingRow title={copy.settings.usdPerMtok} description={copy.settings.usdPerMtokDesc}>
          <Input
            aria-label={copy.settings.usdPerMtok}
            className="h-8 w-28 text-right tabular-nums"
            type="number"
            min={0}
            value={rate}
            onChange={(e) => setRate(e.target.value)}
          />
        </SettingRow>
      </SettingSection>
      <SettingsSaveBar
        open={dirty}
        onDiscard={() => {
          setBudget(savedBudget);
          setRate(savedRate);
        }}
        onSave={async () => {
          await host.patch({ maxBudgetUsd: Number(budget) || 0, usdPerMtok: Number(rate) || 0 });
          toast.success(copy.app.controlSaved);
        }}
      />
    </>
  );
}

/** PolicySettings 是安全与隐私页：审批、运行环境、数据与记忆。预算在模型页。host 读取 cfg，调用 patch，写到 IsolationSection 和 DataSection 的 host。 */
export function PolicySettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  return (
    <>
      <SettingSection id="policy-gate" title={copy.settings.sections.policyGate}>
        <SettingRow title={copy.settings.gateMode} description={copy.settings.gateModeDesc} align="start">
          <SettingSegmented
            id="gate"
            ariaLabel={copy.settings.gateMode}
            value={host.cfg.gateMode || "manual"}
            options={[
              { value: "manual", label: copy.settings.gateManual },
              { value: "autosafe", label: copy.settings.gateAutoSafe },
              { value: "skip", label: copy.settings.gateSkip },
            ]}
            onChange={(v) => void host.patch({ gateMode: v })}
          />
        </SettingRow>
        <SettingRow title={copy.settings.autoAllow} description={copy.settings.autoAllowDesc}>
          <Switch checked={!!host.cfg.autoAllow} onCheckedChange={(v) => void host.patch({ autoAllow: v })} />
        </SettingRow>
        <SettingRow title={copy.settings.crashResume} description={copy.settings.crashResumeDesc}>
          <Switch
            checked={host.cfg.crashResume !== false}
            onCheckedChange={(v) => void host.patch({ crashResume: v })}
          />
        </SettingRow>
      </SettingSection>

      <IsolationSection host={host} />

      <DataSection host={host} />
    </>
  );
}

type PwshInstall = {
  ready?: boolean;
  needed?: boolean;
  phase?: string;
  percent?: number;
  error?: string;
  bin?: string;
  version?: string;
  shell?: string;
};

/** 安装还在解析、下载、解压或校验时为真。phase 是 resolving、downloading、extracting 或 verifying 时返回真。 */
function pwshBusy(phase?: string) {
  return phase === "resolving" || phase === "downloading" || phase === "extracting" || phase === "verifying";
}

/** 显示 PowerShell 运行时的状态，并能安装。放在运行环境一块的末尾。 */
function PwshRows() {
  const copy = useCopy();
  const [st, setSt] = useState<PwshInstall>({});
  /** 把状态收成对象。s 为真时才继续，用 typeof 看类型，作为分支结果。 */
  function apply(s: any) {
    setSt(s && typeof s === "object" ? s : {});
  }
  useEffect(() => {
    void api.runtime.pwshStatus().then(apply).catch(() => {});
  }, []);
  const phase = String(st.phase || "");
  const working = pwshBusy(phase);
  useEffect(() => {
    if (!working) return;
    const t = window.setInterval(() => {
      void api.runtime.pwshStatus().then(apply).catch(() => {});
    }, 400);
    return () => window.clearInterval(t);
  }, [working]);
  const needed = st.needed !== false;
  const ready = !!st.ready;
  return (
    <div data-setting-section-highlight-target="policy-shell">
      <SettingRow title={copy.settings.pwsh} description={`${st.shell || st.bin || copy.settings.pwshFallback} · ${needed ? copy.settings.pwshHint : copy.settings.pwshNotNeeded}`} list border={false}>
        <span
          className={cn(
            "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-ui-footnote font-medium",
            ready ? "bg-success/15 text-success" : working ? "bg-accent/15 text-accent" : "bg-danger/10 text-danger",
          )}
        >
          <span
            className={cn("size-1.5 rounded-full", ready ? "bg-success" : working ? "bg-accent" : "bg-danger")}
            aria-hidden
          />
          {ready
            ? copy.settings.pwshReady
            : working
              ? `${copy.settings.pwshInstalling} ${Math.max(0, Math.min(100, Math.round(Number(st.percent) || 0)))}%`
              : phase === "error"
                ? copy.settings.pwshInstallFailed
                : needed
                  ? copy.settings.pwshMissing
                  : copy.settings.pwshReady}
        </span>
        {needed && !ready && !working ? (
          <Button
            size="sm"
            variant="lift"
            onClick={() => {
              void api.runtime
                .pwshInstall()
                .then(apply)
                .catch((e) => toast.error(api.errMessage(e)));
            }}
          >
            <Download aria-hidden />
            {copy.settings.pwshInstall}
          </Button>
        ) : null}
      </SettingRow>
      {working ? <ProgressHairline value={Number(st.percent) || 0} /> : null}
      {phase === "error" && !ready && st.error ? (
        <p className="pb-2 text-ui-footnote leading-[1.45] text-danger">{String(st.error)}</p>
      ) : null}
    </div>
  );
}

/** 投影隔离报告。host 读取 doctor、vault。 */
function IsolationSection({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [rep, setRep] = useState<any>(host.doctor?.isolation || {});
  useEffect(() => {
    void api.isolationReport().then(setRep).catch(() => {});
  }, [host.doctor]);
  const os = rep?.os || {};
  const vault = rep?.vault || host.vault || {};
  const sandbox = !!(os.sandbox ?? os.Sandbox);
  return (
    <SettingSection
      id="policy-isolation"
      title={copy.settings.sections.policyIsolation}
      footnote={`${copy.settings.isolationHint} ${copy.settings.isolationSleep}`}
    >
      <SettingValueRow title={copy.settings.isolationKind} value={str(os.kind || os.Kind, "none")} mono />
      <SettingRow title={copy.settings.isolationSandbox}>
        <StatusPill ok={sandbox} label={sandbox ? copy.settings.yes : copy.settings.no} />
      </SettingRow>
      <SettingValueRow title={copy.settings.isolationSigned} value={str(os.signed || os.Signed, "unsigned")} mono />
      <SettingValueRow title={copy.settings.isolationVault} value={str(vault.source || vault.Source, "empty")} mono />
      <SettingValueRow
        title={copy.settings.isolationBrowser}
        value={rep?.browser_isolated || rep?.browserIsolated ? "isolated profile" : "unset"}
        mono
      />
      <SettingValueRow title={copy.settings.awakeRequired} value={copy.settings.yes} />
      <SettingValueRow title={copy.settings.lidCloseStops} value={copy.settings.yes} />
      <PwshRows />
    </SettingSection>
  );
}

/** DataSection 是数据入口：记忆去今天栏的记忆页改，诊断包导出到本机。 */
function DataSection({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [count, setCount] = useState<number | null>(null);
  useEffect(() => {
    void api.memoryList().then((rows) => setCount(Array.isArray(rows) ? rows.length : 0)).catch(() => setCount(null));
  }, []);
  return (
    <SettingSection id="policy-data" title={copy.settings.sections.policyData} description={copy.settings.dataDesc}>
      <SettingRow title={copy.settings.dataMemory} description={count === null ? copy.settings.dataMemoryDesc : `${copy.settings.dataMemoryDesc} ${copy.settings.dataMemoryCount.replace("{n}", String(count))}`}>
        <Button size="sm" variant="lift" onClick={() => openMemoryPage()}>
          {copy.settings.dataOpenMemory}
        </Button>
      </SettingRow>
      {host.onExportDiagnostics ? (
        <SettingRow title={copy.settings.exportBundle} description={copy.settings.dataExportDesc} border={false}>
          <Button
            size="sm"
            variant="lift"
            onClick={async () => {
              try {
                await host.onExportDiagnostics?.();
                toast.success(copy.settings.exportedBundle);
              } catch (e) {
                toast.error(String(e));
              }
            }}
          >
            <Download aria-hidden />
            {copy.settings.exportBundle}
          </Button>
        </SettingRow>
      ) : null}
    </SettingSection>
  );
}

/** openMemoryPage 关掉设置回到会话，让今天栏翻到记忆。今天栏还没挂上时，等它挂上再翻。 */
function openMemoryPage() {
  useUI.getState().closeSettings();
  openDesk({ tab: "memory" });
}

/** 画一颗成功或失败的状态。ok 为真时才继续。label 作为子节点渲染。 */
function StatusPill({ ok, label }: { ok: boolean; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-ui-footnote font-medium",
        ok ? "bg-success/15 text-success" : "bg-lift text-muted",
      )}
    >
      <span className={cn("size-1.5 rounded-full", ok ? "bg-success" : "bg-muted")} aria-hidden />
      {label}
    </span>
  );
}

/**
 * AboutSettings 是关于页：字标、一句话介绍、当前版本，以及检查和安装更新。
 * host 读取 cfg、health，调用 onCheckUpdate，调用 patch，调用 onApplyUpdate。
 */
export function AboutSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [url, setUrl] = useState(host.cfg.updateUrl);
  const [applyOpen, setApplyOpen] = useState(false);
  const [ota, setOta] = useState<any>(null);
  const phase = String(ota?.phase || "idle");
  const ready = phase === "ready" || !!ota?.downloaded;
  const percent = Number(ota?.percent || 0);
  useEffect(() => setUrl(host.cfg.updateUrl), [host.cfg.updateUrl]);
  useEffect(() => {
    let stop = false;
    void api.updateStatus().then((s) => {
      if (!stop) setOta(s);
    }).catch(() => {});
    return () => {
      stop = true;
    };
  }, []);
  useEffect(() => {
    if (phase !== "checking" && phase !== "downloading") return;
    const id = window.setInterval(() => {
      void api.updateStatus().then(setOta).catch(() => {});
    }, 1500);
    return () => window.clearInterval(id);
  }, [phase]);
  const busy = phase === "checking" || phase === "downloading";
  // 版本那一行下面的一句状态：在查、在下、可以装了，或者上次什么时候查过。
  const status = phase === "checking"
    ? copy.settings.checking
    : phase === "downloading"
      ? `${copy.settings.downloading} ${percent}%`
      : ready
        ? `${copy.settings.updateReady} ${String(ota?.manifest?.version || ota?.asset?.filename || "")}`.trim()
        : ota?.last_check
          ? `${copy.settings.lastCheck} ${String(ota.last_check)}`
          : undefined;

  return (
    <>
      <div className="mb-8 flex items-center gap-4 px-1">
        <MarkWell />
        <div className="min-w-0 flex-1">
          <div className="text-ui-title-s font-semibold text-foreground">Yoyo</div>
          <p className="mt-0.5 text-ui-body leading-[1.5] text-muted">{copy.settings.aboutBody}</p>
        </div>
      </div>

      <SettingSection id="advanced-updates" title={copy.settings.sections.advancedUpdates}>
        <SettingRow title={copy.settings.currentVersion} description={status}>
          <span className="font-mono text-ui-footnote tabular-nums text-muted">{host.health.version}</span>
          <Button
            size="sm"
            variant="lift"
            disabled={busy}
            onClick={async () => {
              const u = await host.onCheckUpdate();
              setOta(u);
              if (u?.error) toast.error(String(u.error));
              else if (u?.downloaded || u?.phase === "ready") toast.success(copy.settings.updateReady);
              else if (u?.newer || u?.phase === "downloading") toast.message(copy.settings.downloading);
              else toast.message(copy.settings.upToDate);
            }}
          >
            {copy.settings.checkUpdate}
          </Button>
          {ready ? (
            <Button size="sm" onClick={() => setApplyOpen(true)}>
              {copy.settings.applyUpdate}
            </Button>
          ) : null}
        </SettingRow>
        {busy ? <ProgressHairline value={phase === "downloading" ? percent : 0} /> : null}
        {ota?.error ? <p className="text-ui-body leading-[1.5] text-danger">{String(ota.error)}</p> : null}
        <SettingRow title={copy.settings.autoCheck} description={copy.settings.autoCheckDesc}>
          <Switch
            checked={host.cfg.updateAutoCheck !== false}
            onCheckedChange={(v) => void host.patch({ updateAutoCheck: v })}
          />
        </SettingRow>
        <SettingRow title={copy.settings.updateUrl} description={copy.settings.updateUrlDesc} stack>
          <Input
            aria-label={copy.settings.updateUrl}
            className={cn(CONTROL_LG, "font-mono text-ui-footnote")}
            placeholder={String(ota?.feed || "")}
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onBlur={() => {
              if (url !== host.cfg.updateUrl) void host.patch({ updateUrl: url });
            }}
          />
        </SettingRow>
      </SettingSection>

      <ConfirmDialog
        open={applyOpen}
        title={copy.settings.applyUpdate}
        body={copy.settings.applyUpdateConfirm}
        danger
        confirmLabel={copy.settings.applyUpdate}
        onCancel={() => setApplyOpen(false)}
        onConfirm={async () => {
          setApplyOpen(false);
          await host.onApplyUpdate();
        }}
      />
    </>
  );
}

/**
 * DiagnosticsSettings 是高级页里的诊断：日志尾部、审计记录和本机检查，可以复制路径、在文件夹里显示、导出支持包。
 * host 读取 logs、journal、doctor、onExportDiagnostics、onDiagnose、onRevealJournal，调用 onRevealLogs，调用 onExportDiagnostics，调用 onDiagnose，调用 onRevealJournal。
 */
export function DiagnosticsSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [level, setLevel] = useState("all");
  const [copiedPath, setCopiedPath] = useState(false);
  const lines = asArray(host.logs?.lines);
  const diagnostic = lines.length
    ? lines
        .filter((r: any) => level === "all" || String(r.level || "").toLowerCase() === level)
        .map((r: any) => r.raw || `${r.ts || ""} ${r.level || ""} ${r.component || ""} ${r.msg || ""}`)
        .join("\n")
    : String(host.logs?.text || "");
  const journalRecords = asArray(host.journal?.journal);
  const journal = journalRecords.length
    ? journalRecords
        .map(
          (r: any) =>
            `${r.time || r.Time || ""} ${r.type || r.Type || ""} ${
              typeof r.payload === "string" ? r.payload : JSON.stringify(r.payload || r.Payload || r)
            }`,
        )
        .join("\n")
    : "";
  const logPath = String(host.logs?.path || host.logs?.dir || "");

  return (
    <>
      <SettingSection
        id="advanced-logs"
        title={copy.settings.sections.advancedLogs}
        description={copy.settings.logsHint}
        bare
        tight
        headerActions={
          <>
            {host.onDiagnose ? (
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  try {
                    await host.onDiagnose?.();
                    toast.success(copy.settings.diagnoseTurn);
                  } catch (e) {
                    toast.error(String(e));
                  }
                }}
              >
                <Stethoscope aria-hidden />
                {copy.settings.diagnoseTurn}
              </Button>
            ) : null}
            {host.onExportDiagnostics ? (
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  try {
                    await host.onExportDiagnostics?.();
                    toast.success(copy.settings.exportedBundle);
                  } catch (e) {
                    toast.error(String(e));
                  }
                }}
              >
                <Download aria-hidden />
                {copy.settings.exportBundle}
              </Button>
            ) : null}
            <Button size="sm" variant="lift" onClick={() => void host.onRevealLogs()}>
              <FolderOpen aria-hidden />
              {copy.settings.reveal}
            </Button>
          </>
        }
      >
        <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-2">
          <SettingSegmented
            id="log-level"
            ariaLabel={copy.settings.filterLevel}
            value={level}
            options={[
              { value: "all", label: "All" },
              { value: "info", label: "Info" },
              { value: "warn", label: "Warn" },
              { value: "error", label: "Error" },
            ]}
            onChange={setLevel}
          />
          {logPath ? (
            <button
              type="button"
              aria-label={copy.settings.copyPath}
              title={copy.settings.copyPath}
              className="ml-auto flex h-7 min-w-0 items-center gap-1.5 rounded-md px-2 font-mono text-ui-footnote text-muted transition-colors hover:bg-lift/70 hover:text-foreground"
              onClick={async () => {
                if (await writeClipboard(logPath)) {
                  setCopiedPath(true);
                  window.setTimeout(() => setCopiedPath(false), 1200);
                  toast.success(copy.settings.pathCopied);
                }
              }}
            >
              <span className="truncate">{logPath}</span>
              <IconSwap
                on={copiedPath}
                className="size-3.5 shrink-0"
                off={<Copy className="size-3.5" aria-hidden />}
                live={<Check className="size-3.5" aria-hidden />}
              />
            </button>
          ) : null}
        </div>
        <div data-testid="diagnostics-tail">
          <SettingCodePanel className="max-h-[360px] min-h-[96px]">
            {diagnostic || copy.settings.emptyLogs}
          </SettingCodePanel>
        </div>
      </SettingSection>

      <SettingSection
        id="advanced-journal"
        title={copy.settings.sections.advancedJournal}
        description={copy.settings.journalHint}
        bare
        tight
        headerActions={
          host.onRevealJournal ? (
            <Button size="sm" variant="lift" onClick={() => void host.onRevealJournal?.()}>
              <FolderOpen aria-hidden />
              {copy.settings.reveal}
            </Button>
          ) : undefined
        }
      >
        <div data-testid="journal-tail">
          <SettingCodePanel className="max-h-[240px] min-h-[72px]">
            {journal || copy.settings.emptyJournal}
          </SettingCodePanel>
        </div>
      </SettingSection>

      <SettingSection id="advanced-doctor" title={copy.settings.sections.advancedDoctor} bare tight>
        <SettingCodePanel className="max-h-64 min-h-[96px]">
          {JSON.stringify(host.doctor || {}, null, 2).slice(0, 8000)}
        </SettingCodePanel>
      </SettingSection>
    </>
  );
}

/** 按色板编号取出名字和说明。id 用来做分支。palettes 读取 ink、inkHint、dim、dimHint、slate、slateHint 等。 */
function paletteCopy(id: PaletteId, palettes: ReturnType<typeof useCopy>["settings"]["palettes"]) {
  switch (id) {
    case "ink": return { name: palettes.ink, hint: palettes.inkHint };
    case "dim": return { name: palettes.dim, hint: palettes.dimHint };
    case "slate": return { name: palettes.slate, hint: palettes.slateHint };
    case "neutral": return { name: palettes.neutral, hint: palettes.neutralHint };
    case "paper": return { name: palettes.paper, hint: palettes.paperHint };
    case "mist": return { name: palettes.mist, hint: palettes.mistHint };
  }
}

/** 画一组色板供挑选。label 作为子节点渲染。ids 做映射。selected 与 id 比较是否相等。onPick 被调用，并传入 id。className 写到 div 的 className。 */
function PaletteGroup<T extends DarkPalette | LightPalette>({
  label,
  ids,
  selected,
  onPick,
  className,
}: {
  label: string;
  ids: readonly T[];
  selected: T;
  onPick: (id: T) => void;
  className?: string;
}) {
  const copy = useCopy();
  return (
    <div className={className}>
      <div className="mb-2 px-1 text-ui-caption font-medium text-muted">{label}</div>
      <div className="grid grid-cols-3 gap-3">
        {ids.map((id) => {
          const meta = paletteCopy(id, copy.settings.palettes);
          const active = selected === id;
          return (
            <button
              type="button"
              key={id}
              aria-pressed={active}
              aria-label={`${meta.name}. ${meta.hint}`}
              className={cn(
                "group cursor-pointer rounded-xl border bg-card p-2 text-left transition-[background-color,border-color,transform] duration-200 ease-[var(--ease-out)]",
                active ? "bg-lift" : "hover:bg-lift/70",
              )}
              onClick={() => onPick(id)}
            >
              <div className="h-[56px] overflow-hidden rounded-xl border border-border">
                <PreviewPane palette={id} />
              </div>
              <div className="mt-2 flex items-center gap-1 text-ui-footnote font-medium text-foreground">
                {active ? <Check className="size-3.5 text-accent" aria-hidden /> : null}
                {meta.name}
              </div>
              <p className="mt-0.5 text-left text-ui-caption leading-[1.4] text-muted">{meta.hint}</p>
            </button>
          );
        })}
      </div>
    </div>
  );
}

/** 预览当前明暗色板。mode 与 system、dark 比较是否相等。dark 作为分支结果，写到 PreviewPane 的 palette。light 作为分支结果，写到 PreviewPane 的 palette。 */
function ThemePreview({ mode, dark, light }: { mode: ThemePref; dark: DarkPalette; light: LightPalette }) {
  if (mode === "system") {
    return (
      <div className="flex h-[56px] overflow-hidden rounded-xl border border-border">
        <div className="w-1/2 overflow-hidden">
          <PreviewPane palette={dark} />
        </div>
        <div className="w-1/2 overflow-hidden border-l border-border">
          <PreviewPane palette={light} />
        </div>
      </div>
    );
  }
  return (
    <div className="h-[56px] overflow-hidden rounded-xl border border-border">
      <PreviewPane palette={mode === "dark" ? dark : light} />
    </div>
  );
}

/** 画出一块色板。palette 传给 builtinTokens，写到 div 的 data-theme-preview。 */
function PreviewPane({ palette }: { palette: PaletteId }) {
  const tokens = builtinTokens(palette);
  return (
    <div
      className="flex h-full w-full"
      data-theme-preview={palette}
      style={{ background: tokens["--background"], borderColor: tokens["--border"] }}
    >
      <div className="h-full w-[28%] border-r" style={{ background: tokens["--sidebar"], borderColor: tokens["--border"] }} />
      <div className="flex-1 space-y-[3px] p-[6px]">
        <div className="h-[3px] w-4/5 rounded-full" style={{ background: tokens["--muted"] }} />
        <div className="h-[3px] w-3/5 rounded-full" style={{ background: tokens["--muted"] }} />
        <div className="mt-[5px] h-[14px] w-full rounded-sm" style={{ background: tokens["--card"], border: `1px solid ${tokens["--border"]}` }} />
        <div className="mt-[4px] h-[4px] w-8 rounded-full" style={{ background: tokens["--accent"] }} />
      </div>
    </div>
  );
}
