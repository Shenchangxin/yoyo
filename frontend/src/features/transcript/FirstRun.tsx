/**
 * 拥有：第一次打开时主对话里的那封短信：三条本机核对过的事，一个问题，几枚选项，可以跳过。
 * 只在第一段空对话里出现；点了选项、跳过，或者第一段对话开始之后，就不再出现。
 * 不拥有：事实从哪来，那是记忆里本机清点出来的几条。点选项只把一句话放进输入框，不替你发。
 */
import { useEffect, useRef, useState, type ReactNode } from "react";
import { ArrowUpRight } from "lucide-react";
import { memoryList } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { memoryPortrait, type MemoryRow } from "../superagent/work-model";
import { PresenceStamp } from "../presence";

const DONE_KEY = "yoyo.first-run-done";
const FACTS_SHOWN = 3;
const LINES_SHOWN = 2;

function isDone(): boolean {
  try {
    return localStorage.getItem(DONE_KEY) === "1";
  } catch {
    return true;
  }
}

function markDone() {
  try {
    localStorage.setItem(DONE_KEY, "1");
  } catch {
    /* 写不进去时下次还会出现，不影响使用。 */
  }
}

/** FirstRun 在第一次的空对话里代替问候。之后渲染 fallback。 */
export function FirstRun(props: { onPrompt?: (text: string) => void; fallback: ReactNode }) {
  const copy = useCopy();
  const d = copy.desk;
  const w = copy.work;
  const [done, setDone] = useState(isDone);
  const [rows, setRows] = useState<MemoryRow[]>([]);
  const shown = useRef(false);
  useEffect(() => {
    if (done) return;
    shown.current = true;
    let alive = true;
    void memoryList().then((list) => { if (alive) setRows(list as MemoryRow[]); }).catch(() => {});
    return () => {
      alive = false;
      if (shown.current) markDone();
    };
  }, [done]);
  if (done) return <>{props.fallback}</>;
  const portrait = memoryPortrait(rows);
  const sep = /[\u4e00-\u9fff]/.test(w.memoryProjectsLine) ? "、" : ", ";
  const names = portrait.lines.slice(0, LINES_SHOWN).map((line) => line.name).filter(Boolean);
  const facts = [
    portrait.host ? w.memoryHostLine.replace("{name}", portrait.host) : "",
    portrait.os ? w.memoryOsLine.replace("{name}", portrait.os) : "",
    names.length ? w.memoryProjectsLine.replace("{names}", names.join(sep)) : "",
    portrait.hours ? w.memoryHoursLine.replace("{hours}", portrait.hours) : "",
    portrait.git ? w.memoryGitLine.replace("{name}", portrait.git) : "",
  ].filter(Boolean).slice(0, FACTS_SHOWN);
  const chips = [
    ...names.map((name) => ({ key: `line:${name}`, label: name, text: d.firstPromptLine.replace("{name}", name) })),
    { key: "code", label: d.firstCode, text: d.firstPromptCode },
    { key: "video", label: d.firstVideo, text: d.firstPromptVideo },
    { key: "learn", label: d.firstLearn, text: d.firstPromptLearn },
    { key: "chores", label: d.firstChores, text: d.firstPromptChores },
  ];
  function finish() {
    markDone();
    shown.current = false;
    setDone(true);
  }
  return (
    <div className="empty-rise flex gap-2.5" data-testid="first-run">
      <PresenceStamp size={36} className="mt-0.5 shrink-0" />
      <div className="min-w-0 flex-1">
        <p className="text-ui-body-l text-foreground">{d.firstHello}</p>
        {facts.length ? (
          <ul className="mt-2 space-y-1 text-ui-body text-foreground/85">
            {facts.map((fact) => (
              <li key={fact} className="flex gap-2">
                <span className="mt-[9px] size-1 shrink-0 rounded-full bg-muted" aria-hidden />
                <span className="min-w-0">{fact}</span>
              </li>
            ))}
          </ul>
        ) : null}
        <p className="mt-3 text-ui-body-l text-foreground">{d.firstAsk}</p>
        <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
          {chips.map((chip) => (
            <button
              key={chip.key}
              type="button"
              className="inline-flex h-7 items-center gap-1 rounded-full border border-border bg-transparent pl-3 pr-2 text-ui-footnote text-foreground/85 transition-colors duration-[var(--duration-fast)] hover:bg-lift hover:text-foreground"
              onClick={() => {
                props.onPrompt?.(chip.text);
                finish();
              }}
            >
              {chip.label}
              <ArrowUpRight className="size-3 text-muted" aria-hidden />
            </button>
          ))}
          <button type="button" className="ml-1 text-ui-footnote text-muted hover:text-foreground" onClick={finish}>
            {d.firstSkip}
          </button>
        </div>
      </div>
    </div>
  );
}
