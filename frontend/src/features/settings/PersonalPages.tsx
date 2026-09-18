import { useEffect, useState } from "react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { asArray, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { SettingRow, SettingSection } from "./SettingChrome";
import type { SettingsHost } from "./host";

export function IsolationSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const [rep, setRep] = useState<any>(host.doctor?.isolation || {});
  useEffect(() => {
    void api.isolationReport().then(setRep).catch(() => {});
  }, [host.doctor]);
  const os = rep?.os || {};
  const vault = rep?.vault || host.vault || {};
  const sandbox = !!(os.sandbox ?? os.Sandbox);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.isolation}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.isolationHint}</p>
      <SettingSection id="isolation-status" title={copy.settings.sections.isolationStatus}>
        <SettingRow title={copy.settings.isolationKind}>
          <span className="text-[12px] text-muted">{str(os.kind || os.Kind, "none")}</span>
        </SettingRow>
        <SettingRow title={copy.settings.isolationSandbox}>
          <span className="text-[12px] text-muted">{sandbox ? copy.settings.yes : copy.settings.no}</span>
        </SettingRow>
        <SettingRow title={copy.settings.isolationSigned}>
          <span className="text-[12px] text-muted">{str(os.signed || os.Signed, "unsigned")}</span>
        </SettingRow>
        <SettingRow title={copy.settings.isolationVault}>
          <span className="text-[12px] text-muted">{str(vault.source || vault.Source, "empty")}</span>
        </SettingRow>
        <SettingRow title={copy.settings.isolationBrowser} border={false}>
          <span className="text-[12px] text-muted">{rep?.browser_isolated || rep?.browserIsolated ? "isolated profile" : "unset"}</span>
        </SettingRow>
      </SettingSection>
      <p className="mt-4 px-1.5 text-[12px] text-muted">{copy.settings.isolationSleep}</p>
    </>
  );
}

export function ConnectorSettings() {
  const copy = useCopy();
  const [data, setData] = useState<{ accounts?: any[]; catalog?: any[] }>({});
  const [provider, setProvider] = useState("gmail");
  const refresh = () => { void api.connectors().then(setData).catch(() => {}); };
  useEffect(() => { refresh(); }, []);
  const catalog = asArray(data.catalog);
  const accounts = asArray(data.accounts);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.connectors}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.connectorsHint}</p>
      <SettingSection id="connectors-catalog" title={copy.settings.sections.connectorsCatalog}>
        {accounts.map((a: any, i: number) => (
          <SettingRow key={str(a.id || a.ID || i)} title={str(a.label || a.Label || a.provider)} border={i < accounts.length - 1}>
            <span className="text-[11px] text-muted">{str(a.kind || a.Kind)}</span>
          </SettingRow>
        ))}
        <div className="flex gap-2 px-5 py-4">
          <Select value={provider} onValueChange={setProvider}>
            <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
            <SelectContent>
              {(catalog.length ? catalog : [{ provider: "gmail" }, { provider: "feishu" }, { provider: "local" }]).map((c: any) => (
                <SelectItem key={str(c.provider)} value={str(c.provider)}>{str(c.label || c.provider)}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button size="sm" onClick={async () => {
            await api.connectorConnect({ provider, kind: "mail", label: provider });
            refresh();
          }}>{copy.settings.connect}</Button>
          <Button size="sm" variant="lift" onClick={async () => {
            const r = await api.connectorAuthURL(provider, "", "http://127.0.0.1:3080/oauth");
            const url = str(r?.url || r?.URL);
            if (url) window.open(url, "_blank", "noopener");
          }}>{copy.settings.oauth}</Button>
        </div>
      </SettingSection>
    </>
  );
}

export function MemorySettings() {
  const copy = useCopy();
  const [items, setItems] = useState<any[]>([]);
  const [text, setText] = useState("");
  const refresh = () => { void api.memoryList().then(setItems).catch(() => {}); };
  useEffect(() => { refresh(); }, []);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.memory}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.memoryHint}</p>
      <SettingSection id="memory-items" title={copy.settings.sections.memoryItems}>
        <div className="flex gap-2 px-5 py-3">
          <Input value={text} onChange={(e) => setText(e.target.value)} placeholder={copy.settings.memoryHint} />
          <Button size="sm" onClick={async () => {
            if (!text.trim()) return;
            await api.memoryWrite("profile", text.trim());
            setText("");
            refresh();
          }}>{copy.settings.save}</Button>
        </div>
        {items.map((it: any, i: number) => (
          <SettingRow key={str(it.id || i)} title={str(it.text).slice(0, 80)} description={str(it.kind) + (it.staging ? " · staging" : "")} border={i < items.length - 1}>
            <div className="flex gap-1">
              {it.staging ? <Button size="sm" variant="lift" onClick={async () => { await api.memoryPromote(it.id); refresh(); }}>{copy.settings.promote}</Button> : null}
              <Button size="sm" variant="ghost" onClick={async () => { await api.memoryForget(it.id); refresh(); }}>{copy.settings.forget}</Button>
            </div>
          </SettingRow>
        ))}
      </SettingSection>
    </>
  );
}

export function AutomationSettings() {
  const copy = useCopy();
  const [jobs, setJobs] = useState<any[]>([]);
  const [prompt, setPrompt] = useState("");
  const [spec, setSpec] = useState("30m");
  const refresh = () => { void api.scheduleList().then(setJobs).catch(() => {}); };
  useEffect(() => { refresh(); }, []);
  return (
    <>
      <h1 className="mb-1 text-[20px] font-bold">{copy.settings.tabs.automations}</h1>
      <p className="mb-6 text-[12px] text-muted">{copy.settings.automationsHint}</p>
      <SettingSection id="automations-jobs" title={copy.settings.sections.automationsJobs}>
        <div className="grid gap-2 px-5 py-3 sm:grid-cols-3">
          <Input placeholder={copy.settings.scheduleSpec} value={spec} onChange={(e) => setSpec(e.target.value)} />
          <Input className="sm:col-span-2" placeholder={copy.settings.schedulePrompt} value={prompt} onChange={(e) => setPrompt(e.target.value)} />
        </div>
        <div className="px-5 pb-3">
          <Button size="sm" disabled={!prompt.trim()} onClick={async () => {
            await api.scheduleCreate({ kind: "heartbeat", spec, prompt, isolate: true });
            setPrompt("");
            refresh();
          }}>{copy.settings.addJob}</Button>
        </div>
        {jobs.map((j: any, i: number) => (
          <SettingRow key={str(j.id || i)} title={str(j.prompt).slice(0, 80)} description={`${str(j.kind)} ${str(j.spec)}`} border={i < jobs.length - 1}>
            <Button size="sm" variant="ghost" onClick={async () => { await api.scheduleCancel(j.id); refresh(); }}>{copy.settings.unload}</Button>
          </SettingRow>
        ))}
      </SettingSection>
    </>
  );
}
