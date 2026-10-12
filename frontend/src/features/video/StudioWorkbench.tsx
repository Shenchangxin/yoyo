import { useCallback, useEffect, useState } from "react";
import { Clapperboard, Film, ImagePlus, Plus, Radio, RefreshCw, Settings, Share2, Sparkles, Trash2, Video } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import * as api from "../../lib/client";
import { ConfirmDialog } from "../ConfirmDialog";
import { PRESET_STYLES, styleLabel } from "./drama-lib";
import { useStudioSelection } from "./workshop-store";

type StudioPane = "radar" | "topics" | "assemble" | "publish";

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
};

const TREND_PLATFORMS = ["weibo", "douyin", "zhihu", "bilibili", "toutiao", "baidu"] as const;
const SCORE_KEYS = ["heat", "timing", "uniqueness", "feasibility", "persona", "conversion", "risk"] as const;
const ASPECTS = ["9:16", "16:9", "1:1"] as const;

function asList<T>(v: unknown): T[] {
  return Array.isArray(v) ? (v as T[]) : [];
}

function readDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result || ""));
    r.onerror = () => reject(r.error);
    r.readAsDataURL(file);
  });
}

function shotReady(sh: StudioShot) {
  return !!(sh.image_hash || sh.video_hash);
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
  const [pane, setPane] = useState<StudioPane>("radar");
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [platforms, setPlatforms] = useState<string[]>(["weibo", "douyin", "zhihu"]);
  const [trends, setTrends] = useState<TrendGroup[]>([]);
  const [topicStatus, setTopicStatus] = useState("");
  const [topics, setTopics] = useState<Topic[]>([]);
  const [draftTitle, setDraftTitle] = useState("");
  const [projects, setProjects] = useState<StudioProject[]>([]);
  const [project, setProject] = useState<StudioProject | null>(null);
  const [kill, setKill] = useState(false);
  const [killTopic, setKillTopic] = useState<Topic | null>(null);
  const [mediaBase, setMediaBase] = useState("");
  const [opsHash, setOpsHash] = useState("");
  const [opsSub, setOpsSub] = useState("");
  const [opsResult, setOpsResult] = useState<{ media_url?: string; cas_hash?: string } | null>(null);
  const [pubPlatforms, setPubPlatforms] = useState<PublishPlatform[]>([]);
  const [publishes, setPublishes] = useState<Publish[]>([]);
  const [draftPlatform, setDraftPlatform] = useState("douyin");
  const [draftCaption, setDraftCaption] = useState("");
  const [draftTags, setDraftTags] = useState("");
  const [postedUrl, setPostedUrl] = useState("");
  const [pubNote, setPubNote] = useState("");
  const [reads, setReads] = useState("");
  const [likes, setLikes] = useState("");
  const [comments, setComments] = useState("");
  const [wrapTitle, setWrapTitle] = useState("");
  const [defaultAspect, setDefaultAspect] = useState<(typeof ASPECTS)[number]>("9:16");
  const [styles, setStyles] = useState<{ value: string; name: string }[]>(PRESET_STYLES.map((row) => ({ value: row.value, name: row.name })));
  const [remotionReady, setRemotionReady] = useState(false);

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
    void api.video.styles().then((list) => {
      const rows = asList<{ value?: string; name?: string }>(list).filter((row) => row.value);
      if (rows.length) setStyles(rows.map((row) => ({ value: String(row.value), name: String(row.name || row.value) })));
    }).catch(() => {});
    void api.video.fetchTrends("weibo,douyin,zhihu", 15).then((raw) => setTrends(asList<TrendGroup>(raw?.trends))).catch(() => {});
  }, [loadTopics, loadProjects]);

  useEffect(() => {
    if (pane !== "publish") return;
    void loadPublishes().catch((e) => toast.error(api.errMessage(e)));
  }, [pane, loadPublishes]);

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

  async function produce(t: Topic) {
    await run("produce", async () => {
      const sid = await sessionId();
      const p = (await api.video.createStudio({ topic_id: t.id, aspect: defaultAspect, session_id: sid })) as StudioProject;
      if (p?.id) {
        setStudioId(p.id);
        setProject(p);
        setPane("assemble");
      }
      await loadTopics();
      await loadProjects();
    });
  }

  async function schedule(t: Topic) {
    await run("schedule", async () => {
      const sid = await sessionId();
      const out = (await api.video.scheduleTopic(t.id, sid, defaultAspect)) as { studio?: StudioProject };
      if (out?.studio?.id) {
        setStudioId(out.studio.id);
        setProject(out.studio);
        setPane("assemble");
      }
      await loadTopics();
      await loadProjects();
      toast.success(s.schedule);
    });
  }

  async function newProject() {
    await run("create", async () => {
      const sid = await sessionId();
      const p = (await api.video.createStudio({ title: s.newProject, aspect: defaultAspect, session_id: sid })) as StudioProject;
      if (p?.id) {
        setStudioId(p.id);
        setProject(p);
        setPane("assemble");
      }
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

  async function speak() {
    if (!current?.id) return;
    if (!String(current.script || "").trim()) {
      toast.error(s.needScript);
      return;
    }
    await run("speak", async () => {
      const next = (await api.video.speakStudio(current.id, current.script || "")) as StudioProject;
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
      setOpsResult(r);
      if (r?.cas_hash) {
        const next = (await api.video.updateStudio({ id: current.id, result_hash: r.cas_hash })) as StudioProject;
        setProject(next);
      }
    });
  }

  async function assemble() {
    if (!current?.id) return;
    const list = current.storyboard?.shots || [];
    if (!list.some(shotReady)) {
      toast.error(s.needShots);
      return;
    }
    await run("assemble", async () => {
      const next = (await api.video.assembleStudio(current.id)) as StudioProject;
      setProject(next);
      await loadProjects();
      if (next?.result_hash) setPane("publish");
    });
  }

  async function saveDraftPublish() {
    if (!current?.id || !current.result_hash) {
      toast.error(s.needAssemble);
      return;
    }
    await run("draft", async () => {
      await api.video.draftPublish({
        platform: draftPlatform,
        title: current.title || s.newProject,
        caption: draftCaption || current.script || "",
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

  const shots = current?.storyboard?.shots || [];

  function setShots(next: StudioShot[], persist = false) {
    if (!current) return;
    const storyboard = { ...(current.storyboard || {}), shots: next };
    setProject({ ...current, storyboard });
    if (persist) void saveProject({ storyboard });
  }

  function patchShot(i: number, extra: Partial<StudioShot>, persist = false) {
    setShots(shots.map((sh, n) => (n === i ? { ...sh, ...extra } : sh)), persist);
  }

  function addShot(extra: Partial<StudioShot> = {}) {
    setShots([...shots, { duration: 3, caption: "", ...extra }], true);
  }

  function removeShot(i: number) {
    setShots(shots.filter((_, n) => n !== i), true);
  }

  async function uploadShot(kind: "image" | "video", file: File, index = -1) {
    await run("upload", async () => {
      const data = await readDataURL(file);
      const put = await api.video.putStudioMedia(data, file.type, kind);
      const hash = String(put?.cas_hash || put?.CASHash || "");
      if (!hash) throw new Error(s.needShots);
      const extra: Partial<StudioShot> = kind === "video" ? { video_hash: hash, image_hash: "" } : { image_hash: hash, video_hash: "" };
      if (index >= 0) patchShot(index, extra, true);
      else addShot({ ...extra, duration: kind === "video" ? 5 : 3 });
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

  const panes: { id: StudioPane; label: string; hint: string; icon: typeof Radio }[] = [
    { id: "radar", label: s.radar, hint: s.radarHint, icon: Radio },
    { id: "topics", label: s.topics, hint: s.topicsHint, icon: Sparkles },
    { id: "assemble", label: s.assemble, hint: s.assembleHint, icon: Film },
    { id: "publish", label: s.publish, hint: s.publishHint, icon: Share2 },
  ];

  const statusLabel: Record<string, string> = {
    inbox: s.inbox,
    ready: s.ready,
    in_production: s.inProduction,
    done: s.done,
    draft: s.draft,
    assembled: s.assembled,
    failed: s.failed,
    opened: s.opened,
    posted: s.posted,
  };

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="studio-workbench" data-pane={pane}>
      {err ? <div className="border-b border-danger/20 bg-danger/[0.11] px-3 py-1.5 text-[12px] text-danger">{err}</div> : null}
      <header className="flex shrink-0 items-center gap-2 border-b border-border/70 px-3 py-2">
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-[14px] font-semibold tracking-[-0.03em]">{current?.title || copy.video.shellStudio}</h1>
          <p className="truncate text-[12px] text-muted">{panes.find((p) => p.id === pane)?.hint}</p>
        </div>
        <Button
          size="sm"
          variant="ghost"
          className="h-7 w-7 px-0"
          data-testid="studio-open-settings"
          title={copy.video.openWorkshopSettings}
          aria-label={copy.video.openWorkshopSettings}
          onClick={() => openSettings("generation", "generation-studio")}
        >
          <Settings className="size-3.5" aria-hidden />
        </Button>
        <Button size="sm" variant="ghost" className="h-7 gap-1 text-[12px]" disabled={!!busy} onClick={() => void newProject()}>
          <Plus className="size-3.5" aria-hidden />
          {s.newProject}
        </Button>
      </header>
      <div className="flex shrink-0 items-center gap-1 border-b border-border/70 px-2 py-1.5">
        {panes.map((p) => (
          <button
            key={p.id}
            type="button"
            data-testid={`studio-pane-${p.id}`}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[12px] font-medium",
              pane === p.id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
            )}
            onClick={() => setPane(p.id)}
          >
            <p.icon className="size-3.5 opacity-70" aria-hidden />
            {p.label}
          </button>
        ))}
        {busy ? <ProgressHairline className="ml-auto w-24" indeterminate /> : null}
      </div>

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
            <button type="button" className="self-start text-[12px] text-muted underline" title={s.searchKeysHint} onClick={() => openSettings("generation", "generation-search")}>
              {s.searchKeys}
            </button>
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
            <div className="flex flex-wrap items-center gap-1.5">
              {(["", "inbox", "ready", "in_production", "done"] as const).map((st) => (
                <button
                  key={st || "all"}
                  type="button"
                  className={cn("rounded-full px-2 py-0.5 text-[11px]", topicStatus === st ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
                  onClick={() => {
                    setTopicStatus(st);
                    void loadTopics(st).catch((e) => toast.error(api.errMessage(e)));
                  }}
                >
                  {st ? statusLabel[st] : s.topics}
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
                {topics.map((t) => (
                  <li key={`${t.id}-${t.updated_at || ""}`} className="rounded-xl border border-border bg-card px-3.5 py-3">
                    <div className="flex items-start gap-2">
                      <div className="min-w-0 flex-1">
                        <p className="text-[13px] font-medium text-foreground">{t.title}</p>
                        <p className="mt-0.5 text-[12px] text-muted">
                          {[t.source_platform, statusLabel[t.status || "inbox"], t.angle, t.persona_page_id ? s.persona : ""].filter(Boolean).join(" · ")}
                        </p>
                        {t.note ? <p className="mt-1 line-clamp-3 text-[12px] leading-[1.5] text-muted">{t.note}</p> : null}
                      </div>
                      <div className="flex shrink-0 flex-col gap-1">
                        <Button size="sm" variant="lift" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void produce(t)}>
                          {s.produce}
                        </Button>
                        <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void schedule(t)}>
                          {s.schedule}
                        </Button>
                        <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void makePersona(t.id, t.title)}>
                          {s.persona}
                        </Button>
                        <Button size="sm" variant="ghost" className="h-7 text-[12px] text-danger" disabled={!!busy} onClick={() => setKillTopic(t)}>
                          {copy.video.delete}
                        </Button>
                      </div>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {SCORE_KEYS.map((key) => (
                        <label key={key} className="inline-flex items-center gap-1 text-[11px] text-muted">
                          {scoreLabel[key]}
                          <input
                            type="number"
                            min={1}
                            max={5}
                            className="h-6 w-10 rounded-md border border-border bg-background px-1 text-[12px] text-foreground"
                            defaultValue={t.scores?.[key] || ""}
                            onBlur={(e) => {
                              const n = Number(e.target.value);
                              const cur = t.scores?.[key] || 0;
                              if (!Number.isFinite(n) || n === cur) return;
                              void patchTopic(t, { scores: { ...(t.scores || {}), [key]: n } });
                            }}
                          />
                        </label>
                      ))}
                    </div>
                    <Input
                      className="mt-2 h-8"
                      placeholder={s.angle}
                      defaultValue={t.angle || ""}
                      onBlur={(e) => {
                        const v = e.target.value.trim();
                        if (v !== (t.angle || "")) void patchTopic(t, { angle: v, status: t.status === "inbox" ? "ready" : t.status });
                      }}
                    />
                  </li>
                ))}
              </ul>
            )}
          </div>
        ) : null}

        {pane === "assemble" ? (
          <div className="flex flex-col gap-3">
            {projects.length ? (
              <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
                {projects.map((p) => (
                  <li key={p.id}>
                    <button
                      type="button"
                      className={cn("flex w-full items-center gap-3 px-3.5 py-2.5 text-left text-[13px]", p.id === studioId ? "bg-lift" : "u-row-hover")}
                      onClick={() => void selectProject(p.id)}
                    >
                      <Film className="size-4 shrink-0 opacity-70" aria-hidden />
                      <span className="min-w-0 flex-1 truncate">{p.title}</span>
                      <span className="text-[11px] text-muted">{statusLabel[p.status || "draft"] || p.status}</span>
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <EmptyState
                icon={<Clapperboard className="size-5" />}
                title={s.emptyProject}
                body={s.emptyProjectHint}
                action={
                  <div className="flex flex-wrap justify-center gap-2">
                    <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void newProject()}>{s.newProject}</Button>
                    <Button size="sm" variant="ghost" onClick={() => setPane("topics")}>{s.topics}</Button>
                  </div>
                }
              />
            )}
            {current ? (
              <>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-[12px] text-muted">{s.aspect}</span>
                  <div className="flex rounded-lg bg-lift p-0.5 text-[12px]">
                    {ASPECTS.map((a) => (
                      <button
                        key={a}
                        type="button"
                        className={cn("rounded-md px-2 py-1", (current.aspect || defaultAspect) === a ? "bg-card text-foreground" : "text-muted")}
                        onClick={() => void saveProject({ aspect: a })}
                      >
                        {a}
                      </button>
                    ))}
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="mr-1 text-[12px] text-muted">{copy.video.style}</span>
                  {styles.map((row) => (
                    <button
                      key={row.value}
                      type="button"
                      className={cn("rounded-full px-2 py-0.5 text-[11px]", (current.style || "3d") === row.value ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
                      onClick={() => void saveProject({ style: row.value })}
                    >
                      {styleLabel(copy.video.styleNames, row.value, row.name)}
                    </button>
                  ))}
                </div>
                <label className="flex flex-col gap-1 text-[12px] font-medium text-foreground">
                  {s.script}
                  <textarea
                    className="no-drag min-h-28 w-full rounded-lg border border-border bg-background px-2.5 py-2 text-[13px] leading-[1.55] text-foreground outline-none focus-visible:border-accent/45 focus-visible:ring-2 focus-visible:ring-accent/20"
                    value={current.script || ""}
                    onChange={(e) => setProject({ ...current, script: e.target.value })}
                    onBlur={(e) => void saveProject({ script: e.target.value })}
                  />
                </label>
                <div className="flex flex-wrap items-center gap-2">
                  <Button size="sm" variant="lift" disabled={!!busy || !String(current.script || "").trim()} onClick={() => void speak()}>{s.speak}</Button>
                  <Button size="sm" variant="lift" disabled={!!busy || !shots.some(shotReady)} onClick={() => void assemble()}>{s.assembleBtn}</Button>
                  <button type="button" className="text-[12px] text-muted underline" onClick={() => openSettings("generation", "generation-speech")}>
                    {s.speechSettings}
                  </button>
                  <Button size="sm" variant="ghost" className="ml-auto text-danger" disabled={!!busy} onClick={() => setKill(true)}>{copy.video.delete}</Button>
                </div>
                {current.error ? <p className="text-[12px] text-danger">{current.error}</p> : null}
                {current.media_url ? (
                  <video className="max-h-80 w-full rounded-xl bg-black" src={current.media_url} poster={current.poster_url || undefined} controls />
                ) : null}
                {current.result_hash ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <Input className="min-w-40 flex-1" placeholder={s.wrap} value={wrapTitle} onChange={(e) => setWrapTitle(e.target.value)} aria-label={s.wrap} />
                    <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void wrapCurrent()}>{s.wrap}</Button>
                    {remotionReady ? (
                      <span className="text-[11px] text-muted">{s.remotionReady}</span>
                    ) : (
                      <button type="button" className="text-[12px] text-muted underline" onClick={() => openSettings("generation", "generation-studio")}>
                        {s.remotionInstallHint}
                      </button>
                    )}
                  </div>
                ) : null}
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="text-[12px] font-medium text-muted">{s.shots}</h2>
                  <label className="ml-auto inline-flex h-7 cursor-pointer items-center gap-1 rounded-md px-2 text-[12px] text-muted hover:bg-lift hover:text-foreground">
                    <ImagePlus className="size-3.5" aria-hidden />
                    {s.addStill}
                    <input type="file" accept="image/*" className="hidden" onChange={(e) => {
                      const f = e.target.files?.[0];
                      e.target.value = "";
                      if (f) void uploadShot("image", f);
                    }} />
                  </label>
                  <label className="inline-flex h-7 cursor-pointer items-center gap-1 rounded-md px-2 text-[12px] text-muted hover:bg-lift hover:text-foreground">
                    <Video className="size-3.5" aria-hidden />
                    {s.addClip}
                    <input type="file" accept="video/*" className="hidden" onChange={(e) => {
                      const f = e.target.files?.[0];
                      e.target.value = "";
                      if (f) void uploadShot("video", f);
                    }} />
                  </label>
                </div>
                {shots.length === 0 ? <p className="text-[12px] text-muted">{s.needShots}</p> : null}
                <ul className="flex flex-col gap-2">
                  {shots.map((sh, i) => {
                    const src = mediaSrc(sh.image_hash || sh.video_hash);
                    return (
                      <li key={`${i}-${sh.image_hash || ""}-${sh.video_hash || ""}`} className="rounded-xl border border-border bg-card p-3">
                        <div className="flex items-start gap-3">
                          {src ? (
                            sh.video_hash ? (
                              <video className="h-16 w-16 shrink-0 rounded-md bg-black object-cover" src={src} muted />
                            ) : (
                              <img className="h-16 w-16 shrink-0 rounded-md object-cover" src={src} alt="" />
                            )
                          ) : (
                            <div className="grid h-16 w-16 shrink-0 place-items-center rounded-md bg-lift text-muted">
                              <Film className="size-4" aria-hidden />
                            </div>
                          )}
                          <div className="grid min-w-0 flex-1 gap-2 sm:grid-cols-2">
                            <Input placeholder={s.caption} value={sh.caption || ""} onChange={(e) => patchShot(i, { caption: e.target.value })} onBlur={() => patchShot(i, { caption: sh.caption || "" }, true)} />
                            <Input type="number" min={1} placeholder={s.duration} value={sh.duration || 3} onChange={(e) => patchShot(i, { duration: Number(e.target.value) || 3 })} onBlur={() => patchShot(i, { duration: sh.duration || 3 }, true)} />
                            <label className="inline-flex h-8 cursor-pointer items-center justify-center gap-1 rounded-md border border-border text-[12px] text-muted hover:bg-lift hover:text-foreground">
                              {s.addStill}
                              <input type="file" accept="image/*" className="hidden" onChange={(e) => {
                                const f = e.target.files?.[0];
                                e.target.value = "";
                                if (f) void uploadShot("image", f, i);
                              }} />
                            </label>
                            <label className="inline-flex h-8 cursor-pointer items-center justify-center gap-1 rounded-md border border-border text-[12px] text-muted hover:bg-lift hover:text-foreground">
                              {s.addClip}
                              <input type="file" accept="video/*" className="hidden" onChange={(e) => {
                                const f = e.target.files?.[0];
                                e.target.value = "";
                                if (f) void uploadShot("video", f, i);
                              }} />
                            </label>
                          </div>
                          <button type="button" className="shrink-0 rounded-md p-1.5 text-muted hover:bg-lift hover:text-danger" title={s.removeShot} aria-label={s.removeShot} onClick={() => removeShot(i)}>
                            <Trash2 className="size-3.5" aria-hidden />
                          </button>
                        </div>
                      </li>
                    );
                  })}
                </ul>
                <details className="rounded-xl border border-border bg-card p-3" data-testid="studio-ops-tools">
                  <summary className="cursor-pointer text-[12px] font-medium text-foreground">{copy.video.opsTools}</summary>
                  <p className="mt-1 mb-2 text-[12px] leading-[1.5] text-muted">{copy.video.opsToolsHint}</p>
                  <div className="grid gap-2 sm:grid-cols-2">
                    <Input placeholder={s.opsHash} value={opsHash} onChange={(e) => setOpsHash(e.target.value)} />
                    <Input placeholder={s.subtitleHash} value={opsSub} onChange={(e) => setOpsSub(e.target.value)} />
                    <Input placeholder={s.wrap} value={wrapTitle} onChange={(e) => setWrapTitle(e.target.value)} className="sm:col-span-2" />
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    <Button size="sm" variant="ghost" disabled={!opsHash || !!busy} onClick={() => void run("reframe", async () => {
                      const r = await api.video.studioReframe(opsHash, current.aspect || defaultAspect);
                      setOpsResult(r);
                    })}>{s.reframe}</Button>
                    <Button size="sm" variant="ghost" disabled={!opsHash || !opsSub || !!busy} onClick={() => void run("burnin", async () => {
                      const r = await api.video.studioBurnin(opsHash, opsSub);
                      setOpsResult(r);
                    })}>{s.burnin}</Button>
                    <Button size="sm" variant="ghost" disabled={!opsHash || !!busy} onClick={() => void run("slides", async () => {
                      const hashes = opsHash.split(/[,\s]+/).map((x) => x.trim()).filter(Boolean);
                      const r = await api.video.studioSlideshow(hashes, current.aspect || defaultAspect, 3);
                      setOpsResult(r);
                    })}>{s.slideshow}</Button>
                    <Button size="sm" variant="ghost" disabled={!opsHash || !!busy} onClick={() => void run("wrap", async () => {
                      const r = await api.video.studioWrap(opsHash, wrapTitle || current.title || s.wrap);
                      setOpsResult(r);
                      if (r?.cas_hash && opsHash === current.result_hash) {
                        const next = (await api.video.updateStudio({ id: current.id, result_hash: r.cas_hash })) as StudioProject;
                        setProject(next);
                      }
                    })}>{s.wrap}</Button>
                    {opsResult?.cas_hash ? (
                      <Button size="sm" variant="lift" onClick={() => {
                        void saveProject({ storyboard: { ...(current.storyboard || {}), shots: [...shots, { video_hash: opsResult.cas_hash, duration: 3 }] } });
                      }}>{s.applyShot}</Button>
                    ) : null}
                  </div>
                  {opsResult?.media_url ? <video className="mt-2 max-h-48 w-full rounded-lg bg-black" src={opsResult.media_url} controls /> : null}
                </details>
              </>
            ) : null}
          </div>
        ) : null}

        {pane === "publish" ? (
          <div className="flex flex-col gap-3">
            <p className="text-[12px] leading-[1.5] text-muted">{s.neverPublish}</p>
            {current?.result_hash ? (
              <div className="flex flex-wrap items-center gap-2">
                <Input className="min-w-40 flex-1" placeholder={s.wrap} value={wrapTitle} onChange={(e) => setWrapTitle(e.target.value)} aria-label={s.wrap} />
                <Button size="sm" variant="lift" disabled={!!busy} onClick={() => void wrapCurrent()}>{s.wrap}</Button>
                {remotionReady ? <span className="text-[11px] text-muted">{s.remotionReady}</span> : (
                  <button type="button" className="text-[12px] text-muted underline" onClick={() => openSettings("generation", "generation-studio")}>{s.remotionInstallHint}</button>
                )}
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
            <Button size="sm" variant="lift" className="self-start" disabled={!!busy || !current?.result_hash} onClick={() => void saveDraftPublish()}>{s.draftPublish}</Button>
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
                {publishes.map((p) => (
                  <li key={p.id} className="rounded-xl border border-border bg-card px-3.5 py-3">
                    <div className="min-w-0">
                      <p className="text-[13px] font-medium text-foreground">{p.title}</p>
                      <p className="mt-0.5 text-[12px] text-muted">
                        {[p.platform, statusLabel[p.status || "draft"] || p.status].filter(Boolean).join(" · ")}
                      </p>
                      {p.posted_url ? <p className="mt-1 truncate text-[12px] text-muted">{p.posted_url}</p> : null}
                      {p.export_path ? <p className="mt-1 truncate text-[11px] text-muted">{p.export_path}</p> : null}
                    </div>
                    <div className="mt-2 flex flex-wrap gap-2">
                      <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void run("export", async () => {
                        await api.video.exportPublish(p.id);
                        await loadPublishes();
                      })}>{s.exportMp4}</Button>
                      <Button size="sm" variant="lift" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void run("open", async () => {
                        await api.video.openPublish(p.id);
                        await loadPublishes();
                      })}>{s.openCreator}</Button>
                      <Button size="sm" variant="ghost" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void run("analytics", async () => {
                        await api.video.openPublishAnalytics(p.id);
                      })}>{s.openAnalytics}</Button>
                      {p.export_path ? (
                        <Button size="sm" variant="ghost" className="h-7 text-[12px]" onClick={() => void api.openPath(p.export_path || "")}>{s.reveal}</Button>
                      ) : null}
                      <Button size="sm" variant="lift" className="h-7 text-[12px]" disabled={!!busy} onClick={() => void run("posted", async () => {
                        await api.video.markPublish(p.id, "posted", {
                          posted_url: postedUrl,
                          note: pubNote,
                          reads: Number(reads) || 0,
                          likes: Number(likes) || 0,
                          comments: Number(comments) || 0,
                        });
                        await loadPublishes();
                        await loadTopics();
                      })}>{s.markPosted}</Button>
                      <Button size="sm" variant="ghost" className="h-7 text-[12px] text-danger" disabled={!!busy} onClick={() => void run("failed", async () => {
                        await api.video.markPublish(p.id, "failed", { note: pubNote });
                        await loadPublishes();
                      })}>{s.markFailed}</Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
            <details className="rounded-xl border border-border bg-card p-3" data-testid="studio-record-stats">
              <summary className="cursor-pointer text-[12px] font-medium text-foreground">{copy.video.recordStats}</summary>
              <p className="mt-1 text-[12px] leading-[1.5] text-muted">{s.analyticsNote}</p>
              <div className="mt-2 grid gap-2 sm:grid-cols-2">
                <Input className="h-8" placeholder={s.postedUrl} value={postedUrl} onChange={(e) => setPostedUrl(e.target.value)} />
                <Input className="h-8" placeholder={s.analyticsNote} value={pubNote} onChange={(e) => setPubNote(e.target.value)} />
                <Input className="h-8" placeholder={s.reads} value={reads} onChange={(e) => setReads(e.target.value)} />
                <Input className="h-8" placeholder={s.likes} value={likes} onChange={(e) => setLikes(e.target.value)} />
                <Input className="h-8 sm:col-span-2" placeholder={s.comments} value={comments} onChange={(e) => setComments(e.target.value)} />
              </div>
            </details>
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
