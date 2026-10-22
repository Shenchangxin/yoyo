import { useEffect, useRef, useState } from "react";
import { Phone, Send } from "lucide-react";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { str } from "../../lib/normalize";
import { cn } from "../../lib/utils";
import { clock, hexAlpha, imageURL, type Idol, type Line } from "./model";

export function IdolChat(props: {
  idol: Idol;
  lines: Line[];
  pending: string;
  sending: boolean;
  notice: string;
  text: string;
  work: boolean;
  canWork: boolean;
  ended: boolean;
  cooling: string;
  viewingTitle: string;
  ready: boolean;
  giftNames: Record<string, string>;
  onText: (v: string) => void;
  onWork: (v: boolean) => void;
  onSend: () => void;
  onNewChat: () => void;
  onOpenGifts: () => void;
  onBack: () => void;
  onOpenWork: (id: string) => void;
}) {
  const c = useCopy().idol;
  const scroller = useRef<HTMLDivElement>(null);
  const stick = useRef(true);
  const locked = !!props.viewingTitle || props.ended || !!props.cooling;

  useEffect(() => {
    const el = scroller.current;
    if (!el || !stick.current) return;
    el.scrollTop = el.scrollHeight;
  }, [props.lines, props.pending, props.sending]);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="idol-chat">
      <div
        ref={scroller}
        className="min-h-0 flex-1 overflow-auto px-4 py-4"
        onScroll={(e) => {
          const el = e.currentTarget;
          stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
        }}
      >
        {!props.ready ? (
          <div className="grid h-full place-items-center text-[13px] text-muted">{c.loading}</div>
        ) : props.lines.length === 0 && !props.pending ? (
          <EmptyState
            icon={<Phone className="size-5" />}
            title={c.emptyChat.replace("{name}", props.idol.name)}
            body={props.cooling ? undefined : c.emptyChatHint}
          />
        ) : (
          <div className="mx-auto flex max-w-2xl flex-col gap-2">
            {props.lines.map((line, i) => (
              <Bubble
                key={line.id || i}
                line={line}
                color={props.idol.color}
                giftNames={props.giftNames}
                you={c.you}
                openWork={c.openWork}
                score={c.score}
                onOpenWork={props.onOpenWork}
              />
            ))}
            {props.pending ? (
              <Bubble line={{ id: "pending", role: "user", text: props.pending }} color={props.idol.color} giftNames={{}} you={c.you} openWork={c.openWork} score={c.score} onOpenWork={props.onOpenWork} />
            ) : null}
            {props.sending ? <p className="text-[12px] text-muted">{c.thinking}</p> : null}
          </div>
        )}
      </div>
      {props.notice ? (
        <div role="status" className="border-t border-warning/30 bg-warning/10 px-4 py-2 text-[12px] text-warning">{props.notice}</div>
      ) : null}
      {props.viewingTitle ? (
        <div className="flex items-center gap-3 border-t border-border bg-background px-4 py-3">
          <p className="min-w-0 flex-1 text-[13px] text-muted">{c.readOnly} · {props.viewingTitle}</p>
          <Button size="sm" variant="lift" onClick={props.onBack}>{c.backToCurrent}</Button>
        </div>
      ) : props.ended ? (
        <div className="flex flex-wrap items-center gap-3 border-t border-border bg-background px-4 py-3">
          <p className="min-w-0 flex-1 text-[13px] text-muted">{c.ended}</p>
          {props.cooling ? <Button size="sm" variant="lift" onClick={props.onOpenGifts}>{c.give}</Button> : null}
          <Button size="sm" onClick={props.onNewChat}>{c.newChat}</Button>
        </div>
      ) : props.cooling ? (
        <div className="flex flex-wrap items-center gap-3 border-t border-border bg-background px-4 py-3">
          <p className="min-w-0 flex-1 text-[13px] text-muted">{c.cooldownUntil.replace("{time}", props.cooling)}</p>
          <Button size="sm" variant="lift" onClick={props.onOpenGifts}>{c.give}</Button>
        </div>
      ) : (
        <form
          className="border-t border-border bg-background px-4 py-3"
          onSubmit={(e) => { e.preventDefault(); if (!locked) props.onSend(); }}
        >
          {props.canWork ? (
            <label className="mb-2 flex items-center gap-2 text-[12px] text-muted" title={c.workHint}>
              <input type="checkbox" checked={props.work} onChange={(e) => props.onWork(e.target.checked)} />
              {c.work}
            </label>
          ) : null}
          <div className="flex items-end gap-2">
            <textarea
              rows={1}
              value={props.text}
              placeholder={c.placeholder}
              aria-label={c.placeholder}
              disabled={props.sending}
              className="max-h-32 min-h-10 flex-1 resize-none rounded-xl border border-border bg-background px-3 py-2 text-[13px] outline-none focus:ring-2 focus:ring-accent/40"
              onChange={(e) => props.onText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  props.onSend();
                }
              }}
            />
            <Button type="submit" size="send" aria-label={c.send} disabled={props.sending || !props.text.trim()} aria-busy={props.sending}>
              <Send className="size-4" />
            </Button>
          </div>
        </form>
      )}
    </div>
  );
}

function Bubble(props: {
  line: Line;
  color: string;
  giftNames: Record<string, string>;
  you: string;
  openWork: string;
  score: string;
  onOpenWork: (id: string) => void;
}) {
  const line = props.line;
  if (line.role === "event") {
    const gift = line.gift ? (props.giftNames[line.gift] || line.text) : line.text;
    return (
      <div className="py-1 text-center text-[12px] text-muted">
        {gift}
        {line.delta ? <span className="ml-2 tabular-nums">{props.score} {line.delta > 0 ? `+${line.delta}` : line.delta}</span> : null}
      </div>
    );
  }
  const mine = line.role === "user";
  return (
    <div className={cn("flex", mine ? "justify-end" : "justify-start")}>
      <div
        className={cn("max-w-[80%] rounded-2xl px-3 py-2 text-[13px] leading-relaxed", mine ? "bg-foreground text-background" : "text-foreground")}
        style={mine ? undefined : { background: hexAlpha(props.color, 0.16) }}
      >
        <div className={cn("mb-0.5 text-[10px]", mine ? "text-background/70" : "text-muted")}>
          {mine ? props.you : null}
          {line.at ? <span className="ml-2 tabular-nums">{clock(line.at)}</span> : null}
        </div>
        {line.text ? <p className="whitespace-pre-wrap">{line.text}</p> : null}
        {line.role === "idol" && line.photo ? <BlobImage id={line.photo} /> : null}
        {line.role === "idol" && line.delta ? (
          <div className="mt-1 text-[11px] text-muted">{props.score} {line.delta > 0 ? `+${line.delta}` : line.delta}</div>
        ) : null}
        {line.role === "idol" && line.work ? (
          <button type="button" className="mt-1 text-[12px] underline" onClick={() => props.onOpenWork(line.work || "")}>{props.openWork}</button>
        ) : null}
      </div>
    </div>
  );
}

function BlobImage({ id }: { id: string }) {
  const [src, setSrc] = useState("");
  useEffect(() => {
    let stop = false;
    void api.idolCall("blob", { id }).then((o) => {
      const data = str(o?.data);
      if (!stop && data) setSrc(imageURL(data));
    }).catch(() => {});
    return () => { stop = true; };
  }, [id]);
  if (!src) return <div className="mt-2 h-24 w-40 rounded-xl bg-black/5" />;
  return <img alt="" src={src} className="mt-2 max-h-56 rounded-xl object-cover" />;
}
