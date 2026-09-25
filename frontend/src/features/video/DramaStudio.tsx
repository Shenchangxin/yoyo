import { useCallback, useEffect, useState } from "react";
import { Clapperboard, Film, ImagePlus, Plus, Upload, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input, Textarea } from "../../components/ui/input";
import { EmptyState } from "../../components/ui/empty-state";
import { Tooltip } from "../../components/ui/tooltip";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
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
  image_model?: string;
  video_model?: string;
  tts_provider_id?: string;
  tts_model?: string;
  video_url?: string;
  poster_url?: string;
};
type Asset = {
  id: string;
  name?: string;
  location?: string;
  role?: string;
  image_url?: string;
  image_hash?: string;
  linked?: boolean;
  final_prompt?: string;
  appearance?: string;
  styling?: string;
  prompt?: string;
  lighting?: string;
  description?: string;
  time_of_day?: string;
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
type Job = {
  id: string;
  type: string;
  status: string;
  error?: string;
  episode_id?: string;
  storyboard_id?: string;
  character_id?: string;
  scene_id?: string;
  prop_id?: string;
  media_url?: string;
  poster_url?: string;
};
type Bundle = {
  drama: Drama;
  episode: Episode;
  characters: Asset[];
  scenes: Asset[];
  props: Asset[];
  shots: Shot[];
  jobs: Job[];
  plan: { chars: number; target_seconds: number; segment_count: number };
  status?: { ffmpeg?: boolean; missing?: string[] };
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

function parseModels(raw: string): string[] {
  const s = (raw || "").trim();
  if (!s) return [];
  if (s.startsWith("[")) {
    try {
      const v = JSON.parse(s);
      if (Array.isArray(v)) return v.map(String).filter(Boolean);
    } catch {
      /* fall through */
    }
  }
  return s.split(",").map((x) => x.trim()).filter(Boolean);
}

function catalogOf(p: { model?: string; models?: string } | undefined, selected = ""): string[] {
  const out: string[] = [];
  const add = (v: string) => {
    const s = (v || "").trim();
    if (s && !out.includes(s)) out.push(s);
  };
  add(String(p?.model || ""));
  add(selected);
  for (const m of parseModels(String(p?.models || ""))) add(m);
  return out;
}

function ProviderModelSelect(props: {
  kind: string;
  label: string;
  providers: any[];
  providerId: string;
  model: string;
  onChange: (providerId: string, model: string) => void;
}) {
  const rows = props.providers.filter((p) => p.service_type === props.kind && (p.is_active !== false || p.id === props.providerId));
  if (rows.length === 0) return null;
  const current = rows.find((p) => p.id === props.providerId);
  const models = catalogOf(current, props.model);
  const model = props.model && models.includes(props.model) ? props.model : (current?.model || models[0] || "");
  const value = props.providerId ? `${props.providerId}::${model}` : "";
  return (
    <select
      className={cn(field, "h-7 max-w-[14rem]")}
      value={value}
      aria-label={props.label}
      onChange={(e) => {
        const raw = e.target.value;
        const i = raw.indexOf("::");
        if (i < 0) {
          props.onChange("", "");
          return;
        }
        props.onChange(raw.slice(0, i), raw.slice(i + 2));
      }}
    >
      <option value="">{props.label}</option>
      {rows.map((p) => {
        const models = catalogOf(p, p.id === props.providerId ? props.model : "");
        const ids = models.length ? models : [""];
        return (
          <optgroup key={p.id} label={`${p.name || p.provider}${p.has_key ? "" : " · —"}`}>
            {ids.map((m) => (
              <option key={`${p.id}::${m}`} value={`${p.id}::${m}`}>{m || p.name || p.provider}</option>
            ))}
          </optgroup>
        );
      })}
    </select>
  );
}

function isNarrator(a: Asset) {
  const t = `${a.name || ""} ${a.role || ""}`.toLowerCase();
  return ["旁白", "画外音", "narrator", "voice-over", "voiceover"].some((k) => t.includes(k));
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

export function DramaStudio(props: { sessionId?: string; onNeedSession: () => void; onClose?: () => void }) {
  const copy = useCopy();
  const openSettings = useUI((s) => s.openSettings);
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
  const [missing, setMissing] = useState<string[]>([]);
  const [ratio, setRatio] = useState("16:9");
  const [kill, setKill] = useState("");
  const [focusShot, setFocusShot] = useState("");
  const [tasksOpen, setTasksOpen] = useState(false);

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
    if (Array.isArray(b?.status?.missing)) setMissing(b.status.missing);
  }, []);

  useEffect(() => {
    void loadList().catch((e) => setErr(api.errMessage(e)));
    void api.video.styles().then((s) => setStyles(Array.isArray(s) ? s : [])).catch(() => {});
    void api.video.providers().then((p) => setProviders(Array.isArray(p) ? p : [])).catch(() => {});
    void api.video.status().then((s) => {
      setFfmpeg(!!s?.ffmpeg);
      if (Array.isArray(s?.missing)) setMissing(s.missing);
    }).catch(() => {});
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
    { id: "video_prompts", label: copy.video.vprompts, pane: "board" as Pane },
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

  const showMissing = missing.includes("image") || missing.includes("video");
  const shots = bundle?.shots || [];
  const focused = shots.find((s) => s.id === focusShot) || shots[0];
  const liveCount = (bundle?.jobs || []).filter((j) => j.status === "queued" || j.status === "running" || j.status === "polling" || j.status === "failed").length;

  return (
    <div className="drama-studio flex h-full min-h-0 flex-col" data-testid="drama-studio">
      {err ? <div className="border-b border-danger/20 bg-danger/[0.11] px-3 py-1.5 text-[12px] text-danger">{err}</div> : null}
      {showMissing ? (
        <div className="flex items-center gap-2 border-b border-border/70 bg-lift/60 px-3 py-1.5 text-[12px] text-muted">
          <span className="min-w-0 flex-1">{copy.video.missingProviders}</span>
          <button type="button" className="underline" onClick={() => openSettings("generation", "generation-image")}>{copy.video.openSettings}</button>
        </div>
      ) : null}
      <header className="drama-toolbar glass-chrome">
        <select
          className={cn(field, "min-w-0 max-w-[9.5rem]")}
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
          className={cn(field, "min-w-0 max-w-[8.5rem]")}
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
        <div className="flex rounded-[8px] bg-lift p-0.5">
          {["16:9", "9:16", "1:1"].map((r) => (
            <button
              key={r}
              type="button"
              title={copy.video.ratio}
              className={cn("rounded-[6px] px-1.5 py-1 font-mono text-[10px] tabular-nums", (drama?.aspect_ratio || ratio) === r ? "bg-card text-foreground" : "text-muted")}
              onClick={() => {
                setRatio(r);
                if (dramaId) void api.video.updateDrama({ id: dramaId, aspect_ratio: r }).then(loadList);
              }}
            >
              {r}
            </button>
          ))}
        </div>
        <div className="process-tabs ml-1 flex h-8 items-stretch gap-0.5" role="tablist" aria-label={copy.video.workshop}>
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
        <Tooltip content={copy.video.importHint}>
          <button type="button" className={iconBtn} aria-label={copy.video.import} onClick={() => void importHuobao()}>
            <Upload className="size-3.5" />
          </button>
        </Tooltip>
        <button
          type="button"
          className={cn("ml-auto inline-flex h-7 items-center gap-1 rounded-[8px] px-2 text-[11px] font-medium", tasksOpen ? "bg-lift text-foreground" : "text-muted hover:bg-lift hover:text-foreground")}
          onClick={() => setTasksOpen((v) => !v)}
        >
          {copy.video.jobs}
          {liveCount ? <span className="tabular-nums">{liveCount}</span> : null}
        </button>
        {props.onClose ? (
          <Tooltip content={copy.video.closeBoard}>
            <button type="button" className={iconBtn} aria-label={copy.video.closeBoard} onClick={props.onClose}>
              <X className="size-3.5" />
            </button>
          </Tooltip>
        ) : null}
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
          <div className="drama-body">
            <aside className="drama-browser">
              <CastPane
                bundle={bundle}
                copy={copy}
                onExtract={() => void stage("extract")}
                onExtractKind={(k) => void stage(k)}
                onPrompts={() => void stage("prompts")}
                onStills={() => void run("stills", () => api.video.generateMissingAssets(episodeId))}
                onGen={(kind, id) => void run("gen", () => api.video.generateAsset(kind, id, episodeId))}
                onUpload={(kind, id, b64) => void run("up", () => api.video.uploadAsset(kind, id, b64))}
                onSave={(kind, id, fields) => void run("asset", () => api.video.saveAsset(kind, id, fields))}
                onCreate={(kind, fields) => void run("new", () => api.video.createAsset(kind, episodeId, fields))}
                onDelete={(kind, id) => void run("del", () => api.video.deleteAsset(kind, id))}
              />
            </aside>
            <section className="drama-viewer" aria-label={copy.video.play}>
              <div className="drama-viewer-stage">
                {pane === "script" ? (
                  <div className="w-full max-w-[42rem] text-[13px] leading-6 text-[#f2ede6]/80">
                    {(ep?.script_content || ep?.content || copy.video.emptyHint).slice(0, 900)}
                  </div>
                ) : pane === "cut" && ep?.video_url ? (
                  <video src={ep.video_url} poster={ep.poster_url} controls />
                ) : focused?.video_url ? (
                  <video src={focused.video_url} poster={focused.poster_url} controls />
                ) : focused?.poster_url ? (
                  <img src={focused.poster_url} alt="" />
                ) : (
                  <div className="grid aspect-video w-full max-w-xl place-items-center rounded-[10px] bg-black/40 text-[12px] text-[#f2ede6]/55">{copy.video.noClip}</div>
                )}
              </div>
              <div className="flex flex-wrap items-center gap-1 border-t border-white/5 px-2 py-1.5">
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
                        else void stage(s.id);
                      }}
                      className={cn(
                        "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[11px] font-medium transition-colors",
                        st === "done" ? "bg-white/10 text-[#f2ede6]" : live ? "bg-accent/20 text-[#f2ede6]" : "text-[#f2ede6]/55 hover:bg-white/8 hover:text-[#f2ede6]",
                      )}
                    >
                      {live ? <span className="pulse-dot" /> : <span className={cn("size-1.5 rounded-full", st === "done" ? "bg-success" : "bg-white/25")} />}
                      {s.label}
                    </button>
                  );
                })}
                {busy ? <span className="ml-1 text-[11px] text-[#f2ede6]/55">{copy.video.stageBusy}</span> : null}
              </div>
            </section>
            <aside className="drama-inspector">
              <div className="mb-3 flex flex-wrap items-center gap-1.5">
                <Input
                  className="h-7 max-w-[7.5rem] text-[12.5px] font-medium"
                  value={drama?.title || ""}
                  aria-label={copy.video.pickSeries}
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
              </div>
              <div className="mb-3 grid gap-1.5">
                <ProviderModelSelect
                  kind="image"
                  label={copy.video.imageProvider}
                  providers={providers}
                  providerId={ep?.image_provider_id || ""}
                  model={ep?.image_model || ""}
                  onChange={(id, model) => { if (ep) void api.video.updateEpisode({ id: ep.id, image_provider_id: id, image_model: model }).then(() => loadBundle(ep.id)); }}
                />
                <ProviderModelSelect
                  kind="video"
                  label={copy.video.videoProvider}
                  providers={providers}
                  providerId={ep?.video_provider_id || ""}
                  model={ep?.video_model || ""}
                  onChange={(id, model) => { if (ep) void api.video.updateEpisode({ id: ep.id, video_provider_id: id, video_model: model }).then(() => loadBundle(ep.id)); }}
                />
                <ProviderModelSelect
                  kind="tts"
                  label={copy.video.kindSpeech}
                  providers={providers}
                  providerId={ep?.tts_provider_id || ""}
                  model={ep?.tts_model || ""}
                  onChange={(id, model) => { if (ep) void api.video.updateEpisode({ id: ep.id, tts_provider_id: id, tts_model: model }).then(() => loadBundle(ep.id)); }}
                />
                <select className={cn(field, "h-7")} value={ep?.resolution || "720p"} aria-label={copy.video.resolution} onChange={(e) => { if (ep) void api.video.updateEpisode({ id: ep.id, resolution: e.target.value }).then(() => loadBundle(ep.id)); }}>
                  {["480p", "720p", "1080p"].map((r) => <option key={r} value={r}>{r}</option>)}
                </select>
              </div>
              {pane === "script" ? <ScriptPane ep={ep!} plan={bundle.plan} copy={copy} onSave={(patch) => void run("save", () => api.video.updateEpisode({ id: ep!.id, ...patch }))} onRewrite={() => void stage("rewrite")} onSkip={() => void run("skip", () => api.video.skipRewrite(episodeId))} /> : null}
              {pane === "board" ? (
                <BoardPane
                  bundle={bundle}
                  copy={copy}
                  focusId={focused?.id}
                  onFocus={setFocusShot}
                  onBoard={() => void stage("storyboard")}
                  onVprompts={() => void stage("video_prompts")}
                  onGenAll={() => void run("clips", () => api.video.generateMissingShots(episodeId))}
                  onGen={(id) => void run("clip", () => api.video.generateShot(id))}
                  onPatch={(s) => void run("shot", () => api.video.updateShot(s))}
                  onApply={(id) => void run("apply", () => api.video.applyJob(id))}
                />
              ) : null}
              {pane === "cut" ? <CutPane bundle={bundle} sel={selShots} setSel={setSelShots} copy={copy} ffmpeg={ffmpeg} compact onMerge={() => {
                const ids = selectedClipIds(bundle.shots || [], selShots);
                if (!ids.length) {
                  toast.message(copy.video.needClips);
                  return;
                }
                void run("merge", () => api.video.merge(episodeId, ids));
              }} /> : null}
              {pane === "cast" ? (
                <p className="text-[12px] leading-5 text-muted">{copy.video.workshopHint}</p>
              ) : null}
              <button
                type="button"
                className={cn("mt-3 h-6 rounded-md px-2 text-[11px] font-medium", kill === "episode" + episodeId ? "text-danger" : "text-muted hover:bg-lift hover:text-foreground")}
                onClick={() => void remove("episode", episodeId)}
              >
                {kill === "episode" + episodeId ? copy.video.confirmDelete : copy.video.delete}
              </button>
            </aside>
          </div>
          {shots.length ? (
            <div className="drama-timeline" aria-label={copy.video.board}>
              {shots.map((s) => (
                <button
                  key={s.id}
                  type="button"
                  className={cn("drama-shot", focused?.id === s.id && "is-on")}
                  onClick={() => { setFocusShot(s.id); setPane("board"); }}
                >
                  {s.poster_url || s.video_url ? (
                    <img src={s.poster_url || ""} alt="" />
                  ) : (
                    <span className="drama-shot-empty grid place-items-center text-[10px]">{String(s.shot_number).padStart(2, "0")}</span>
                  )}
                  <span className="truncate font-mono text-[10px] tabular-nums">{String(s.shot_number).padStart(2, "0")} {s.title}</span>
                </button>
              ))}
            </div>
          ) : null}
          {tasksOpen ? (
            <div className="drama-task-drawer">
              <JobStrip jobs={bundle.jobs || []} copy={copy} onRetry={(id) => void run("retry", () => api.video.retryJob(id))} onCancel={(id) => void run("cancel", () => api.video.cancelJob(id))} />
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}

function ScriptPane({ ep, plan, copy, onSave, onRewrite, onSkip }: { ep: Episode; plan: Bundle["plan"]; copy: CopyT; onSave: (p: Partial<Episode>) => void; onRewrite: () => void; onSkip: () => void }) {
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
        <div className="mb-1.5 flex items-center justify-between gap-2 text-[11px] text-muted">
          <span>{copy.video.screenplay}</span>
          <span className="flex gap-1">
            <Button size="sm" variant="lift" onClick={onSkip} disabled={!content.trim()}>{copy.video.skipRewrite}</Button>
            <Button size="sm" onClick={onRewrite}>{copy.video.rewrite}</Button>
          </span>
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
  onExtract: () => void; onExtractKind: (k: string) => void; onPrompts: () => void; onStills: () => void;
  onGen: (kind: string, id: string) => void; onUpload: (kind: string, id: string, b64: string) => void;
  onSave: (kind: string, id: string, fields: Record<string, string>) => void;
  onCreate: (kind: string, fields: Record<string, string>) => void;
  onDelete: (kind: string, id: string) => void;
}) {
  const c = props.copy.video;
  return (
    <div>
      <div className="mb-3 flex flex-wrap gap-2">
        <Button size="sm" onClick={props.onExtract}>{c.extract}</Button>
        <Button size="sm" variant="lift" onClick={() => props.onExtractKind("extract_characters")}>{c.extractChars}</Button>
        <Button size="sm" variant="lift" onClick={() => props.onExtractKind("extract_scenes")}>{c.extractScenes}</Button>
        <Button size="sm" variant="lift" onClick={() => props.onExtractKind("extract_props")}>{c.extractProps}</Button>
        <Button size="sm" variant="lift" onClick={props.onPrompts}>{c.prompts}</Button>
        <Button size="sm" variant="lift" onClick={props.onStills}>{c.generateAll}</Button>
      </div>
      <div className="grid gap-4">
        <AssetCol title={c.characters} items={props.bundle.characters} name={(a) => a.name || ""} kind="character" createLabel={c.addCharacter} copy={props.copy} onGen={props.onGen} onUpload={props.onUpload} onSave={props.onSave} onCreate={props.onCreate} onDelete={props.onDelete} />
        <AssetCol title={c.scenes} items={props.bundle.scenes} name={(a) => a.location || ""} kind="scene" createLabel={c.addScene} copy={props.copy} onGen={props.onGen} onUpload={props.onUpload} onSave={props.onSave} onCreate={props.onCreate} onDelete={props.onDelete} />
        <AssetCol title={c.props} items={props.bundle.props} name={(a) => a.name || ""} kind="prop" createLabel={c.addProp} copy={props.copy} onGen={props.onGen} onUpload={props.onUpload} onSave={props.onSave} onCreate={props.onCreate} onDelete={props.onDelete} />
      </div>
    </div>
  );
}

function AssetCol(props: {
  title: string; items: Asset[]; name: (a: Asset) => string; kind: string; createLabel: string;
  copy: CopyT; onGen: (kind: string, id: string) => void; onUpload: (kind: string, id: string, b64: string) => void;
  onSave: (kind: string, id: string, fields: Record<string, string>) => void;
  onCreate: (kind: string, fields: Record<string, string>) => void;
  onDelete: (kind: string, id: string) => void;
}) {
  const rows = props.items.filter((a) => a.linked !== false);
  const [open, setOpen] = useState("");
  const [draft, setDraft] = useState("");
  const c = props.copy.video;
  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between px-0.5">
        <span className="text-[11px] font-medium text-muted">{props.title}</span>
        <button type="button" className="text-[11px] text-muted hover:text-foreground" onClick={() => {
          const name = window.prompt(props.createLabel);
          if (!name?.trim()) return;
          if (props.kind === "scene") props.onCreate("scene", { location: name.trim() });
          else props.onCreate(props.kind, { name: name.trim() });
        }}>{props.createLabel}</button>
      </div>
      {rows.length === 0 ? (
        <p className="px-0.5 py-3 text-[12px] text-muted">{c.noStills}</p>
      ) : (
        <ul>
          {rows.map((a, i) => {
            const narrator = props.kind === "character" && isNarrator(a);
            return (
              <li key={a.id} className={cn("py-2", i ? "border-t border-border/50" : "")}>
                <div className="flex items-center gap-2">
                  {a.image_url ? (
                    <img src={a.image_url} alt="" className="size-10 shrink-0 rounded-md object-cover" />
                  ) : (
                    <div className="grid size-10 shrink-0 place-items-center rounded-md bg-lift text-[10px] text-muted">—</div>
                  )}
                  <button type="button" className="min-w-0 flex-1 text-left" onClick={() => { setOpen(open === a.id ? "" : a.id); setDraft(""); }}>
                    <div className="truncate text-[12.5px] font-medium">{props.name(a)}</div>
                    <div className="truncate text-[11px] text-muted">{narrator ? c.narrator : (a.final_prompt || a.appearance || a.description || a.prompt || "")}</div>
                  </button>
                  <label className="grid size-7 cursor-pointer place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground" title={c.upload}>
                    <ImagePlus className="size-3.5" />
                    <input type="file" accept="image/*" className="hidden" onChange={(e) => {
                      const f = e.target.files?.[0];
                      if (!f) return;
                      const r = new FileReader();
                      r.onload = () => props.onUpload(props.kind, a.id, String(r.result || ""));
                      r.readAsDataURL(f);
                    }} />
                  </label>
                  {narrator ? null : <Button size="sm" variant="ghost" onClick={() => props.onGen(props.kind, a.id)}>{c.generate}</Button>}
                </div>
                {open === a.id ? (
                  <div className="mt-2 grid gap-2 rounded-lg bg-lift/50 p-2">
                    {props.kind === "character" ? (
                      <>
                        <Input className="h-7 text-[12px]" defaultValue={a.appearance || ""} placeholder={c.appearance} onBlur={(e) => props.onSave("character", a.id, { appearance: e.target.value })} />
                        <Input className="h-7 text-[12px]" defaultValue={a.styling || ""} placeholder={c.styling} onBlur={(e) => props.onSave("character", a.id, { styling: e.target.value })} />
                      </>
                    ) : null}
                    {props.kind === "scene" ? (
                      <Input className="h-7 text-[12px]" defaultValue={a.lighting || ""} placeholder={c.lighting} onBlur={(e) => props.onSave("scene", a.id, { lighting: e.target.value })} />
                    ) : null}
                    {props.kind === "prop" ? (
                      <Input className="h-7 text-[12px]" defaultValue={a.description || ""} placeholder={c.finalPrompt} onBlur={(e) => props.onSave("prop", a.id, { description: e.target.value })} />
                    ) : null}
                    <Textarea className="min-h-[72px] text-[12px]" defaultValue={a.final_prompt || ""} placeholder={c.finalPrompt} onBlur={(e) => props.onSave(props.kind, a.id, { final_prompt: e.target.value })} />
                    <div className="flex justify-end">
                      <button type="button" className="text-[11px] text-danger" onClick={() => {
                        if (draft !== a.id) { setDraft(a.id); return; }
                        props.onDelete(props.kind, a.id);
                      }}>{draft === a.id ? c.confirmDelete : c.delete}</button>
                    </div>
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function BoardPane(props: {
  bundle: Bundle; copy: CopyT;
  focusId?: string;
  onFocus?: (id: string) => void;
  onBoard: () => void; onVprompts: () => void; onGenAll: () => void; onGen: (id: string) => void;
  onPatch: (s: Record<string, any>) => void; onApply: (id: string) => void;
}) {
  const c = props.copy.video;
  const shots = props.bundle.shots || [];
  const [focus, setFocus] = useState(props.focusId || "");
  useEffect(() => { if (props.focusId) setFocus(props.focusId); }, [props.focusId]);
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
          {shots.map((s, i) => {
            const takes = (props.bundle.jobs || []).filter((j) => j.storyboard_id === s.id && j.type === "video" && j.status === "succeeded");
            const open = focus === s.id;
            const selected = (props.focusId || focus) === s.id;
            return (
              <article key={s.id} className={cn("grid gap-2 py-2", i ? "border-t border-border/50" : "", selected && "bg-lift/40")}>
                <div className="min-w-0">
                  <div className="mb-1 flex items-center gap-2">
                    <button type="button" className="font-mono text-[11px] tabular-nums text-muted" onClick={() => { setFocus(s.id); props.onFocus?.(s.id); }}>
                      {String(s.shot_number).padStart(2, "0")}
                    </button>
                    <Input className="h-7 flex-1 text-[13px]" defaultValue={s.title} onFocus={() => { setFocus(s.id); props.onFocus?.(s.id); }} onBlur={(e) => props.onPatch({ id: s.id, title: e.target.value })} />
                    <Input
                      className="h-7 w-14 font-mono text-[12px] tabular-nums"
                      type="number"
                      min={2}
                      max={30}
                      defaultValue={s.duration}
                      aria-label={c.durationLabel}
                      onBlur={(e) => props.onPatch({ id: s.id, duration: Number(e.target.value) || s.duration })}
                    />
                    <span className="text-[11px] text-muted">{c.seconds}</span>
                    {s.video_url ? <a className="text-[11px] underline" href={s.video_url} download>{c.download}</a> : null}
                    <Button size="sm" variant="lift" onClick={() => props.onGen(s.id)}>{c.generate}</Button>
                  </div>
                  <p className="mb-2 text-[12.5px] leading-5 text-muted">{s.description}</p>
                  <DramaMentionField
                    assets={shotMentions(s, props.bundle)}
                    defaultValue={s.video_prompt}
                    placeholder={c.mention}
                    onBlur={(e) => props.onPatch({ id: s.id, video_prompt: e.target.value })}
                  />
                  <button type="button" className="mt-2 text-[11px] text-muted hover:text-foreground" onClick={() => setFocus(open ? "" : s.id)}>{c.bindRefs}</button>
                  {open ? (
                    <div className="mt-2 grid gap-2 rounded-lg bg-lift/50 p-2 text-[12px]">
                      <label className="flex items-center gap-2">
                        <span className="w-16 text-muted">{c.scenes}</span>
                        <select className={cn(field, "h-7 flex-1")} value={s.scene_id || ""} onChange={(e) => props.onPatch({ id: s.id, scene_id: e.target.value })}>
                          <option value="">{c.unbind}</option>
                          {(props.bundle.scenes || []).map((sc) => <option key={sc.id} value={sc.id}>{sc.location}</option>)}
                        </select>
                      </label>
                      <div>
                        <div className="mb-1 text-muted">{c.characters}</div>
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
                        <div className="mb-1 text-muted">{c.props}</div>
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
                      <div>
                        <div className="mb-1 text-muted">{c.history}</div>
                        {takes.length === 0 ? <p className="text-[11px] text-muted">{c.noHistory}</p> : takes.map((j) => (
                          <div key={j.id} className="flex items-center gap-2 py-1">
                            {j.poster_url ? <img src={j.poster_url} alt="" className="h-8 w-12 rounded object-cover" /> : <div className="h-8 w-12 rounded bg-lift" />}
                            {j.media_url ? <a className="text-[11px] underline" href={j.media_url} download>{c.download}</a> : null}
                            <button type="button" className="text-[11px] underline" onClick={() => props.onApply(j.id)}>{c.setMain}</button>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}
                </div>
              </article>
            );
          })}
        </div>
      )}
    </div>
  );
}

function CutPane(props: { bundle: Bundle; sel: Record<string, boolean>; setSel: (v: Record<string, boolean>) => void; copy: CopyT; ffmpeg: boolean; compact?: boolean; onMerge: () => void }) {
  const c = props.copy.video;
  const shots = props.bundle.shots || [];
  const ep = props.bundle.episode;
  const n = selectedClipIds(shots, props.sel).length;
  return (
    <div>
      {!props.ffmpeg ? <p className="mb-3 text-[12.5px] text-danger">{c.ffmpegMissing}</p> : null}
      {!props.compact && ep.video_url ? <video src={ep.video_url} poster={ep.poster_url} controls className="mb-4 max-h-[360px] w-full rounded-[10px] bg-background" /> : null}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Button onClick={props.onMerge} disabled={!props.ffmpeg || n === 0}>{c.export}</Button>
        <span className="text-[12px] tabular-nums text-muted">{n} {c.clipCount}</span>
        <button type="button" className="text-[12px] text-muted underline" onClick={() => {
          const next: Record<string, boolean> = {};
          for (const s of shots) next[s.id] = !!s.video_url;
          props.setSel(next);
        }}>{c.selectAll}</button>
        <button type="button" className="text-[12px] text-muted underline" onClick={() => {
          const next: Record<string, boolean> = {};
          for (const s of shots) next[s.id] = false;
          props.setSel(next);
        }}>{c.selectNone}</button>
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
