import { useEffect, useState, type ReactNode } from "react";
import { ArrowUpRight } from "lucide-react";
import { useCopy } from "../lib/i18n";
import * as api from "../lib/client";
import { asArray, str } from "../lib/normalize";
import { cn } from "../lib/utils";
import type { Notice } from "../lib/protocol";

export function InboxMenu(props: {
  notices?: Notice[];
  onNotice?: (n: Notice) => void;
  onClearNotices?: () => void;
  onOpenThread?: (id: string) => void;
  onResolve?: (id: string, decision: string) => void;
}) {
  const copy = useCopy();
  const [q, setQ] = useState<any>({});
  const refresh = () => { void api.reviewQueue().then(setQ).catch(() => {}); };
  useEffect(() => { refresh(); }, []);
  const drafts = asArray(q.drafts);
  const inbox = asArray(q.inbox);
  const offers = asArray(q.offers);
  const notices = props.notices || [];
  if (!drafts.length && !inbox.length && !offers.length && !notices.length) {
    return (
      <div className="px-3 py-6 text-center" data-testid="inbox-menu">
        <p className="text-[13px] font-medium text-foreground/85">{copy.review.nothingWaiting}</p>
        <p className="mt-1 text-[12px] leading-[1.55] text-muted">{copy.review.noQueue}</p>
      </div>
    );
  }
  return (
    <div className="max-h-[28rem] overflow-auto py-1" data-testid="inbox-menu">
      {notices.length ? (
        <section>
          <MenuLabel count={notices.length}>{copy.rail.notifications}</MenuLabel>
          <ul>
            {notices.slice(0, 12).map((n) => (
              <li key={n.id}>
                <button
                  type="button"
                  className="flex w-full flex-col px-3 py-2.5 text-left hover:bg-lift/50"
                  onClick={() => props.onNotice?.(n)}
                >
                  <span className="truncate text-[13px] font-medium text-foreground">{n.title}</span>
                  {n.body ? <span className="mt-0.5 line-clamp-2 text-[12px] leading-[1.5] text-muted">{n.body}</span> : null}
                </button>
              </li>
            ))}
          </ul>
          {props.onClearNotices ? (
            <div className="px-3 pb-2">
              <TextAction onClick={props.onClearNotices}>{copy.rail.clearNotifications}</TextAction>
            </div>
          ) : null}
        </section>
      ) : null}
      {offers.length ? (
        <section>
          <MenuLabel count={offers.length}>{copy.review.approvals}</MenuLabel>
          <ul>
            {offers.map((o: any) => {
              const id = str(o.id || o.ID);
              const req = o.request || o.Request || {};
              const session = str(req.session_id || req.SessionID);
              return (
                <li key={id} className="px-3 py-2.5">
                  <div className="flex items-start gap-2">
                    <div className="min-w-0 flex-1">
                      <div className="text-[13px] font-medium text-foreground">{str(req.action || req.Action, "approval")}</div>
                      <div className="mt-0.5 truncate font-mono text-[11px] text-muted">{str(req.command || req.Command || req.path || req.Path)}</div>
                    </div>
                    {session && props.onOpenThread ? (
                      <IconAction label={copy.review.openThread} onClick={() => props.onOpenThread?.(session)}>
                        <ArrowUpRight className="size-3.5" aria-hidden />
                      </IconAction>
                    ) : null}
                  </div>
                  <div className="mt-2 flex items-center gap-1">
                    <TextAction tone="primary" onClick={() => { void props.onResolve?.(id, "once"); refresh(); }}>{copy.review.once}</TextAction>
                    <TextAction tone="danger" onClick={() => { void props.onResolve?.(id, "deny"); refresh(); }}>{copy.review.deny}</TextAction>
                  </div>
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
      {inbox.length ? (
        <section>
          <MenuLabel count={inbox.length}>{copy.review.inbox}</MenuLabel>
          <ul>
            {inbox.map((it: any) => {
              const id = str(it.id || it.ID);
              const session = str(it.session_id || it.SessionID);
              const offer = offers.find((o: any) => str((o.request || o.Request || {}).session_id || (o.request || o.Request || {}).SessionID) === session);
              const offerId = str(offer?.id || offer?.ID);
              return (
                <li key={id} className="px-3 py-2.5">
                  <div className="flex items-start gap-2">
                    <div className="min-w-0 flex-1">
                      <div className="text-[13px] font-medium text-foreground">{str(it.title || it.Title)}</div>
                      <div className="mt-0.5 text-[12px] leading-[1.5] text-muted">{str(it.body || it.Body).slice(0, 200)}</div>
                    </div>
                    {session && props.onOpenThread ? (
                      <IconAction label={copy.review.openThread} onClick={() => { void api.inboxRead(id); props.onOpenThread?.(session); }}>
                        <ArrowUpRight className="size-3.5" aria-hidden />
                      </IconAction>
                    ) : null}
                  </div>
                  <div className="mt-2 flex items-center gap-1">
                    {offerId ? (
                      <>
                        <TextAction tone="primary" onClick={() => { void props.onResolve?.(offerId, "once"); void api.inboxRead(id); refresh(); }}>{copy.review.once}</TextAction>
                        <TextAction tone="danger" onClick={() => { void props.onResolve?.(offerId, "deny"); void api.inboxDismiss(id); refresh(); }}>{copy.review.deny}</TextAction>
                      </>
                    ) : (
                      <TextAction onClick={() => { void api.inboxDismiss(id); refresh(); }}>{copy.review.dismiss}</TextAction>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
      {drafts.length ? (
        <section>
          <MenuLabel count={drafts.length}>{copy.review.drafts}</MenuLabel>
          <ul>
            {drafts.map((d: any, i: number) => (
              <li key={str(d.id || i)} className="px-3 py-2.5">
                <div className="text-[13px] font-medium text-foreground">{str(d.to)}</div>
                <div className="mt-0.5 text-[12px] text-muted">{str(d.subject)}</div>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}

export function inboxBadgeCount(inbox: any[], queue: any, notices = 0): number {
  const unread = inbox.filter((it: any) => it.unread || it.Unread).length;
  const offers = asArray(queue?.offers).length;
  const drafts = asArray(queue?.drafts).length;
  return unread + offers + drafts + notices;
}

function MenuLabel({ children, count }: { children: ReactNode; count?: number }) {
  return (
    <div className="flex h-8 items-center gap-2 px-3 text-[11px] font-medium text-muted">
      <span>{children}</span>
      {count ? <span className="tabular-nums text-muted/60">{count}</span> : null}
    </div>
  );
}

function IconAction({ label, onClick, children }: { label: string; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" aria-label={label} className="grid size-6 shrink-0 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" onClick={onClick}>
      {children}
    </button>
  );
}

function TextAction({ children, onClick, tone }: { children: ReactNode; onClick: () => void; tone?: "primary" | "danger" | "ghost" }) {
  return (
    <button
      type="button"
      className={cn(
        "h-6 rounded-md px-2 text-[11px] font-medium transition-colors",
        tone === "primary" && "bg-foreground text-background hover:opacity-90",
        tone === "danger" && "text-muted hover:bg-danger/10 hover:text-danger",
        (!tone || tone === "ghost") && "text-muted hover:bg-lift hover:text-foreground",
      )}
      onClick={onClick}
    >
      {children}
    </button>
  );
}
