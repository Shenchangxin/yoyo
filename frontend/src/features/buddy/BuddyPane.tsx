/**
 * 拥有：今天栏里的 Buddy 页，和今天、任务、记忆并排。默认是名单：要处理的、在这台电脑上发现的、在线的、能装的；
 * 点一项进它的详情；偏好、添加自定义 agent、日志也都在这一栏里翻页，不再弹对话框、也不另开一个栏目。
 * 不拥有：取数和轮询（buddy-store.ts）、动作（buddy-actions.ts）、判断（buddy-model.ts）。
 * 翻到哪一页由两处决定：仓库里的 dialog（偏好、自定义、日志、配置）和 useUI 的 buddyFocus（选中的 agent）。
 */

import { useEffect, useMemo, useState } from "react";
import { Loader2, Plus, RefreshCw, SlidersHorizontal } from "lucide-react";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { StatusDot } from "../../components/ui/status-dot";
import type { BuddyView } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { DayCard, DayMore, DayPill, DayRound, DaySection, DayTile, DayTitle } from "../superagent/day-ui";
import { BuddyAgentPage } from "./BuddyAgentPage";
import { BuddyCustomPage, BuddyLogPage, BuddyPrefsPage } from "./BuddyPages";
import { AgentAvatar, BuddyConfirms, BuddyMore, BuddyPrimary } from "./BuddyParts";
import { buddyDo } from "./buddy-actions";
import { buddyPrimary, buddySections, buddyStatusText, buddyTone, fill, hasExternal, progressBar, type BuddyCopy } from "./buddy-model";
import { useBuddy } from "./buddy-store";

/** RECOMMENDED 是一个外部 agent 都没装时推荐的三个。 */
const RECOMMENDED = ["claude", "codex", "gemini"];

/** Skeleton 是第一次取数时的几行占位。 */
function Skeleton({ c }: { c: BuddyCopy }) {
  return (
    <div role="status" aria-label={c.loading} className="mt-4 space-y-2" data-testid="buddy-skeleton">
      {[0, 1, 2, 3].map((i) => (
        <DayTile key={i} className="flex items-center gap-3 py-3.5">
          <div className="size-9 animate-pulse rounded-full bg-lift/80" />
          <div className="flex-1 space-y-1.5">
            <div className="h-3 w-32 animate-pulse rounded-sm bg-lift/80" />
            <div className="h-2.5 w-20 animate-pulse rounded-sm bg-lift/60" />
          </div>
        </DayTile>
      ))}
    </div>
  );
}

/** Row 是名单里的一行：头像、名字和版本、状态句，右边是主按钮和更多。点名字进详情。安装中时行底一条进度细线。 */
function Row({ v, onOpen }: { v: BuddyView; onOpen: (id: string) => void }) {
  const c = useCopy().buddy;
  const tone = buddyTone(v);
  const status = buddyStatusText(v, c);
  const bar = progressBar(v);
  const quiet = v.state === "unsupported";
  return (
    <div className="relative flex min-w-0 items-center gap-3 py-3" data-testid={`buddy-row-${v.id}`}>
      <AgentAvatar v={v} quiet={quiet} />
      <button
        type="button"
        className="min-w-0 flex-1 cursor-pointer rounded-lg text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
        onClick={() => onOpen(v.id)}
      >
        <span className="flex min-w-0 items-baseline gap-1.5">
          <span className={cn("truncate text-ui-body-l font-medium", quiet && "text-muted")}>{v.name}</span>
          {v.version ? <span className="shrink-0 text-ui-caption tabular-nums text-muted">{fill(c.detail.version, { v: v.version })}</span> : null}
        </span>
        <span className="mt-0.5 flex min-w-0 items-start gap-1.5 text-ui-footnote leading-5 text-muted" title={status}>
          {quiet ? null : <StatusDot tone={tone.tone} still={tone.still} className="mt-[7px]" />}
          <span className="line-clamp-2 min-w-0" data-testid={`buddy-status-${v.id}`}>{status}</span>
        </span>
      </button>
      <div className="flex shrink-0 items-center gap-0.5">
        <BuddyPrimary v={v} small hideTest />
        <BuddyMore v={v} />
      </div>
      {v.state === "installing" ? <ProgressHairline className="absolute inset-x-0 bottom-0" value={bar.value} indeterminate={bar.indeterminate} /> : null}
    </div>
  );
}

/** Group 是名单里的一组：标题和个数，下面一张小卡片，行与行之间一条细线。空组不画。 */
function Group(props: { title: string; hint?: string; list: BuddyView[]; person?: boolean; onOpen: (id: string) => void; testId?: string }) {
  if (!props.list.length) return null;
  return (
    <DaySection title={props.title} count={props.list.length} person={props.person} hint={props.hint} testId={props.testId}>
      <DayTile className="divide-y divide-border/50">
        {props.list.map((v) => (
          <Row key={v.id} v={v} onOpen={props.onOpen} />
        ))}
      </DayTile>
    </DaySection>
  );
}

/** Welcome 是一个外部 agent 都没有时的那张卡：一句话说清它是干什么的，下面三个推荐，点了就装。 */
function Welcome({ list, c }: { list: BuddyView[]; c: BuddyCopy }) {
  const picks = RECOMMENDED.map((id) => list.find((v) => v.id === id)).filter((v): v is BuddyView => !!v);
  return (
    <DayCard className="mt-4" testId="buddy-welcome">
      <h2 className="text-ui-title font-semibold tracking-[-0.03em]">{c.emptyTitle}</h2>
      <p className="mt-1.5 text-ui-body leading-6 text-muted">{c.emptyBody}</p>
      {picks.length ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {picks.map((v) => {
            const action = buddyPrimary(v);
            const installing = v.state === "installing";
            return (
              <DayPill key={v.id} tone="soft" disabled={installing || (action !== "install" && action !== "installSystem")} onClick={() => void buddyDo(v, action)}>
                {installing ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
                {fill(c.installNamed, { name: v.name })}
              </DayPill>
            );
          })}
        </div>
      ) : null}
    </DayCard>
  );
}

/** ListPage 是名单页。一个外部 agent 都没有时先给欢迎卡，推荐里还没装的不再在下面重复。 */
function ListPage({ c }: { c: BuddyCopy }) {
  const list = useBuddy((s) => s.list);
  const ready = useBuddy((s) => s.loaded && !s.detecting);
  const error = useBuddy((s) => s.error);
  const openDialog = useBuddy((s) => s.openDialog);
  const setFocus = useUI((s) => s.setBuddyFocus);
  const [moreOpen, setMoreOpen] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const empty = ready && !hasExternal(list);
  const sections = useMemo(() => {
    const s = buddySections(list);
    if (!empty) return s;
    const hide = (v: BuddyView) => RECOMMENDED.includes(v.id) && v.state === "not_installed" && !s.found.includes(v);
    return { ...s, online: [], available: s.available.filter((v) => !hide(v)), more: s.more.filter((v) => !hide(v)) };
  }, [list, empty]);
  return (
    <div className="px-1 pb-8 pt-2" data-testid="buddy-pane">
      <div className="flex items-center justify-between gap-3">
        <DayTitle>{c.title}</DayTitle>
        <div className="flex shrink-0 items-center gap-1.5">
          <DayRound
            label={c.refresh}
            disabled={refreshing}
            onClick={() => {
              setRefreshing(true);
              void useBuddy.getState().load("force").finally(() => setRefreshing(false));
            }}
          >
            <RefreshCw className={cn("size-3.5", refreshing && "animate-spin")} aria-hidden />
          </DayRound>
          <DayRound label={c.prefs} onClick={() => openDialog({ kind: "prefs" })}>
            <SlidersHorizontal className="size-3.5" aria-hidden />
          </DayRound>
        </div>
      </div>
      <p className="mt-1.5 text-ui-footnote leading-5 text-muted">{c.paneIntro}</p>
      <BuddyConfirms />
      {!ready ? (
        error && !list.length ? <p role="alert" className="mt-4 px-1 text-ui-body text-danger">{fill(c.loadFailed, { error })}</p> : <Skeleton c={c} />
      ) : (
        <div data-testid="buddy-overview">
          {empty ? <Welcome list={list} c={c} /> : null}
          <Group title={c.sectionNeeds} list={sections.needs} person onOpen={setFocus} />
          <Group title={c.sectionFound} hint={c.foundHint} list={sections.found} onOpen={setFocus} testId="buddy-found" />
          <Group title={c.sectionOnline} list={sections.online} onOpen={setFocus} />
          <Group title={c.sectionAvailable} list={sections.available} onOpen={setFocus} />
          {sections.more.length && moreOpen ? <Group title={c.sectionMore} list={sections.more} onOpen={setFocus} /> : null}
          {sections.more.length ? (
            <DayMore aria-expanded={moreOpen} onClick={() => setMoreOpen(!moreOpen)}>
              {moreOpen ? c.fewer : fill(c.moreCount, { n: sections.more.length })}
            </DayMore>
          ) : null}
          <button
            type="button"
            className="day-tile mt-4 flex w-full items-center gap-3 px-4 py-3.5 text-left"
            onClick={() => openDialog({ kind: "custom" })}
          >
            <Plus className="size-4 shrink-0 text-muted" aria-hidden />
            <span className="min-w-0 flex-1">
              <span className="block truncate text-ui-body-l font-medium">{c.addCustom}</span>
              <span className="mt-0.5 block truncate text-ui-footnote leading-5 text-muted">{c.custom.body}</span>
            </span>
          </button>
        </div>
      )}
    </div>
  );
}

/**
 * BuddyPane 是今天栏里的 Buddy 页。打开时立刻取一次名单。
 * 要去配置一项时直接翻到它的详情，那里最上面就是要填的东西；选中的项找不到时（比如刚移除）回到名单。
 */
export function BuddyPane() {
  const c = useCopy().buddy;
  const list = useBuddy((s) => s.list);
  const dialog = useBuddy((s) => s.dialog);
  const openDialog = useBuddy((s) => s.openDialog);
  const focus = useUI((s) => s.buddyFocus);
  const setFocus = useUI((s) => s.setBuddyFocus);
  useEffect(() => {
    void useBuddy.getState().load();
  }, []);
  useEffect(() => {
    // 「配置」不再弹对话框：翻到那一项的详情，要填的东西就在最上面。翻过去之后这张单子就用完了。
    if (dialog?.kind !== "setup") return;
    setFocus(dialog.id);
    openDialog(null);
  }, [dialog, setFocus, openDialog]);
  const close = () => openDialog(null);
  if (dialog?.kind === "prefs") return <BuddyPrefsPage onBack={close} />;
  if (dialog?.kind === "custom") return <BuddyCustomPage onBack={close} />;
  if (dialog?.kind === "log") {
    const target = list.find((v) => v.id === dialog.id);
    if (target) return <BuddyLogPage v={target} onBack={close} />;
  }
  const focused = focus ? list.find((v) => v.id === focus) : undefined;
  if (focused) return <BuddyAgentPage key={focused.id} v={focused} onBack={() => setFocus("")} />;
  return <ListPage c={c} />;
}
