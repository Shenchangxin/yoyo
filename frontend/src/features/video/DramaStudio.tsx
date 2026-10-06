import { useCallback, useEffect, useRef, useState } from "react";
import { Check, ChevronDown, Ellipsis, Film, Plus, Upload, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { Tooltip } from "../../components/ui/tooltip";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import * as api from "../../lib/client";
import { subscribeItems } from "../../lib/stream";
import { ConfirmDialog } from "../ConfirmDialog";
import { useDramaSelection } from "./workshop-store";
import { ingestSeriesFile } from "./bind";
import { modelOptionName, useConfigStore } from "@yingce/stores/use-config-store";
import { commitEpisodeTitle, commitSeriesTitle, firstLook, isAgentStep, jobMoving, phaseDone, phaseQueue, pipeStatus, selectedClipIds, shownEpisodeTitle, shownSeriesTitle, type Bundle, type Drama, type Episode, type EpisodePlan, type Job, type Phase, type PhaseStep, type Shot } from "./drama-lib";
import { BoardInspector, CastGrid, CastInspector, CutInspector, EmptyDesk, EpisodeMap, GenerationSettings, JobDrawer, ScriptStage } from "./drama-stages";

const iconBtn =
  "grid size-7 shrink-0 place-items-center rounded-md text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30";

const PHASES: Phase[] = ["script", "cast", "board", "cut"];

export function DramaStudio(props: { sessionId?: string; onNeedSession: () => void | Promise<string | void>; onClose?: () => void }) {
  const copy = useCopy();
  const openSettings = useUI((s) => s.openSettings);
  const dramaId = useDramaSelection((s) => s.dramaId);
  const episodeId = useDramaSelection((s) => s.episodeId);
  const setDramaId = useDramaSelection((s) => s.setDramaId);
  const setEpisodeId = useDramaSelection((s) => s.setEpisodeId);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [bundle, setBundle] = useState<Bundle | null>(null);
  const [pane, setPane] = useState<Phase>("script");
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [selShots, setSelShots] = useState<Record<string, boolean>>({});
  const [styles, setStyles] = useState<{ value: string; name: string }[]>([]);
  const [providers, setProviders] = useState<any[]>([]);
  const [ffmpeg, setFfmpeg] = useState(true);
  const [ffmpegInstall, setFfmpegInstall] = useState<{ phase?: string; percent?: number; error?: string }>({});
  const [missing, setMissing] = useState<string[]>([]);
  const [ratio, setRatio] = useState("9:16");
  const [draftTitle, setDraftTitle] = useState("");
  const [focusShot, setFocusShot] = useState("");
  const [focusAsset, setFocusAsset] = useState({ kind: "", id: "" });
  const [tasksOpen, setTasksOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [killEp, setKillEp] = useState(false);
  const [killAsset, setKillAsset] = useState<{ kind: string; id: string } | null>(null);
  const [plan, setPlan] = useState<EpisodePlan | null>(null);
  const [dramaJobs, setDramaJobs] = useState<Job[]>([]);
  const queueRef = useRef<PhaseStep[]>([]);
  const waitingTurn = useRef(false);
  const kickRef = useRef<() => Promise<void>>(async () => {});
  const scriptQueueRef = useRef(false);

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
    if (b?.status?.ffmpeg_install && typeof b.status.ffmpeg_install === "object") setFfmpegInstall(b.status.ffmpeg_install);
    if (Array.isArray(b?.status?.missing)) setMissing(b.status.missing);
  }, []);

  const loadPlan = useCallback(async (id: string) => {
    if (!id) {
      setPlan(null);
      return;
    }
    const p = await api.video.getPlan(id).catch(() => null) as EpisodePlan | null;
    if (!p || !p.status) {
      setPlan(null);
      return;
    }
    setPlan(p);
  }, []);

  useEffect(() => {
    void loadList().catch((e) => setErr(api.errMessage(e)));
    void api.video.styles().then((s) => setStyles(Array.isArray(s) ? s : [])).catch(() => {});
    void api.video.providers().then((p) => setProviders(Array.isArray(p) ? p : [])).catch(() => {});
    void api.video.status().then((s) => {
      setFfmpeg(!!s?.ffmpeg);
      if (s?.ffmpeg_install && typeof s.ffmpeg_install === "object") setFfmpegInstall(s.ffmpeg_install);
      if (Array.isArray(s?.missing)) setMissing(s.missing);
    }).catch(() => {});
  }, [loadList]);

  const ffmpegWorking = ffmpegInstall.phase === "resolving" || ffmpegInstall.phase === "downloading" || ffmpegInstall.phase === "extracting" || ffmpegInstall.phase === "verifying";
  useEffect(() => {
    if (!ffmpegWorking) return;
    const t = window.setInterval(() => {
      void api.video.status().then((s) => {
        setFfmpeg(!!s?.ffmpeg);
        if (s?.ffmpeg_install && typeof s.ffmpeg_install === "object") setFfmpegInstall(s.ffmpeg_install);
      }).catch(() => {});
    }, 400);
    return () => window.clearInterval(t);
  }, [ffmpegWorking]);

  useEffect(() => {
    if (!dramaId) {
      setEpisodes([]);
      setPlan(null);
      setDramaJobs([]);
      return;
    }
    void loadPlan(dramaId).catch(() => setPlan(null));
    void api.video.dramaJobs(dramaId).then((list) => setDramaJobs(Array.isArray(list) ? list : [])).catch(() => setDramaJobs([]));
    void api.video.episodes(dramaId).then((list) => {
      const eps = Array.isArray(list) ? list : [];
      setEpisodes(eps);
    }).catch((e) => setErr(api.errMessage(e)));
  }, [dramaId, loadPlan]);

  useEffect(() => {
    if (!dramaId || plan?.status === "draft") return;
    if (!episodeId && episodes[0]) setEpisodeId(episodes[0].id);
  }, [dramaId, episodeId, episodes, plan?.status, setEpisodeId]);

  useEffect(() => {
    if (!episodeId) {
      setBundle(null);
      return;
    }
    void loadBundle(episodeId).catch((e) => setErr(api.errMessage(e)));
  }, [episodeId, loadBundle]);

  useEffect(() => {
    setFocusShot("");
    setFocusAsset({ kind: "", id: "" });
    setSelShots({});
  }, [episodeId]);

  useEffect(() => {
    if (pane !== "cast") return;
    if (bundle?.episode?.id && episodeId && bundle.episode.id !== episodeId) return;
    const hit = firstLook(bundle);
    if (!hit) return;
    const ids = new Set([
      ...(bundle?.characters || []).filter((a) => a.linked !== false).map((a) => a.id),
      ...(bundle?.scenes || []).filter((a) => a.linked !== false).map((a) => a.id),
      ...(bundle?.props || []).filter((a) => a.linked !== false).map((a) => a.id),
    ]);
    if (focusAsset.id && ids.has(focusAsset.id)) return;
    setFocusAsset(hit);
  }, [pane, bundle, episodeId, focusAsset.id]);

  useEffect(() => subscribeItems((item) => {
    if (item.source === "video") {
      const epId = String(item.payload?.episode_id || "");
      if (epId && epId === episodeId) void loadBundle(episodeId);
    }
    if (item.type === "turn_end" && dramaId) void loadPlan(dramaId);
    if ((item.type === "turn_end" || item.type === "error") && waitingTurn.current) {
      if (item.type === "error") {
        waitingTurn.current = false;
        scriptQueueRef.current = false;
        queueRef.current = [];
        setBusy("");
        setErr(item.text || copy.video.failed);
        return;
      }
      waitingTurn.current = false;
      queueRef.current = queueRef.current.slice(1);
      if (episodeId) void loadBundle(episodeId);
      if (scriptQueueRef.current && dramaId) {
        void (async () => {
          const sid = props.sessionId || "";
          if (!sid) { scriptQueueRef.current = false; setBusy(""); return; }
          const r = await api.video.queueScripts(dramaId, sid).catch(() => null) as { done?: boolean; episode_id?: string } | null;
          if (!r || r.done || !r.episode_id) {
            scriptQueueRef.current = false;
            setBusy("");
            return;
          }
          setEpisodeId(r.episode_id);
          waitingTurn.current = true;
          setBusy("rewrite");
        })();
        return;
      }
      void kickRef.current();
    }
  }), [episodeId, dramaId, loadBundle, loadPlan, copy.video.failed, props.sessionId, setEpisodeId]);

  const liveJobs = [...(bundle?.jobs || []), ...dramaJobs].some((j) => jobMoving(j.status));
  useEffect(() => {
    if (!dramaId || (!liveJobs && !busy)) return;
    const t = window.setInterval(() => {
      if (episodeId) void loadBundle(episodeId);
      void api.video.dramaJobs(dramaId).then((list) => setDramaJobs(Array.isArray(list) ? list : [])).catch(() => {});
    }, 1600);
    return () => window.clearInterval(t);
  }, [dramaId, episodeId, liveJobs, busy, loadBundle]);

  useEffect(() => {
    if (!props.sessionId) return;
    if (episodeId) {
      void api.video.bind(props.sessionId, episodeId, dramaId).catch(() => {});
      return;
    }
    if (dramaId) void api.video.bind(props.sessionId, "", dramaId).catch(() => {});
  }, [props.sessionId, episodeId, dramaId]);

  const drama = dramas.find((d) => d.id === dramaId) || bundle?.drama;
  const ep = bundle?.episode;
  const shots = bundle?.shots || [];
  const focused = shots.find((s) => s.id === focusShot) || shots[0];
  const liveCount = [...(bundle?.jobs || []), ...dramaJobs].filter((j) => jobMoving(j.status) || j.status === "failed").length;
  const showMissing = missing.includes("image") || missing.includes("video");

  useEffect(() => {
    const head = queueRef.current[0];
    if (!waitingTurn.current || !head || !isAgentStep(head) || !ep) return;
    if (pipeStatus(ep.pipeline, head) !== "done") return;
    waitingTurn.current = false;
    queueRef.current = queueRef.current.slice(1);
    void kickRef.current();
  }, [ep]);

  async function sessionId() {
    if (props.sessionId) return props.sessionId;
    const id = await props.onNeedSession();
    return (typeof id === "string" && id) || "";
  }

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

  async function startStage(name: string) {
    const sid = await sessionId();
    if (!sid) {
      throw new Error(copy.video.failed);
    }
    if (!episodeId) throw new Error("episode");
    await api.video.bind(sid, episodeId);
    const textModel = useConfigStore.getState().config.textModel;
    if (textModel) await api.setSessionModel(sid, modelOptionName(textModel)).catch(() => {});
    await api.video.stage(sid, episodeId, name);
  }

  async function kickQueue() {
    const head = queueRef.current[0];
    if (!head) {
      waitingTurn.current = false;
      setBusy("");
      if (episodeId) await loadBundle(episodeId);
      return;
    }
    setErr("");
    setBusy(head);
    try {
      if (head === "stills") {
        queueRef.current = queueRef.current.slice(1);
        await api.video.generateMissingAssets(episodeId);
        if (episodeId) await loadBundle(episodeId);
        await kickQueue();
        return;
      }
      if (head === "clips") {
        queueRef.current = queueRef.current.slice(1);
        await api.video.generateMissingShots(episodeId);
        if (episodeId) await loadBundle(episodeId);
        await kickQueue();
        return;
      }
      if (head === "merge") {
        queueRef.current = [];
        const ids = selectedClipIds(shots, selShots);
        if (!ids.length) {
          toast.message(copy.video.needClips);
          setPane("board");
          setBusy("");
          return;
        }
        await api.video.merge(episodeId, ids);
        if (episodeId) await loadBundle(episodeId);
        setBusy("");
        return;
      }
      waitingTurn.current = true;
      await startStage(head);
    } catch (e) {
      waitingTurn.current = false;
      queueRef.current = [];
      setBusy("");
      setErr(api.errMessage(e));
      toast.error(api.errMessage(e));
    }
  }
  kickRef.current = kickQueue;

  async function ensureScript() {
    if (!ep) return false;
    if ((ep.script_content || "").trim()) return true;
    if (!(ep.content || "").trim()) {
      toast.message(copy.video.needSource);
      setPane("script");
      return false;
    }
    toast.message(copy.video.usingSource);
    try {
      await api.video.skipRewrite(episodeId);
      if (episodeId) await loadBundle(episodeId);
      return true;
    } catch (e) {
      toast.error(api.errMessage(e));
      return false;
    }
  }

  async function createDrama() {
    await run("create", async () => {
      const d = await api.video.createDrama({ title: commitSeriesTitle(draftTitle), style: "3d", aspect_ratio: ratio });
      setDramaId(d.id);
      const created = await api.video.createEpisode(d.id, "", "");
      setEpisodeId(created.id);
      setPane("script");
      setDraftTitle("");
      setCreating(false);
    });
  }

  async function createEpisode() {
    if (!dramaId) return;
    await run("episode", async () => {
      const created = await api.video.createEpisode(dramaId, "", "");
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

  async function importNovel() {
    const paths = await api.pickFiles();
    const path = (paths || []).find((p) => /\.(txt|md|markdown|docx|pdf)$/i.test(p)) || (paths || [])[0];
    if (!path) return;
    await run("ingest", async () => {
      const id = await ingestSeriesFile(path);
      setDramaId(id);
      setEpisodeId("");
      await loadPlan(id);
      const sid = props.sessionId || await sessionId();
      if (sid) {
        await api.video.bind(sid, "", id);
        await api.video.stage(sid, "", "outline", id);
      }
    });
  }

  async function confirmMap() {
    if (!dramaId) return;
    await run("commit", async () => {
      const p = await api.video.commitEpisodes(dramaId) as EpisodePlan;
      setPlan(p);
      const eps = await api.video.episodes(dramaId);
      const list = Array.isArray(eps) ? eps : [];
      setEpisodes(list);
      if (list[0]) {
        setEpisodeId(list[0].id);
        setPane("script");
      }
    });
  }

  async function queueAllScripts() {
    if (!dramaId) return;
    const sid = await sessionId();
    if (!sid) return;
    scriptQueueRef.current = true;
    setBusy("rewrite");
    waitingTurn.current = true;
    const r = await api.video.queueScripts(dramaId, sid) as { done?: boolean; episode_id?: string };
    await loadPlan(dramaId);
    if (r?.episode_id) setEpisodeId(r.episode_id);
    if (r?.done || !r?.episode_id) {
      scriptQueueRef.current = false;
      waitingTurn.current = false;
      setBusy("");
    }
  }

  async function extractBible() {
    if (!dramaId) return;
    const sid = await sessionId();
    if (!sid) return;
    waitingTurn.current = true;
    setBusy("extract");
    await api.video.bind(sid, episodeId, dramaId);
    await api.video.stage(sid, episodeId, "bible", dramaId);
  }

  async function runPhase() {
    if (!episodeId) {
      setCreating(true);
      return;
    }
    if (busy) return;
    if (pane === "script") {
      if (!(ep?.content || "").trim() && !(ep?.script_content || "").trim()) {
        toast.message(copy.video.needSource);
        return;
      }
    } else if (pane === "cast" || pane === "board") {
      if (!(await ensureScript())) return;
    } else if (!ffmpeg) {
      void api.video.installFFmpeg().then((s) => {
        setFfmpeg(!!s?.ffmpeg);
        if (s?.ffmpeg_install && typeof s.ffmpeg_install === "object") setFfmpegInstall(s.ffmpeg_install);
      }).catch((e) => toast.error(api.errMessage(e)));
      return;
    }
    queueRef.current = phaseQueue(pane, bundle);
    waitingTurn.current = false;
    await kickQueue();
  }

  const phaseHint: Record<Phase, string> = {
    script: copy.video.phaseHintScript,
    cast: copy.video.phaseHintCast,
    board: copy.video.phaseHintBoard,
    cut: copy.video.phaseHintCut,
  };
  const phaseLabel: Record<Phase, string> = {
    script: copy.video.script,
    cast: copy.video.cast,
    board: copy.video.board,
    cut: copy.video.cut,
  };
  const ctaLabel = !ffmpeg && pane === "cut"
    ? copy.video.ffmpegInstall
    : pane === "script" ? copy.video.writeScript
      : pane === "cast" ? copy.video.extractLooks
        : pane === "board" ? copy.video.breakShots
          : copy.video.exportCut;

  const lookAsset = focusAsset.kind === "character" ? (bundle?.characters || []).find((a) => a.id === focusAsset.id)
    : focusAsset.kind === "scene" ? (bundle?.scenes || []).find((a) => a.id === focusAsset.id)
      : (bundle?.props || []).find((a) => a.id === focusAsset.id);

  function reorder(from: number, to: number) {
    if (from === to || from < 0 || to < 0 || from >= shots.length || to >= shots.length) return;
    const next = shots.slice();
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    void run("order", async () => {
      for (let i = 0; i < next.length; i++) {
        if (next[i].shot_number !== i + 1) await api.video.updateShot({ id: next[i].id, shot_number: i + 1 });
      }
    });
  }

  const showMap = !!dramaId && plan?.status === "draft" && (plan.episodes?.length || 0) > 0;
  const showDesk = !!episodeId && !!bundle && !showMap;

  return (
    <div className="drama-studio flex h-full min-h-0 flex-col" data-testid="drama-studio" data-phase={pane}>
      {err ? <div className="border-b border-danger/20 bg-danger/[0.11] px-3 py-1.5 text-[12px] text-danger">{err}</div> : null}
      {showMissing ? (
        <div className="flex items-center gap-2 border-b border-border/70 bg-lift/60 px-3 py-1.5 text-[12px] text-muted">
          <span className="min-w-0 flex-1">{copy.video.missingProviders}</span>
          <button type="button" className="underline" onClick={() => openSettings("generation", "generation-image")}>{copy.video.openSettings}</button>
        </div>
      ) : null}
      <header className="drama-toolbar glass-chrome">
        <ProjectMenu
          dramas={dramas}
          episodes={episodes}
          dramaId={dramaId}
          episodeId={episodeId}
          copy={copy}
          onPick={(d, epId) => { setDramaId(d); setEpisodeId(epId); }}
          onNewDrama={() => { setDraftTitle(""); setCreating(true); }}
          onNewEpisode={() => void createEpisode()}
        />
        {drama?.aspect_ratio ? (
          <Tooltip content={copy.video.ratioLocked}>
            <span className="font-mono text-[10px] tabular-nums text-muted">{drama.aspect_ratio}</span>
          </Tooltip>
        ) : null}
        <div className="drama-phase" role="tablist" aria-label={copy.video.desk}>
          {PHASES.map((id, i) => {
            const done = phaseDone(ep, bundle, id);
            return (
              <span key={id} className="flex items-center">
                <button
                  type="button"
                  role="tab"
                  data-testid={`drama-phase-${id}`}
                  aria-selected={pane === id}
                  className={cn("drama-phase-step", pane === id && "is-on", done && "is-done")}
                  onClick={() => setPane(id)}
                >
                  <span className="drama-phase-dot">{done ? <Check className="size-2.5" /> : i + 1}</span>
                  {phaseLabel[id]}
                </button>
                {i < PHASES.length - 1 ? <span className="drama-phase-track" aria-hidden /> : null}
              </span>
            );
          })}
        </div>
        <Button size="sm" data-testid="drama-phase-cta" disabled={!!busy || (!showDesk && !showMap)} onClick={() => showMap ? void confirmMap() : void runPhase()}>
          {busy ? copy.video.stageBusy : showMap ? copy.video.mapConfirm : ctaLabel}
        </Button>
        <button
          type="button"
          className={cn("ml-auto inline-flex h-7 items-center gap-1 rounded-[8px] px-2 text-[11px] font-medium", tasksOpen ? "bg-lift text-foreground" : "text-muted hover:bg-lift hover:text-foreground")}
          onClick={() => setTasksOpen((v) => !v)}
        >
          {copy.video.jobs}
          {liveCount ? <span className="tabular-nums">{liveCount}</span> : null}
        </button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button type="button" className={iconBtn} aria-label={copy.video.more}>
              <Ellipsis className="size-3.5" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onSelect={() => void importNovel()}>
              <Upload className="size-3.5 shrink-0 opacity-70" />
              {copy.video.mapFile}
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => void importHuobao()}>
              <Upload className="size-3.5 shrink-0 opacity-70" />
              {copy.video.import}
            </DropdownMenuItem>
            {plan?.status === "committed" ? <DropdownMenuItem onSelect={() => void queueAllScripts()}>{copy.video.mapQueueScripts}</DropdownMenuItem> : null}
            {dramaId ? <DropdownMenuItem onSelect={() => void extractBible()}>{copy.video.mapBible}</DropdownMenuItem> : null}
            {pane === "cast" ? <DropdownMenuItem onSelect={() => void run("stills", () => api.video.generateMissingAssets(episodeId))}>{copy.video.fillMissingLooks}</DropdownMenuItem> : null}
            {pane === "board" ? <DropdownMenuItem onSelect={() => void run("clips", () => api.video.generateMissingShots(episodeId))}>{copy.video.fillMissingShots}</DropdownMenuItem> : null}
            {pane === "script" ? <DropdownMenuItem onSelect={() => void run("skip", () => api.video.skipRewrite(episodeId))}>{copy.video.skipRewrite}</DropdownMenuItem> : null}
          </DropdownMenuContent>
        </DropdownMenu>
        {props.onClose ? (
          <Tooltip content={copy.video.closeBoard}>
            <button type="button" className={iconBtn} data-testid="drama-close-board" aria-label={copy.video.closeBoard} onClick={props.onClose}>
              <X className="size-3.5" />
            </button>
          </Tooltip>
        ) : null}
      </header>
      {showDesk ? (
        <p className="drama-guide" data-testid="drama-guide">{phaseHint[pane]}</p>
      ) : showMap ? (
        <p className="drama-guide" data-testid="drama-guide">{copy.video.confirmMapHint}</p>
      ) : null}
      {showMap ? (
        <EpisodeMap
          plan={plan!}
          copy={copy}
          busy={!!busy}
          onConfirm={() => void confirmMap()}
          onSplit={(n) => void run("split", async () => { setPlan(await api.video.splitPlanEpisode(dramaId, n) as EpisodePlan); })}
          onMerge={(n) => void run("merge", async () => { setPlan(await api.video.mergePlanEpisodes(dramaId, n) as EpisodePlan); })}
          onBible={() => void extractBible()}
        />
      ) : !showDesk ? (
        <EmptyDesk copy={copy} onNameSeries={() => { setDraftTitle(""); setCreating(true); }} />
      ) : (
        <>
          <div className="drama-body">
            <section className="drama-viewer" aria-label={copy.video.play}>
              <div className="drama-viewer-stage">
                {pane === "script" ? (
                  <ScriptStage key={ep!.id} ep={ep!} plan={bundle?.plan} copy={copy} onSave={(patch) => void run("save", () => api.video.updateEpisode({ id: ep!.id, ...patch }))} />
                ) : pane === "cast" ? (
                  <CastGrid
                    bundle={bundle!}
                    copy={copy}
                    focusId={focusAsset.id}
                    onFocus={(kind, id) => setFocusAsset({ kind, id })}
                    onCreate={(kind, fields) => void run("new", () => api.video.createAsset(kind, episodeId, fields))}
                  />
                ) : pane === "cut" && ep?.video_url ? (
                  <video src={ep.video_url} poster={ep.poster_url} controls />
                ) : focused?.video_url ? (
                  <video src={focused.video_url} poster={focused.poster_url} controls />
                ) : focused?.poster_url ? (
                  <img src={focused.poster_url} alt="" />
                ) : (
                  <div className="grid aspect-video w-full max-w-xl place-items-center rounded-[10px] bg-black/40 text-[12px] text-[#f4f4f0]/55">{copy.video.noClip}</div>
                )}
              </div>
            </section>
            <aside className="drama-inspector">
              <div className="mb-3 grid gap-1.5">
                <label className="grid gap-0.5">
                  <span className="text-[11px] text-muted">{copy.video.seriesName}</span>
                  <Input
                    className="h-7 text-[13px] font-medium"
                    value={shownSeriesTitle(drama?.title, copy.video.untitled)}
                    aria-label={copy.video.seriesName}
                    onChange={(e) => {
                      const title = e.target.value;
                      setDramas((list) => list.map((d) => d.id === dramaId ? { ...d, title } : d));
                    }}
                    onBlur={(e) => {
                      const title = commitSeriesTitle(e.target.value);
                      if (!dramaId || !title) return;
                      void api.video.updateDrama({ id: dramaId, title });
                    }}
                  />
                </label>
                <label className="grid gap-0.5">
                  <span className="text-[11px] text-muted">{copy.video.episodeName}</span>
                  <Input
                    className="h-7 text-[13px]"
                    value={shownEpisodeTitle(ep?.title, ep?.episode_number, copy.video.episodeN)}
                    aria-label={copy.video.episodeName}
                    onChange={(e) => {
                      const title = e.target.value;
                      setBundle((b) => b ? { ...b, episode: { ...b.episode, title } } : b);
                      setEpisodes((list) => list.map((x) => x.id === ep?.id ? { ...x, title } : x));
                    }}
                    onBlur={(e) => {
                      if (!ep) return;
                      void api.video.updateEpisode({ id: ep.id, title: commitEpisodeTitle(e.target.value) });
                    }}
                  />
                </label>
              </div>
              {pane === "script" ? (
                <button
                  type="button"
                  className="h-6 self-start rounded-md px-2 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground"
                  onClick={() => void run("skip", () => api.video.skipRewrite(episodeId))}
                  disabled={!(ep?.content || "").trim()}
                >
                  {copy.video.skipRewrite}
                </button>
              ) : null}
              {pane === "cast" && lookAsset && focusAsset.kind ? (
                <CastInspector
                  key={lookAsset.id}
                  kind={focusAsset.kind}
                  asset={lookAsset}
                  copy={copy}
                  onGen={() => void run("gen", () => api.video.generateAsset(focusAsset.kind, lookAsset.id, episodeId))}
                  onUpload={(b64) => void run("up", () => api.video.uploadAsset(focusAsset.kind, lookAsset.id, b64))}
                  onSave={(fields) => void run("asset", () => api.video.saveAsset(focusAsset.kind, lookAsset.id, fields))}
                  onDelete={() => setKillAsset({ kind: focusAsset.kind, id: lookAsset.id })}
                />
              ) : pane === "cast" ? (
                <p className="text-[12px] leading-5 text-muted">{copy.video.phaseHintCast}</p>
              ) : null}
              {pane === "board" && focused ? (
                <BoardInspector
                  key={focused.id}
                  bundle={bundle!}
                  shot={focused}
                  copy={copy}
                  onPatch={(s) => void run("shot", () => api.video.updateShot(s))}
                  onGen={() => void run("clip", () => api.video.generateShot(focused.id))}
                  onApply={(id) => void run("apply", () => api.video.applyJob(id))}
                />
              ) : pane === "board" ? (
                <EmptyState icon={<Film className="size-4" />} title={copy.video.noClip} body={copy.video.phaseHintBoard} />
              ) : null}
              {pane === "cut" ? (
                <CutInspector
                  bundle={bundle!}
                  sel={selShots}
                  setSel={setSelShots}
                  copy={copy}
                  ffmpeg={ffmpeg}
                  ffmpegInstall={ffmpegInstall}
                  onInstall={() => {
                    void api.video.installFFmpeg().then((s) => {
                      setFfmpeg(!!s?.ffmpeg);
                      if (s?.ffmpeg_install && typeof s.ffmpeg_install === "object") setFfmpegInstall(s.ffmpeg_install);
                    }).catch((e) => toast.error(api.errMessage(e)));
                  }}
                />
              ) : null}
              {ep ? (
                <GenerationSettings
                  ep={ep}
                  dramaStyle={drama?.style || "3d"}
                  styles={styles}
                  providers={providers}
                  copy={copy}
                  onStyle={(style) => { if (dramaId) void api.video.updateDrama({ id: dramaId, style }).then(loadList); }}
                  onEpisode={(patch) => { if (ep) void api.video.updateEpisode({ id: ep.id, ...patch }).then(() => loadBundle(ep.id)); }}
                />
              ) : null}
              <button type="button" className="mt-3 h-6 rounded-md px-2 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground" onClick={() => setKillEp(true)}>
                {copy.video.deleteEpisode}
              </button>
            </aside>
          </div>
          {pane === "board" || pane === "cut" ? (
            <div className="drama-timeline" aria-label={copy.video.board}>
              {(shots.length ? shots : Array.from({ length: Math.max(bundle?.plan.segment_count || 3, 3) }, (_, i) => ({
                id: `ghost-${i}`,
                shot_number: i + 1,
                title: "",
                description: "",
                video_prompt: "",
                duration: 0,
                status: "",
              } as Shot))).map((s, i) => {
                const live = (bundle?.jobs || []).some((j) => j.storyboard_id === s.id && jobMoving(j.status));
                const on = focused?.id === s.id;
                const included = selShots[s.id] !== false && !!s.video_url;
                const ghost = s.id.startsWith("ghost-");
                return (
                  <div
                    key={s.id}
                    draggable={pane === "cut" && !!s.video_url}
                    className={cn("drama-shot", on && "is-on", live && "is-live", pane === "cut" && s.video_url && !included && "is-off")}
                    onDragStart={(e) => { e.dataTransfer.setData("text/plain", String(i)); }}
                    onDragOver={(e) => e.preventDefault()}
                    onDrop={(e) => {
                      e.preventDefault();
                      reorder(Number(e.dataTransfer.getData("text/plain")), i);
                    }}
                  >
                    <button
                      type="button"
                      className="drama-shot-body"
                      disabled={ghost}
                      onClick={() => { if (!ghost) setFocusShot(s.id); }}
                    >
                      {s.poster_url || s.video_url ? (
                        <img src={s.poster_url || ""} alt="" />
                      ) : (
                        <span className="drama-shot-empty grid place-items-center text-[10px]">{String(s.shot_number).padStart(2, "0")}</span>
                      )}
                      <span className="truncate font-mono text-[10px] tabular-nums">{String(s.shot_number).padStart(2, "0")} {s.title}</span>
                    </button>
                    {pane === "cut" && s.video_url ? (
                      <button
                        type="button"
                        className={cn("drama-shot-check", included && "is-on")}
                        aria-pressed={included}
                        aria-label={included ? copy.video.selectNone : copy.video.selectAll}
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelShots({ ...selShots, [s.id]: !included });
                        }}
                      >
                        {included ? <Check className="size-2.5" /> : null}
                      </button>
                    ) : null}
                  </div>
                );
              })}
            </div>
          ) : null}
          {tasksOpen ? (
            <div className="drama-task-drawer">
              <JobDrawer jobs={dramaJobs.length ? dramaJobs : (bundle?.jobs || [])} bundle={bundle} episodes={episodes} copy={copy} onRetry={(id) => void run("retry", () => api.video.retryJob(id))} onCancel={(id) => void run("cancel", () => api.video.cancelJob(id))} />
            </div>
          ) : null}
        </>
      )}
      <ConfirmDialog
        open={creating}
        title={copy.video.newDrama}
        body={
          <span className="grid gap-2 pt-2">
            <label className="grid gap-1">
              <span className="text-[11px] text-muted">{copy.video.createName}</span>
              <Input value={draftTitle} placeholder={copy.video.untitled} onChange={(e) => setDraftTitle(e.target.value)} />
            </label>
            <span className="flex rounded-[8px] bg-lift p-0.5">
              {["9:16", "16:9", "1:1"].map((r) => (
                <button key={r} type="button" className={cn("flex-1 rounded-[6px] px-1.5 py-1 font-mono text-[10px]", ratio === r ? "bg-card text-foreground" : "text-muted")} onClick={() => setRatio(r)}>{r}</button>
              ))}
            </span>
          </span>
        }
        confirmLabel={copy.video.createStart}
        onCancel={() => setCreating(false)}
        onConfirm={() => createDrama()}
      />
      <ConfirmDialog
        open={killEp}
        title={copy.video.deleteEpisode}
        body={copy.video.deleteEpisodeBody}
        danger
        confirmLabel={copy.video.delete}
        onCancel={() => setKillEp(false)}
        onConfirm={async () => {
          if (!episodeId) return;
          await api.video.deleteEpisode(episodeId);
          setEpisodeId("");
          setKillEp(false);
          await loadList();
        }}
      />
      <ConfirmDialog
        open={!!killAsset}
        title={copy.video.delete}
        body={copy.video.confirmDelete}
        danger
        confirmLabel={copy.video.delete}
        onCancel={() => setKillAsset(null)}
        onConfirm={async () => {
          if (!killAsset) return;
          await api.video.deleteAsset(killAsset.kind, killAsset.id);
          setKillAsset(null);
          setFocusAsset({ kind: "", id: "" });
          if (episodeId) await loadBundle(episodeId);
        }}
      />
    </div>
  );
}

function ProjectMenu(props: {
  dramas: Drama[];
  episodes: Episode[];
  dramaId: string;
  episodeId: string;
  copy: ReturnType<typeof useCopy>;
  onPick: (dramaId: string, episodeId: string) => void;
  onNewDrama: () => void;
  onNewEpisode: () => void;
}) {
  const drama = props.dramas.find((d) => d.id === props.dramaId);
  const episode = props.episodes.find((e) => e.id === props.episodeId);
  const label = [
    shownSeriesTitle(drama?.title, props.copy.video.untitled),
    episode ? shownEpisodeTitle(episode.title, episode.episode_number, props.copy.video.episodeN) : "",
  ].filter(Boolean).join(" / ") || props.copy.video.noProject;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="inline-flex h-7 max-w-[12rem] items-center gap-1 rounded-md px-1.5 text-[12px] font-medium hover:bg-lift" aria-label={props.copy.video.pickEpisode}>
          <Film className="size-3 shrink-0 opacity-70" />
          <span className="truncate">{label}</span>
          <ChevronDown className="size-3 shrink-0 opacity-70" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72 max-h-80 overflow-y-auto">
        <DropdownMenuItem onSelect={props.onNewDrama}>
          <Plus className="size-3.5 shrink-0 opacity-70" />
          {props.copy.video.newDrama}
        </DropdownMenuItem>
        {props.dramaId ? (
          <DropdownMenuItem onSelect={props.onNewEpisode}>
            <Film className="size-3.5 shrink-0 opacity-70" />
            {props.copy.video.newEpisode}
          </DropdownMenuItem>
        ) : null}
        {props.dramas.length ? <DropdownMenuSeparator /> : null}
        {props.dramas.map((d) => (
          <DramaGroup key={d.id} drama={d} active={d.id === props.dramaId ? props.episodeId : ""} copy={props.copy} onPick={props.onPick} />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function DramaGroup(props: {
  drama: Drama;
  active: string;
  copy: ReturnType<typeof useCopy>;
  onPick: (dramaId: string, episodeId: string) => void;
}) {
  const [eps, setEps] = useState<Episode[]>([]);
  useEffect(() => {
    void api.video.episodes(props.drama.id).then((list) => setEps(Array.isArray(list) ? list : [])).catch(() => setEps([]));
  }, [props.drama.id]);
  return (
    <>
      <DropdownMenuLabel className="truncate">{shownSeriesTitle(props.drama.title, props.copy.video.untitled)}</DropdownMenuLabel>
      {eps.length ? eps.map((ep) => (
        <DropdownMenuItem key={ep.id} onSelect={() => props.onPick(props.drama.id, ep.id)}>
          <span className="min-w-0 flex-1 truncate pl-1">{shownEpisodeTitle(ep.title, ep.episode_number, props.copy.video.episodeN)}</span>
          {props.active === ep.id ? <Check className="size-3.5 shrink-0" /> : null}
        </DropdownMenuItem>
      )) : (
        <div className="px-2.5 py-1 text-[11px] text-muted">{props.copy.video.untitledEp}</div>
      )}
    </>
  );
}
