/**
 * 拥有：MCP 和插件的设置。MCP 排在连接页，插件排在高级页。
 * 不拥有：连接应用。启动、停止和替换会调用 api 改状态。
 */
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Plus } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { asArray, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { ConfirmDialog } from "../ConfirmDialog";
import { FormDialog } from "../FormDialog";
import {
  SettingActionRow,
  SettingEmpty,
  SettingRow,
  SettingSection,
} from "./SettingChrome";
import type { SettingsHost } from "./host";

/** 画一行输入。label 用作标签，作为子节点渲染。value 写到 Input 的 value。onChange 被调用，并传入 e.target.value。 */
function Field({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="flex flex-col gap-[var(--space-item)] text-ui-body font-medium text-foreground">
      <span>{label}</span>
      <Input className="h-8 w-full text-ui-body font-normal" value={value} onChange={(e) => onChange(e.target.value)} />
    </label>
  );
}

/** McpSection 编辑 MCP 服务，并能启动、停止和替换。host 读取 plugins，调用 onReplaceMcp，调用 onStopMcp，调用 onStartMcpHttp，调用 onStartMcp。 */
export function McpSection({ host }: { host: SettingsHost }) {
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

  /** 清掉正在填写的服务。 */
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
            className="no-drag h-44 w-full rounded-md border border-border bg-background p-3 font-mono text-ui-footnote leading-[1.6] text-foreground outline-none focus-visible:border-foreground/25"
            value={json}
            spellCheck={false}
            autoComplete="off"
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
          <SettingActionRow
            border={false}
            chevron={false}
            title={copy.settings.addServer}
            trailing={<Plus className="size-4" aria-hidden />}
            onClick={() => setAdding(true)}
          />
          <FormDialog open={adding} title={copy.settings.addServer} description={copy.settings.mcpHint} onClose={reset}>
            <div className="flex flex-col gap-3">
              <Field label={copy.settings.mcpName} value={name} onChange={setName} />
              <Field label={copy.settings.mcpCommand} value={command} onChange={setCommand} />
              <Field label={copy.settings.mcpArgs} value={args} onChange={setArgs} />
              <Field label={copy.settings.mcpEndpoint} value={endpoint} onChange={setEndpoint} />
              <div className="dialog-actions">
                <Button size="sm" variant="lift" onClick={reset}>{copy.dialog.cancel}</Button>
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
          </FormDialog>
        </>
      )}
    </SettingSection>
  );
}

/** 列出插件，并能卸载。host 读取 plugins，调用 onUnload。 */
export function FiberSection({ host }: { host: SettingsHost }) {
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
