import { useCallback, useEffect, useState } from "react";
import { Clapperboard, Film, ImagePlus, Plus, Upload } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input, Textarea } from "../../components/ui/input";
import { EmptyState } from "../../components/ui/empty-state";
import { Tooltip } from "../../components/ui/tooltip";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { subscribeItems } from "../../lib/stream";
import { DramaMentionField, type MentionAsset } from "./DramaMentionField";
import { useDramaSelection } from "./workshop-store";

type Drama = {
  id: string;
  title: string;
  style: string;
  aspect_ratio: string;
  episode_count: number;
};
type Episode = {
  id: string;
  drama_id: string;
  title: string;
  content: string;
  script_content: string;
  resolution: string;
  pipeline: string;
  image_provider_id: string;
  video_provider_id: string;
  video_url?: string;
  poster_url?: string;
};
type Asset = {
  id: string;
  name?: string;
  location?: string;
  image_url?: string;
  image_hash?: string;
  linked?: boolean;
  final_prompt?: string;
  appearance?: string;
  prompt?: string;
  description?: string;
};
type Shot = {
  id: string;
  shot_number: number;
  title: string;
  description: string;
  video_prompt: string;
  duration: number;
  status: string;
  video_url?: string;
  poster_url?: string;
  character_ids?: string[];
  scene_id?: string;
  prop_ids?: string[];
};
type Job = { id: string; type: string; status: string; error?: string; episode_id?: string };
type Bundle = {
  drama: Drama;
  episode: Episode;
  characters: Asset[];
  scenes: Asset[];
  props: Asset[];
  shots: Shot[];
  jobs: Job[];
  plan: { chars: number; target_seconds: number; segment_count: number };
  status?: { ffmpeg?: boolean; media_base?: string };
};
type Pane = "script" | "cast" | "board" | "cut";
type CopyT = ReturnType<typeof useCopy>;

const field =
  "no-drag h-8 rounded-lg border border-border bg-background px-2 text-[12.5px] text-foreground outline-none focus-visible:border-foreground/25 focus-visible:ring-1 focus-visible:ring-foreground/15";

function pipeStatus(raw: string, stage: string) {
  try {
    const m = JSON.parse(raw || "{}");
    return String(m?.[stage]?.status || "");
  } catch {
    return "";
  }
}

function selectedClipIds(shots: Shot[], sel: Record<string, boolean>) {
  return shots.filter((s) => s.video_url && sel[s.id] !== false).map((s) => s.id);
}

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

export function DramaStudio(props: { sessionId?: string; onNeedSession: () => void }) {
  const copy = useCopy();
  const dramaId = useDramaSelection((s) => s.dramaId);
  const episodeId = useDramaSelection((s) => s.episodeId);
  const setDramaId = useDramaSelection((s) => s.setDramaId);
  const setEpisodeId = useDramaSelection((s) => s.setEpisodeId);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [bundle, setBundle] = useState<Bundle | null>(null);
  const [pane, setPane] = useState<Pane>("script");
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [selShots, setSelShots] = useState<Record<string, boolean>>({});
  const [styles, setStyles] = useState<{ value: string; name: string }[]>([]);
  const [providers, setProviders] = useState<any[]>([]);
  const [ffmpeg, setFfmpeg] = useState(true);
  const [ratio, setRatio] = useState("16:9");
  const [kill, setKill] = useState("");

  const loadList = useCallback(async () => {
    const list = (await api.video.listDramas()) as Drama[];
    setDramas(Array.isArray(list) ? list : []);
  }, []);

  const loadBundle = useCallback(async (id: string) => {
    if (!id) {
      setBundle(null);
      return;
    }
    const b = (await api.video.bundle(id)) as Bundle;
    setBundle(b);
    setFfmpeg(b?.status?.ffmpeg !== false);
  }, []);

  useEffect(() => {
    void loadList().catch((e) => setErr(api.errMessage(e)));
    void api.video.styles().then((s) => setStyles(Array.isArray(s) ? s : [])).catch(() => {});
    void api.video.providers().then((p) => setProviders(Array.isArray(p) ? p : [])).catch(() => {});
    void api.video.status().then((s) => setFfmpeg(!!s?.ffmpeg)).catch(() => {});
  }, [loadList]);

  useEffect(() => {
    if (!dramaId) {
      setEpisodes([]);
      return;
    }
    void api.video.episodes(dramaId).then((list) => {
      const eps = Array.isArray(list) ? list : [];
      setEpisodes(eps);
      if (!episodeId && eps[0]) setEpisodeId(eps[0].id);
    }).catch((e) => setErr(api.errMessage(e)));
  }, [dramaId]);

  useEffect(() => {
    if (!episodeId) {
      setBundle(null);
      return;
    }
    void loadBundle(episodeId).catch((e) => setErr(api.errMessage(e)));
  }, [episodeId, loadBundle]);

  useEffect(() => subscribeItems((item) => {
    if (item.source !== "video") return;
    const ep = String(item.payload?.episode_id || "");
    if (ep && ep === episodeId) void loadBundle(episodeId);
  }), [episodeId, loadBundle]);

  const liveJobs = (bundle?.jobs || []).some((j) => j.status === "queued" || j.status === "running" || j.status === "polling");
  useEffect(() => {
    if (!episodeId || !liveJobs) return;
    const t = window.setInterval(() => { void loadBundle(episodeId); }, 2000);
    return () => window.clearInterval(t);
  }, [episodeId, liveJobs, loadBundle]);

  useEffect(() => {
    if (!props.sessionId || !episodeId) return;
    void api.video.bind(props.sessionId, episodeId).catch(() => {});
  }, [props.sessionId, episodeId]);

  const drama = dramas.find((d) => d.id === dramaId) || bundle?.drama;
  const ep = bundle?.episode;

  async function run(label: string, fn: () => Promise<any>) {
    setBusy(label);
    setErr("");
    try {
      await fn();
      if (episodeId) await loadBundle(episodeId);
      await loadList();
      if (dramaId) {
        const eps = await api.video.episodes(dramaId);
        setEpisodes(Array.isArray(eps) ? eps : []);
      }
    } catch (e) {
      setErr(api.errMessage(e));
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function stage(name: string) {
    if (!props.sessionId) {
      toast.message(copy.video.needSession);
      props.onNeedSession();
      return;
    }
    if (!episodeId) return;
    await run(name, async () => {
      await api.video.bind(props.sessionId!, episodeId);
      await api.video.stage(props.sessionId!, episodeId, name);
    });
  }

  async function createDrama() {
    await run("create", async () => {
      const d = await api.video.createDrama({ title: copy.video.untitled, style: "3d", aspect_ratio: ratio });
      setDramaId(d.id);
      const created = await api.video.createEpisode(d.id, copy.video.untitledEp, "");
      setEpisodeId(created.id);
      setPane("script");
    });
  }

  async function createEpisode() {
    if (!dramaId) return;
    await run("episode", async () => {
      const created = await api.video.createEpisode(dramaId, copy.video.untitledEp, "");
      setEpisodeId(created.id);
      setPane("script");
    });
  }

  async function importHuobao() {
    const paths = await api.pickFiles();
    const db = paths.find((p) => p.toLowerCase().endsWith(".sqlite") || p.toLowerCase().endsWith(".db")) || paths[0];
    if (!db) return;
    await run("import", async () => {
      await api.video.importHuobao(db, "");
      toast.success(copy.video.import);
    });
  }

  async function remove(kind: "drama" | "episode", id: string) {
    if (kill !== kind + id) {
      setKill(kind + id);
      window.setTimeout(() => setKill((v) => (v === kind + id ? "" : v)), 2500);
      return;
    }
    setKill("");
    await run("delete", async () => {
      if (kind === "drama") {
        await api.video.deleteDrama(id);
        if (dramaId === id) {
          setDramaId("");
          setEpisodeId("");
          setBundle(null);
        }
      } else {
        await api.video.deleteEpisode(id);
        if (episodeId === id) setEpisodeId("");
      }
    });
  }

  const pipe = [
    { id: "rewrite", label: copy.video.rewrite, pane: "script" as Pane },
    { id: "extract", label: copy.video.extract, pane: "cast" as Pane },
    { id: "prompts", label: copy.video.prompts, pane: "cast" as Pane },
    { id: "assets", label: copy.video.stills, pane: "cast" as Pane },
    { id: "storyboard", label: copy.video.storyboard, pane: "board" as Pane },
    { id: "gen", label: copy.video.clips, pane: "board" as Pane },
    { id: "merge", label: copy.video.stitch, pane: "cut" as Pane },
  ];

  const panes: { id: Pane; label: string }[] = [
    { id: "script", label: copy.video.script },
    { id: "cast", label: copy.video.cast },
    { id: "board", label: copy.video.board },
    { id: "cut", label: copy.video.cut },
  ];

  const iconBtn =
    "grid size-7 shrink-0 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30";

  return (
    <div className="flex h-full min-h-0 flex-col">
      {err ? <div className="border-b border-danger/20 bg-danger/[0.11] px-3 py-1.5 text-[12px] text-danger">{err}</div> : null}
      <header className="flex min-h-10 shrink-0 flex-wrap items-center gap-1.5 border-b border-border/70 px-2 py-1.5">
        <select
          className={cn(field, "min-w-0 max-w-[9.5rem] flex-1")}
          value={dramaId}
          aria-label={copy.video.pickSeries}
          onChange={(e) => { setDramaId(e.target.value); setEpisodeId(""); setKill(""); }}
        >
          <option value="">{copy.video.pickSeries}</option>
          {dramas.map((d) => (
            <option key={d.id} value={d.id}>{d.title || copy.video.untitled}</option>
          ))}
        </select>
        <select
          className={cn(field, "min-w-0 max-w-[8.5rem] flex-1")}
          value={episodeId}
          disabled={!dramaId}
          aria-label={copy.video.pickEpisode}
          onChange={(e) => setEpisodeId(e.target.value)}
        >
          <option value="">{copy.video.pickEpisode}</option>
          {episodes.map((e) => (
            <option key={e.id} value={e.id}>{e.title || copy.video.untitledEp}</option>
          ))}
        </select>
        <Tooltip content={copy.video.newDrama}>
          <button type="button" className={iconBtn} aria-label={copy.video.newDrama} onClick={() => void createDrama()}>
            <Plus className="size-3.5" />
          </button>
        </Tooltip>
        <Tooltip content={copy.video.newEpisode}>
          <button type="button" className={iconBtn} aria-label={copy.video.newEpisode} disabled={!dramaId} onClick={() => void createEpisode()}>
            <Film className="size-3.5" />
          </button>
        </Tooltip>
        <div className="flex rounded-md bg-lift p-0.5">
          {["16:9", "9:16"].map((r) => (
            <button
              key={r}
              type="button"
              title={copy.video.locked}
              className={cn("rounded px-1.5 py-1 font-mono text-[10px] tabular-nums", (drama?.aspect_ratio || ratio) === r ? "bg-panel text-foreground" : "text-muted")}
              onClick={() => setRatio(r)}
            >
              {r}
            </button>
          ))}
        </div>
        <Tooltip content={copy.video.importHint}>
          <button type="button" className={iconBtn} aria-label={copy.video.import} onClick={() => void importHuobao()}>
            <Upload className="size-3.5" />
          </button>
        </Tooltip>
      </header>
      {!episodeId || !bundle ? (
        <EmptyState
          className="flex-1"
          icon={<Clapperboard className="size-5" />}
          title={copy.video.empty}
          body={copy.video.emptyHint}
          action={<Button size="sm" onClick={() => void createDrama()}>{copy.video.newDrama}</Button>}
        />
      ) : (
        <>
          <div className="flex shrink-0 flex-wrap items-center gap-1.5 border-b border-border/70 px-2 py-1.5">
            <Input
              className="h-7 max-w-[7.5rem] text-[12.5px] font-medium"
              value={drama?.title || ""}
              aria-label={copy.video.title}
              onChange={(e) => {
                const title = e.target.value;
                setDramas((list) => list.map((d) => d.id === dramaId ? { ...d, title } : d));
              }}
              onBlur={(e) => { if (dramaId) void api.video.updateDrama({ id: dramaId, title: e.target.value }); }}
            />
            <Input
              className="h-7 max-w-[7rem] text-[12px]"
              value={ep?.title || ""}
              aria-label={copy.video.untitledEp}
              onChange={(e) => {
                const title = e.target.value;
                setBundle((b) => b ? { ...b, episode: { ...b.episode, title } } : b);
                setEpisodes((list) => list.map((x) => x.id === ep?.id ? { ...x, title } : x));
              }}
              onBlur={(e) => { if (ep) void api.video.updateEpisode({ id: ep.id, title: e.target.value }); }}
            />
            <select className={cn(field, "h-7")} value={drama?.style || "3d"} aria-label={copy.video.style} onChange={(e) => { if (dramaId) void api.video.updateDrama({ id: dramaId, style: e.target.value }).then(loadList); }}>
              {styles.map((s) => <option key={s.value} value={s.value}>{s.name}</option>)}
            </select>
            {providers.some((p) => p.service_type === "image") ? (
              <select className={cn(field, "h-7 max-w-[7.5rem]")} value={ep?.image_provider_id || ""} aria-label={copy.video.imageProvider} onChange={(e) => { if (ep) void api.video.updateEpisode({ id: ep.id, image_provider_id: e.target.value }).then(() => loadBundle(ep.id)); }}>
                <option value="">{copy.video.imageProvider}</option>
                {providers.filter((p) => p.service_type === "image").map((p: any) => <option key={p.id} value={p.id}>{p.name}{p.has_key ? "" : " · —"}</option>)}
              </select>
            ) : null}
            {providers.some((p) => p.service_type === "video") ? (
              <select className={cn(field, "h-7 max-w-[7.5rem]")} value={ep?.video_provider_id || ""} aria-label={copy.video.videoProvider} onChange={(e) => { if (ep) void api.video.updateEpisode({ id: ep.id, video_provider_id: e.target.value }).then(() => loadBundle(ep.id)); }}>
                <option value="">{copy.video.videoProvider}</option>
                {providers.filter((p) => p.service_type === "video").map((p: any) => <option key={p.id} value={p.id}>{p.name}{p.has_key ? "" : " · —"}</option>)}
              </select>
            ) : null}
            <select className={cn(field, "h-7")} value={ep?.resolution || "720p"} aria-label={copy.video.resolution} onChange={(e) => { if (ep) void api.video.updateEpisode({ id: ep.id, resolution: e.target.value }).then(() => loadBundle(ep.id)); }}>
              {["480p", "720p", "1080p"].map((r) => <option key={r} value={r}>{r}</option>)}
            </select>
            <button
              type="button"
              className={cn("ml-auto h-6 rounded-md px-2 text-[11px] font-medium", kill === "episode" + episodeId ? "text-danger" : "text-muted hover:bg-lift hover:text-foreground")}
              onClick={() => void remove("episode", episodeId)}
            >
              {kill === "episode" + episodeId ? copy.video.confirmDelete : copy.video.delete}
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-1 border-b border-border/70 px-2 py-1.5">
            {pipe.map((s) => {
              const st = pipeStatus(ep?.pipeline || "", s.id);
              const live = st === "running" || busy === s.id;
              return (
                <button
                  key={s.id}
                  type="button"
                  disabled={!!busy}
                  onClick={() => {
                    setPane(s.pane);
                    if (s.id === "assets") void run("stills", () => api.video.generateMissingAssets(episodeId));
                    else if (s.id === "gen") void run("clips", () => api.video.generateMissingShots(episodeId));
                    else if (s.id === "merge") setPane("cut");
                    else void stage(s.id === "prompts" ? "prompts" : s.id);
                  }}
                  className={cn(
                    "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium transition-colors",
                    st === "done" ? "bg-lift text-foreground" : live ? "bg-accent/[0.11] text-foreground" : "text-muted hover:bg-lift/60 hover:text-foreground",
                  )}
                >
                  {live ? <span className="pulse-dot" /> : <span className={cn("size-1.5 rounded-full", st === "done" ? "bg-success" : "bg-muted/50")} />}
                  {s.label}
                </button>
              );
            })}
            {busy ? <span className="ml-1 text-[11px] text-muted">{copy.video.stageBusy}</span> : null}
          </div>
          <div className="flex h-9 shrink-0 items-center border-b border-border/70 px-2">
            <div className="process-tabs flex h-9 items-stretch gap-0.5" role="tablist" aria-label={copy.video.workshop}>
              {panes.map((p) => (
                <button
                  key={p.id}
                  type="button"
                  role="tab"
                  aria-selected={pane === p.id}
                  className={cn(
                    "relative flex h-full cursor-pointer items-center px-2 text-[12px] font-medium transition-colors",
                    pane === p.id ? "text-foreground" : "text-muted hover:text-foreground",
                  )}
                  onClick={() => setPane(p.id)}
                >
                  {p.label}
                  <span className={cn("absolute inset-x-2 -bottom-px h-[1.5px] rounded-full bg-foreground transition-opacity duration-150", pane === p.id ? "opacity-100" : "opacity-0")} aria-hidden />
                </button>
              ))}
            </div>
          </div>
          <div className="min-h-0 flex-1 overflow-auto px-3 py-2.5">
            {pane === "script" ? <ScriptPane ep={ep!} plan={bundle.plan} copy={copy} onSave={(patch) => void run("save", () => api.video.updateEpisode({ id: ep!.id, ...patch }))} onRewrite={() => void stage("rewrite")} /> : null}
            {pane === "cast" ? <CastPane bundle={bundle} copy={copy} onExtract={() => void stage("extract")} onPrompts={() => void stage("prompts")} onStills={() => void run("stills", () => api.video.generateMissingAssets(episodeId))} onGen={(kind, id) => void run("gen", () => api.video.generateAsset(kind, id, episodeId))} onUpload={(kind, id, b64) => void run("up", () => api.video.uploadAsset(kind, id, b64))} /> : null}
            {pane === "board" ? <BoardPane bundle={bundle} copy={copy} onBoard={() => void stage("storyboard")} onVprompts={() => void stage("video_prompts")} onGenAll={() => void run("clips", () => api.video.generateMissingShots(episodeId))} onGen={(id) => void run("clip", () => api.video.generateShot(id))} onPatch={(s) => void run("shot", () => api.video.updateShot(s))} /> : null}
            {pane === "cut" ? <CutPane bundle={bundle} sel={selShots} setSel={setSelShots} copy={copy} ffmpeg={ffmpeg} onMerge={() => {
              const ids = selectedClipIds(bundle.shots || [], selShots);
              if (!ids.length) {
                toast.message(copy.video.needClips);
                return;
              }
              void run("merge", () => api.video.merge(episodeId, ids));
            }} /> : null}
          </div>
          <JobStrip jobs={bundle.jobs || []} copy={copy} onRetry={(id) => void run("retry", () => api.video.retryJob(id))} onCancel={(id) => void run("cancel", () => api.video.cancelJob(id))} />
        </>
      )}
    </div>
  );
}

function ScriptPane({ ep, plan, copy, onSave, onRewrite }: { ep: Episode; plan: Bundle["plan"]; copy: CopyT; onSave: (p: Partial<Episode>) => void; onRewrite: () => void }) {
  const [content, setContent] = useState(ep.content);
  const [script, setScript] = useState(ep.script_content);
  useEffect(() => { setContent(ep.content); setScript(ep.script_content); }, [ep.id, ep.content, ep.script_content]);
  return (
    <div className="grid gap-3">
      <label className="block">
        <div className="mb-1.5 text-[11px] text-muted">{copy.video.novel}</div>
        <Textarea className="min-h-[160px] rounded-[10px] border border-border bg-card px-3 py-2 text-[13px] leading-5" value={content} onChange={(e) => setContent(e.target.value)} onBlur={() => onSave({ content, script_content: script })} />
      </label>
      <label className="block">
        <div className="mb-1.5 flex items-center justify-between text-[11px] text-muted">
          <span>{copy.video.screenplay}</span>
          <Button size="sm" onClick={onRewrite}>{copy.video.rewrite}</Button>
        </div>
        <Textarea className="min-h-[160px] rounded-[10px] border border-border bg-card px-3 py-2 font-mono text-[12.5px] leading-5" value={script} onChange={(e) => setScript(e.target.value)} onBlur={() => onSave({ content, script_content: script })} />
      </label>
      <div className="flex flex-wrap items-center gap-2 text-[12px] text-muted">
        <span className="rounded-md bg-lift px-2 py-1">{copy.video.duration}</span>
        <span className="tabular-nums">{plan.chars} {copy.video.chars}</span>
        <span className="tabular-nums">{plan.target_seconds}{copy.video.seconds}</span>
        <span className="tabular-nums">{plan.segment_count} {copy.video.segments}</span>
      </div>
    </div>
  );
}

function CastPane(props: {
  bundle: Bundle; copy: CopyT;
  onExtract: () => void; onPrompts: () => void; onStills: () => void;
  onGen: (kind: string, id: string) => void; onUpload: (kind: string, id: string, b64: string) => void;
}) {
  const c = props.copy.video;
  return (
    <div>
      <div className="mb-3 flex flex-wrap gap-2">
        <Button size="sm" onClick={props.onExtract}>{c.extract}</Button>
        <Button size="sm" variant="lift" onClick={props.onPrompts}>{c.prompts}</Button>
        <Button size="sm" variant="lift" onClick={props.onStills}>{c.generateAll}</Button>
      </div>
      <div className="grid gap-4">
        <AssetCol title={c.characters} items={props.bundle.characters} name={(a) => a.name || ""} kind="character" copy={props.copy} onGen={props.onGen} onUpload={props.onUpload} />
        <AssetCol title={c.scenes} items={props.bundle.scenes} name={(a) => a.location || ""} kind="scene" copy={props.copy} onGen={props.onGen} onUpload={props.onUpload} />
        <AssetCol title={c.props} items={props.bundle.props} name={(a) => a.name || ""} kind="prop" copy={props.copy} onGen={props.onGen} onUpload={props.onUpload} />
      </div>
    </div>
  );
}

function AssetCol(props: {
  title: string; items: Asset[]; name: (a: Asset) => string; kind: string;
  copy: CopyT; onGen: (kind: string, id: string) => void; onUpload: (kind: string, id: string, b64: string) => void;
}) {
  const rows = props.items.filter((a) => a.linked !== false);
  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between px-0.5">
        <span className="text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80">{props.title}</span>
        <span className="font-mono text-[11px] tabular-nums text-muted/70">{rows.length}</span>
      </div>
      {rows.length === 0 ? (
        <p className="px-0.5 py-3 text-[12px] text-muted">{props.copy.video.noStills}</p>
      ) : (
        <ul>
          {rows.map((a, i) => (
            <li key={a.id} className={cn("flex items-center gap-2 py-2", i ? "border-t border-border/50" : "")}>
              {a.image_url ? (
                <img src={a.image_url} alt="" className="size-10 shrink-0 rounded-md object-cover" />
              ) : (
                <div className="grid size-10 shrink-0 place-items-center rounded-md bg-lift text-[10px] text-muted">—</div>
              )}
              <div className="min-w-0 flex-1">
                <div className="truncate text-[12.5px] font-medium">{props.name(a)}</div>
                <div className="truncate text-[11px] text-muted">{a.final_prompt || a.appearance || a.description || a.prompt || ""}</div>
              </div>
              <label className="grid size-7 cursor-pointer place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" title={props.copy.video.upload}>
                <ImagePlus className="size-3.5" />
                <input type="file" accept="image/*" className="hidden" onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (!f) return;
                  const r = new FileReader();
                  r.onload = () => props.onUpload(props.kind, a.id, String(r.result || ""));
                  r.readAsDataURL(f);
                }} />
              </label>
              <Button size="sm" variant="ghost" onClick={() => props.onGen(props.kind, a.id)}>{props.copy.video.generate}</Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function BoardPane(props: {
  bundle: Bundle; copy: CopyT;
  onBoard: () => void; onVprompts: () => void; onGenAll: () => void; onGen: (id: string) => void; onPatch: (s: Partial<Shot> & { id: string }) => void;
}) {
  const c = props.copy.video;
  const shots = props.bundle.shots || [];
  return (
    <div>
      <div className="mb-3 flex flex-wrap gap-2">
        <Button size="sm" onClick={props.onBoard}>{c.storyboard}</Button>
        <Button size="sm" variant="lift" onClick={props.onVprompts}>{c.vprompts}</Button>
        <Button size="sm" variant="lift" onClick={props.onGenAll}>{c.generateAll}</Button>
      </div>
      {shots.length === 0 ? (
        <EmptyState icon={<Film className="size-4" />} title={c.noClip} body={c.emptyHint} />
      ) : (
        <div className="space-y-0">
          {shots.map((s, i) => (
            <article key={s.id} className={cn("grid gap-3 py-3", i ? "border-t border-border/50" : "")}>
              {s.video_url ? (
                <video src={s.video_url} poster={s.poster_url} controls className="aspect-video w-full rounded-lg bg-background" />
              ) : s.poster_url ? (
                <img src={s.poster_url} alt="" className="aspect-video w-full rounded-lg object-cover" />
              ) : (
                <div className="grid aspect-video place-items-center rounded-lg bg-lift text-[11px] text-muted">{c.noClip}</div>
              )}
              <div className="min-w-0">
                <div className="mb-1 flex items-center gap-2">
                  <span className="font-mono text-[11px] tabular-nums text-muted">{String(s.shot_number).padStart(2, "0")}</span>
                  <Input className="h-7 flex-1 text-[13px]" defaultValue={s.title} onBlur={(e) => props.onPatch({ id: s.id, title: e.target.value })} />
                  <span className="font-mono text-[11px] tabular-nums text-muted">{s.duration}{c.seconds}</span>
                  <Button size="sm" variant="lift" onClick={() => props.onGen(s.id)}>{c.generate}</Button>
                </div>
                <p className="mb-2 text-[12.5px] leading-5 text-muted">{s.description}</p>
                <DramaMentionField
                  assets={shotMentions(s, props.bundle)}
                  defaultValue={s.video_prompt}
                  placeholder={c.mention}
                  onBlur={(e) => props.onPatch({ id: s.id, video_prompt: e.target.value })}
                />
              </div>
            </article>
          ))}
        </div>
      )}
    </div>
  );
}

function CutPane(props: { bundle: Bundle; sel: Record<string, boolean>; setSel: (v: Record<string, boolean>) => void; copy: CopyT; ffmpeg: boolean; onMerge: () => void }) {
  const c = props.copy.video;
  const shots = props.bundle.shots || [];
  const ep = props.bundle.episode;
  const n = selectedClipIds(shots, props.sel).length;
  return (
    <div>
      {!props.ffmpeg ? <p className="mb-3 text-[12.5px] text-danger">{c.ffmpegMissing}</p> : null}
      {ep.video_url ? <video src={ep.video_url} poster={ep.poster_url} controls className="mb-4 max-h-[360px] w-full rounded-[10px] bg-background" /> : null}
      <div className="mb-3 flex items-center gap-2">
        <Button onClick={props.onMerge} disabled={!props.ffmpeg || n === 0}>{c.export}</Button>
        <span className="text-[12px] tabular-nums text-muted">{n} {c.clipCount}</span>
        <span className="text-[12px] text-muted">{c.selectAll}</span>
      </div>
      <div className="grid gap-2">
        {shots.map((s) => (
          <label key={s.id} className={cn("cursor-pointer overflow-hidden rounded-[10px] border bg-card", props.sel[s.id] !== false && s.video_url ? "border-foreground/40" : "border-border/80")}>
            {s.video_url ? <video src={s.video_url} poster={s.poster_url} className="aspect-video w-full object-cover" /> : <div className="grid aspect-video place-items-center bg-lift text-[11px] text-muted">{c.noClip}</div>}
            <div className="flex items-center gap-2 px-2.5 py-2 text-[12px]">
              <input type="checkbox" className="accent-foreground" checked={props.sel[s.id] !== false && !!s.video_url} disabled={!s.video_url} onChange={(e) => props.setSel({ ...props.sel, [s.id]: e.target.checked })} />
              <span className="min-w-0 flex-1 truncate">{s.title || s.shot_number}</span>
              <span className="font-mono text-[11px] tabular-nums text-muted">{s.duration}{c.seconds}</span>
            </div>
          </label>
        ))}
      </div>
    </div>
  );
}

function JobStrip(props: { jobs: Job[]; copy: CopyT; onRetry: (id: string) => void; onCancel: (id: string) => void }) {
  const live = props.jobs.filter((j) => j.status === "queued" || j.status === "running" || j.status === "polling" || j.status === "failed").slice(0, 6);
  if (!live.length) return null;
  const c = props.copy.video;
  const label: Record<string, string> = { queued: c.queued, running: c.running, polling: c.polling, succeeded: c.succeeded, failed: c.failed };
  return (
    <div className="flex gap-2 overflow-x-auto border-t border-border/70 px-3 py-2">
      {live.map((j) => {
        const moving = j.status === "queued" || j.status === "running" || j.status === "polling";
        return (
          <div key={j.id} className="flex shrink-0 items-center gap-2 rounded-lg bg-lift px-2 py-1 text-[11px]">
            {moving ? <span className="pulse-dot" /> : null}
            <span className="font-medium">{j.type}</span>
            <span className={j.status === "failed" ? "text-danger" : "text-muted"}>{label[j.status] || j.status}</span>
            {j.status === "failed" ? <button type="button" className="underline" onClick={() => props.onRetry(j.id)}>{c.retry}</button> : null}
            {moving ? <button type="button" className="underline" onClick={() => props.onCancel(j.id)}>{c.cancel}</button> : null}
          </div>
        );
      })}
    </div>
  );
}
