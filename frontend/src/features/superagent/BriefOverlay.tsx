/**
 * 拥有：主会话里点开一份历史简报时的弹层。点按钮把任务放进输入框，不直接发出。
 * 不拥有：简报什么时候推进。那在 desk-feed。
 */

import { useState, useSyncExternalStore } from "react";
import { Button } from "../../components/ui/button";
import { useCopy } from "../../lib/i18n";
import { briefDay, briefPrompt, briefStorageKey, parseStoredBrief, serializeStoredBrief, type BriefCard, type BriefSlot, type MorningBrief } from "./work-model";
import { BriefStack } from "./WorkPane";
import { BRIEF_EVENT } from "./desk-feed";

type OpenBrief = { id: string; chip: string; slot: string; day: string; brief: string };

let current: OpenBrief | null = null;
const listeners = new Set<() => void>();

/** askBrief 打开一份已经推进过的简报。 */
export function askBrief(next: OpenBrief) {
  current = next;
  listeners.forEach((fn) => fn());
}

function closeBrief() {
  current = null;
  listeners.forEach((fn) => fn());
}

function subscribe(fn: () => void) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

/** BriefOverlay 盖在会话上。没有打开的简报时不占地方。 */
export function BriefOverlay(props: { onDraft: (text: string) => void }) {
  const copy = useCopy();
  const open = useSyncExternalStore(subscribe, () => current, () => null);
  if (!open) return null;
  return <BriefDialog open={open} onClose={closeBrief} onDraft={props.onDraft} title={copy.work.brief} closeLabel={copy.work.briefClose} filledLabel={copy.work.briefFilled} />;
}

function BriefDialog(props: { open: OpenBrief; onClose: () => void; onDraft: (text: string) => void; title: string; closeLabel: string; filledLabel: string }) {
  const [brief, setBrief] = useState<MorningBrief | null>(() => resolveBrief(props.open));
  function useCard(card: BriefCard, which: "primary" | "secondary") {
    const text = briefPrompt(card, which).trim();
    if (brief) {
      const next = { ...brief, picked: { ...brief.picked, [card.id]: which } };
      setBrief(next);
      rememberPick(props.open, next);
    }
    if (text) props.onDraft(text);
    props.onClose();
  }
  function mark(key: string, value: string) {
    if (!brief) return;
    const next = { ...brief, picked: { ...brief.picked, [key]: value } };
    setBrief(next);
    rememberPick(props.open, next);
  }
  return (
    <div className="absolute inset-0 z-40 flex items-center justify-center bg-black/30 p-6" onClick={props.onClose}>
      <section className="flex max-h-[min(720px,100%)] w-full max-w-xl flex-col overflow-hidden rounded-xl border border-border bg-background shadow-[var(--shadow-card)]" onClick={(event) => event.stopPropagation()}>
        <div className="flex shrink-0 items-center justify-between px-4 py-3">
          <h2 className="text-ui-title-s font-semibold">{props.open.chip || props.title}</h2>
          <Button size="sm" variant="lift" onClick={props.onClose}>
            {props.closeLabel}
          </Button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-5">
          {brief ? (
            <BriefStack
              brief={brief}
              onUse={useCard}
              onMark={mark}
              onDraft={(text) => { props.onDraft(text); props.onClose(); }}
              filledLabel={props.filledLabel}
            />
          ) : null}
        </div>
      </section>
    </div>
  );
}

function resolveBrief(open: OpenBrief): MorningBrief | null {
  const slot = open.slot === "evening" ? "evening" : "morning";
  const stored = open.day === briefDay() ? parseStoredBrief(localStorage.getItem(briefStorageKey(slot)), open.day) : null;
  const fromMessage = parsePayload(open.brief);
  if (fromMessage) return { ...fromMessage, picked: { ...fromMessage.picked, ...(stored?.picked || {}) } };
  return stored;
}

function parsePayload(raw: string): MorningBrief | null {
  try {
    const body = JSON.parse(raw) as MorningBrief;
    if (!body || !Array.isArray(body.cards) || body.cards.length < 1) return null;
    return { cards: body.cards, picked: body.picked || {}, welcome: body.welcome === true, woven: body.woven === true };
  } catch {
    return null;
  }
}

function rememberPick(open: OpenBrief, next: MorningBrief) {
  const slot: BriefSlot = open.slot === "evening" ? "evening" : "morning";
  if (open.day !== briefDay()) return;
  const stored = parseStoredBrief(localStorage.getItem(briefStorageKey(slot)), open.day);
  const base = stored || next;
  try {
    localStorage.setItem(briefStorageKey(slot), serializeStoredBrief(open.day, { ...base, picked: next.picked }));
  } catch {
    /* 点选写不进去时，弹层这一次仍然留着。 */
  }
  window.dispatchEvent(new Event(BRIEF_EVENT));
}
