import { useCallback, useEffect, useState } from "react";
import { ChevronRight, Clapperboard, MoreHorizontal, Plus, Radio, RefreshCw, Settings, Share2, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { writeClipboard } from "../../lib/clipboard";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import * as api from "../../lib/client";
import { ConfirmDialog } from "../ConfirmDialog";
import { CreateShortField, ScorePills, StudioIdChip } from "./studio-chrome";
import { StudioProduce } from "./StudioProduce";
import { type StudioCatalog, type StudioProject as ProduceProject } from "./studio-model";
import { studioNext, studioNextCopy, type StudioPane } from "./studio-next";
import { SCORE_KEYS, TOPIC_FILTERS, topicIsScheduled, topicPrimaryAction, type TopicFilter } from "./studio-topic";
import { useStudioSelection } from "./workshop-store";

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
};

const TREND_PLATFORMS = ["weibo", "douyin", "zhihu", "bilibili", "toutiao", "baidu"] as const;
const ASPECTS = ["9:16", "16:9", "1:1"] as const;

function asList<T>(v: unknown): T[] {
  return Array.isArray(v) ? (v as T[]) : [];
}

export function StudioWorkbench(props: {
  sessionId?: string;
  onNeedSession: () => void | Promise<string | void>;
  onClose?: () => void;
}) {
  const copy = useCopy();
  const s = copy.video.studio;
  const openSettings = useUI((s) => s.openSettings);
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
  const [kill, setKill] = useState(false);
  const [killTopic, setKillTopic] = useState<Topic | null>(null);
  const [mediaBase, setMediaBase] = useState("");
  const [pubPlatforms, setPubPlatforms] = useState<PublishPlatform[]>([]);
  const [publishes, setPublishes] = useState<Publish[]>([]);
  const [draftPlatform, setDraftPlatform] = useState("douyin");
  const [draftCaption, setDraftCaption] = useState("");
  const [draftTags, setDraftTags] = useState("");
  const [wrapTitle, setWrapTitle] = useState("");
  const [defaultAspect, setDefaultAspect] = useState<(typeof ASPECTS)[number]>("9:16");
  const [remotionReady, setRemotionReady] = useState(false);
  const [catalog, setCatalog] = useState<StudioCatalog>({});

  const current = project && project.id === studioId ? project : projects.find((p) => p.id === studioId) || project;

  async function sessionId() {
    if (props.sessionId) return props.sessionId;
    const id = await props.onNeedSession();
    return typeof id === "string" && id ? id : "";
  }

  const loadTopics = useCallback(async (status = topicStatus) => {
    const list = await api.video.listTopics(status);
    setTopics(asList<Topic>(list));
  }, [topicStatus]);

  const loadProjects = useCallback(async () => {
    const list = await api.video.listStudio();
    const rows = asList<StudioProject>(list);
    setProjects(rows);
    const currentId = useStudioSelection.getState().studioId;
    if (currentId && !rows.some((row) => row.id === currentId)) {
      setStudioId("");
      setProject(null);
    }
  }, [setStudioId]);

  const loadProject = useCallback(async (id: string) => {
    if (!id) {
      setProject(null);
      return;
    }
    const p = (await api.video.getStudio(id)) as StudioProject;
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
      if (p) setDraftPlatform(p);
    }).catch(() => {});
    void api.video.status().then((st) => {
      setMediaBase(String(st?.media_base || "").replace(/\/+$/, ""));
      setRemotionReady(!!st?.remotion?.ready);
    }).catch(() => {});
    void api.video.fetchTrends("weibo,douyin,zhihu", 15).then((raw) => setTrends(asList<TrendGroup>(raw?.trends))).catch(() => {});
    void api.video.studioCatalog().then((row) => setCatalog((row || {}) as StudioCatalog)).catch(() => {});
  }, [loadTopics, loadProjects]);

  useEffect(() => {
    if (pane !== "publish") return;
    void loadPublishes().catch((e) => toast.error(api.errMessage(e)));
  }, [pane, loadPublishes]);

  useEffect(() => {
    const text = current?.copy_text || current?.delivery?.copy_text || "";
    if (text) setDraftCaption(text);
  }, [current?.id, current?.copy_text, current?.delivery?.copy_text]);

  useEffect(() => {
    if (!studioId) {
      setProject(null);
      return;
    }
    void loadProject(studioId).catch(() => {
      setProject(null);
      setStudioId("");
    });
  }, [studioId, loadProject, setStudioId]);

  useEffect(() => {
    if (!props.sessionId || !studioId) return;
    void api.video.studioBind(props.sessionId, studioId).catch((e) => {
      if (/not found/i.test(api.errMessage(e))) setStudioId("");
    });
  }, [props.sessionId, studioId, setStudioId]);

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
  }, [current, current?.id, current?.title, current?.gate, current?.furthest_gate, current?.status, current?.script, current?.script_status, current?.narration_hash, current?.skip_audio, current?.result_hash, current?.delivery?.status, pane, setStudioMeta]);

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
    const sid = await sessionId();
    if (sid) await api.video.studioBind(sid, p.id).catch(() => {});
  }

  async function produce(t: Topic) {
    if (t.studio_project_id) {
      await run("open", async () => {
        const p = (await api.video.getStudio(t.studio_project_id || "")) as StudioProject;
        await openShort(p);
        await loadProjects();
      });
      return;
    }
    await run("produce", async () => {
      const sid = await sessionId();
      const p = (await api.video.createStudio({ topic_id: t.id, aspect: defaultAspect, session_id: sid })) as StudioProject;
      await openShort(p);
      await loadTopics();
      await loadProjects();
    });
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
      const sid = await sessionId();
      const p = (await api.video.createStudio({ title: name, aspect: defaultAspect, session_id: sid })) as StudioProject;
      setNaming(false);
      await openShort(p);
      await loadProjects();
    });
  }

  async function selectProject(id: string) {
    setStudioId(id);
    const sid = await sessionId();
    if (sid) await api.video.studioBind(sid, id).catch(() => {});
    setPane("assemble");
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

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="studio-workbench" data-pane={pane}>
      {err ? <div className="border-b border-danger/20 bg-danger/[0.11] px-3 py-1.5 text-[12px] text-danger">{err}</div> : null}
      <header className="flex shrink-0 items-center gap-2 border-b border-border/70 px-3 py-2">
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            {projects.length > 1 ? (
              <label className="min-w-0 flex-1">
                <span className="sr-only">{s.pickShort}</span>
                <select
                  className="h-7 w-full max-w-full truncate rounded-md border-0 bg-transparent text-[14px] font-semibold tracking-[-0.03em] outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
                  value={studioId}
                  aria-label={s.pickShort}
                  data-testid="studio-workbench-bound"
                  onChange={(e) => {
                    if (e.target.value) void selectProject(e.target.value);
                  }}
                >
                  {!studioId ? <option value="">{copy.video.shellStudio}</option> : null}
                  {projects.map((p) => (
                    <option key={p.id} value={p.id}>{p.title}</option>
                  ))}
                </select>
              </label>
            ) : (
              <h1 className="truncate text-[14px] font-semibold tracking-[-0.03em]" data-testid="studio-workbench-bound">{current?.title || copy.video.shellStudio}</h1>
            )}
          </div>
          <p className="truncate text-[12px] text-muted" data-testid="studio-next">
            {[
              current ? (statusLabel[current.status || ""] || current.status) : "",
              current ? (gateLabel[current.gate || current.furthest_gate || ""] || "") : "",
              nextLabel,
            ].filter(Boolean).join(" · ")}
          </p>
          {current?.id ? (
            <StudioIdChip
              projectId={current.id}
              sessionId={props.sessionId}
              projectLabel={s.projectId}
              sessionLabel={s.sessionId}
              copyLabel={s.copyId}
              copiedLabel={s.copiedId}
              inspectHint={s.inspectHint}
            />
          ) : null}
        </div>
        {pane !== next.pane ? (
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
        <Button size="sm" variant="ghost" className="h-7 w-7 shrink-0 px-0" disabled={!!busy} title={s.newProject} aria-label={s.newProject} data-testid="studio-new-short" onClick={() => setNaming((v) => !v)}>
          <Plus className="size-3.5" aria-hidden />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          className="h-7 w-7 shrink-0 px-0"
          data-testid="studio-open-settings"
          title={copy.video.openWorkshopSettings}
          aria-label={copy.video.openWorkshopSettings}
          onClick={() => openSettings("generation", "generation-studio")}
        >
          <Settings className="size-3.5" aria-hidden />
        </Button>
        {current ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="h-7 w-7 shrink-0 px-0" aria-label={copy.video.delete}>
                <MoreHorizontal className="size-3.5" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => { void writeClipboard(current.id).then((ok) => { if (ok) toast.success(s.copiedId); }); }}>
                {s.copyId}
              </DropdownMenuItem>
              {props.sessionId ? (
                <DropdownMenuItem onSelect={() => { void writeClipboard(props.sessionId || "").then((ok) => { if (ok) toast.success(s.copiedId); }); }}>
                  {s.sessionId}
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuItem className="text-danger" onSelect={() => setKill(true)}>
                {copy.video.delete}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </header>
      <nav className="flex shrink-0 items-center gap-0.5 border-b border-border/70 px-2 py-1.5" aria-label={copy.video.shellStudio} data-testid="studio-pipeline">
        {panes.map((p, i) => (
          <span key={p.id} className="flex min-w-0 items-center">
            {i > 0 ? <span className="mx-0.5 text-[11px] text-muted/35" aria-hidden>/</span> : null}
            <button
              type="button"
              data-testid={`studio-pane-${p.id}`}
              title={p.hint}
              className={cn(
                "inline-flex items-center gap-1 rounded-md px-2 py-1 text-[12px] font-medium",
                pane === p.id ? "bg-foreground text-background" : paneDone[p.id] ? "text-foreground hover:bg-lift/70" : "text-muted hover:bg-lift/70 hover:text-foreground",
              )}
              onClick={() => setPane(p.id)}
            >
              <span className="tabular-nums text-[10px] opacity-70">{i + 1}</span>
              {p.label}
            </button>
          </span>
        ))}
        {busy ? <ProgressHairline className="ml-auto w-24" indeterminate /> : null}
      </nav>
      {naming ? (
        <div className="shrink-0 border-b border-border/70 px-4 py-2">
          <CreateShortField
            autoFocus
            disabled={!!busy}
            placeholder={s.createTitle}
            submitLabel={s.create}
            hint={s.createTitleHint}
            testId="studio-create"
            onSubmit={(title) => void newProject(title)}
          />
        </div>
      ) : null}

      <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
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
            {trends.length === 0 ? (
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
              trends.map((g) => (
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
              <Input className="h-8" placeholder={s.title} value={draftTitle} onChange={(e) => setDraftTitle(e.target.value)} onKeyDown={(e) => e.key === "Enter" && void saveDraftTopic()} />
              <Button size="sm" variant="lift" className="h-8" disabled={!draftTitle.trim() || !!busy} onClick={() => void saveDraftTopic()}>
                {s.newTopic}
              </Button>
            </div>
            {topics.length === 0 ? (
              <EmptyState
                icon={<Sparkles className="size-5" />}
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
                {topics.map((t) => {
                  const primary = topicPrimaryAction(t);
                  const queued = topicIsScheduled(t);
                  return (
                  <li key={`${t.id}-${t.updated_at || ""}`} className="rounded-xl border border-border bg-card px-3.5 py-3" data-testid={`studio-topic-${t.id}`} data-status={t.status || "inbox"}>
                    <div className="flex items-start gap-2">
                      <div className="min-w-0 flex-1">
                        <p className="text-[13px] font-medium text-foreground">{t.title}</p>
                        <p className="mt-0.5 text-[12px] text-muted">
                          {[t.source_platform, statusLabel[t.status || "inbox"], t.angle].filter(Boolean).join(" · ")}
                        </p>
                        {t.note ? <p className="mt-1 line-clamp-2 text-[12px] leading-[1.5] text-muted">{t.note}</p> : null}
                      </div>
                      <div className="flex shrink-0 flex-col items-end gap-1">
                        <Button size="sm" variant="default" className="h-7 text-[12px]" disabled={!!busy} data-testid={primary === "open" ? "studio-topic-open" : "studio-topic-produce"} onClick={() => void produce(t)}>
                          {primary === "open" ? s.openShort : s.produce}
                        </Button>
                        {queued ? (
                          <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} data-testid="studio-topic-unschedule" onClick={() => void unschedule(t)}>
                            {s.unschedule}
                          </Button>
                        ) : null}
                      </div>
                    </div>
                    {!t.angle && t.status !== "done" && t.status !== "in_production" ? (
                      <Input
                        className="mt-2 h-8"
                        placeholder={s.writeAngle}
                        defaultValue=""
                        data-testid="studio-topic-angle"
                        onBlur={(e) => {
                          const v = e.target.value.trim();
                          if (v) void patchTopic(t, { angle: v, status: queued ? "scheduled" : "ready" });
                        }}
                      />
                    ) : null}
                    {queued ? <p className="mt-2 text-[11px] text-muted">{s.scheduleHint}</p> : null}
                    <ScorePills
                      scores={t.scores}
                      labels={scoreLabel}
                      avgLabel={s.scoreAvg}
                      disabled={!!busy}
                      onChange={(scores) => void patchTopic(t, { scores })}
                    />
                    <details className="mt-1">
                      <summary className="cursor-pointer text-[12px] text-muted" data-testid="studio-topic-more">{s.moreTopic}</summary>
                      <div className="mt-2 flex flex-wrap gap-1.5">
                        <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void run("expand", async () => {
                          await api.video.expandTopic(t.id);
                          await loadTopics();
                        })}>
                          {s.desk.expand}
                        </Button>
                        {!queued && primary === "produce" ? (
                          <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} data-testid="studio-topic-schedule" onClick={() => void schedule(t)}>
                            {s.schedule}
                          </Button>
                        ) : null}
                        <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void makePersona(t.id, t.title)}>
                          {s.persona}
                        </Button>
                        {t.angle ? (
                          <Input
                            className="h-8 min-w-40 flex-1"
                            placeholder={s.angle}
                            defaultValue={t.angle}
                            onBlur={(e) => {
                              const v = e.target.value.trim();
                              if (v !== (t.angle || "")) void patchTopic(t, { angle: v, status: t.status === "inbox" ? "ready" : t.status });
                            }}
                          />
                        ) : null}
                        <Button size="sm" variant="ghost" className="h-7 text-[12px] text-danger" disabled={!!busy} onClick={() => setKillTopic(t)}>
                          {copy.video.delete}
                        </Button>
                      </div>
                      {t.persona_page_id ? <p className="mt-1 text-[11px] text-muted">{s.personaHint}</p> : null}
                    </details>
                    {(t.options || []).length ? (
                      <ul className="mt-2 divide-y divide-border/70 border-t border-border/70" data-testid="studio-topic-options">
                        {t.options!.map((opt) => (
                          <li key={opt.id} className="flex items-start gap-2 py-2">
                            <div className="min-w-0 flex-1">
                              <p className="text-[12px] font-medium text-foreground">{opt.title}</p>
                              <p className="text-[11px] leading-[1.45] text-muted">{[opt.angle, opt.summary].filter(Boolean).join(" · ")}</p>
                            </div>
                            <Button size="sm" variant="ghost" className="h-7 shrink-0 text-[12px]" disabled={!!busy} onClick={() => void run("choose", async () => {
                              await api.video.chooseTopicOption(t.id, opt.id);
                              await loadTopics();
                            })}>{s.desk.choose}</Button>
                          </li>
                        ))}
                      </ul>
                    ) : null}
                  </li>
                  );
                })}
              </ul>
            )}
          </div>
        ) : null}

        {pane === "assemble" ? (
          <div className="flex flex-col gap-3">
            {!current ? (
              <EmptyState
                icon={<Clapperboard className="size-5" />}
                title={s.emptyProject}
                body={s.emptyProjectHint}
                action={
                  <div className="flex w-full max-w-sm flex-col gap-2">
                    <CreateShortField
                      disabled={!!busy}
                      placeholder={s.createTitle}
                      submitLabel={s.create}
                      testId="studio-assemble-create"
                      onSubmit={(title) => void newProject(title)}
                    />
                    <Button size="sm" variant="ghost" onClick={() => setPane("topics")}>{s.topics}</Button>
                  </div>
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
          <div className="flex flex-col gap-3">
            <p className="text-[12px] leading-[1.5] text-muted">{s.neverPublish}</p>
            {current ? (
              <div className="flex flex-col gap-2" data-testid="studio-publish-pack">
                <p className={cn("text-[12px]", current.delivery?.status === "publishable" ? "text-success" : "text-warning")}>
                  {current.delivery?.status === "publishable" ? s.desk.publishable : (current.delivery?.note || s.desk.draftPack)}
                </p>
                {current.delivery?.status !== "publishable" ? (
                  <Button size="sm" variant="default" className="self-start" onClick={() => setPane("assemble")}>{s.nextDo}</Button>
                ) : null}
                {current.media_url ? <video className="max-h-48 w-full rounded-xl bg-black" src={current.media_url} poster={current.cover_url || current.poster_url || undefined} controls /> : null}
                {current.copy_text ? <pre className="whitespace-pre-wrap text-[12px] leading-[1.55] text-muted">{current.copy_text}</pre> : null}
              </div>
            ) : null}
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="mr-1 text-[12px] text-muted">{s.platforms}</span>
              {(pubPlatforms.length ? pubPlatforms : [{ id: "douyin", label: "抖音" }, { id: "xhs", label: "小红书" }, { id: "bilibili", label: "B站" }]).map((p) => (
                <button
                  key={p.id}
                  type="button"
                  className={cn("rounded-full px-2 py-0.5 text-[11px]", draftPlatform === p.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
                  onClick={() => setDraftPlatform(p.id)}
                >
                  {p.label || p.id}
                </button>
              ))}
            </div>
            <Input className="h-8" placeholder={s.caption} value={draftCaption} onChange={(e) => setDraftCaption(e.target.value)} />
            <Input className="h-8" placeholder={s.tags} value={draftTags} onChange={(e) => setDraftTags(e.target.value)} />
            <Button size="sm" variant="lift" className="self-start" disabled={!!busy || current?.delivery?.status !== "publishable"} onClick={() => void saveDraftPublish()}>{s.draftPublish}</Button>
            {publishes.length === 0 ? (
              <EmptyState
                icon={<Share2 className="size-5" />}
                title={s.emptyPublish}
                body={s.emptyPublishHint}
                action={
                  <Button size="sm" variant="ghost" onClick={() => setPane("assemble")}>
                    {s.assemble}
                  </Button>
                }
              />
            ) : (
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
            )}
          </div>
        ) : null}
      </div>

      <ConfirmDialog
        open={kill}
        title={copy.video.delete}
        body={s.deleteProject}
        danger
        confirmLabel={copy.video.delete}
        onCancel={() => setKill(false)}
        onConfirm={async () => {
          if (!current?.id) return;
          await api.video.deleteStudio(current.id);
          setKill(false);
          setStudioId("");
          setProject(null);
          await loadProjects();
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
          await api.video.deleteTopic(killTopic.id);
          setKillTopic(null);
          await loadTopics();
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
