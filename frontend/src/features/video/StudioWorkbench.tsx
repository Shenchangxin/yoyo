import { useCallback, useEffect, useState } from "react";
import { ArrowLeft, Check, ChevronDown, ChevronRight, Clapperboard, List, MoreHorizontal, Plus, Radio, RefreshCw, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { writeClipboard } from "../../lib/clipboard";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { ConfirmDialog } from "../ConfirmDialog";
import { CreateShortField, StudioIdChip } from "./studio-chrome";
import { TopicCard } from "./studio-topic-card";
import { StudioProduce } from "./StudioProduce";
import { DeliveryShelf } from "./DeliveryShelf";
import { type StudioCatalog, type StudioProject as ProduceProject } from "./studio-model";
import { studioNext, studioNextCopy, type StudioPane } from "./studio-next";
import { SCORE_KEYS, TOPIC_FILTERS, filmInFlight, topicCanFilm, topicMatchesQuery, trendMatchesQuery, type TopicFilter } from "./studio-topic";
import { asStudioProject, studioSessionId } from "./studio-session";
import { useStudioSelection } from "./workshop-store";
import type { VideoProject } from "../../lib/protocol";

type TrendItem = { title?: string; hot?: string; url?: string };
type TrendGroup = { platform?: string; label?: string; items?: TrendItem[]; ok?: boolean };
type Topic = {
  id: string;
  title: string;
  source_platform?: string;
  source_title?: string;
  angle?: string;
  format?: string;
  note?: string;
  scores?: Record<string, number>;
  status?: string;
  studio_project_id?: string;
  persona_page_id?: string;
  options?: { id: string; title?: string; angle?: string; summary?: string }[];
  updated_at?: string;
};
type StudioShot = { image_hash?: string; video_hash?: string; caption?: string; duration?: number; motion?: string; prompt?: string };
type PublishPlatform = { id: string; label?: string; creator_url?: string; analytics_url?: string };
type Publish = {
  id: string;
  project_id?: string;
  topic_id?: string;
  platform?: string;
  title?: string;
  caption?: string;
  tags?: string;
  media_hash?: string;
  export_path?: string;
  creator_url?: string;
  analytics_url?: string;
  posted_url?: string;
  status?: string;
  note?: string;
  reads?: number;
  likes?: number;
  comments?: number;
};
type StudioProject = {
  id: string;
  title: string;
  topic_id?: string;
  aspect?: string;
  style?: string;
  script?: string;
  script_status?: string;
  skip_audio?: boolean;
  storyboard?: { shots?: StudioShot[]; size?: string };
  narration_hash?: string;
  subtitle_hash?: string;
  bgm_hash?: string;
  result_hash?: string;
  status?: string;
  error?: string;
  media_url?: string;
  poster_url?: string;
  narration_url?: string;
  cover_url?: string;
  cover_hash?: string;
  copy_text?: string;
  cover_urls?: { hash?: string; url?: string }[];
  versions?: { n: number; result_hash?: string }[];
  delivery?: { status?: string; note?: string; copy_text?: string; files?: { kind?: string; hash?: string }[] };
  gate?: string;
  furthest_gate?: string;
  identity?: { character_id?: string; name?: string };
  shots?: { image_hash?: string; video_hash?: string }[];
  auto_phase?: string;
  auto_error?: string;
  template_id?: string;
  target_duration?: number;
  platform?: string;
  poster_hash?: string;
};

const TREND_PLATFORMS = ["weibo", "douyin", "zhihu", "bilibili", "toutiao", "baidu"] as const;
const ASPECTS = ["9:16", "16:9", "1:1"] as const;

function asList<T>(v: unknown): T[] {
  return Array.isArray(v) ? (v as T[]) : [];
}

function shelfFilms(projects: StudioProject[], current: StudioProject | null) {
  const live = new Set(["queued", "script", "voice", "visuals", "compose", "done", "failed"]);
  const rows = projects.filter((p) => !!p.result_hash || live.has(p.auto_phase || ""));
  const top = rows.slice(0, 12);
  if (current?.id && (current.result_hash || current.auto_phase) && !top.some((p) => p.id === current.id)) top.unshift(current);
  return top;
}

export function StudioWorkbench(props: {
  sessionId?: string;
  projects?: VideoProject[];
  onNeedSession: () => void | Promise<string | void>;
  onOpenProject?: (p: VideoProject) => void | Promise<void>;
  onRefreshHistory?: () => void;
  onClose?: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const studioId = useStudioSelection((st) => st.studioId);
  const setStudioId = useStudioSelection((st) => st.setStudioId);
  const setStudioMeta = useStudioSelection((st) => st.setStudioMeta);
  const [pane, setPane] = useState<StudioPane>(studioId ? "assemble" : "radar");
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [platforms, setPlatforms] = useState<string[]>(["weibo", "douyin", "zhihu"]);
  const [trends, setTrends] = useState<TrendGroup[]>([]);
  const [topicStatus, setTopicStatus] = useState<TopicFilter>("");
  const [naming, setNaming] = useState(false);
  const [topics, setTopics] = useState<Topic[]>([]);
  const [draftTitle, setDraftTitle] = useState("");
  const [projects, setProjects] = useState<StudioProject[]>([]);
  const [project, setProject] = useState<StudioProject | null>(null);
  const [kill, setKill] = useState<StudioProject | null>(null);
  const [switchOpen, setSwitchOpen] = useState(false);
  const [killTopic, setKillTopic] = useState<Topic | null>(null);
  const [mediaBase, setMediaBase] = useState("");
  const [pubPlatforms, setPubPlatforms] = useState<PublishPlatform[]>([]);
  const [publishes, setPublishes] = useState<Publish[]>([]);
  const [draftPlatform, setDraftPlatform] = useState("");
  const [preferredPlatform, setPreferredPlatform] = useState("");
  const [draftCaption, setDraftCaption] = useState("");
  const [draftTags, setDraftTags] = useState("");
  const [wrapTitle, setWrapTitle] = useState("");
  const [defaultAspect, setDefaultAspect] = useState<(typeof ASPECTS)[number]>("9:16");
  const [remotionReady, setRemotionReady] = useState(false);
  const [catalog, setCatalog] = useState<StudioCatalog>({});
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [radarLoading, setRadarLoading] = useState(true);

  const current = (project && project.id === studioId ? project : null) || projects.find((p) => p.id === studioId) || null;
  const copilotSessionId = studioSessionId(studioId, props.projects || []);

  async function attachCopilot(p: { id: string; title?: string; status?: string; gate?: string }) {
    if (!p?.id || !props.onOpenProject) return;
    await props.onOpenProject(asStudioProject({
      id: p.id,
      title: p.title || "",
      sessionId: studioSessionId(p.id, props.projects || []),
      status: p.status,
      gate: p.gate,
    }));
  }

  const loadTopics = useCallback(async (status = topicStatus) => {
    const list = await api.video.listTopics(status);
    setTopics(asList<Topic>(list));
  }, [topicStatus]);

  const loadProjects = useCallback(async () => {
    const list = await api.video.listStudio();
    const rows = asList<StudioProject>(list);
    setProjects(rows);
  }, []);

  const loadProject = useCallback(async (id: string) => {
    if (!id) {
      setProject(null);
      return;
    }
    const p = (await api.video.getStudio(id)) as StudioProject;
    if (useStudioSelection.getState().studioId !== id) return;
    setProject(p);
    setProjects((prev) => {
      const rest = prev.filter((x) => x.id !== p.id);
      return [p, ...rest];
    });
  }, []);

  const loadPublishes = useCallback(async (projectId = studioId) => {
    const list = await api.video.listPublishes(projectId || "");
    setPublishes(asList<Publish>(list));
  }, [studioId]);

  useEffect(() => {
    void loadTopics().catch((e) => toast.error(api.errMessage(e)));
    void loadProjects().catch((e) => toast.error(api.errMessage(e)));
    void api.video.publishPlatforms().then((rows) => setPubPlatforms(asList<PublishPlatform>(rows))).catch(() => {});
    void api.video.settings().then((cfg) => {
      const a = String(cfg?.studio_aspect || "9:16");
      setDefaultAspect(a === "16:9" || a === "1:1" ? a : "9:16");
      const p = String(cfg?.studio_publish_platform || "").trim();
      if (p) setPreferredPlatform(p);
    }).catch(() => {});
    void api.video.status().then((st) => {
      setMediaBase(String(st?.media_base || "").replace(/\/+$/, ""));
      setRemotionReady(!!st?.remotion?.ready);
    }).catch(() => {});
    void api.video.fetchTrends("weibo,douyin,zhihu", 15).then((raw) => setTrends(asList<TrendGroup>(raw?.trends))).catch(() => {}).finally(() => setRadarLoading(false));
    void api.video.studioCatalog().then((row) => setCatalog((row || {}) as StudioCatalog)).catch(() => {});
  }, [loadProjects]);

  useEffect(() => {
    if (pane !== "publish") return;
    void loadPublishes().catch((e) => toast.error(api.errMessage(e)));
  }, [pane, loadPublishes]);

  const filmLive = projects.some((p) => filmInFlight(p.auto_phase));
  useEffect(() => {
    if (!filmLive) return;
    const t = window.setInterval(() => {
      void loadProjects();
      const id = useStudioSelection.getState().studioId;
      if (id) void loadProject(id);
      void loadTopics();
    }, 2000);
    return () => window.clearInterval(t);
  }, [filmLive, loadProjects, loadProject, loadTopics]);

  useEffect(() => {
    setDraftPlatform("");
  }, [current?.id]);

  useEffect(() => {
    const text = current?.copy_text || current?.delivery?.copy_text || "";
    if (text) setDraftCaption(text);
  }, [current?.id, current?.copy_text, current?.delivery?.copy_text]);

  useEffect(() => {
    if (!studioId) {
      setProject(null);
      return;
    }
    const id = studioId;
    void loadProject(id).catch(() => {
      if (useStudioSelection.getState().studioId !== id) return;
      setProject(null);
      setStudioId("");
    });
  }, [studioId, loadProject, setStudioId]);

  useEffect(() => {
    if (!studioId) return;
    const hit = (props.projects || []).find((row) => row.kind === "studio" && row.id === studioId);
    void attachCopilot({
      id: studioId,
      title: hit?.title || useStudioSelection.getState().title,
      status: hit?.status,
      gate: hit?.gate,
    }).catch(() => {
      /* copilot bind must not bounce the short back home */
    });
    // Attach once per short. Switching shorts changes studioId; creating one does too.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- attachCopilot is stable enough via studioId
  }, [studioId]);

  useEffect(() => {
    if (studioId) {
      setPane((p) => (p === "radar" || p === "topics" ? "assemble" : p));
      return;
    }
    setPane((p) => (p === "assemble" || p === "publish" ? "radar" : p));
  }, [studioId]);

  useEffect(() => {
    const next = studioNext(current, pane);
    if (!current) {
      setStudioMeta({ pane, gate: "", status: "", nextKey: next.key });
      return;
    }
    setStudioMeta({
      title: current.title || "",
      pane,
      gate: next.gate || current.gate || current.furthest_gate || "",
      status: current.status || "",
      nextKey: next.key,
    });
  }, [current, current?.id, current?.title, current?.gate, current?.furthest_gate, current?.status, current?.script, current?.script_status, current?.narration_hash, current?.skip_audio, current?.result_hash, current?.auto_phase, current?.delivery?.status, pane, setStudioMeta]);

  async function run(label: string, fn: () => Promise<void>) {
    setBusy(label);
    setErr("");
    try {
      await fn();
    } catch (e) {
      const msg = api.errMessage(e);
      setErr(msg);
      toast.error(msg);
    } finally {
      setBusy("");
    }
  }

  async function refreshTrends() {
    await run("radar", async () => {
      const raw = await api.video.fetchTrends(platforms.join(","), 15);
      setTrends(asList<TrendGroup>(raw?.trends));
    });
  }

  async function collect(platform: string, item: TrendItem) {
    if (!item.title) return;
    await run("collect", async () => {
      await api.video.topicFromTrend(platform, item.title || "", item.hot || "");
      await loadTopics();
      setPane("topics");
      toast.success(s.collected);
    });
  }

  async function saveDraftTopic() {
    const title = draftTitle.trim();
    if (!title) return;
    await run("topic", async () => {
      await api.video.saveTopic({ title, format: "short", status: "inbox" });
      setDraftTitle("");
      await loadTopics();
    });
  }

  async function patchTopic(t: Topic, extra: Partial<Topic>) {
    await run("topic", async () => {
      await api.video.saveTopic({ ...t, ...extra });
      await loadTopics();
    });
  }

  async function openShort(p: StudioProject) {
    if (!p?.id) return;
    setStudioId(p.id);
    setProject(p);
    setPane("assemble");
    try {
      await attachCopilot(p);
    } catch {
      /* stay on this short even if the copilot session fails to bind */
    }
  }

  async function openDelivery(id: string) {
    if (!id) return;
    const hit = projects.find((row) => row.id === id);
    setStudioId(id);
    setProject(hit || null);
    setPane("publish");
    setDraftPlatform("");
    try {
      const p = (await api.video.getStudio(id)) as StudioProject;
      if (useStudioSelection.getState().studioId !== id) return;
      setProject(p);
      await attachCopilot(p);
    } catch {
      /* stay on delivery even if the copilot session fails to bind */
    }
  }

  async function filmTopics(ids: string[]) {
    const unique = [...new Set(ids.map((id) => id.trim()).filter(Boolean))];
    if (!unique.length) {
      toast.error(s.needTopicFirst);
      return;
    }
    await run("auto", async () => {
      const rows = asList<StudioProject>(await api.video.autoTopics(unique));
      setPicked(new Set());
      await loadTopics();
      await loadProjects();
      props.onRefreshHistory?.();
      const first = rows.find((row) => filmInFlight(row.auto_phase)) || rows[0];
      if (!first?.id) return;
      setStudioId(first.id);
      setProject(first);
      setPane("publish");
      setDraftPlatform("");
      setDraftCaption(first.copy_text || "");
      try {
        await attachCopilot(first);
      } catch {
        /* the films still land on delivery */
      }
    });
  }

  async function produce(t: Topic) {
    const film = projects.find((p) => p.id === t.studio_project_id);
    if (t.studio_project_id && !topicCanFilm(t, film)) {
      await openDelivery(t.studio_project_id);
      return;
    }
    await filmTopics([t.id]);
  }

  function goHome() {
    setStudioId("");
    setProject(null);
    setPane("topics");
    setNaming(false);
  }

  function togglePick(id: string) {
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function produceSelected() {
    const rows = topics.filter((t) => picked.has(t.id) && topicCanFilm(t, projects.find((p) => p.id === t.studio_project_id)));
    if (!rows.length) {
      toast.error(s.needTopicFirst);
      return;
    }
    await filmTopics(rows.map((t) => t.id));
  }

  async function schedule(t: Topic) {
    await run("schedule", async () => {
      const out = (await api.video.scheduleTopic(t.id)) as { duplicate?: boolean };
      await loadTopics();
      toast.success(out?.duplicate ? s.scheduleExists : s.scheduleQueued);
    });
  }

  async function unschedule(t: Topic) {
    await run("unschedule", async () => {
      await api.video.unscheduleTopic(t.id);
      await loadTopics();
    });
  }

  async function newProject(title: string) {
    const name = title.trim();
    if (!name) {
      toast.error(s.needNamedShort);
      return;
    }
    await run("create", async () => {
      const p = (await api.video.createStudio({ title: name, aspect: defaultAspect })) as StudioProject;
      setNaming(false);
      await openShort(p);
      await loadProjects();
      props.onRefreshHistory?.();
    });
  }

  async function selectProject(id: string) {
    const hit = projects.find((row) => row.id === id);
    setStudioId(id);
    setPane(hit?.result_hash || hit?.auto_phase ? "publish" : "assemble");
    setDraftPlatform("");
    try {
      await attachCopilot({ id, title: hit?.title, status: hit?.status, gate: hit?.gate });
    } catch {
      /* stay on this short even if the copilot session fails to bind */
    }
  }

  function shortLabel(title?: string) {
    return (title || "").trim() || s.newProject;
  }

  function askDelete(p: StudioProject) {
    setSwitchOpen(false);
    setKill(p);
  }

  async function removeShort(id: string) {
    await api.video.deleteStudio(id);
    setKill(null);
    if (studioId === id) goHome();
    await loadProjects();
    props.onRefreshHistory?.();
  }

  async function saveProject(patch: Partial<StudioProject>) {
    if (!current?.id) return;
    await run("save", async () => {
      const next = (await api.video.updateStudio({ id: current.id, ...patch })) as StudioProject;
      setProject(next);
    });
  }

  async function wrapCurrent() {
    if (!current?.id) return;
    const hash = current.result_hash;
    if (!hash) {
      toast.error(s.wrapNeedVideo);
      return;
    }
    const title = wrapTitle.trim() || current.title || s.wrap;
    await run("wrap", async () => {
      const r = await api.video.studioWrap(hash, title);
      if (r?.cas_hash) {
        const next = (await api.video.updateStudio({ id: current.id, result_hash: r.cas_hash })) as StudioProject;
        setProject(next);
      }
    });
  }

  async function saveDraftPublish() {
    if (!current?.id || !current.result_hash) {
      toast.error(s.needAssemble);
      return;
    }
    if (!draftPlatform) {
      toast.error(s.filmPickPlatform);
      return;
    }
    if (current.delivery?.status !== "publishable") {
      toast.error(s.desk.needPack);
      return;
    }
    await run("draft", async () => {
      await api.video.draftPublish({
        platform: draftPlatform,
        title: current.title || s.newProject,
        caption: draftCaption || current.copy_text || current.script || "",
        tags: draftTags,
        project_id: current.id,
        topic_id: current.topic_id || "",
        media_hash: current.result_hash || "",
      });
      await loadPublishes(current.id);
      toast.success(s.draftPublish);
    });
  }

  async function makePersona(topicId = "", name = "") {
    await run("persona", async () => {
      const label = name.trim() || current?.title || s.persona;
      const out = await api.video.createPersona(label, topicId);
      if (topicId) await loadTopics();
      toast.success(out?.title || s.persona);
    });
  }

  function mediaSrc(hash?: string) {
    if (!hash) return "";
    return mediaBase ? `${mediaBase}/media/${hash}` : "";
  }

  function trendLabel(id: string) {
    const labels: Record<(typeof TREND_PLATFORMS)[number], string> = {
      weibo: s.trendWeibo,
      douyin: s.trendDouyin,
      zhihu: s.trendZhihu,
      bilibili: s.trendBilibili,
      toutiao: s.trendToutiao,
      baidu: s.trendBaidu,
    };
    return labels[id as (typeof TREND_PLATFORMS)[number]] || id;
  }

  const scoreLabel: Record<(typeof SCORE_KEYS)[number], string> = {
    heat: s.scoreHeat,
    timing: s.scoreTiming,
    uniqueness: s.scoreUniqueness,
    feasibility: s.scoreFeasibility,
    persona: s.scorePersona,
    conversion: s.scoreConversion,
    risk: s.scoreRisk,
  };

  const panes: { id: StudioPane; label: string; hint: string }[] = [
    { id: "radar", label: s.radar, hint: s.radarHint },
    { id: "topics", label: s.topics, hint: s.topicsHint },
    { id: "assemble", label: s.assemble, hint: s.assembleHint },
    { id: "publish", label: s.publish, hint: s.publishHint },
  ];
  const next = studioNext(current, pane);
  const nextLabel = studioNextCopy(s, next.key);
  const paneDone: Record<StudioPane, boolean> = {
    radar: topics.length > 0,
    topics: !!current,
    assemble: !!current?.result_hash,
    publish: publishes.some((p) => p.status === "posted" || p.status === "opened"),
  };

  const statusLabel: Record<string, string> = {
    inbox: s.inbox,
    ready: s.ready,
    scheduled: s.scheduled,
    in_production: s.inProduction,
    done: s.done,
    draft: s.draft,
    assembled: s.assembled,
    failed: s.failed,
    opened: s.opened,
    posted: s.posted,
  };
  const gateLabel: Record<string, string> = {
    setup: s.desk.setup,
    script: s.desk.script,
    audio: s.desk.audio,
    visuals: s.desk.visuals,
    compose: s.desk.compose,
    delivery: s.desk.delivery,
  };
  const atHome = !studioId;
  const navPanes = panes.filter((p) => atHome ? p.id === "radar" || p.id === "topics" : p.id === "assemble" || p.id === "publish");
  const shownTopics = topics.filter((t) => topicMatchesQuery(t, query));
  const shownTrends = trends
    .map((g) => ({ ...g, items: (g.items || []).filter((item) => trendMatchesQuery(item, query)) }))
    .filter((g) => (g.items || []).length);
  const pickCount = [...picked].filter((id) => topics.some((t) => t.id === id && topicCanFilm(t, projects.find((p) => p.id === t.studio_project_id)))).length;

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="studio-workbench" data-pane={pane} data-bound={current ? "true" : "false"} data-home={atHome ? "true" : "false"}>
      {err ? <div className="border-b border-danger/20 bg-danger/[0.11] px-3 py-1.5 text-[12px] text-danger">{err}</div> : null}
      <header className="flex shrink-0 items-center gap-2 border-b border-border/70 px-3 py-2">
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-0.5">
            {current ? (
              <Button size="sm" variant="ghost" className="h-7 w-7 shrink-0 px-0" data-testid="studio-back-home" title={s.backHome} aria-label={s.backHome} onClick={goHome}>
                <ArrowLeft className="size-3.5" aria-hidden />
              </Button>
            ) : null}
            <h1 className="min-w-0 truncate text-[14px] font-semibold tracking-[-0.03em]" data-testid="studio-workbench-bound">
              {current?.title || copy.video.shellStudio}
            </h1>
            {projects.length > 0 ? (
              <DropdownMenu open={switchOpen} onOpenChange={setSwitchOpen}>
                <DropdownMenuTrigger asChild>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-6 w-6 shrink-0 px-0"
                    aria-label={s.pickShort}
                    title={s.pickShort}
                    data-testid="studio-switch-short"
                  >
                    <ChevronDown className="size-3.5 text-muted" aria-hidden />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-80 max-h-80 overflow-y-auto">
                  <DropdownMenuItem data-testid="studio-switch-new" onSelect={() => setNaming(true)}>
                    <Plus className="size-3.5 shrink-0 opacity-70" aria-hidden />
                    {s.newProject}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuLabel>{s.recentShorts}</DropdownMenuLabel>
                  {projects.map((p) => {
                    const meta = [
                      statusLabel[p.status || ""] || p.status || "",
                      gateLabel[p.gate || p.furthest_gate || ""] || "",
                    ].filter(Boolean).join(" · ");
                    return (
                      <div key={p.id} className="flex items-start gap-0.5">
                        <DropdownMenuItem
                          className="min-w-0 flex-1 items-start"
                          data-testid={`studio-switch-${p.id}`}
                          onSelect={() => { void selectProject(p.id); }}
                        >
                          <span className="min-w-0 flex-1">
                            <span className="block truncate text-[13px]">{shortLabel(p.title)}</span>
                            {meta ? <span className="mt-0.5 block truncate text-[11px] text-muted">{meta}</span> : null}
                          </span>
                          {p.id === studioId ? <Check className="mt-0.5 size-3.5 shrink-0" aria-hidden /> : null}
                        </DropdownMenuItem>
                        <button
                          type="button"
                          className="mt-0.5 grid size-7 shrink-0 place-items-center rounded-md text-muted hover:bg-danger/10 hover:text-danger"
                          data-testid={`studio-delete-${p.id}`}
                          aria-label={s.deleteShort}
                          title={s.deleteShort}
                          onPointerDown={(e) => e.preventDefault()}
                          onClick={() => askDelete(p)}
                        >
                          <Trash2 className="size-3.5" aria-hidden />
                        </button>
                      </div>
                    );
                  })}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null}
          </div>
          <p className="truncate text-[12px] text-muted" data-testid="studio-next">
            {current
              ? [statusLabel[current.status || ""] || current.status, gateLabel[current.gate || current.furthest_gate || ""] || "", pane === next.pane ? "" : nextLabel].filter(Boolean).join(" · ")
              : s.homeHint}
          </p>
          {current?.id ? (
            <StudioIdChip
              projectId={current.id}
              sessionId={copilotSessionId}
              projectLabel={s.projectId}
              sessionLabel={s.sessionId}
              copyLabel={s.copyId}
              copiedLabel={s.copiedId}
              inspectHint={s.inspectHint}
            />
          ) : null}
        </div>
        {!atHome && pane !== next.pane ? (
          <Button
            size="sm"
            variant="default"
            className="h-7 shrink-0 gap-1 text-[12px]"
            data-testid="studio-next-do"
            onClick={() => setPane(next.pane)}
          >
            {s.nextDo}
            <ChevronRight className="size-3.5" aria-hidden />
          </Button>
        ) : null}
        {atHome ? (
          <Button size="sm" variant="ghost" className="h-7 shrink-0 gap-1 text-[12px]" disabled={!!busy} data-testid="studio-new-short" onClick={() => setNaming(true)}>
            <Plus className="size-3.5" aria-hidden />
            {s.newProject}
          </Button>
        ) : null}
        {current ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="h-7 w-7 shrink-0 px-0" data-testid="studio-more" aria-label={copy.video.more}>
                <MoreHorizontal className="size-3.5" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => { void writeClipboard(current.id).then((ok) => { if (ok) toast.success(s.copiedId); }); }}>
                {s.copyId}
              </DropdownMenuItem>
              {copilotSessionId ? (
                <DropdownMenuItem onSelect={() => { void writeClipboard(copilotSessionId).then((ok) => { if (ok) toast.success(s.copiedId); }); }}>
                  {s.sessionId}
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuItem className="text-danger" data-testid="studio-delete-menu" onSelect={() => askDelete(current)}>
                {s.deleteShort}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </header>
      <nav className="flex shrink-0 flex-wrap items-center gap-x-1 gap-y-1 border-b border-border/70 px-2 py-1.5" aria-label={copy.video.shellStudio} data-testid="studio-pipeline">
        {navPanes.map((p, i) => (
          <button
            key={p.id}
            type="button"
            data-testid={`studio-pane-${p.id}`}
            title={p.hint}
            aria-current={pane === p.id ? "page" : undefined}
            className={cn(
              "inline-flex items-center gap-1 rounded-md px-2 py-1 text-[12px] font-medium",
              pane === p.id ? "bg-foreground text-background" : paneDone[p.id] ? "text-foreground hover:bg-lift/70" : "text-muted hover:bg-lift/70 hover:text-foreground",
            )}
            onClick={() => setPane(p.id)}
          >
            <span className="tabular-nums text-[10px] opacity-70">{i + 1}</span>
            {p.label}
          </button>
        ))}
        {atHome ? (
          <div className="relative ml-auto min-w-40 flex-1">
            <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted" aria-hidden />
            <Input
              className="h-7 border-transparent bg-lift/80 pl-7 text-[12px]"
              value={query}
              placeholder={s.searchTopics}
              aria-label={s.searchTopics}
              data-testid="studio-home-search"
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
        ) : null}
        {busy ? <ProgressHairline className={cn("w-24", atHome ? "" : "ml-auto")} indeterminate /> : null}
      </nav>
      {naming ? (
        <div className="shrink-0 border-b border-border/70 px-4 py-3" data-testid="studio-start">
          <CreateShortField
            autoFocus
            disabled={!!busy}
            placeholder={s.createTitle}
            submitLabel={s.create}
            hint={s.startHint}
            testId="studio-create"
            onSubmit={(title) => void newProject(title)}
          />
        </div>
      ) : null}

      <div className="min-h-0 flex-1 overflow-auto px-4 py-3" data-testid={atHome ? "studio-home" : "studio-project"}>
        {pane === "radar" ? (
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="mr-1 text-[12px] text-muted">{s.platforms}</span>
              {TREND_PLATFORMS.map((p) => {
                const on = platforms.includes(p);
                return (
                  <button
                    key={p}
                    type="button"
                    aria-pressed={on}
                    className={cn("rounded-full px-2 py-0.5 text-[11px]", on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
                    onClick={() => setPlatforms((prev) => {
                      if (on && prev.length <= 1) return prev;
                      return on ? prev.filter((x) => x !== p) : [...prev, p];
                    })}
                  >
                    {trendLabel(p)}
                  </button>
                );
              })}
              <Button size="sm" variant="lift" className="ml-auto h-7 gap-1 text-[12px]" disabled={!!busy} onClick={() => void refreshTrends()}>
                <RefreshCw className="size-3.5" aria-hidden />
                {s.refresh}
              </Button>
            </div>
            {radarLoading && shownTrends.length === 0 ? (
              <EmptyState
                icon={<Radio className="size-5" />}
                title={copy.video.loadingRadar}
                body={s.refresh}
              />
            ) : shownTrends.length === 0 ? (
              <EmptyState
                icon={<Radio className="size-5" />}
                title={s.emptyRadar}
                body={s.emptyRadarHint}
                action={
                  <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void refreshTrends()}>
                    {s.refresh}
                  </Button>
                }
              />
            ) : (
              shownTrends.map((g) => (
                <section key={g.platform || g.label} className="overflow-hidden rounded-xl border border-border bg-card">
                  <h2 className="flex items-center gap-2 px-3.5 py-2 text-[12px] font-medium text-muted">
                    {g.label || g.platform}
                    {!g.ok ? <span className="text-danger/80">{s.offline}</span> : null}
                  </h2>
                  <ul className="divide-y divide-border">
                    {(g.items || []).map((item, i) => (
                      <li key={`${g.platform}-${i}`} className="flex items-center gap-3 px-3.5 py-2.5">
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-[13px] text-foreground">{item.title}</p>
                          {item.hot ? <p className="text-[11px] text-muted">{item.hot}</p> : null}
                        </div>
                        <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void collect(g.platform || "", item)}>
                          {s.collect}
                        </Button>
                      </li>
                    ))}
                  </ul>
                </section>
              ))
            )}
          </div>
        ) : null}

        {pane === "topics" ? (
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-1.5" data-testid="studio-topic-filters">
              {TOPIC_FILTERS.map((st) => (
                <button
                  key={st || "all"}
                  type="button"
                  data-testid={`studio-topic-filter-${st || "all"}`}
                  aria-pressed={topicStatus === st}
                  className={cn("rounded-full px-2 py-0.5 text-[11px]", topicStatus === st ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
                  onClick={() => {
                    setTopicStatus(st);
                    void loadTopics(st).catch((e) => toast.error(api.errMessage(e)));
                  }}
                >
                  {st ? statusLabel[st] : s.allTopics}
                </button>
              ))}
            </div>
            <div className="flex gap-2">
              <Input className="h-8" placeholder={s.newTopicPh} value={draftTitle} onChange={(e) => setDraftTitle(e.target.value)} onKeyDown={(e) => e.key === "Enter" && void saveDraftTopic()} />
              <Button size="sm" variant="lift" className="h-8" disabled={!draftTitle.trim() || !!busy} onClick={() => void saveDraftTopic()}>
                {s.newTopic}
              </Button>
            </div>
            {pickCount > 0 ? (
              <div className="flex items-center gap-3 rounded-xl border border-foreground/15 bg-card px-3 py-2.5" data-testid="studio-topic-picked">
                <div className="min-w-0 flex-1">
                  <p className="text-[13px] font-medium text-foreground">{s.selectedCount.replace("{n}", String(pickCount))}</p>
                  <p className="mt-0.5 text-[12px] leading-[1.45] text-muted">{s.filmHint}</p>
                </div>
                <Button size="sm" variant="default" className="h-8 shrink-0 text-[12px]" disabled={!!busy} data-testid="studio-new-from-topics" onClick={() => void produceSelected()}>
                  {s.desk.oneClick}
                </Button>
              </div>
            ) : (
              <p className="text-[12px] leading-[1.5] text-muted">{s.topicsHint}</p>
            )}
            {shownTopics.length === 0 ? (
              <EmptyState
                icon={<List className="size-5" />}
                title={s.emptyTopics}
                body={s.emptyTopicsHint}
                action={
                  <Button size="sm" variant="ghost" onClick={() => setPane("radar")}>
                    {s.radar}
                  </Button>
                }
              />
            ) : (
              <ul className="flex flex-col gap-2">
                {shownTopics.map((t) => (
                  <TopicCard
                    key={t.id}
                    topic={t}
                    busy={!!busy}
                    picked={picked.has(t.id)}
                    statusLabel={statusLabel[t.status || "inbox"] || t.status || ""}
                    deleteLabel={copy.video.delete}
                    scoreLabels={scoreLabel}
                    film={projects.find((p) => p.id === t.studio_project_id)}
                    copy={s}
                    onPick={() => togglePick(t.id)}
                    onProduce={() => void produce(t)}
                    onUnschedule={() => void unschedule(t)}
                    onPatch={(extra) => void patchTopic(t, extra)}
                    onExpand={() => void run("expand", async () => {
                      await api.video.expandTopic(t.id);
                      await loadTopics();
                    })}
                    onSchedule={() => void schedule(t)}
                    onPersona={() => void makePersona(t.id, t.title)}
                    onChoose={(optionId) => void run("choose", async () => {
                      await api.video.chooseTopicOption(t.id, optionId);
                      await loadTopics();
                    })}
                    onDelete={() => setKillTopic(t)}
                  />
                ))}
              </ul>
            )}
          </div>
        ) : null}

        {pane === "assemble" ? (
          <div className="flex flex-col gap-3">
            {!current ? (
              <EmptyState
                icon={<Clapperboard className="size-5" />}
                title={studioId ? copy.video.loadingList : s.emptyProject}
                body={studioId ? undefined : s.emptyProjectHint}
                action={
                  studioId ? undefined : <Button size="sm" variant="ghost" onClick={() => setPane("topics")}>{s.topics}</Button>
                }
              />
            ) : null}
            {current ? (
              <>
                <StudioProduce
                  current={current as ProduceProject}
                  catalog={catalog}
                  busy={busy}
                  mediaSrc={mediaSrc}
                  onChange={(next) => setProject(next as StudioProject)}
                  run={run}
                  saveProject={async (patch) => { await saveProject(patch as Partial<StudioProject>); }}
                  wrapTitle={wrapTitle}
                  setWrapTitle={setWrapTitle}
                  remotionReady={remotionReady}
                  onWrap={() => wrapCurrent()}
                  onPublish={() => setPane("publish")}
                />
              </>
            ) : null}
          </div>
        ) : null}

        {pane === "publish" ? (
          <DeliveryShelf
            films={shelfFilms(projects, current)}
            current={current}
            catalog={catalog}
            busy={busy}
            platforms={pubPlatforms.length ? pubPlatforms : [{ id: "douyin", label: "抖音" }, { id: "xhs", label: "小红书" }, { id: "bilibili", label: "B站" }]}
            platform={draftPlatform}
            suggested={current?.platform || preferredPlatform}
            caption={draftCaption}
            tags={draftTags}
            onSelect={(id) => { void openDelivery(id); }}
            onPlatform={setDraftPlatform}
            onCaption={setDraftCaption}
            onTags={setDraftTags}
            onSave={() => void saveDraftPublish()}
            onRetry={() => {
              if (!current?.id) return;
              void run("auto", async () => {
                const p = (await api.video.autoStudio(current.id)) as StudioProject;
                setProject(p);
                await loadProjects();
              });
            }}
            onEdit={() => setPane("assemble")}
            onPack={() => {
              if (!current?.id) return;
              void run("package", async () => {
                const p = (await api.video.packageStudio(current.id)) as StudioProject;
                setProject(p);
              });
            }}
            onBack={() => {
              setStudioId("");
              setProject(null);
              setPane("topics");
            }}
            mediaSrc={mediaSrc}
            drafts={publishes.length ? (
              <ul className="flex flex-col gap-2">
                {publishes.map((row) => (
                  <PublishDraftCard
                    key={row.id}
                    draft={row}
                    busy={busy}
                    statusLabel={statusLabel[row.status || "draft"] || row.status || ""}
                    onRun={run}
                    onReload={async () => {
                      await loadPublishes();
                      await loadTopics();
                    }}
                  />
                ))}
              </ul>
            ) : null}
          />
        ) : null}
      </div>

      <ConfirmDialog
        open={!!kill}
        title={s.deleteShort}
        body={s.deleteNamed.replace("{title}", shortLabel(kill?.title))}
        danger
        confirmLabel={s.deleteShort}
        onCancel={() => setKill(null)}
        onConfirm={async () => {
          if (!kill?.id) return;
          try {
            await removeShort(kill.id);
          } catch (e) {
            toast.error(api.errMessage(e));
          }
        }}
      />
      <ConfirmDialog
        open={!!killTopic}
        title={copy.video.delete}
        body={s.deleteTopic}
        danger
        confirmLabel={copy.video.delete}
        onCancel={() => setKillTopic(null)}
        onConfirm={async () => {
          if (!killTopic?.id) return;
          try {
            await api.video.deleteTopic(killTopic.id);
            setKillTopic(null);
            await loadTopics();
          } catch (e) {
            toast.error(api.errMessage(e));
          }
        }}
      />
    </div>
  );
}

function PublishDraftCard(props: {
  draft: Publish;
  busy: string;
  statusLabel: string;
  onRun: (label: string, fn: () => Promise<void>) => Promise<void>;
  onReload: () => Promise<void>;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const p = props.draft;
  const [postedUrl, setPostedUrl] = useState(p.posted_url || "");
  const [note, setNote] = useState(p.note || "");
  const [reads, setReads] = useState(p.reads ? String(p.reads) : "");
  const [likes, setLikes] = useState(p.likes ? String(p.likes) : "");
  const [comments, setComments] = useState(p.comments ? String(p.comments) : "");
  return (
    <li className="rounded-xl border border-border bg-card px-3.5 py-3">
      <div className="min-w-0">
        <p className="text-[13px] font-medium text-foreground">{p.title}</p>
        <p className="mt-0.5 text-[12px] text-muted">{[p.platform, props.statusLabel].filter(Boolean).join(" · ")}</p>
        {p.posted_url ? <p className="mt-1 truncate text-[12px] text-muted">{p.posted_url}</p> : null}
        {p.export_path ? <p className="mt-1 truncate text-[11px] text-muted">{p.export_path}</p> : null}
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!props.busy} onClick={() => void props.onRun("export", async () => {
          await api.video.exportPublish(p.id);
          await props.onReload();
        })}>{s.exportMp4}</Button>
        <Button size="sm" variant="default" className="h-7 text-[12px]" disabled={!!props.busy} onClick={() => void props.onRun("open", async () => {
          await api.video.openPublish(p.id);
          await props.onReload();
        })}>{s.openCreator}</Button>
        <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!props.busy} onClick={() => void props.onRun("analytics", async () => {
          await api.video.openPublishAnalytics(p.id);
        })}>{s.openAnalytics}</Button>
        {p.export_path ? (
          <Button size="sm" variant="ghost" className="h-7 text-[12px]" onClick={() => void api.openPath(p.export_path || "")}>{s.reveal}</Button>
        ) : null}
        <Button size="sm" variant="ghost" className="h-7 text-[12px] text-danger" disabled={!!props.busy} onClick={() => void props.onRun("failed", async () => {
          await api.video.markPublish(p.id, "failed", { note });
          await props.onReload();
        })}>{s.markFailed}</Button>
      </div>
      <details className="mt-2" data-testid="studio-record-stats">
        <summary className="cursor-pointer text-[12px] text-muted">{copy.video.recordStats}</summary>
        <p className="mt-1 text-[11px] leading-[1.45] text-muted">{s.analyticsNote}</p>
        <div className="mt-2 grid gap-2 sm:grid-cols-2">
          <Input className="h-8" placeholder={s.postedUrl} value={postedUrl} onChange={(e) => setPostedUrl(e.target.value)} />
          <Input className="h-8" placeholder={s.analyticsNote} value={note} onChange={(e) => setNote(e.target.value)} />
          <Input className="h-8" placeholder={s.reads} value={reads} onChange={(e) => setReads(e.target.value)} />
          <Input className="h-8" placeholder={s.likes} value={likes} onChange={(e) => setLikes(e.target.value)} />
          <Input className="h-8 sm:col-span-2" placeholder={s.comments} value={comments} onChange={(e) => setComments(e.target.value)} />
        </div>
        <Button size="sm" variant="lift" className="mt-2 h-7 text-[12px]" disabled={!!props.busy} onClick={() => void props.onRun("posted", async () => {
          await api.video.markPublish(p.id, "posted", {
            posted_url: postedUrl,
            note,
            reads: Number(reads) || 0,
            likes: Number(likes) || 0,
            comments: Number(comments) || 0,
          });
          await props.onReload();
        })}>{s.markPosted}</Button>
      </details>
    </li>
  );
}
