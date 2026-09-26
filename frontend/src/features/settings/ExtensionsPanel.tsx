import { useEffect, useState } from "react";
import { toast } from "sonner";
import { ExternalLink, Plus } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { asArray, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { ConfirmDialog } from "../ConfirmDialog";
import {
  SettingActionRow,
  SettingEmpty,
  SettingRow,
  SettingSection,
  SettingsPageHeader,
} from "./SettingChrome";
import type { SettingsHost } from "./host";

function Field({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
      <span>{label}</span>
      <Input className="h-8 w-full text-[12.5px] font-normal" value={value} onChange={(e) => onChange(e.target.value)} />
    </label>
  );
}

export function ExtensionsSettings({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.extensions} description={copy.settings.tabHints.extensions} />
      <McpSection host={host} />
      <FiberSection host={host} />
      <ConnectorSection />
    </>
  );
}

function McpSection({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const servers = asArray(host.plugins?.mcp || host.plugins?.servers);
  const [adding, setAdding] = useState(false);
  const [jsonMode, setJsonMode] = useState(false);
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");
  const [endpoint, setEndpoint] = useState("");
  const [json, setJson] = useState("[]");

  useEffect(() => {
    const rows = servers.map((row: any) => ({
      name: str(row?.name || row?.Name || row),
      command: str(row?.command || row?.Command),
      args: asArray(row?.args || row?.Args).map(String),
      endpoint: str(row?.endpoint || row?.Endpoint),
    }));
    setJson(JSON.stringify(rows, null, 2));
  }, [host.plugins]);

  function reset() {
    setName("");
    setCommand("");
    setArgs("");
    setEndpoint("");
    setAdding(false);
  }

  return (
    <SettingSection
      id="extensions-mcp"
      title={copy.settings.sections.extensionsMcp}
      footnote={jsonMode ? copy.settings.mcpJsonDesc : copy.settings.mcpHint}
      actions={
        <Button size="sm" variant="ghost" aria-pressed={jsonMode} onClick={() => setJsonMode((v) => !v)}>
          {copy.settings.editJson}
        </Button>
      }
    >
      {jsonMode ? (
        <div className="p-3">
          <textarea
            aria-label={copy.settings.mcpJson}
            className="no-drag h-44 w-full rounded-[9px] border border-border bg-background p-3 font-mono text-[11.5px] leading-[1.6] text-foreground outline-none focus-visible:border-foreground/25"
            value={json}
            onChange={(e) => setJson(e.target.value)}
          />
          <div className="mt-2.5 flex justify-end">
            <Button
              size="sm"
              onClick={async () => {
                try {
                  const parsed = JSON.parse(json);
                  const next = (Array.isArray(parsed) ? parsed : [])
                    .map((r: any) => ({
                      name: String(r.name || ""),
                      command: String(r.command || ""),
                      args: Array.isArray(r.args) ? r.args.map(String) : [],
                      endpoint: String(r.endpoint || ""),
                    }))
                    .filter((s: { name: string; command: string; endpoint: string }) => s.name && (s.command || s.endpoint));
                  await host.onReplaceMcp(next);
                  toast.success(copy.app.controlSaved);
                } catch (e: any) {
                  toast.error(e?.message || "invalid JSON");
                }
              }}
            >
              {copy.settings.applyJson}
            </Button>
          </div>
        </div>
      ) : (
        <>
          {servers.length === 0 ? (
            <SettingEmpty border>{copy.settings.noMcp}</SettingEmpty>
          ) : (
            servers.map((row: any, i: number) => {
              const n = str(row?.name || row?.Name || row);
              const cmd = str(row?.command || row?.Command) || str(row?.endpoint || row?.Endpoint);
              return (
                <SettingRow key={n || i} list title={n} description={cmd || undefined}>
                  <Button size="sm" variant="ghost" onClick={() => void host.onStopMcp(n)}>
                    {copy.settings.mcpStop}
                  </Button>
                </SettingRow>
              );
            })
          )}
          {adding ? (
            <SettingRow stack border={false}>
              <div className="flex w-full flex-col gap-[var(--space-group)]">
                <Field label={copy.settings.mcpName} value={name} onChange={setName} />
                <Field label={copy.settings.mcpCommand} value={command} onChange={setCommand} />
                <Field label={copy.settings.mcpArgs} value={args} onChange={setArgs} />
                <Field label={copy.settings.mcpEndpoint} value={endpoint} onChange={setEndpoint} />
                <div className="flex items-center justify-end gap-2">
                  <Button size="sm" variant="ghost" onClick={reset}>
                    {copy.settings.discard}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={!name.trim() || !endpoint.trim()}
                    onClick={async () => {
                      await host.onStartMcpHttp?.(name.trim(), endpoint.trim());
                      reset();
                    }}
                  >
                    {copy.settings.mcpHttp}
                  </Button>
                  <Button
                    size="sm"
                    disabled={!name.trim() || !command.trim()}
                    onClick={async () => {
                      await host.onStartMcp(name.trim(), command.trim(), args.split(/\s+/).filter(Boolean));
                      reset();
                    }}
                  >
                    {copy.settings.mcpStart}
                  </Button>
                </div>
              </div>
            </SettingRow>
          ) : (
            <SettingActionRow
              border={false}
              chevron={false}
              title={copy.settings.addServer}
              trailing={<Plus className="size-4" aria-hidden />}
              onClick={() => setAdding(true)}
            />
          )}
        </>
      )}
    </SettingSection>
  );
}

function FiberSection({ host }: { host: SettingsHost }) {
  const copy = useCopy();
  const fibers = asArray(host.plugins?.fibers || host.plugins?.Fibers);
  const [pending, setPending] = useState("");
  return (
    <SettingSection id="extensions-fibers" title={copy.settings.sections.extensionsFibers} footnote={copy.settings.fibersHint}>
      {fibers.length === 0 ? (
        <SettingEmpty>{copy.settings.noFibers}</SettingEmpty>
      ) : (
        fibers.map((name: any, i: number) => (
          <SettingRow key={str(name)} list title={str(name)}>
            <Button size="sm" variant="danger" onClick={() => setPending(str(name))}>
              {copy.settings.unload}
            </Button>
          </SettingRow>
        ))
      )}
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
    </SettingSection>
  );
}

function ConnectorSection() {
  const copy = useCopy();
  const [data, setData] = useState<{ accounts?: any[]; catalog?: any[] }>({});
  const [provider, setProvider] = useState("gmail");
  const [clientId, setClientId] = useState("");
  const [code, setCode] = useState("");
  const refresh = () => {
    void api.connectors().then(setData).catch(() => {});
  };
  useEffect(() => {
    refresh();
  }, []);
  const catalog = asArray(data.catalog);
  const accounts = asArray(data.accounts);
  const options = catalog.length ? catalog : [{ provider: "gmail" }, { provider: "outlook" }, { provider: "feishu" }, { provider: "local" }];
  return (
    <SettingSection
      id="extensions-connectors"
      title={copy.settings.sections.extensionsConnectors}
      footnote={copy.settings.connectorsHint}
    >
      {accounts.length === 0 ? (
        <SettingEmpty border>{copy.settings.noConnectors}</SettingEmpty>
      ) : (
        accounts.map((a: any, i: number) => (
          <SettingRow
            key={str(a.id || a.ID || i)}
            list
            title={str(a.label || a.Label || a.provider)}
            description={str(a.kind || a.Kind) || undefined}
          >
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                await api.connectorDisconnect(str(a.id || a.ID));
                refresh();
              }}
            >
              {copy.settings.disconnect}
            </Button>
          </SettingRow>
        ))
      )}
      <SettingRow title={copy.settings.provider} stack border={false}>
        <div className="flex w-full flex-col gap-[var(--space-group)]">
          <Select value={provider} onValueChange={setProvider}>
            <SelectTrigger className="h-8 w-full" aria-label={copy.settings.provider}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {options.map((c: any) => (
                <SelectItem key={str(c.provider)} value={str(c.provider)}>
                  {str(c.label || c.provider)}{c.mcp_pack || c.MCPPack ? " · MCP" : ""}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Field label={copy.settings.oauthClientId} value={clientId} onChange={setClientId} />
          <Field label={copy.settings.oauthCode} value={code} onChange={setCode} />
          <div className="flex items-center justify-end gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={async () => {
                const r = await api.connectorAuthURL(provider, clientId, "http://127.0.0.1:3080/oauth");
                const url = str(r?.url || r?.URL);
                if (url) window.open(url, "_blank", "noopener");
              }}
            >
              <ExternalLink aria-hidden />
              {copy.settings.oauth}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={!code.trim()}
              onClick={async () => {
                await api.connectorComplete(provider, code.trim(), clientId);
                setCode("");
                refresh();
              }}
            >
              {copy.settings.completeOAuth}
            </Button>
            <Button
              size="sm"
              onClick={async () => {
                await api.connectorConnect({
                  provider,
                  kind: str(catalog.find((c: any) => str(c.provider) === provider)?.kind || catalog.find((c: any) => str(c.provider) === provider)?.Kind || "mail"),
                  label: provider,
                });
                refresh();
              }}
            >
              {copy.settings.connect}
            </Button>
          </div>
        </div>
      </SettingRow>
    </SettingSection>
  );
}
