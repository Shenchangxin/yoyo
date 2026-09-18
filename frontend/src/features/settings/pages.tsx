import { useEffect, useState } from "react";
import { toast } from "sonner";
import { applyLocale, getLocale, useCopy, type Locale } from "../../lib/i18n";
import { DEFAULT_KEYMAP, formatShortcut, mergeKeymap, type KeymapId } from "../../lib/keymap";
import { useTheme, type ThemePref } from "../../lib/theme";
import { applyUiScale } from "../../lib/scale";
import { chrome } from "../../lib/chrome";
import { asArray, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import { Slider } from "../../components/ui/slider";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { ConfirmDialog } from "../ConfirmDialog";
import { SettingRow, SettingSection } from "./SettingChrome";
import { cn } from "../../lib/utils";
import type { SettingsHost } from "./host";

export type { SettingsHost } from "./host";
export { ProviderSettings } from "./ProviderPanel";

export function GeneralSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const cfg = host.cfg;
  const scale = cfg.uiScale && cfg.uiScale > 0 ? cfg.uiScale : 1;
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.general}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.hint}</p>
      <SettingSection id="general-basics" title={copy.settings.sections.generalBasics}>
        <SettingRow title={copy.settings.workspace} description={copy.settings.workspaceDesc}>
          <div className="flex gap-2">
            <Input
              className="h-8 w-56 text-[12px]"
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
              {copy.settings.browse}
            </Button>
          </div>
        </SettingRow>
        <SettingRow title={copy.settings.notifications} description={copy.settings.notificationsDesc} border={false}>
          <Switch
            checked={cfg.notificationsEnabled !== false}
            onCheckedChange={(v) => void host.patch({ notificationsEnabled: v })}
          />
        </SettingRow>
      </SettingSection>
      <SettingSection id="general-app" title={copy.settings.sections.generalApp}>
        <SettingRow title={copy.settings.notifyUnfocused} description={copy.settings.notifyUnfocusedDesc}>
          <Switch
            checked={!!cfg.notifyWhenUnfocusedOnly}
            onCheckedChange={(v) => void host.patch({ notifyWhenUnfocusedOnly: v })}
          />
        </SettingRow>
        <SettingRow title={copy.settings.uiScale} description={copy.settings.uiScaleDesc} border={false}>
          <div className="flex items-center gap-3">
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
            <span className="w-10 text-right text-[12px] tabular-nums text-muted">{Math.round(scale * 100)}%</span>
          </div>
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
  const { pref, setPref, resolved } = useTheme();
  const locale = (host.cfg.locale === "zh-CN" ? "zh-CN" : host.cfg.locale === "en" ? "en" : getLocale()) as Locale;
  const modes: { id: ThemePref; label: string }[] = [
    { id: "system", label: copy.settings.system },
    { id: "dark", label: copy.settings.dark },
    { id: "light", label: copy.settings.light },
  ];
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.appearance}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.control.appearanceHint}</p>
      <SettingSection id="appearance-mode" title={copy.settings.sections.appearanceMode} description={copy.settings.colorModeDesc}>
        <div className="flex flex-wrap gap-3 px-5 py-4">
          {modes.map((m) => (
            <button
              type="button"
              key={m.id}
              className={cn(
                "w-[108px] rounded-xl border p-2 text-left transition-colors",
                pref === m.id ? "border-foreground/40 ring-1 ring-foreground/20" : "border-border hover:border-muted",
              )}
              onClick={() => {
                setPref(m.id);
                void host.patch({ theme: m.id });
              }}
            >
              <ThemePreview mode={m.id} />
              <div className="mt-2 text-center text-[12px] font-medium">{m.label}</div>
            </button>
          ))}
        </div>
        <p className="border-t border-border/80 px-5 py-3 text-[12px] text-muted">
          {copy.settings.resolved} {resolved}
        </p>
      </SettingSection>
      <SettingSection id="appearance-language" title={copy.settings.sections.appearanceLanguage}>
        <SettingRow title={copy.settings.language} description={copy.settings.languageDesc} border={false}>
          <Select
            value={locale}
            onValueChange={(v) => {
              const next = v as Locale;
              applyLocale(next);
              void host.patch({ locale: next });
            }}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="en">{copy.settings.english}</SelectItem>
              <SelectItem value="zh-CN">{copy.settings.chinese}</SelectItem>
            </SelectContent>
          </Select>
        </SettingRow>
      </SettingSection>
    </>
  );
}

export function PolicySettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [budget, setBudget] = useState(String(host.cfg.maxBudgetUsd || 0));
  const [rate, setRate] = useState(String(host.cfg.usdPerMtok || 0));
  useEffect(() => {
    setBudget(String(host.cfg.maxBudgetUsd || 0));
    setRate(String(host.cfg.usdPerMtok || 0));
  }, [host.cfg]);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.policy}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.control.policyHint}</p>
      <SettingSection id="policy-gate" title={copy.settings.sections.policyGate}>
        <SettingRow title={copy.settings.autoAllow} description={copy.settings.autoAllowDesc}>
          <Switch checked={!!host.cfg.autoAllow} onCheckedChange={(v) => void host.patch({ autoAllow: v })} />
        </SettingRow>
        <SettingRow title={copy.settings.gateMode} description={copy.settings.gateModeDesc}>
          <Select value={host.cfg.gateMode || "manual"} onValueChange={(v) => void host.patch({ gateMode: v })}>
            <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="manual">{copy.settings.gateManual}</SelectItem>
              <SelectItem value="autosafe">{copy.settings.gateAutoSafe}</SelectItem>
              <SelectItem value="skip">{copy.settings.gateSkip}</SelectItem>
            </SelectContent>
          </Select>
        </SettingRow>
        <SettingRow title={copy.settings.crashResume} description={copy.settings.crashResumeDesc}>
          <Switch checked={host.cfg.crashResume !== false} onCheckedChange={(v) => void host.patch({ crashResume: v })} />
        </SettingRow>
        <SettingRow title={copy.settings.searchUrl} description={copy.settings.searchUrlDesc} border={false}>
          <Input className="h-8 w-56 text-[12px]" value={host.cfg.searchUrl || ""} onChange={(e) => void host.patch({ searchUrl: e.target.value })} />
        </SettingRow>
      </SettingSection>
      <SettingSection id="policy-budget" title={copy.settings.sections.policyBudget}>
        <SettingRow title={copy.settings.maxBudget}>
          <Input className="h-8 w-28 text-[12px]" type="number" value={budget} onChange={(e) => setBudget(e.target.value)} />
        </SettingRow>
        <SettingRow title={copy.settings.usdPerMtok} border={false}>
          <Input className="h-8 w-28 text-[12px]" type="number" value={rate} onChange={(e) => setRate(e.target.value)} />
        </SettingRow>
      </SettingSection>
      <div className="px-1.5">
        <Button
          onClick={async () => {
            await host.patch({ maxBudgetUsd: Number(budget) || 0, usdPerMtok: Number(rate) || 0 });
            toast.success(copy.app.controlSaved);
          }}
        >
          {copy.settings.savePolicy}
        </Button>
      </div>
    </>
  );
}

export function McpSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const mcp = asArray(host.plugins?.mcp || host.plugins?.servers);
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");
  const [endpoint, setEndpoint] = useState("");
  const [json, setJson] = useState("[]");
  useEffect(() => {
    const rows = mcp.map((row: any) => ({
      name: str(row?.name || row?.Name || row),
      command: str(row?.command || row?.Command),
      args: asArray(row?.args || row?.Args).map(String),
      endpoint: str(row?.endpoint || row?.Endpoint),
    }));
    setJson(JSON.stringify(rows, null, 2));
  }, [host.plugins]);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.mcp}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.mcpHint}</p>
      <SettingSection id="mcp-servers" title={copy.settings.sections.mcpServers}>
        {mcp.length === 0 ? (
          <p className="px-5 py-4 text-[13px] text-muted">{copy.settings.noMcp}</p>
        ) : (
          mcp.map((row: any, i: number) => {
            const n = str(row?.name || row?.Name || row);
            return (
              <SettingRow key={n || i} title={n} border={i < mcp.length - 1}>
                <Button size="sm" variant="ghost" onClick={() => void host.onStopMcp(n)}>
                  {copy.settings.mcpStop}
                </Button>
              </SettingRow>
            );
          })
        )}
        <div className="grid gap-2 border-t border-border px-5 py-4 sm:grid-cols-3">
          <Input placeholder={copy.settings.mcpName} value={name} onChange={(e) => setName(e.target.value)} />
          <Input placeholder={copy.settings.mcpCommand} value={command} onChange={(e) => setCommand(e.target.value)} />
          <Input placeholder={copy.settings.mcpArgs} value={args} onChange={(e) => setArgs(e.target.value)} />
        </div>
        <div className="grid gap-2 px-5 pb-3 sm:grid-cols-[1fr_auto]">
          <Input placeholder={copy.settings.mcpEndpoint} value={endpoint} onChange={(e) => setEndpoint(e.target.value)} />
          <Button
            size="sm"
            variant="lift"
            disabled={!name.trim() || !endpoint.trim()}
            onClick={() => void host.onStartMcpHttp?.(name.trim(), endpoint.trim())}
          >
            {copy.settings.mcpHttp}
          </Button>
        </div>
        <div className="px-5 pb-4">
          <Button
            size="sm"
            disabled={!name.trim() || !command.trim()}
            onClick={() => void host.onStartMcp(name.trim(), command.trim(), args.split(/\s+/).filter(Boolean))}
          >
            {copy.settings.mcpStart}
          </Button>
        </div>
      </SettingSection>
      <SettingSection id="mcp-json" title={copy.settings.sections.mcpJson} description={copy.settings.mcpJsonDesc}>
        <div className="p-4">
          <textarea
            className="no-drag h-40 w-full rounded-lg border border-border bg-background p-3 font-mono text-[12px] text-foreground"
            value={json}
            onChange={(e) => setJson(e.target.value)}
          />
          <Button
            className="mt-3"
            size="sm"
            onClick={async () => {
              try {
                const parsed = JSON.parse(json);
                const servers = (Array.isArray(parsed) ? parsed : []).map((r: any) => ({
                  name: String(r.name || ""),
                  command: String(r.command || ""),
                  args: Array.isArray(r.args) ? r.args.map(String) : [],
                  endpoint: String(r.endpoint || ""),
                })).filter((s: { name: string; command: string; endpoint: string }) => s.name && (s.command || s.endpoint));
                await host.onReplaceMcp(servers);
                toast.success(copy.app.controlSaved);
              } catch (e: any) {
                toast.error(e?.message || "invalid JSON");
              }
            }}
          >
            {copy.settings.applyJson}
          </Button>
        </div>
      </SettingSection>
    </>
  );
}

export function PluginsSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const fibers = asArray(host.plugins?.fibers || host.plugins?.Fibers);
  const [pending, setPending] = useState("");
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.plugins}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.control.pluginsHint}</p>
      <SettingSection id="plugins-fibers" title={copy.settings.sections.pluginsFibers}>
        {fibers.length === 0 ? (
          <p className="px-5 py-4 text-[13px] text-muted">{copy.settings.noFibers}</p>
        ) : (
          fibers.map((name: any, i: number) => (
            <SettingRow key={str(name)} title={str(name)} border={i < fibers.length - 1}>
              <Button size="sm" variant="danger" onClick={() => setPending(str(name))}>
                {copy.settings.unload}
              </Button>
            </SettingRow>
          ))
        )}
      </SettingSection>
      <ConfirmDialog
        open={!!pending}
        title={copy.settings.unload}
        body={copy.settings.unloadConfirm}
        danger
        confirmLabel={copy.settings.unload}
        onCancel={() => setPending("")}
        onConfirm={async () => {
          if (pending) await host.onUnload(pending);
          setPending("");
        }}
      />
    </>
  );
}

export function ShortcutsSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const keymap = mergeKeymap(host.cfg.keymap);
  const [armed, setArmed] = useState<KeymapId | null>(null);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.shortcuts}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.recordHint}</p>
      <SettingSection id="shortcuts-map" title={copy.settings.sections.shortcutsMap}>
        {(Object.keys(DEFAULT_KEYMAP) as KeymapId[]).map((id, i, all) => (
          <SettingRow key={id} title={id} border={i < all.length - 1}>
            <button
              type="button"
              aria-label={id}
              className="h-8 min-w-[7.5rem] rounded-lg border border-border bg-background px-3 text-[12px] text-foreground"
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

export function UpdatesSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [url, setUrl] = useState(host.cfg.updateUrl);
  const [applyOpen, setApplyOpen] = useState(false);
  useEffect(() => setUrl(host.cfg.updateUrl), [host.cfg.updateUrl]);
  const channel = host.cfg.updateChannel || "nightly";
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.updates}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.updateUrlDesc}</p>
      <SettingSection id="updates-channel" title={copy.settings.sections.updatesChannel}>
        <SettingRow title={copy.settings.currentVersion}>
          <span className="text-[12px] tabular-nums text-muted">{host.health.version}</span>
        </SettingRow>
        <SettingRow title={copy.settings.channel}>
          <Select value={channel} onValueChange={(v) => void host.patch({ updateChannel: v })}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="stable">stable</SelectItem>
              <SelectItem value="beta">beta</SelectItem>
              <SelectItem value="nightly">nightly</SelectItem>
            </SelectContent>
          </Select>
        </SettingRow>
        <SettingRow title={copy.settings.updateUrl} border={false}>
          <Input className="h-8 w-64 text-[12px]" value={url} onChange={(e) => setUrl(e.target.value)} onBlur={() => { if (url !== host.cfg.updateUrl) void host.patch({ updateUrl: url }); }} />
        </SettingRow>
      </SettingSection>
      <div className="flex gap-2 px-1.5">
        <Button
          variant="lift"
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
        <Button variant="danger" onClick={() => setApplyOpen(true)}>
          {copy.settings.applyUpdate}
        </Button>
      </div>
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

export function LogsSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const recs = asArray(host.logs?.journal || host.logs);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.logs}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.logsHint}</p>
      <SettingSection id="logs-journal" title={copy.settings.sections.logsJournal}>
        <div className="flex justify-end gap-2 border-b border-border px-4 py-2">
          <Button size="sm" variant="lift" onClick={() => void host.onRevealLogs()}>
            {copy.settings.reveal}
          </Button>
        </div>
        <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap px-4 py-3 font-mono text-[11px] text-muted">
          {recs.length
            ? recs
                .map((r: any) => `${r.time || r.Time || ""} ${r.type || r.Type || ""} ${typeof r.payload === "string" ? r.payload : JSON.stringify(r.payload || r.Payload || r)}`)
                .join("\n")
            : host.logs
              ? JSON.stringify(host.logs, null, 2).slice(0, 12000)
              : copy.settings.emptyLogs}
        </pre>
      </SettingSection>
      <SettingSection id="logs-doctor" title={copy.settings.sections.logsDoctor}>
        <pre className="max-h-64 overflow-auto whitespace-pre-wrap px-4 py-3 font-mono text-[11px] text-muted">
          {JSON.stringify(host.doctor || {}, null, 2).slice(0, 8000)}
        </pre>
      </SettingSection>
    </>
  );
}

export function DangerSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [applyOpen, setApplyOpen] = useState(false);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.danger}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.checkoutConfirm}</p>
      <SettingSection id="danger-zone" title={copy.settings.sections.dangerZone}>
        <SettingRow title={copy.settings.applyUpdate} description={copy.settings.applyUpdateConfirm} border={false}>
          <Button variant="danger" size="sm" onClick={() => setApplyOpen(true)}>
            {copy.settings.applyUpdate}
          </Button>
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

function ThemePreview({ mode }: { mode: ThemePref }) {
  if (mode === "system") {
    return (
      <div className="flex h-14 overflow-hidden rounded-md border border-border">
        <div className="theme-preview-dark w-1/2 p-1.5">
          <div className="theme-preview-inner h-full rounded-sm" />
        </div>
        <div className="theme-preview-light w-1/2 p-1.5">
          <div className="theme-preview-inner h-full rounded-sm" />
        </div>
      </div>
    );
  }
  return (
    <div className={cn("h-14 overflow-hidden rounded-md border border-border p-1.5", mode === "dark" ? "theme-preview-dark" : "theme-preview-light")}>
      <div className="theme-preview-inner h-full rounded-sm" />
    </div>
  );
}
