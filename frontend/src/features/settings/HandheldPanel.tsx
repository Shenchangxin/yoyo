/**
 * 拥有：设置的「手机」页：出示配对码、批准等待的手机、收回已配对的手机。
 * 不拥有：通用设置。配对、批准和收回会调用 api 改状态。
 */
import { useEffect, useState } from "react";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { SettingEmpty, SettingRow, SettingSection } from "./SettingChrome";

type Pending = { id?: string; name?: string };
type Device = { id?: string; name?: string; revoked?: boolean };

/** HandheldSettings 是「手机」整页：上面一块管配对，下面一块列出已配对的手机。每两秒读一次状态。 */
export function HandheldSettings() {
  const copy = useCopy();
  const [status, setStatus] = useState<any>(null);
  const [offer, setOffer] = useState<any>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  /** 拉取远程配对状态。 */
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
  const noLan = listening && Array.isArray(status?.hosts) && status.hosts.length === 0;
  const waiting = pending.length === 0 && browsers.length === 0;

  return (
    <>
      <SettingSection id="advanced-handheld" title={copy.settings.handheldPairing} description={copy.settings.handheldBody}>
        <SettingRow
          title={listening ? copy.settings.handheldListening : copy.settings.handheldOff}
          description={status?.fingerprint ? `${copy.settings.handheldFingerprint} ${status.fingerprint}` : undefined}
        >
          {listening ? (
            <span className="inline-flex items-center gap-1.5 text-ui-footnote tabular-nums text-muted">
              <span className="size-1.5 rounded-full bg-status-live" aria-hidden />
              {status?.port ? status.port : null}
            </span>
          ) : null}
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
          <Button
            size="sm"
            variant={offer ? "lift" : "default"}
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
        </SettingRow>
        {noLan ? <p className="text-ui-body leading-[1.5] text-danger">{copy.settings.handheldNoLan}</p> : null}
        {error ? <p className="text-ui-body leading-[1.5] text-danger" role="alert">{error}</p> : null}
        {offer?.png ? (
          <div className="flex flex-wrap items-start gap-4">
            <img
              alt={copy.settings.sections.advancedHandheld}
              className="size-44 shrink-0 rounded-md bg-white p-2"
              src={`data:image/png;base64,${offer.png}`}
            />
            <div className="min-w-0 flex-1 basis-56">
              <p className="text-ui-body leading-[1.55] text-muted">{copy.settings.handheldBrowser}</p>
              <p className="prose-select mt-2 break-all font-mono text-ui-caption leading-[1.5] text-muted/80">{offer.uri}</p>
            </div>
          </div>
        ) : null}
        {browsers.map((item) => (
          <SettingRow key={item.id || "browser"} list title={copy.settings.handheldBrowserOn} />
        ))}
        {pending.map((item) => (
          <SettingRow key={item.id} list title={item.name || item.id}>
            <Button size="sm" variant="ghost" onClick={() => void api.remoteDeny(String(item.id)).then(load)}>{copy.settings.handheldDeny}</Button>
            <Button size="sm" onClick={() => void api.remoteApprove(String(item.id)).then(load)}>{copy.settings.handheldApprove}</Button>
          </SettingRow>
        ))}
        {waiting && (listening || offer) ? <SettingEmpty compact>{copy.settings.handheldWaiting}</SettingEmpty> : null}
      </SettingSection>

      <SettingSection id="handheld-devices" title={copy.settings.handheldDevices}>
        {devices.length === 0 ? <SettingEmpty compact>{copy.settings.handheldEmpty}</SettingEmpty> : null}
        {devices.map((item) => (
          <SettingRow key={item.id} list title={item.name || item.id}>
            <Button size="sm" variant="ghost" onClick={() => void api.remoteRevoke(String(item.id)).then(load)}>{copy.settings.handheldRevoke}</Button>
          </SettingRow>
        ))}
      </SettingSection>
    </>
  );
}
