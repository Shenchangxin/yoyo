import { useEffect, useState } from "react";
import { Film, ImagePlus } from "lucide-react";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { Input, Textarea } from "../../components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { DramaMentionField, type MentionAsset } from "./DramaMentionField";
import {
  assetName,
  catalogOf,
  isNarrator,
  jobMoving,
  linkedAssets,
  providerKey,
  selectedClipIds,
  serviceKind,
  type Asset,
  type Bundle,
  type Episode,
  type EpisodePlan,
  type Job,
  type Shot,
} from "./drama-lib";

type CopyT = ReturnType<typeof useCopy>;

const field =
  "no-drag h-8 rounded-lg border border-border bg-background px-2 text-[13px] text-foreground outline-none focus-visible:border-foreground/25 focus-visible:ring-1 focus-visible:ring-foreground/15";

function shotMentions(s: Shot, bundle: Bundle): MentionAsset[] {
  const out: MentionAsset[] = [];
  const chars = new Set(s.character_ids || []);
  const props = new Set(s.prop_ids || []);
  const scene = s.scene_id || "";
  for (const c of bundle.characters || []) if (c.name && (chars.size === 0 || chars.has(c.id))) out.push({ name: c.name, kind: "character" });
  for (const sc of bundle.scenes || []) if (sc.location && (!scene || sc.id === scene)) out.push({ name: sc.location, kind: "scene" });
  for (const p of bundle.props || []) if (p.name && (props.size === 0 || props.has(p.id))) out.push({ name: p.name, kind: "prop" });
  if (out.length === 0) {
    for (const c of bundle.characters || []) if (c.name) out.push({ name: c.name, kind: "character" });
    for (const sc of bundle.scenes || []) if (sc.location) out.push({ name: sc.location, kind: "scene" });
    for (const p of bundle.props || []) if (p.name) out.push({ name: p.name, kind: "prop" });
  }
  return out.sort((a, b) => b.name.length - a.name.length);
}

export function ProviderModelSelect(props: {
  kind: string;
  label: string;
  providers: any[];
  providerId: string;
  model: string;
  onChange: (providerId: string, model: string) => void;
}) {
  const copy = useCopy();
  const openSettings = useUI((s) => s.openSettings);
  const want = serviceKind(props.kind);
  const settingSection = want === "tts" ? "generation-speech" : `generation-${want}`;
  const rows = props.providers.filter((p) => {
    if (serviceKind(p.service_type ?? p.serviceType) !== want) return false;
    const selected = providerKey(p.id) === providerKey(props.providerId) && providerKey(p.id) !== "";
    return p.is_active !== false || p.has_key || selected;
  });
  if (rows.length === 0) {
    return (
      <button
        type="button"
        className={cn(field, "h-7 max-w-[14rem] text-left text-muted")}
        onClick={() => openSettings("generation", settingSection)}
      >
        {props.label} · {copy.video.openSettings}
      </button>
    );
  }
  const current = rows.find((p) => providerKey(p.id) === providerKey(props.providerId)) || rows.find((p) => p.is_default) || rows[0];
  const models = catalogOf(current, props.model);
  const model = props.model && models.includes(props.model) ? props.model : (current?.model || models[0] || "");
  const value = current ? `${current.id}::${model}` : "";
  return (
    <Select
      value={value || undefined}
      onValueChange={(raw) => {
        const i = raw.indexOf("::");
        if (i < 0) {
          props.onChange("", "");
          return;
        }
        props.onChange(raw.slice(0, i), raw.slice(i + 2));
      }}
    >
      <SelectTrigger className={cn(field, "h-7 max-w-[14rem]")} aria-label={props.label}>
        <SelectValue placeholder={props.label} />
      </SelectTrigger>
      <SelectContent>
        {rows.map((p) => {
          const models = catalogOf(p, providerKey(p.id) === providerKey(props.providerId) ? props.model : "");
          const ids = models.length ? models : [""];
          return ids.map((m) => (
            <SelectItem key={`${p.id}::${m}`} value={`${p.id}::${m}`}>
              {`${p.name || p.provider}${m ? ` · ${m}` : ""}`}
            </SelectItem>
          ));
        })}
      </SelectContent>
    </Select>
  );
}

export function GenerationSettings(props: {
  ep: Episode;
  dramaStyle: string;
  styles: { value: string; name: string }[];
  providers: any[];
  copy: CopyT;
  onStyle: (style: string) => void;
  onEpisode: (patch: Record<string, string>) => void;
}) {
  const c = props.copy.video;
  const ep = props.ep;
  return (
    <details className="drama-gen-fold mt-3">
      <summary className="text-[11px] font-medium text-muted">{c.genSettings}</summary>
      <div className="mt-2 grid gap-1.5">
        <select className={cn(field, "h-7")} value={props.dramaStyle || "3d"} aria-label={c.style} onChange={(e) => props.onStyle(e.target.value)}>
          {props.styles.map((s) => <option key={s.value} value={s.value}>{s.name}</option>)}
        </select>
        <ProviderModelSelect kind="image" label={c.imageProvider} providers={props.providers} providerId={ep.image_provider_id || ""} model={ep.image_model || ""} onChange={(id, model) => props.onEpisode({ image_provider_id: id, image_model: model })} />
        <ProviderModelSelect kind="video" label={c.videoProvider} providers={props.providers} providerId={ep.video_provider_id || ""} model={ep.video_model || ""} onChange={(id, model) => props.onEpisode({ video_provider_id: id, video_model: model })} />
        <ProviderModelSelect kind="tts" label={c.kindSpeech} providers={props.providers} providerId={ep.tts_provider_id || ""} model={ep.tts_model || ""} onChange={(id, model) => props.onEpisode({ tts_provider_id: id, tts_model: model })} />
        <select className={cn(field, "h-7")} value={ep.resolution || "720p"} aria-label={c.resolution} onChange={(e) => props.onEpisode({ resolution: e.target.value })}>
          {["480p", "720p", "1080p"].map((r) => <option key={r} value={r}>{r}</option>)}
        </select>
      </div>
    </details>
  );
}

export function ScriptStage(props: { ep: Episode; copy: CopyT; plan?: Bundle["plan"]; onSave: (p: Partial<Episode>) => void }) {
  const [content, setContent] = useState(props.ep.content);
  const [script, setScript] = useState(props.ep.script_content);
  useEffect(() => {
    setContent(props.ep.content);
    setScript(props.ep.script_content);
  }, [props.ep.id, props.ep.content, props.ep.script_content]);
  const c = props.copy.video;
  const plan = props.plan;
  return (
    <div className="drama-script-split">
      {plan ? (
        <p className="drama-script-plan">{plan.chars} {c.chars} · {plan.target_seconds} {c.seconds} · {plan.segment_count} {c.segments}</p>
      ) : null}
      <label className="drama-script-col block">
        <div className="mb-1.5 text-[11px] text-muted">{c.novel}</div>
        <Textarea
          className="min-h-[calc(100%-1.5rem)] rounded-[10px] border border-border bg-card px-3 py-2 text-[13px] leading-6"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          onBlur={() => props.onSave({ content, script_content: script })}
        />
      </label>
      <label className="drama-script-col block">
        <div className="mb-1.5 text-[11px] text-muted">{c.screenplay}</div>
        <Textarea
          className="min-h-[calc(100%-1.5rem)] rounded-[10px] border border-border bg-card px-3 py-2 font-mono text-[13px] leading-6"
          value={script}
          onChange={(e) => setScript(e.target.value)}
          onBlur={() => props.onSave({ content, script_content: script })}
        />
      </label>
    </div>
  );
}

export function CastGrid(props: {
  bundle: Bundle;
  copy: CopyT;
  focusId: string;
  onFocus: (kind: string, id: string) => void;
  onCreate: (kind: string, fields: Record<string, string>) => void;
}) {
  const c = props.copy.video;
  const groups: { kind: string; title: string; items: Asset[]; create: string }[] = [
    { kind: "character", title: c.characters, items: linkedAssets(props.bundle.characters), create: c.addCharacter },
    { kind: "scene", title: c.scenes, items: linkedAssets(props.bundle.scenes), create: c.addScene },
    { kind: "prop", title: c.props, items: linkedAssets(props.bundle.props), create: c.addProp },
  ];
  return (
    <div className="grid gap-5 px-1 py-2">
      {groups.map((g) => (
        <LookGroup key={g.kind} {...g} copy={props.copy} focusId={props.focusId} onFocus={props.onFocus} onCreate={props.onCreate} />
      ))}
    </div>
  );
}

function LookGroup(props: {
  kind: string;
  title: string;
  items: Asset[];
  create: string;
  copy: CopyT;
  focusId: string;
  onFocus: (kind: string, id: string) => void;
  onCreate: (kind: string, fields: Record<string, string>) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState("");
  const c = props.copy.video;
  return (
    <section>
      <div className="mb-2 flex items-baseline justify-between px-3">
        <span className="text-[11px] font-medium text-muted">{props.title}</span>
        <button type="button" className="text-[11px] text-muted hover:text-foreground" onClick={() => setAdding(true)}>{props.create}</button>
      </div>
      {adding ? (
        <form
          className="mb-2 flex gap-2 px-3"
          onSubmit={(e) => {
            e.preventDefault();
            const n = name.trim();
            if (!n) return;
            props.onCreate(props.kind, props.kind === "scene" ? { location: n } : { name: n });
            setName("");
            setAdding(false);
          }}
        >
          <Input className="h-7 text-[12px]" autoFocus value={name} placeholder={c.addPlaceholder} onChange={(e) => setName(e.target.value)} onBlur={() => { if (!name.trim()) setAdding(false); }} />
          <Button size="sm" type="submit">{props.create}</Button>
        </form>
      ) : null}
      {props.items.length === 0 ? (
        <p className="px-3 py-2 text-[12px] text-muted">{c.noStills}</p>
      ) : (
        <div className="drama-look-grid">
          {props.items.map((a) => {
            const narrator = props.kind === "character" && isNarrator(a);
            return (
              <button
                key={a.id}
                type="button"
                className={cn("drama-look-card", props.focusId === a.id && "is-on")}
                onClick={() => props.onFocus(props.kind, a.id)}
              >
                {a.image_url ? <img src={a.image_url} alt="" /> : <span className="drama-look-empty grid place-items-center text-[11px]">{narrator ? c.narrator : "—"}</span>}
                <span className="truncate text-[12px] font-medium">{assetName(a, props.kind)}</span>
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}

export function CastInspector(props: {
  kind: string;
  asset: Asset;
  copy: CopyT;
  onGen: () => void;
  onUpload: (b64: string) => void;
  onSave: (fields: Record<string, string>) => void;
  onDelete: () => void;
}) {
  const c = props.copy.video;
  const narrator = props.kind === "character" && isNarrator(props.asset);
  const a = props.asset;
  return (
    <div className="grid gap-2">
      <div className="text-[13px] font-medium">{assetName(a, props.kind)}</div>
      {narrator ? <p className="text-[12px] text-muted">{c.narrator}</p> : null}
      {props.kind === "character" ? (
        <>
          <Input className="h-7 text-[12px]" defaultValue={a.appearance || ""} placeholder={c.appearance} onBlur={(e) => props.onSave({ appearance: e.target.value })} />
          <Input className="h-7 text-[12px]" defaultValue={a.styling || ""} placeholder={c.styling} onBlur={(e) => props.onSave({ styling: e.target.value })} />
        </>
      ) : null}
      {props.kind === "scene" ? (
        <Input className="h-7 text-[12px]" defaultValue={a.lighting || ""} placeholder={c.lighting} onBlur={(e) => props.onSave({ lighting: e.target.value })} />
      ) : null}
      {props.kind === "prop" ? (
        <Input className="h-7 text-[12px]" defaultValue={a.description || ""} placeholder={c.finalPrompt} onBlur={(e) => props.onSave({ description: e.target.value })} />
      ) : null}
      <Textarea className="min-h-[72px] rounded-lg border border-border bg-card px-2 py-1.5 text-[12px]" defaultValue={a.final_prompt || ""} placeholder={c.finalPrompt} onBlur={(e) => props.onSave({ final_prompt: e.target.value })} />
      <div className="flex flex-wrap items-center gap-1.5">
        <label className="grid size-7 cursor-pointer place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" title={c.upload}>
          <ImagePlus className="size-3.5" />
          <input type="file" accept="image/*" className="hidden" onChange={(e) => {
            const f = e.target.files?.[0];
            if (!f) return;
            const r = new FileReader();
            r.onload = () => props.onUpload(String(r.result || ""));
            r.readAsDataURL(f);
          }} />
        </label>
        {narrator ? null : <Button size="sm" variant="lift" onClick={props.onGen}>{c.generate}</Button>}
        <button type="button" className="ml-auto text-[11px] text-danger" onClick={props.onDelete}>{c.delete}</button>
      </div>
    </div>
  );
}

export function BoardInspector(props: {
  bundle: Bundle;
  shot: Shot;
  copy: CopyT;
  onPatch: (s: Record<string, any>) => void;
  onGen: () => void;
  onApply: (id: string) => void;
}) {
  const c = props.copy.video;
  const s = props.shot;
  const takes = (props.bundle.jobs || []).filter((j) => j.storyboard_id === s.id && j.type === "video" && j.status === "succeeded");
  const hasCast = (s.character_ids || []).length > 0 || !!s.scene_id;
  const mentions = shotMentions(s, props.bundle);
  return (
    <div className="grid gap-2">
      <div className="flex items-center gap-2">
        <span className="font-mono text-[11px] tabular-nums text-muted">{String(s.shot_number).padStart(2, "0")}</span>
        <Input className="h-7 flex-1 text-[13px]" defaultValue={s.title} onBlur={(e) => props.onPatch({ id: s.id, title: e.target.value })} />
        <Input className="h-7 w-14 font-mono text-[12px] tabular-nums" type="number" min={2} max={30} defaultValue={s.duration} aria-label={c.durationLabel} onBlur={(e) => props.onPatch({ id: s.id, duration: Number(e.target.value) || s.duration })} />
        <span className="text-[11px] text-muted">{c.seconds}</span>
      </div>
      <p className="text-[13px] leading-5 text-muted">{s.description}</p>
      {!hasCast ? <p className="text-[12px] text-muted">{c.missingCast}</p> : null}
      <DramaMentionField
        assets={mentions}
        defaultValue={s.video_prompt}
        placeholder={mentions.map((a) => `@${a.name}`).join(" ") || c.mention}
        onBlur={(e) => props.onPatch({ id: s.id, video_prompt: e.target.value })}
      />
      <label className="flex items-center gap-2 text-[12px]">
        <span className="w-16 text-muted">{c.scenes}</span>
        <select className={cn(field, "h-7 flex-1")} value={s.scene_id || ""} onChange={(e) => props.onPatch({ id: s.id, scene_id: e.target.value })}>
          <option value="">{c.unbind}</option>
          {(props.bundle.scenes || []).map((sc) => <option key={sc.id} value={sc.id}>{sc.location}</option>)}
        </select>
      </label>
      <div>
        <div className="mb-1 text-[11px] text-muted">{c.characters}</div>
        <div className="flex flex-wrap gap-1.5">
          {(props.bundle.characters || []).filter((ch) => ch.linked !== false && !isNarrator(ch)).map((ch) => {
            const on = (s.character_ids || []).includes(ch.id);
            return (
              <button
                key={ch.id}
                type="button"
                className={cn("rounded-full px-2 py-0.5 text-[11px]", on ? "bg-foreground text-background" : "bg-lift text-muted")}
                onClick={() => {
                  const next = on ? (s.character_ids || []).filter((id) => id !== ch.id) : [...(s.character_ids || []), ch.id];
                  props.onPatch({ id: s.id, character_ids: next });
                }}
              >{ch.name}</button>
            );
          })}
        </div>
      </div>
      <div>
        <div className="mb-1 text-[11px] text-muted">{c.props}</div>
        <div className="flex flex-wrap gap-1.5">
          {(props.bundle.props || []).filter((p) => p.linked !== false).map((p) => {
            const on = (s.prop_ids || []).includes(p.id);
            return (
              <button
                key={p.id}
                type="button"
                className={cn("rounded-full px-2 py-0.5 text-[11px]", on ? "bg-foreground text-background" : "bg-lift text-muted")}
                onClick={() => {
                  const next = on ? (s.prop_ids || []).filter((id) => id !== p.id) : [...(s.prop_ids || []), p.id];
                  props.onPatch({ id: s.id, prop_ids: next });
                }}
              >{p.name}</button>
            );
          })}
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {s.video_url ? <a className="text-[11px] underline" href={s.video_url} download>{c.download}</a> : null}
        <Button size="sm" variant="lift" onClick={props.onGen}>{c.anotherTake}</Button>
      </div>
      <div>
        <div className="mb-1 text-[11px] text-muted">{c.history}</div>
        {takes.length === 0 ? <p className="text-[11px] text-muted">{c.noHistory}</p> : (
          <div className="flex flex-wrap gap-2">
            {takes.map((j) => (
              <button key={j.id} type="button" className="w-[72px] text-left" onClick={() => props.onApply(j.id)}>
                {j.poster_url ? <img src={j.poster_url} alt="" className="h-12 w-full rounded object-cover" /> : <div className="h-12 rounded bg-lift" />}
                <span className="text-[10px] text-muted">{c.setMain}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export function CutInspector(props: {
  bundle: Bundle;
  sel: Record<string, boolean>;
  setSel: (v: Record<string, boolean>) => void;
  copy: CopyT;
  ffmpeg: boolean;
  ffmpegInstall?: { phase?: string; percent?: number; error?: string };
  onInstall: () => void;
}) {
  const c = props.copy.video;
  const shots = props.bundle.shots || [];
  const n = selectedClipIds(shots, props.sel).length;
  const phase = String(props.ffmpegInstall?.phase || "");
  const installing = phase === "resolving" || phase === "downloading" || phase === "extracting" || phase === "verifying";
  return (
    <div>
      {!props.ffmpeg ? (
        <p className="mb-3 text-[13px] text-danger">
          {installing ? `${c.ffmpegInstalling} ${Math.max(0, Math.min(100, Math.round(Number(props.ffmpegInstall?.percent) || 0)))}%` : phase === "error" ? c.ffmpegInstallFailed : c.ffmpegMissing}
        </p>
      ) : null}
      {installing ? <ProgressHairline className="mb-3" value={Number(props.ffmpegInstall?.percent) || 0} /> : null}
      <div className="mb-3 flex flex-wrap items-center gap-2 text-[12px] text-muted">
        <span className="tabular-nums">{n} {c.clipCount}</span>
        <button type="button" className="underline" onClick={() => {
          const next: Record<string, boolean> = {};
          for (const s of shots) next[s.id] = !!s.video_url;
          props.setSel(next);
        }}>{c.selectAll}</button>
        <button type="button" className="underline" onClick={() => {
          const next: Record<string, boolean> = {};
          for (const s of shots) next[s.id] = false;
          props.setSel(next);
        }}>{c.selectNone}</button>
      </div>
    </div>
  );
}

export function JobDrawer(props: { jobs: Job[]; bundle: Bundle | null; episodes?: { id: string; title: string; episode_number?: number }[]; copy: CopyT; onRetry: (id: string) => void; onCancel: (id: string) => void }) {
  const live = props.jobs.filter((j) => jobMoving(j.status) || j.status === "failed");
  if (!live.length) return null;
  const c = props.copy.video;
  const label: Record<string, string> = { queued: c.queued, running: c.running, polling: c.polling, succeeded: c.succeeded, failed: c.failed };
  const groups: { key: string; caption: string; jobs: Job[] }[] = [];
  const seen = new Map<string, number>();
  for (const j of live) {
        const key = `${j.episode_id || ""}:${j.storyboard_id || j.character_id || j.scene_id || j.prop_id || j.id}`;
    const at = seen.get(key);
    if (at === undefined) {
      seen.set(key, groups.length);
      groups.push({ key, caption: jobCaption(j, props.bundle, props.episodes, c), jobs: [j] });
    } else {
      groups[at].jobs.push(j);
    }
  }
  return (
    <div className="drama-task-list">
      {groups.map((g) => (
        <div key={g.key} className="drama-task-group">
          <div className="drama-task-group-head">{g.caption}</div>
          {g.jobs.map((j) => {
            const moving = jobMoving(j.status);
            return (
              <div key={j.id} className="relative flex items-center gap-2 rounded-lg bg-lift px-2 py-1.5 pb-2 text-[11px]">
                {moving ? <span className="pulse-dot" /> : null}
                <span className={j.status === "failed" ? "text-danger" : "text-muted"}>{label[j.status] || j.status}</span>
                {j.error ? <span className="min-w-0 truncate text-danger">{j.error}</span> : null}
                {j.status === "failed" ? <button type="button" className="underline" onClick={() => props.onRetry(j.id)}>{c.retry}</button> : null}
                {moving ? <button type="button" className="underline" onClick={() => props.onCancel(j.id)}>{c.cancel}</button> : null}
                {moving ? <ProgressHairline className="absolute inset-x-0 bottom-0 rounded-none" indeterminate /> : null}
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
}

function jobCaption(j: Job, bundle: Bundle | null, episodes: { id: string; title: string; episode_number?: number }[] | undefined, c: CopyT["video"]) {
  const ep = (episodes || []).find((e) => e.id === j.episode_id);
  const epLabel = ep && ep.id !== bundle?.episode?.id ? `${ep.title || c.untitledEp} · ` : "";
  const shot = (bundle?.shots || []).find((s) => s.id === j.storyboard_id);
  if (shot) return `${epLabel}${c.jobClip} · ${String(shot.shot_number).padStart(2, "0")}`;
  const ch = (bundle?.characters || []).find((a) => a.id === j.character_id);
  if (ch) return `${epLabel}${c.jobLook} · ${ch.name}`;
  const sc = (bundle?.scenes || []).find((a) => a.id === j.scene_id);
  if (sc) return `${epLabel}${c.jobLook} · ${sc.location}`;
  const p = (bundle?.props || []).find((a) => a.id === j.prop_id);
  if (p) return `${epLabel}${c.jobLook} · ${p.name}`;
  if (epLabel) return `${epLabel}${j.type === "video" ? c.jobClip : c.jobLook}`;
  return j.type === "video" ? c.jobClip : c.jobLook;
}

export function EmptyDesk(props: {
  copy: CopyT;
  onNameSeries: () => void;
}) {
  const c = props.copy.video;
  return (
    <EmptyState
      className="flex-1"
      icon={<Film className="size-5" />}
      title={c.empty}
      body={c.emptyChatFirst}
      action={
        <Button type="button" variant="ghost" onClick={props.onNameSeries}>
          {c.nameFirst}
        </Button>
      }
    />
  );
}

export function EpisodeMap(props: {
  plan: EpisodePlan;
  copy: CopyT;
  busy?: boolean;
  onConfirm: () => void;
  onSplit: (n: number) => void;
  onMerge: (n: number) => void;
  onBible: () => void;
}) {
  const c = props.copy.video;
  const rows = props.plan.episodes || [];
  return (
    <div className="drama-map" data-testid="drama-episode-map">
      <div className="drama-map-head">
        <div>
          <h2>{c.mapTitle}</h2>
          <p>{c.mapHint}</p>
        </div>
        <div className="drama-map-actions">
          <Button type="button" variant="ghost" size="sm" disabled={!!props.busy} onClick={props.onBible}>{c.mapBible}</Button>
          <Button type="button" size="sm" data-testid="drama-map-confirm" disabled={!!props.busy || rows.length === 0} onClick={props.onConfirm}>{c.mapConfirm}</Button>
        </div>
      </div>
      {rows.length === 0 ? (
        <p className="drama-map-empty">{c.mapEmpty}</p>
      ) : (
        <ol className="drama-map-list">
          {rows.map((row, i) => (
            <li key={`${row.n}-${row.start}`} className="drama-map-row">
              <div className="drama-map-n">{String(row.n).padStart(2, "0")}</div>
              <div className="drama-map-copy">
                <strong>{row.title || c.untitledEp}</strong>
                {row.logline ? <span>{row.logline}</span> : null}
                {row.hook ? <em>{c.mapHook} · {row.hook}</em> : null}
              </div>
              <div className="drama-map-meta">
                <span>{row.target_seconds || 0}{c.seconds}</span>
                <span>{Math.max(0, (row.end || 0) - (row.start || 0))}{c.chars}</span>
              </div>
              <div className="drama-map-row-actions">
                <button type="button" disabled={!!props.busy} onClick={() => props.onSplit(row.n)}>{c.mapSplit}</button>
                {i < rows.length - 1 ? <button type="button" disabled={!!props.busy} onClick={() => props.onMerge(row.n)}>{c.mapMerge}</button> : null}
              </div>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
