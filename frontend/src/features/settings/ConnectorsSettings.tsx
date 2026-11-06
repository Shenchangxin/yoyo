/**
 * 拥有：连接应用的目录、授权和读到的第一页。
 * 不拥有：工作来源怎么轮询，消息机器人怎么收发。点到那两类时打开对应的设置。
 */
import { useEffect, useState } from "react";
import { ExternalLink } from "lucide-react";
import { toast } from "sonner";
import { FormDialog } from "../FormDialog";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import * as api from "../../lib/client";
import { getLocale, useCopy } from "../../lib/i18n";
import { asArray, str } from "../../lib/normalize";
import { useUI } from "../../lib/store";
import { WorkSources, type WorkSourceName } from "../superagent/SourceSettings";
import { catalogRows, ConnectorDirectory, type CatalogRow } from "./ConnectorDirectory";
import { connectorBlurb, connectorName, resolvedAccess } from "./connector-directory";
import { SettingEmpty, SettingRow, SettingSection } from "./SettingChrome";

function Field(props: { label: string; value: string; onChange: (v: string) => void; secret?: boolean; placeholder?: string }) {
  return (
    <label className="flex flex-col gap-1 text-[13px] font-medium">
      <span>{props.label}</span>
      <Input className="h-8 font-normal" type={props.secret ? "password" : "text"} value={props.value} placeholder={props.placeholder} autoComplete="off" spellCheck={false} onChange={(e) => props.onChange(e.target.value)} />
    </label>
  );
}

type Mode = "local" | "oauth" | "source" | "im" | "phone" | "token";

const ADDRESS = new Set([
  "qq-mail", "netease-mail", "local",
  "wps", "tencent-docs", "weread", "tencent-meeting",
  "linkedin", "quickbooks", "zoho-invoice",
]);

function modeOf(row: CatalogRow): Mode {
  const access = resolvedAccess(row.provider, row.access);
  if (access === "local") return "local";
  if (access === "oauth") return "oauth";
  if (access === "source") return "source";
  if (row.provider === "dingtalk" || row.provider === "wecom" || row.provider === "wechat") return "im";
  if (row.platform === "ios" || row.platform === "android" || row.provider.endsWith("-operate") || row.provider.startsWith("mac-") && row.access !== "local") return "phone";
  return "token";
}

function titles(raw: unknown): string[] {
  return asArray(raw).map((item) => str((item as { title?: string }).title)).filter(Boolean);
}

/** ConnectorsSettings 是连接页。工作来源在上，目录在下。 */
export function ConnectorsSettings() {
  const copy = useCopy();
  const openSettings = useUI((s) => s.openSettings);
  const [data, setData] = useState<{ accounts?: unknown[]; catalog?: unknown[] }>({});
  const [picked, setPicked] = useState<CatalogRow | null>(null);
  const [source, setSource] = useState<WorkSourceName | null>(null);
  const [clientId, setClientId] = useState("");
  const [secret, setSecret] = useState("");
  const [code, setCode] = useState("");
  const [token, setToken] = useState("");
  const [endpoint, setEndpoint] = useState("");
  const [seen, setSeen] = useState<string[]>([]);
  const [problem, setProblem] = useState("");
  const [busy, setBusy] = useState(false);

  const refresh = () => {
    void api.connectors().then(setData).catch(() => {});
  };
  useEffect(() => { refresh(); }, []);

  const catalog = catalogRows(asArray(data.catalog));
  const accounts = asArray(data.accounts);
  const connected = accounts.map((a) => str((a as { provider?: string; Provider?: string }).provider || (a as { Provider?: string }).Provider));

  async function run(provider: string, nextToken = "", nextEndpoint = "", surface: "dialog" | "toast" = "dialog") {
    setBusy(true);
    setProblem("");
    setSeen([]);
    try {
      const out = await api.connectorUse(provider, nextToken, nextEndpoint);
      const rows = titles(out?.items || out?.Items);
      setSeen(rows);
      if (rows.length) {
        toast.success(rows.slice(0, 3).join(" · "));
        setPicked(null);
      } else if (surface === "toast") toast(copy.settings.connectorsEmpty);
      else setProblem(copy.settings.connectorsEmpty);
      refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e || "");
      if (surface === "toast") toast.error(msg);
      else setProblem(msg);
    } finally {
      setBusy(false);
    }
  }

  function pick(row: CatalogRow) {
    const mode = modeOf(row);
    setPicked({ ...row, access: resolvedAccess(row.provider, row.access) });
    setSeen([]);
    setProblem("");
    setToken("");
    setEndpoint("");
    setCode("");
    if (mode === "source") {
      setPicked(null);
      setSource(row.provider === "feishu" ? "feishu" : "github");
      return;
    }
    if (mode === "im") {
      setPicked(null);
      openSettings("messaging");
      toast(copy.settings.connectorsIm);
      return;
    }
    if (mode === "local" || mode === "phone") {
      setPicked(null);
      void run(row.provider, "", "", "toast");
    }
  }

  const mode = picked ? modeOf(picked) : null;
  const mail = picked?.provider === "qq-mail" || picked?.provider === "netease-mail" || picked?.provider === "local";

  return (
    <>
      <WorkSources openSource={source} onOpenSource={setSource} />
      <SettingSection id="extensions-connectors" title={copy.settings.appsTitle} footnote={copy.settings.connectorsHint}>
        {accounts.length === 0 ? <SettingEmpty border>{copy.settings.noConnectors}</SettingEmpty> : accounts.map((a, i) => {
          const row = a as { id?: string; ID?: string; label?: string; Label?: string; provider?: string; kind?: string; Kind?: string };
          return (
            <SettingRow key={str(row.id || row.ID || i)} list title={str(row.label || row.Label || row.provider)} description={str(row.kind || row.Kind) || undefined}>
              <Button size="sm" variant="ghost" onClick={async () => { await api.connectorDisconnect(str(row.id || row.ID)); refresh(); }}>
                {copy.settings.disconnect}
              </Button>
            </SettingRow>
          );
        })}
        <ConnectorDirectory catalog={catalog} connected={connected} onPick={pick} />
        <FormDialog
          open={!!picked && (mode === "oauth" || mode === "token")}
          title={picked ? connectorName(picked.provider, picked.label, getLocale()) : ""}
          description={picked ? connectorBlurb(picked.provider, getLocale()) : ""}
          onClose={() => { if (!busy) setPicked(null); }}
        >
          {picked && mode === "oauth" ? (
            <div className="flex flex-col gap-3">
              <Field label={copy.settings.oauthClientId} value={clientId} onChange={setClientId} />
              <Field label={copy.settings.oauthSecret} value={secret} onChange={setSecret} secret />
              <Field label={copy.settings.oauthCode} value={code} onChange={setCode} />
              {problem ? <p className="text-[12px] leading-5 text-danger">{problem}</p> : null}
              <div className="dialog-actions">
                <Button size="sm" variant="lift" disabled={busy} onClick={() => setPicked(null)}>{copy.dialog.cancel}</Button>
                <Button size="sm" variant="outline" onClick={async () => {
                  const r = await api.connectorAuthURL(picked.provider, clientId, "http://127.0.0.1:3080/oauth");
                  const url = str(r?.url || r?.URL);
                  if (url) window.open(url, "_blank", "noopener");
                }}>
                  <ExternalLink aria-hidden />
                  {copy.settings.oauth}
                </Button>
                <Button size="sm" disabled={!code.trim() || busy} onClick={async () => {
                  setBusy(true);
                  setProblem("");
                  try {
                    await api.connectorComplete(picked.provider, code.trim(), clientId, secret);
                    setCode("");
                    await run(picked.provider);
                  } catch (e) {
                    setProblem(e instanceof Error ? e.message : String(e || ""));
                    setBusy(false);
                  }
                }}>
                  {copy.settings.completeOAuth}
                </Button>
              </div>
            </div>
          ) : null}
          {picked && mode === "token" ? (
            <div className="flex flex-col gap-3">
              {ADDRESS.has(picked.provider) ? (
                <Field
                  label={mail ? copy.settings.connectorsMailbox : copy.settings.connectorsEndpoint}
                  value={endpoint}
                  onChange={setEndpoint}
                  placeholder={mail ? "name@example.com" : "https://"}
                />
              ) : null}
              <Field label={mail ? copy.settings.connectorsPassword : copy.settings.connectorsToken} value={token} onChange={setToken} secret />
              {problem ? <p className="text-[12px] leading-5 text-danger">{problem}</p> : null}
              {seen.length ? <p className="text-[12px] leading-5 text-muted">{seen.slice(0, 4).join(" · ")}</p> : null}
              <div className="dialog-actions">
                <Button size="sm" variant="lift" disabled={busy} onClick={() => setPicked(null)}>{copy.dialog.cancel}</Button>
                <Button size="sm" disabled={!token.trim() || busy} onClick={() => void run(picked.provider, token.trim(), endpoint.trim())}>
                  {copy.settings.connect}
                </Button>
              </div>
            </div>
          ) : null}
        </FormDialog>
      </SettingSection>
    </>
  );
}
