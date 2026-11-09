/**
 * 拥有：Buddy 栏目的左侧栏：标题和在线个数、在线 / 需要处理 / 可安装 / 更多四组、「更多」的折叠、「添加自定义 agent」，以及上下键移动和回车。
 * 不拥有：取数（buddy-store.ts）和右侧的总览与详情。点一行只改 buddyFocus。
 */

import { Plus } from "lucide-react";
import { useMemo, useState, type KeyboardEvent } from "react";
import type { BuddyView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { StatusDot } from "../../components/ui/status-dot";
import { BuddyMark } from "./BuddyMark";
import { buddySections, buddyStatusText, buddyTone, fill, onlineCount, type BuddyCopy } from "./buddy-model";
import { useBuddy } from "./buddy-store";

const ROW =
  "mb-px flex min-h-8 w-full items-center gap-2 rounded-lg px-2.5 py-[6px] text-left text-ui-body transition-[background-color,color] duration-[var(--duration-fast)] ease-[var(--ease-out)]";

/** rowTone 是一行的底色：选中时抬起，否则悬停时浅浅抬起。 */
function rowTone(on: boolean) {
  return on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground";
}

/** moveFocus 在侧栏的行之间按上下键移动焦点。回车由按钮自己处理。 */
function moveFocus(e: KeyboardEvent<HTMLElement>) {
  if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
  const rows = Array.from(e.currentTarget.querySelectorAll<HTMLButtonElement>("[data-buddy-row]"));
  const at = rows.indexOf(document.activeElement as HTMLButtonElement);
  const next = rows[e.key === "ArrowDown" ? Math.min(rows.length - 1, at + 1) : Math.max(0, at - 1)];
  if (next) {
    e.preventDefault();
    next.focus();
  }
}

/** BuddyRailRow 是一个 agent：字形、名字，右边是状态点；安装中时右边是百分比。 */
function BuddyRailRow({ v, on, c, onPick }: { v: BuddyView; on: boolean; c: BuddyCopy; onPick: (id: string) => void }) {
  const tone = buddyTone(v);
  const status = buddyStatusText(v, c);
  return (
    <button
      type="button"
      data-buddy-row
      aria-current={on ? "page" : undefined}
      title={status}
      className={cn(ROW, rowTone(on))}
      onClick={() => onPick(v.id)}
    >
      <BuddyMark v={v} className="size-4" />
      <span className="min-w-0 flex-1 truncate">{v.name}</span>
      {v.state === "installing" ? (
        <span className="shrink-0 text-ui-caption tabular-nums text-muted">{v.progress?.percent ? `${v.progress.percent}%` : "…"}</span>
      ) : (
        <StatusDot tone={tone.tone} still={tone.still} className="mr-0.5" />
      )}
      <span className="sr-only">{status}</span>
    </button>
  );
}

/** BuddyRailGroup 是一组：上面一条细线和组名，下面是行。空组不画。 */
function BuddyRailGroup(props: { label: string; list: BuddyView[]; focus: string; c: BuddyCopy; onPick: (id: string) => void }) {
  if (!props.list.length) return null;
  return (
    <div role="group" aria-label={props.label} className="mt-1.5 border-t border-border/60 pt-1.5 first:mt-0 first:border-t-0 first:pt-0">
      <p className="px-2.5 pb-1 pt-0.5 text-ui-caption text-muted">{props.label}</p>
      {props.list.map((v) => (
        <BuddyRailRow key={v.id} v={v} on={props.focus === v.id} c={props.c} onPick={props.onPick} />
      ))}
    </div>
  );
}

/** BuddyRailNav 画出 Buddy 的侧栏。还没取到列表时画三行骨架。 */
export function BuddyRailNav() {
  const copy = useCopy();
  const c = copy.buddy;
  const list = useBuddy((s) => s.list);
  const loaded = useBuddy((s) => s.loaded && !s.detecting);
  const openDialog = useBuddy((s) => s.openDialog);
  const focus = useUI((s) => s.buddyFocus);
  const setFocus = useUI((s) => s.setBuddyFocus);
  const [moreOpen, setMoreOpen] = useState(false);
  const sections = useMemo(() => buddySections(list), [list]);
  const more = sections.more;
  const focusInMore = more.some((v) => v.id === focus);
  const showMore = moreOpen || focusInMore;
  return (
    <nav aria-label={c.title} data-testid="buddy-rail" className="px-1.5 pb-1" onKeyDown={moveFocus}>
      <button
        type="button"
        data-buddy-row
        aria-current={focus === "" ? "page" : undefined}
        aria-label={c.overview}
        className={cn(ROW, rowTone(focus === ""), "font-medium")}
        onClick={() => setFocus("")}
      >
        <span className="min-w-0 flex-1 truncate">{c.title}</span>
        {loaded ? <span className="shrink-0 text-ui-caption font-normal text-muted">{fill(c.onlineCount, { n: onlineCount(list) })}</span> : null}
      </button>
      {!loaded ? (
        <div aria-label={c.loading} className="mt-1.5 space-y-1.5 px-2.5" role="status">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-5 animate-pulse rounded-md bg-lift/70" />
          ))}
        </div>
      ) : (
        <div className="mt-1.5">
          <BuddyRailGroup label={c.sectionOnline} list={sections.online} focus={focus} c={c} onPick={setFocus} />
          <BuddyRailGroup label={c.sectionNeeds} list={sections.needs} focus={focus} c={c} onPick={setFocus} />
          <BuddyRailGroup label={c.sectionAvailable} list={sections.available} focus={focus} c={c} onPick={setFocus} />
          <div role="group" aria-label={c.sectionMore} className="mt-1.5 border-t border-border/60 pt-1.5">
            {more.length ? (
              <>
                {showMore ? (
                  <>
                    <p className="px-2.5 pb-1 pt-0.5 text-ui-caption text-muted">{c.sectionMore}</p>
                    {more.map((v) => (
                      <BuddyRailRow key={v.id} v={v} on={focus === v.id} c={c} onPick={setFocus} />
                    ))}
                  </>
                ) : null}
                {focusInMore ? null : (
                  <button type="button" data-buddy-row className={cn(ROW, rowTone(false))} aria-expanded={showMore} onClick={() => setMoreOpen(!moreOpen)}>
                    <span className="min-w-0 flex-1 truncate">{showMore ? c.fewer : fill(c.moreCount, { n: more.length })}</span>
                  </button>
                )}
              </>
            ) : null}
            <button type="button" data-buddy-row className={cn(ROW, rowTone(false))} onClick={() => openDialog({ kind: "custom" })}>
              <Plus className="size-3.5 shrink-0 opacity-70" aria-hidden />
              <span className="min-w-0 flex-1 truncate">{c.addCustom}</span>
            </button>
          </div>
        </div>
      )}
    </nav>
  );
}
