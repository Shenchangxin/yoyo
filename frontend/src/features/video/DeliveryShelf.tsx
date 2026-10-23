import type { ReactNode } from "react";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { filmInFlight } from "./studio-topic";
import type { StudioCatalog, StudioProject } from "./studio-model";
import { Clapperboard } from "lucide-react";

const STEPS = ["script", "voice", "visuals", "compose"] as const;

type Platform = { id: string; label?: string };

export function DeliveryShelf(props: {
  films: StudioProject[];
  current: StudioProject | null;
  catalog: StudioCatalog;
  busy: string;
  platforms: Platform[];
  platform: string;
  suggested?: string;
  caption: string;
  tags: string;
  onSelect: (id: string) => void;
  onPlatform: (id: string) => void;
  onCaption: (value: string) => void;
  onTags: (value: string) => void;
  onSave: () => void;
  onRetry: () => void;
  onEdit: () => void;
  onPack: () => void;
  onBack: () => void;
  mediaSrc: (hash?: string) => string;
  drafts?: ReactNode;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const d = s.desk;
  const cur = props.current;
  if (!cur) {
    return (
      <EmptyState
        icon={<Clapperboard className="size-5" />}
        title={s.filmEmpty}
        body={s.filmEmptyHint}
        action={<Button size="sm" variant="ghost" onClick={props.onBack}>{s.topics}</Button>}
      />
    );
  }
  const phase = cur.auto_phase || "";
  const live = filmInFlight(phase);
  const failed = phase === "failed" && !cur.result_hash;
  const ready = !!cur.result_hash && !live;
  const videoSrc = cur.media_url || props.mediaSrc(cur.result_hash);
  const poster = cur.cover_url || cur.poster_url || props.mediaSrc(cur.cover_hash || cur.poster_hash);
  const plan = filmPlan(cur, props.catalog, s.filmSeconds);
  const portrait = (cur.aspect || "9:16") !== "16:9";
  const square = cur.aspect === "1:1";
  const publishable = cur.delivery?.status === "publishable";
  const stepAt = STEPS.indexOf(phase as (typeof STEPS)[number]);

  return (
    <div className="flex flex-col gap-4" data-testid="studio-delivery-shelf">
      {props.films.length > 1 ? (
        <div className="flex flex-col gap-2">
          <p className="text-[12px] text-muted">{s.filmShelf}</p>
          <ul className="flex gap-2 overflow-x-auto pb-1">
            {props.films.map((film) => {
              const on = film.id === cur.id;
              const thumb = film.cover_url || film.poster_url || props.mediaSrc(film.cover_hash || film.poster_hash || film.result_hash);
              const label = phaseText(s, d, film.auto_phase, !!film.result_hash);
              return (
                <li key={film.id} className="w-[6.75rem] shrink-0">
                  <button
                    type="button"
                    data-testid={`studio-delivery-film-${film.id}`}
                    aria-pressed={on}
                    className={cn(
                      "w-full rounded-xl border bg-card p-1.5 text-left",
                      on ? "border-foreground" : "border-border hover:border-foreground/40",
                    )}
                    onClick={() => props.onSelect(film.id)}
                  >
                    <span className={cn("relative block overflow-hidden rounded-lg bg-black", film.aspect === "16:9" ? "aspect-video" : film.aspect === "1:1" ? "aspect-square" : "aspect-[9/16]")}>
                      {thumb ? <img src={thumb} alt="" className="h-full w-full object-cover" /> : (
                        <span className="grid h-full place-items-center px-1 text-center text-[10px] leading-[1.35] text-white/70">{label}</span>
                      )}
                    </span>
                    <span className="mt-1.5 block line-clamp-2 text-[12px] font-medium leading-[1.35] text-foreground">{film.title}</span>
                    <span className="mt-0.5 block truncate text-[11px] text-muted">{label}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        </div>
      ) : null}

      <section className="flex flex-col gap-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="truncate text-[15px] font-medium text-foreground">{cur.title}</h2>
            {plan ? <p className="mt-1 text-[12px] leading-[1.45] text-muted">{s.filmPlan.replace("{plan}", plan)}</p> : null}
          </div>
          <Button size="sm" variant="ghost" className="h-7 shrink-0 text-[12px]" onClick={props.onEdit}>{s.filmEdit}</Button>
        </div>

        <div
          className={cn(
            "relative overflow-hidden rounded-xl bg-black",
            square ? "mx-auto aspect-square w-full max-w-[20rem]" : portrait ? "mx-auto aspect-[9/16] w-full max-w-[17.5rem]" : "aspect-video w-full",
          )}
        >
          {live ? <ProgressHairline className="absolute inset-x-0 top-0 z-10" indeterminate /> : null}
          {ready && videoSrc ? (
            <video
              key={cur.result_hash || cur.id}
              className="h-full w-full object-contain"
              src={videoSrc}
              poster={poster || undefined}
              controls
              playsInline
              data-testid="studio-delivery-video"
            />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 px-5 text-center">
              <p className="text-[13px] font-medium text-white">{phaseText(s, d, phase, ready)}</p>
              {ready || failed ? null : <p className="text-[12px] leading-[1.45] text-white/65">{phase === "queued" ? s.filmQueued : s.filmWorking}</p>}
            </div>
          )}
        </div>

        {live && phase !== "queued" ? (
          <ol className="flex flex-wrap gap-x-3 gap-y-1">
            {STEPS.map((id, i) => {
              const on = phase === id;
              const past = stepAt > i;
              const label = id === "script" ? d.phaseScript : id === "voice" ? d.phaseVoice : id === "visuals" ? d.phaseVisuals : d.phaseCompose;
              return (
                <li key={id} className={cn("inline-flex items-center gap-1.5 text-[12px]", on ? "font-medium text-foreground" : past ? "text-foreground" : "text-muted")}>
                  <span className={cn("size-1.5 rounded-full", on || past ? "bg-foreground" : "bg-border")} />
                  {label}
                </li>
              );
            })}
          </ol>
        ) : null}

        {failed ? (
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-danger/30 bg-danger/5 px-3 py-2.5">
            <p className="min-w-0 text-[12px] leading-[1.45] text-danger">{cur.auto_error || d.phaseFailed}</p>
            <Button size="sm" variant="default" className="h-7 text-[12px]" disabled={!!props.busy} onClick={props.onRetry}>{s.filmRetry}</Button>
          </div>
        ) : null}
      </section>

      {ready ? (
        <section className="flex flex-col gap-3 border-t border-border/80 pt-4" data-testid="studio-delivery-post">
          <div>
            <h3 className="text-[13px] font-medium text-foreground">{s.filmPost}</h3>
            <p className="mt-1 text-[12px] leading-[1.45] text-muted">{publishable ? s.filmReady : (cur.delivery?.note || d.draftPack)}</p>
          </div>
          {!publishable ? (
            <Button size="sm" variant="default" className="h-7 self-start text-[12px]" disabled={!!props.busy} onClick={props.onPack}>{d.package}</Button>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-1.5">
                {props.platforms.map((p) => {
                  const on = props.platform === p.id;
                  return (
                    <button
                      key={p.id}
                      type="button"
                      data-testid={`studio-delivery-platform-${p.id}`}
                      aria-pressed={on}
                      className={cn(
                        "inline-flex h-8 items-center gap-1 rounded-full px-3 text-[12px]",
                        on ? "bg-foreground text-background" : "border border-border bg-card text-foreground hover:bg-lift/70",
                      )}
                      onClick={() => props.onPlatform(p.id)}
                    >
                      {p.label || p.id}
                      {props.suggested === p.id ? <span className={cn("text-[10px]", on ? "text-background/70" : "text-muted")}>{s.filmSuggest}</span> : null}
                    </button>
                  );
                })}
              </div>
              <Input className="h-8" placeholder={s.caption} value={props.caption} onChange={(e) => props.onCaption(e.target.value)} />
              <Input className="h-8" placeholder={s.tags} value={props.tags} onChange={(e) => props.onTags(e.target.value)} />
              <div className="flex flex-wrap items-center gap-2">
                <Button size="sm" variant="default" className="h-8 text-[12px]" disabled={!!props.busy || !props.platform} title={props.platform ? undefined : s.filmPickPlatform} onClick={props.onSave}>{s.draftPublish}</Button>
                {!props.platform ? <p className="text-[12px] text-muted">{s.filmPickPlatform}</p> : null}
              </div>
              <p className="text-[12px] leading-[1.45] text-muted">{s.neverPublish}</p>
            </>
          )}
        </section>
      ) : null}

      {props.drafts ? (
        <section className="flex flex-col gap-2 border-t border-border/80 pt-4">
          <h3 className="text-[13px] font-medium text-foreground">{s.filmDrafts}</h3>
          {props.drafts}
        </section>
      ) : ready ? (
        <p className="text-[12px] text-muted">{s.filmNoDrafts}</p>
      ) : null}
    </div>
  );
}

function filmPlan(p: StudioProject, catalog: StudioCatalog, seconds: string) {
  const tpl = catalog.templates?.find((t) => t.id === p.template_id)?.title;
  const length = p.target_duration ? seconds.replace("{n}", String(p.target_duration)) : "";
  return [tpl, p.aspect, length].filter(Boolean).join(" · ");
}

function phaseText(s: { filmQueued: string }, d: { phaseScript: string; phaseVoice: string; phaseVisuals: string; phaseCompose: string; phaseDone: string; phaseFailed: string; phaseQueued: string }, phase: string | undefined, hasVideo: boolean) {
  if (hasVideo && phase !== "failed" && !filmInFlight(phase)) return d.phaseDone;
  switch (phase) {
    case "queued":
      return d.phaseQueued || s.filmQueued;
    case "script":
      return d.phaseScript;
    case "voice":
      return d.phaseVoice;
    case "visuals":
      return d.phaseVisuals;
    case "compose":
      return d.phaseCompose;
    case "failed":
      return d.phaseFailed;
    case "done":
      return d.phaseDone;
    default:
      return d.phaseQueued;
  }
}
