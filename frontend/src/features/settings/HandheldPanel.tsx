import { useEffect, useState } from "react";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { SettingSection } from "./SettingChrome";

type Pending = { id?: string; name?: string };
type Device = { id?: string; name?: string; revoked?: boolean };

export function HandheldSettings() {
  const copy = useCopy();
  const [status, setStatus] = useState<any>(null);
  const [offer, setOffer] = useState<any>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    try {
      setStatus(await api.remoteStatus());
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void load();
    const id = window.setInterval(() => { void load(); }, 2000);
    return () => window.clearInterval(id);
  }, []);

  const pending = (status?.pending || []) as Pending[];
  const browsers = (status?.browsers || []) as Pending[];
  const devices = ((status?.devices || []) as Device[]).filter((d) => !d.revoked);
  const listening = !!status?.listening;

  return (
    <SettingSection id="advanced-handheld" title={copy.settings.sections.advancedHandheld} description={copy.settings.handheldBody}>
      <div className="flex items-center gap-2 text-[12.5px] text-muted">
        <span className={listening ? "size-1.5 rounded-full bg-accent" : "size-1.5 rounded-full bg-muted"} aria-hidden />
        <span>{listening ? copy.settings.handheldListening : copy.settings.handheldOff}</span>
        {status?.port ? <span className="tabular-nums text-muted/80">{status.port}</span> : null}
      </div>
      {status?.fingerprint ? (
        <p className="mt-2 break-all font-mono text-[11.5px] text-muted">
          {copy.settings.handheldFingerprint} {status.fingerprint}
        </p>
      ) : null}
      {listening && Array.isArray(status?.hosts) && status.hosts.length === 0 ? (
        <p className="mt-2 text-[12.5px] text-danger">{copy.settings.handheldNoLan}</p>
      ) : null}
      {error ? <p className="mt-2 text-[12.5px] text-danger">{error}</p> : null}
      <div className="mt-3 flex flex-wrap gap-2">
        <Button
          size="sm"
          disabled={busy}
          onClick={() => {
            if (offer) {
              setOffer(null);
              return;
            }
            setBusy(true);
            void api.remotePair().then((next) => {
              setOffer(next);
              return load();
            }).catch((e) => setError(e instanceof Error ? e.message : String(e))).finally(() => setBusy(false));
          }}
        >
          {offer ? copy.settings.handheldHide : copy.settings.handheldShow}
        </Button>
        {listening ? (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setOffer(null);
              void api.remoteDisable().then(load).catch((e) => setError(e instanceof Error ? e.message : String(e)));
            }}
          >
            {copy.settings.handheldDisable}
          </Button>
        ) : null}
      </div>
      {offer?.png ? (
        <div className="mt-4 flex flex-col items-start gap-2">
          <img
            alt={copy.settings.sections.advancedHandheld}
            className="h-44 w-44 bg-white p-2"
            src={`data:image/png;base64,${offer.png}`}
          />
          <p className="max-w-sm break-all font-mono text-[11px] text-muted">{offer.uri}</p>
          <p className="max-w-sm text-[12.5px] text-muted">{copy.settings.handheldBrowser}</p>
        </div>
      ) : null}
      <ul className="mt-4 divide-y divide-border/50">
        {pending.length === 0 && browsers.length === 0 ? <li className="py-2 text-[12.5px] text-muted">{copy.settings.handheldWaiting}</li> : null}
        {browsers.map((item) => (
          <li key={item.id || "browser"} className="py-2 text-[13px]">{copy.settings.handheldBrowserOn}</li>
        ))}
        {pending.map((item) => (
          <li key={item.id} className="flex items-center justify-between gap-3 py-2">
            <span className="text-[13px]">{item.name || item.id}</span>
            <span className="flex gap-2">
              <Button size="sm" onClick={() => void api.remoteApprove(String(item.id)).then(load)}>{copy.settings.handheldApprove}</Button>
              <Button size="sm" variant="ghost" onClick={() => void api.remoteDeny(String(item.id)).then(load)}>{copy.settings.handheldDeny}</Button>
            </span>
          </li>
        ))}
      </ul>
      <ul className="divide-y divide-border/50">
        {devices.length === 0 ? <li className="py-2 text-[12.5px] text-muted">{copy.settings.handheldEmpty}</li> : null}
        {devices.map((item) => (
          <li key={item.id} className="flex items-center justify-between gap-3 py-2">
            <span className="text-[13px]">{item.name || item.id}</span>
            <Button size="sm" variant="ghost" onClick={() => void api.remoteRevoke(String(item.id)).then(load)}>{copy.settings.handheldRevoke}</Button>
          </li>
        ))}
      </ul>
    </SettingSection>
  );
}
