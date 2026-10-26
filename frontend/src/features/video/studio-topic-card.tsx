/**
 * 本文件拥有一张选题卡的投影：勾选、开拍或打开、角度和分数。
 * 不拥有选题列表。动作都交给回调，本文件不改服务端。
 */
import { Check, MoreHorizontal } from "lucide-react";
import { Button } from "../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { Input } from "../../components/ui/input";
import { cn } from "../../lib/utils";
import { TopicScoreSheet } from "./studio-chrome";
import { filmInFlight, topicCanFilm, topicIsScheduled, type ScoreKey } from "./studio-topic";

/** TopicRow 是选题卡要显示的标题、来源、角度、分数、状态和可选切口。 */
export type TopicRow = {
  id: string;
  title: string;
  source_platform?: string;
  angle?: string;
  note?: string;
  scores?: Record<string, number>;
  status?: string;
  studio_project_id?: string;
  persona_page_id?: string;
  options?: { id: string; title?: string; angle?: string; summary?: string }[];
  updated_at?: string;
};

type TopicCopy = {
  newProject: string;
  openShort: string;
  writeAngle: string;
  angle: string;
  schedule: string;
  unschedule: string;
  scheduleHint: string;
  scoreAvg: string;
  scoreEmpty: string;
  scoreHint: string;
  moreTopic: string;
  persona: string;
  personaDone: string;
  personaHint: string;
  angles: string;
  newFromTopics: string;
  filmTopic: string;
  viewFilm: string;
  viewProgress: string;
  filmRetry: string;
  desk: { expand: string; choose: string };
};

/** TopicCard 按成片阶段显示打开、看进度、重试或开拍。已完成或制作中的角度不可改。props 读 topic、film、copy、busy、picked、statusLabel、scoreLabels 和 deleteLabel，并把勾选、开拍、取消排期、改卡、展开、排期、人设、选切口和删除交给对应回调。 */
export function TopicCard(props: {
  topic: TopicRow;
  busy: boolean;
  picked: boolean;
  statusLabel: string;
  deleteLabel: string;
  scoreLabels: Record<ScoreKey, string>;
  film?: { auto_phase?: string; result_hash?: string };
  copy: TopicCopy;
  onPick: () => void;
  onProduce: () => void;
  onUnschedule: () => void;
  onPatch: (extra: Partial<TopicRow>) => void;
  onExpand: () => void;
  onSchedule: () => void;
  onPersona: () => void;
  onChoose: (optionId: string) => void;
  onDelete: () => void;
}) {
  const t = props.topic;
  const s = props.copy;
  const queued = topicIsScheduled(t);
  const ready = !!props.film?.result_hash && props.film?.auto_phase !== "failed";
  const live = filmInFlight(props.film?.auto_phase);
  const failed = props.film?.auto_phase === "failed" && !props.film?.result_hash;
  const pending = !!t.studio_project_id && !props.film;
  const pickable = topicCanFilm(t, props.film);
  const opening = ready || live || pending;
  const label = ready ? s.viewFilm : live || pending ? s.viewProgress : failed ? s.filmRetry : s.filmTopic;
  const locked = t.status === "done" || t.status === "in_production";

  return (
    <li className="rounded-xl border border-border bg-card px-3.5 py-3" data-testid={`studio-topic-${t.id}`} data-status={t.status || "inbox"}>
      <div className="flex items-start gap-2">
        {pickable ? (
          <button
            type="button"
            className={cn(
              "mt-0.5 grid size-7 shrink-0 place-items-center rounded-md border",
              props.picked ? "border-foreground bg-foreground text-background" : "border-border text-muted hover:bg-lift/70",
            )}
            data-testid={`studio-topic-pick-${t.id}`}
            aria-pressed={props.picked}
            aria-label={s.filmTopic}
            onClick={props.onPick}
          >
            {props.picked ? <Check className="size-3.5" aria-hidden /> : null}
          </button>
        ) : null}
        <div className="min-w-0 flex-1">
          <p className="text-[13px] font-medium text-foreground">{t.title}</p>
          <p className="mt-0.5 text-[12px] text-muted">
            {[t.source_platform, props.statusLabel, t.angle, t.persona_page_id ? s.personaDone : ""].filter(Boolean).join(" · ")}
          </p>
          {t.note ? <p className="mt-1 line-clamp-2 text-[12px] leading-[1.5] text-muted">{t.note}</p> : null}
        </div>
        <div className="flex shrink-0 items-start gap-0.5">
          <div className="flex flex-col items-end gap-1">
            <Button size="sm" variant="default" className="h-7 text-[12px]" disabled={props.busy} data-testid={opening ? "studio-topic-open" : "studio-topic-produce"} onClick={props.onProduce}>
              {label}
            </Button>
            {queued ? (
              <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={props.busy} data-testid="studio-topic-unschedule" onClick={props.onUnschedule}>
                {s.unschedule}
              </Button>
            ) : null}
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="h-7 w-7 px-0" data-testid="studio-topic-more" aria-label={s.moreTopic}>
                <MoreHorizontal className="size-3.5" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-52">
              <DropdownMenuItem disabled={props.busy} onSelect={props.onExpand}>
                {s.desk.expand}
              </DropdownMenuItem>
              {!queued && !t.studio_project_id ? (
                <DropdownMenuItem disabled={props.busy} data-testid="studio-topic-schedule" onSelect={props.onSchedule}>
                  {s.schedule}
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuItem disabled={props.busy} title={s.personaHint} onSelect={props.onPersona}>
                {s.persona}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-danger" disabled={props.busy} onSelect={props.onDelete}>
                {props.deleteLabel}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      {!locked ? (
        <Input
          className="mt-1.5 h-7 border-transparent bg-transparent px-0 text-[12px] shadow-none focus-visible:border-border focus-visible:bg-card focus-visible:px-2"
          placeholder={s.writeAngle}
          defaultValue={t.angle || ""}
          data-testid="studio-topic-angle"
          aria-label={s.angle}
          onBlur={/** e 取出去掉空白的角度，没变化就不改，否则交给 onPatch。 */ (e) => {
            const v = e.target.value.trim();
            if (v === (t.angle || "")) return;
            if (!v && !t.angle) return;
            props.onPatch({ angle: v, status: v ? (queued ? "scheduled" : t.status === "inbox" || !t.status ? "ready" : t.status) : t.status });
          }}
        />
      ) : null}
      {queued ? <p className="mt-1.5 text-[11px] text-muted">{s.scheduleHint}</p> : null}
      <TopicScoreSheet
        scores={t.scores}
        labels={props.scoreLabels}
        avgLabel={s.scoreAvg}
        emptyLabel={s.scoreEmpty}
        hint={s.scoreHint}
        disabled={props.busy}
        onChange={/** scores 交给 onPatch。 */ (scores) => props.onPatch({ scores })}
      />
      {(t.options || []).length ? (
        <div className="mt-2 border-t border-border/70 pt-2" data-testid="studio-topic-options">
          <p className="px-0.5 pb-1 text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80">{s.angles}</p>
          <ul className="divide-y divide-border/70">
            {t.options!.map(/** opt 显示 title、angle 和 summary，点选时把 id 交给 onChoose。 */ (opt) => (
              <li key={opt.id} className="flex items-start gap-2 py-2">
                <div className="min-w-0 flex-1">
                  <p className="text-[12px] font-medium text-foreground">{opt.title}</p>
                  <p className="text-[11px] leading-[1.45] text-muted">{[opt.angle, opt.summary].filter(Boolean).join(" · ")}</p>
                </div>
                <Button size="sm" variant="ghost" className="h-7 shrink-0 text-[12px]" disabled={props.busy} onClick={() => props.onChoose(opt.id)}>
                  {s.desk.choose}
                </Button>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </li>
  );
}
