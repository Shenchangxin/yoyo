import { useEffect, useState } from "react";
import { Check, FolderOpen } from "lucide-react";
import { toast } from "sonner";
import * as api from "../../lib/client";
import { applyLocale, getLocale, useCopy, type Locale } from "../../lib/i18n";
import { DEFAULT_KEYMAP, formatShortcut, mergeKeymap, type KeymapId } from "../../lib/keymap";
import { DARK_PALETTES, LIGHT_PALETTES, useTheme, type DarkPalette, type LightPalette, type PaletteId, type ThemePref } from "../../lib/theme";
import { applyUiScale } from "../../lib/scale";
import { chrome } from "../../lib/chrome";
import { asArray, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import { Slider } from "../../components/ui/slider";
import { ConfirmDialog } from "../ConfirmDialog";
import {
  CONTROL_LG,
  CONTROL_SM,
  SettingActionRow,
  SettingCodePanel,
  SettingRow,
  SettingSection,
  SettingSegmented,
  SettingValueRow,
  SettingsPageHeader,
  SettingsSaveBar,
} from "./SettingChrome";
import { cn } from "../../lib/utils";
import type { SettingsHost } from "./host";

export type { SettingsHost } from "./host";
export { ProviderSettings } from "./ProviderPanel";

export function GeneralSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const cfg = host.cfg;
  const notifications = cfg.notificationsEnabled !== false;
  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.general} description={copy.settings.tabHints.general} />

      <SettingSection
        id="general-workspace"
        title={copy.settings.sections.generalWorkspace}
        footnote={copy.settings.workspaceDesc}
      >
        <SettingRow stack border={false}>
          <div className="flex items-center gap-2">
            <Input
              aria-label={copy.settings.workspace}
              className="h-8 flex-1 font-mono text-[12px]"
              value={cfg.workspace}
              onChange={(e) => void host.patch({ workspace: e.target.value })}
            />
            <Button
              size="sm"
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
      </SettingSection>

      <SettingSection id="general-notifications" title={copy.settings.sections.generalNotifications}>
        <SettingRow title={copy.settings.notifications} description={copy.settings.notificationsDesc}>
          <Switch checked={notifications} onCheckedChange={(v) => void host.patch({ notificationsEnabled: v })} />
        </SettingRow>
        <SettingRow title={copy.settings.notifyUnfocused} description={copy.settings.notifyUnfocusedDesc} border={false}>
          <Switch
            disabled={!notifications}
            checked={!!cfg.notifyWhenUnfocusedOnly}
            onCheckedChange={(v) => void host.patch({ notifyWhenUnfocusedOnly: v })}
          />
        </SettingRow>
      </SettingSection>

      <SettingSection id="general-desktop" title={copy.settings.sections.generalDesktop}>
        <SettingRow title={copy.settings.closeToTray} description={copy.settings.closeToTrayDesc}>
          <Switch checked={!!cfg.closeToTray} onCheckedChange={(v) => void host.patch({ closeToTray: v })} />
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
      </SettingSection>
    </>
  );
}

export function AppearanceSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const { pref, setPref, resolved, darkPalette, lightPalette, setDarkPalette, setLightPalette } = useTheme();
  const locale = (host.cfg.locale === "zh-CN" ? "zh-CN" : host.cfg.locale === "en" ? "en" : getLocale()) as Locale;
  const scale = host.cfg.uiScale && host.cfg.uiScale > 0 ? host.cfg.uiScale : 1;
  const modes: { id: ThemePref; label: string }[] = [
    { id: "system", label: copy.settings.system },
    { id: "dark", label: copy.settings.dark },
    { id: "light", label: copy.settings.light },
  ];
  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.appearance} description={copy.settings.tabHints.appearance} />

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
                  "group cursor-pointer rounded-[12px] border bg-card p-2 text-left transition-colors",
                  active ? "border-accent/70 ring-2 ring-accent/25" : "border-border hover:border-muted/50",
                )}
                onClick={() => {
                  setPref(m.id);
                  void host.patch({ theme: m.id });
                }}
              >
                <ThemePreview mode={m.id} dark={darkPalette} light={lightPalette} />
                <div className="mt-2 flex items-center justify-center gap-1 text-[12px] font-medium text-foreground">
                  {active ? <Check className="size-3.5 text-accent" aria-hidden /> : null}
                  {m.label}
                </div>
              </button>
            );
          })}
        </div>
      </SettingSection>

      <SettingSection
        id="appearance-palette"
        title={copy.settings.sections.appearancePalette}
        footnote={copy.settings.palettesDesc}
        bare
      >
        <PaletteGroup
          label={copy.settings.paletteGroupDark}
          ids={DARK_PALETTES}
          selected={darkPalette}
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

      <SettingSection id="appearance-display" title={copy.settings.sections.appearanceDisplay}>
        <SettingRow title={copy.settings.language} description={copy.settings.languageDesc}>
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
        <SettingRow title={copy.settings.uiScale} description={copy.settings.uiScaleDesc} border={false}>
          <div className="flex w-[208px] items-center gap-3">
            <Slider
              min={0.85}
              max={1.25}
              step={0.05}
              value={[scale]}
              onValueChange={([v]) => {
                applyUiScale(v);
                void host.patch({ uiScale: v });
              }}
            />
            <span className="w-9 shrink-0 text-right text-[12px] tabular-nums text-muted">
              {Math.round(scale * 100)}%
            </span>
          </div>
        </SettingRow>
      </SettingSection>
    </>
  );
}

export function ShortcutsSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const keymap = mergeKeymap(host.cfg.keymap);
  const [armed, setArmed] = useState<KeymapId | null>(null);
  const ids = Object.keys(DEFAULT_KEYMAP) as KeymapId[];
  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.shortcuts} description={copy.settings.tabHints.shortcuts} />
      <SettingSection
        id="shortcuts-map"
        title={copy.settings.sections.shortcutsMap}
        footnote={copy.settings.recordHint}
      >
        {ids.map((id, i) => (
          <SettingRow key={id} title={copy.settings.keys[id] || id} border={i < ids.length - 1}>
            <button
              type="button"
              aria-label={id}
              className={cn(
                "h-8 min-w-[7.5rem] rounded-[9px] border px-3 text-[12px] tabular-nums transition-colors",
                armed === id
                  ? "border-accent/70 bg-accent/10 text-foreground"
                  : "border-border bg-background text-foreground hover:bg-lift/60",
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
              {armed === id ? copy.settings.recording : keymap[id]}
            </button>
          </SettingRow>
        ))}
      </SettingSection>
    </>
  );
}

export function PolicySettings({ host }: { host: SettingsHost }) {
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
      <SettingsPageHeader title={copy.settings.tabs.policy} description={copy.settings.tabHints.policy} />

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
        <SettingRow title={copy.settings.searchUrl} description={copy.settings.searchUrlDesc} border={false}>
          <Input
            aria-label={copy.settings.searchUrl}
            className={CONTROL_LG}
            value={host.cfg.searchUrl || ""}
            onChange={(e) => void host.patch({ searchUrl: e.target.value })}
          />
        </SettingRow>
      </SettingSection>

      <SettingSection id="policy-budget" title={copy.settings.sections.policyBudget}>
        <SettingRow title={copy.settings.maxBudget}>
          <Input
            aria-label={copy.settings.maxBudget}
            className={CONTROL_SM}
            type="number"
            value={budget}
            onChange={(e) => setBudget(e.target.value)}
          />
        </SettingRow>
        <SettingRow title={copy.settings.usdPerMtok} border={false}>
          <Input
            aria-label={copy.settings.usdPerMtok}
            className={CONTROL_SM}
            type="number"
            value={rate}
            onChange={(e) => setRate(e.target.value)}
          />
        </SettingRow>
      </SettingSection>

      <IsolationSection host={host} />

      <SettingsSaveBar
        open={dirty}
        saveLabel={copy.settings.savePolicy}
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
        border={false}
      />
    </SettingSection>
  );
}

function StatusPill({ ok, label }: { ok: boolean; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11.5px] font-medium",
        ok ? "bg-success/15 text-success" : "bg-lift text-muted",
      )}
    >
      <span className={cn("size-1.5 rounded-full", ok ? "bg-success" : "bg-muted")} aria-hidden />
      {label}
    </span>
  );
}

export function AdvancedSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [url, setUrl] = useState(host.cfg.updateUrl);
  const [applyOpen, setApplyOpen] = useState(false);
  useEffect(() => setUrl(host.cfg.updateUrl), [host.cfg.updateUrl]);
  const channel = host.cfg.updateChannel || "nightly";
  const records = asArray(host.logs?.journal || host.logs);
  const journal = records.length
    ? records
        .map(
          (r: any) =>
            `${r.time || r.Time || ""} ${r.type || r.Type || ""} ${
              typeof r.payload === "string" ? r.payload : JSON.stringify(r.payload || r.Payload || r)
            }`,
        )
        .join("\n")
    : host.logs
      ? JSON.stringify(host.logs, null, 2).slice(0, 12000)
      : copy.settings.emptyLogs;

  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.advanced} description={copy.settings.tabHints.advanced} />

      <SettingSection
        id="advanced-updates"
        title={copy.settings.sections.advancedUpdates}
        footnote={copy.settings.updateUrlDesc}
        actions={
          <Button
            size="sm"
            variant="ghost"
            onClick={async () => {
              const u = await host.onCheckUpdate();
              if (u?.error) toast.error(String(u.error));
              else if (u?.downloaded) toast.success(`downloaded ${u.size || 0} bytes`);
              else if (u?.newer) toast.message("newer build verified — apply staged");
              else toast.message(u?.staged ? `staged ${u.size || 0} bytes` : copy.settings.noStaged);
            }}
          >
            {copy.settings.checkUpdate}
          </Button>
        }
      >
        <SettingValueRow title={copy.settings.currentVersion} value={host.health.version} mono />
        <SettingRow title={copy.settings.channel}>
          <SettingSegmented
            id="channel"
            ariaLabel={copy.settings.channel}
            value={channel}
            options={[
              { value: "stable", label: "Stable" },
              { value: "beta", label: "Beta" },
              { value: "nightly", label: "Nightly" },
            ]}
            onChange={(v) => void host.patch({ updateChannel: v })}
          />
        </SettingRow>
        <SettingRow title={copy.settings.updateUrl} border={false}>
          <Input
            aria-label={copy.settings.updateUrl}
            className={cn(CONTROL_LG, "font-mono")}
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onBlur={() => {
              if (url !== host.cfg.updateUrl) void host.patch({ updateUrl: url });
            }}
          />
        </SettingRow>
      </SettingSection>

      <SettingSection
        id="advanced-logs"
        title={copy.settings.sections.advancedLogs}
        footnote={copy.settings.logsHint}
        actions={
          <Button size="sm" variant="ghost" onClick={() => void host.onRevealLogs()}>
            {copy.settings.reveal}
          </Button>
        }
      >
        <SettingCodePanel className="max-h-[360px] min-h-[96px]">{journal}</SettingCodePanel>
      </SettingSection>

      <SettingSection id="advanced-doctor" title={copy.settings.sections.advancedDoctor}>
        <SettingCodePanel className="max-h-64 min-h-[96px]">
          {JSON.stringify(host.doctor || {}, null, 2).slice(0, 8000)}
        </SettingCodePanel>
      </SettingSection>

      <SettingSection id="advanced-danger" title={copy.settings.sections.advancedDanger}>
        <SettingActionRow
          danger
          border={false}
          title={copy.settings.applyUpdate}
          description={copy.settings.applyUpdateConfirm}
          onClick={() => setApplyOpen(true)}
        />
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
      <div className="mb-2 px-1 text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80">{label}</div>
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
                "group cursor-pointer rounded-[12px] border bg-card p-2 text-left transition-colors",
                active ? "border-accent/70 ring-2 ring-accent/25" : "border-border hover:border-muted/50",
              )}
              onClick={() => onPick(id)}
            >
              <div className="h-[52px] overflow-hidden rounded-[7px] border border-border">
                <PreviewPane palette={id} />
              </div>
              <div className="mt-2 flex items-center justify-center gap-1 text-[12px] font-medium text-foreground">
                {active ? <Check className="size-3.5 text-accent" aria-hidden /> : null}
                {meta.name}
              </div>
              <p className="mt-0.5 text-center text-[11px] leading-[1.4] text-muted">{meta.hint}</p>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function ThemePreview({ mode, dark, light }: { mode: ThemePref; dark: DarkPalette; light: LightPalette }) {
  if (mode === "system") {
    return (
      <div className="flex h-[52px] overflow-hidden rounded-[7px] border border-border">
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
    <div className="h-[52px] overflow-hidden rounded-[7px] border border-border">
      <PreviewPane palette={mode === "dark" ? dark : light} />
    </div>
  );
}

function PreviewPane({ palette }: { palette: PaletteId }) {
  return (
    <div className="flex h-full w-full" data-theme-preview={palette}>
      <div className="theme-preview-rail h-full w-[28%] border-r" />
      <div className="flex-1 space-y-[3px] p-[6px]">
        <div className="theme-preview-line h-[3px] w-4/5 rounded-full" />
        <div className="theme-preview-line h-[3px] w-3/5 rounded-full" />
        <div className="theme-preview-bubble mt-[5px] h-[14px] w-full rounded-[3px]" />
      </div>
    </div>
  );
}
