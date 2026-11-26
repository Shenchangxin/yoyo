/**
 * 本文件拥有对话流里的制作简报卡：做什么、用哪个账号、本机兜底什么、缺什么，以及操作者怎么回答了确认卡。
 * 不拥有确认。确认在输入框上方的 hold 提问卡上完成，这里只照着工具结果画。
 */
import { CircleAlert } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import type { Item } from "../../lib/protocol";
import { toolResultBody } from "../../lib/tool-summary";
import { parseVideoBrief } from "../../lib/video-brief";
import type { FilmMode } from "./film-cast";

/** BriefCard 在结果还没回来时不画。props 的 result 是 video_brief 的工具结果，call 不使用。 */
export function BriefCard(props: { call?: Item; result?: Item }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const film = copy.video.film;
  const view = props.result ? parseVideoBrief(toolResultBody(props.result)) : null;
  if (!view) return null;
  const intents = c.briefIntents as Record<string, string>;
  const caps = c.briefCaps as Record<string, string>;
  const kind = view.intent === "film" && view.format ? film[view.format as FilmMode] || view.format : intents[view.intent] || view.intent;
  const facts = [kind, view.seconds ? c.briefSeconds.replace("{n}", String(view.seconds)) : "", view.aspect, view.subject].filter(Boolean);
  const status = view.confirmed === "go" ? c.briefGo
    : view.confirmed === "edit" ? c.briefEdit
      : view.confirmed === "stop" ? c.briefStop
        : view.confirmed === "unconfirmed" ? c.briefWaiting
          : view.missing.length ? "" : view.spend ? "" : c.briefFree;
  return (
    <section className="rounded-xl border border-border bg-card px-3.5 py-3" data-testid="video-brief-card" aria-label={c.briefTitle}>
      <div className="text-ui-badge font-medium uppercase tracking-[0.08em] text-muted/80">{c.briefTitle}</div>
      <p className="mt-1 text-ui-body font-medium text-foreground">{facts.join(" · ")}</p>
      {view.needs.length ? (
        <ul className="mt-2 divide-y divide-border/50 text-ui-footnote">
          {view.needs.map(/** cap 画一行：账号名、本机，或缺。 */ (cap) => {
            const names = view.have[cap] || [];
            const missing = view.missing.includes(cap);
            return (
              <li key={cap} className="flex h-7 items-center gap-2" data-testid={`video-brief-${cap}`}>
                <span className="w-16 shrink-0 text-muted">{caps[cap] || cap}</span>
                {missing ? (
                  <span className="inline-flex items-center gap-1 text-warning"><CircleAlert className="size-3.5" aria-hidden />{c.briefMissing}</span>
                ) : names.length ? (
                  <span className="truncate text-foreground/85">{names[0]}</span>
                ) : (
                  <span className="text-foreground/85">{c.briefLocal}</span>
                )}
              </li>
            );
          })}
        </ul>
      ) : null}
      {view.missing.length ? <p className="mt-2 text-ui-footnote text-muted">{c.briefMissingHint}</p> : null}
      {status ? <p className="mt-2 text-ui-caption text-muted" data-testid="video-brief-status">{status}</p> : null}
    </section>
  );
}