/**
 * 拥有：工单交给人的「最后一下」：要人做的那一下、价格、截止，以及打开、我办好了、不用了三个按钮。
 * 不拥有：工人怎么准备（superagent 的 finish 报告）。按钮只把一句话交给工单，工人收到后自己收尾；工人从不替人付款或提交。
 */

import { useState } from "react";
import { openURL, orchestratorAct } from "../../lib/client";
import { useCopy } from "../../lib/i18n";

const BTN = "h-7 rounded-lg px-2.5 text-ui-footnote font-medium transition-colors duration-[var(--duration-fast)] disabled:opacity-50";

/** FinishCard 只在报告带了 finish 时出现。点过之后收成一句回执。 */
export function FinishCard(props: { ticket: string; finish: Record<string, unknown> }) {
  const t = useCopy().transcript;
  const what = String(props.finish.what || "").trim();
  const url = String(props.finish.url || "").trim();
  const price = String(props.finish.price || "").trim();
  const deadline = String(props.finish.deadline || "").trim();
  const [sent, setSent] = useState<"" | "done" | "skip">("");
  const [busy, setBusy] = useState(false);
  if (!what) return null;
  async function tell(kind: "done" | "skip") {
    if (busy || !props.ticket) return;
    setBusy(true);
    try {
      await orchestratorAct("say", props.ticket, "", kind === "done" ? t.finishDoneSay : t.finishSkipSay);
      setSent(kind);
    } catch {
      // 没送到时按钮留着，可以再点。
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="mt-2.5 rounded-lg border border-border bg-lift/40 px-3 py-2.5" data-testid="finish-card">
      <p className="text-ui-caption text-muted">{t.finishHead}</p>
      <p className="mt-0.5 text-ui-body font-medium">{what}</p>
      {price || deadline ? (
        <p className="mt-0.5 text-ui-footnote tabular-nums text-muted">
          {[price ? `${t.finishPrice}${price}` : "", deadline ? `${t.finishDeadline}${deadline}` : ""].filter(Boolean).join(" · ")}
        </p>
      ) : null}
      {sent ? (
        <p className="mt-2 text-ui-footnote text-muted" data-testid="finish-sent">{sent === "done" ? t.finishDoneSent : t.finishSkipSent}</p>
      ) : (
        <div className="mt-2 flex flex-wrap items-center gap-1.5">
          {url ? (
            <button type="button" className={`${BTN} bg-foreground text-background`} onClick={() => { void openURL(url); }}>
              {t.finishOpen}
            </button>
          ) : null}
          <button type="button" className={`${BTN} border border-border hover:bg-lift`} disabled={busy} onClick={() => { void tell("done"); }}>
            {t.finishDone}
          </button>
          <button type="button" className={`${BTN} text-muted hover:bg-lift hover:text-foreground`} disabled={busy} onClick={() => { void tell("skip"); }}>
            {t.finishSkip}
          </button>
        </div>
      )}
    </div>
  );
}
