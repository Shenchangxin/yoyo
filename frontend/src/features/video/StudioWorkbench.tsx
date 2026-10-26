/**
 * 本文件拥有短片工作台的雷达、选题、制作和交付切换。不拥有每一步的表单细节。
 * 收集、开拍、保存、打包和发布草稿会改服务端。成片进行中每 2000 毫秒重读，不是另一条模型循环。
 */
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

/** 把未知值收成数组。 v 是数组就原样留下，否则返回空数组。 */
function asList<T>(v: unknown): T[] {
  return Array.isArray(v) ? (v as T[]) : [];
}

/** 留下有成片或正在自动阶段里的短片，最多 12 条，当前这条不在里面就放到最前。 projects 留下有成片或自动阶段还在名单里的，最多 12 条。current 有 id 且有成片或自动阶段、又不在这 12 条里，就放到最前。 */
function shelfFilms(projects: StudioProject[], current: StudioProject | null) {
  const live = new Set(["queued", "script", "voice", "visuals", "compose", "done", "failed"]);
  const rows = projects.filter(/** p 有成片哈希，或自动阶段还在 live 里，才留下。 */ (p) => !!p.result_hash || live.has(p.auto_phase || ""));
  const top = rows.slice(0, 12);
  if (current?.id && (current.result_hash || current.auto_phase) && !top.some(/** p 用 id 和当前短片比对。 */ (p) => p.id === current.id)) top.unshift(current);
  return top;
}

/** StudioWorkbench 在雷达、选题、制作和发布之间切换当前短片。 props 用 projects 算副驾会话并找当前短片，有 onOpenProject 就把短片交给它，做成片或删改后调用 onRefreshHistory。sessionId、onNeedSession、onClose 函数体没有读取。 */
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
  const studioId = useStudioSelection(/** st 取出 studioId。 */ (st) => st.studioId);
  const setStudioId = useStudioSelection(/** st 取出 setStudioId。 */ (st) => st.setStudioId);
  const setStudioMeta = useStudioSelection(/** st 取出 setStudioMeta。 */ (st) => st.setStudioMeta);
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

  const current = (project && project.id === studioId ? project : null) || projects.find(/** p 用 id 和当前短片 id 比对。 */ (p) => p.id === studioId) || null;
  const copilotSessionId = studioSessionId(studioId, props.projects || []);

  /** 让父级打开这条短片的副驾。没有打开回调就返回。 p 没有 id 或没有打开回调就返回，否则把 id、title、会话、status 和 gate 交给 onOpenProject。 */
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

  /** 按状态读取选题。 */
  const loadTopics = useCallback(/** status 交给 listTopics，结果收成选题数组。 */ async (status = topicStatus) => {
    const list = await api.video.listTopics(status);
    setTopics(asList<Topic>(list));
  }, [topicStatus]);

  /** 读取短片列表。 */
  const loadProjects = useCallback(async () => {
    const list = await api.video.listStudio();
    const rows = asList<StudioProject>(list);
    setProjects(rows);
  }, []);

  /** 读取一条短片。选择已经换了就丢掉结果。 id 为空就清空工程，否则交给 getStudio；返回时选择已经不是这个 id 就丢掉。 */
  const loadProject = useCallback(async (id: string) => {
    if (!id) {
      setProject(null);
      return;
    }
    const p = (await api.video.getStudio(id)) as StudioProject;
    if (useStudioSelection.getState().studioId !== id) return;
    setProject(p);
    setProjects(/** prev 去掉和 p 相同 id 的项，再把 p 放到最前。 */ (prev) => {
      const rest = prev.filter(/** x 的 id 和 p 不同才留下。 */ (x) => x.id !== p.id);
      return [p, ...rest];
    });
  }, []);

  /** 读取发布草稿。 */
  const loadPublishes = useCallback(/** projectId 空着当空字符串，交给 listPublishes。 */ async (projectId = studioId) => {
    const list = await api.video.listPublishes(projectId || "");
    setPublishes(asList<Publish>(list));
  }, [studioId]);

  useEffect(() => {
    void loadTopics().catch(/** e 交给 errMessage 后弹出。 */ (e) => toast.error(api.errMessage(e)));
    void loadProjects().catch(/** e 交给 errMessage 后弹出。 */ (e) => toast.error(api.errMessage(e)));
    void api.video.publishPlatforms().then(/** rows 收成发布平台数组。 */ (rows) => setPubPlatforms(asList<PublishPlatform>(rows))).catch(() => {});
    void api.video.settings().then(/** cfg 取出 studio_aspect，只接受 16:9 和 1:1，否则用 9:16；有 studio_publish_platform 就记成偏好平台。 */ (cfg) => {
      const a = String(cfg?.studio_aspect || "9:16");
      setDefaultAspect(a === "16:9" || a === "1:1" ? a : "9:16");
      const p = String(cfg?.studio_publish_platform || "").trim();
      if (p) setPreferredPlatform(p);
    }).catch(() => {});
    void api.video.status().then(/** st 取出 media_base 并去掉结尾斜杠，再看 remotion.ready。 */ (st) => {
      setMediaBase(String(st?.media_base || "").replace(/\/+$/, ""));
      setRemotionReady(!!st?.remotion?.ready);
    }).catch(() => {});
    void api.video.fetchTrends("weibo,douyin,zhihu", 15).then(/** raw 取出 trends 收成数组。 */ (raw) => setTrends(asList<TrendGroup>(raw?.trends))).catch(() => {}).finally(() => setRadarLoading(false));
    void api.video.studioCatalog().then(/** row 空着当空对象，写成目录。 */ (row) => setCatalog((row || {}) as StudioCatalog)).catch(() => {});
  }, [loadProjects]);

  useEffect(() => {
    if (pane !== "publish") return;
    void loadPublishes().catch(/** e 交给 errMessage 后弹出。 */ (e) => toast.error(api.errMessage(e)));
  }, [pane, loadPublishes]);

  const filmLive = projects.some(/** p 把 auto_phase 交给 filmInFlight。 */ (p) => filmInFlight(p.auto_phase));
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
    const hit = (props.projects || []).find(/** row 要是短片，并且 id 和当前短片相同。 */ (row) => row.kind === "studio" && row.id === studioId);
    void attachCopilot({
      id: studioId,
      title: hit?.title || useStudioSelection.getState().title,
      status: hit?.status,
      gate: hit?.gate,
    }).catch(() => {
      /* 副驾绑定失败也不能把短片弹回首页 */
    });
    // 每个短片只挂一次。切换短片会改 studioId，新建也会。
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 靠 studioId 足够稳定
  }, [studioId]);

  useEffect(() => {
    if (studioId) {
      setPane(/** p 是 radar 或 topics 时收成 assemble，否则原样返回。 */ (p) => (p === "radar" || p === "topics" ? "assemble" : p));
      return;
    }
    setPane(/** p 是 assemble 或 publish 时收成 radar，否则原样返回。 */ (p) => (p === "assemble" || p === "publish" ? "radar" : p));
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

  /** 跑一次请求，失败就记下错误。 label 写成忙碌标记。fn 执行，失败就记下并弹出错误。 */
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

  /** 按勾选的平台重拉热榜，每边 15 条。 */
  async function refreshTrends() {
    await run("radar", async () => {
      const raw = await api.video.fetchTrends(platforms.join(","), 15);
      setTrends(asList<TrendGroup>(raw?.trends));
    });
  }

  /** 把一条热榜收成选题，并切到选题栏。 platform 和条目的标题、热度一起交给 topicFromTrend。item 没有 title 就返回，否则把 title 和 hot 交给 topicFromTrend。 */
  async function collect(platform: string, item: TrendItem) {
    if (!item.title) return;
    await run("collect", async () => {
      await api.video.topicFromTrend(platform, item.title || "", item.hot || "");
      await loadTopics();
      setPane("topics");
      toast.success(s.collected);
    });
  }

  /** 用草稿标题存一条收件箱选题。 */
  async function saveDraftTopic() {
    const title = draftTitle.trim();
    if (!title) return;
    await run("topic", async () => {
      await api.video.saveTopic({ title, format: "short", status: "inbox" });
      setDraftTitle("");
      await loadTopics();
    });
  }

  /** 把选题的改动写回去。 t 展开后并上 extra 再保存。extra 并进这条选题。 */
  async function patchTopic(t: Topic, extra: Partial<Topic>) {
    await run("topic", async () => {
      await api.video.saveTopic({ ...t, ...extra });
      await loadTopics();
    });
  }

  /** 打开这条短片到制作栏，并挂上副驾。 p 没有 id 就返回，否则记下短片、切到制作，并尝试挂上副驾。 */
  async function openShort(p: StudioProject) {
    if (!p?.id) return;
    setStudioId(p.id);
    setProject(p);
    setPane("assemble");
    try {
      await attachCopilot(p);
    } catch {
      /* 副驾会话绑不上也留在这条短片 */
    }
  }

  /** 打开这条短片的交付栏，并重新读一次工程。 id 为空就返回，否则用它找短片、记下选择并重新读工程。 */
  async function openDelivery(id: string) {
    if (!id) return;
    const hit = projects.find(/** row 用 id 和传入的 id 比对。 */ (row) => row.id === id);
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
      /* 副驾绑不上也留在交付 */
    }
  }

  /** 把这些选题做成片，并打开第一条的交付栏。 ids 去掉空白并去重，空的就提示，否则交给 autoTopics。 */
  async function filmTopics(ids: string[]) {
    const unique = [...new Set(ids.map(/** id 去掉空白。 */ (id) => id.trim()).filter(Boolean))];
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
      const first = rows.find(/** row 把 auto_phase 交给 filmInFlight。 */ (row) => filmInFlight(row.auto_phase)) || rows[0];
      if (!first?.id) return;
      setStudioId(first.id);
      setProject(first);
      setPane("publish");
      setDraftPlatform("");
      setDraftCaption(first.copy_text || "");
      try {
        await attachCopilot(first);
      } catch {
        /* 成片仍然落到交付 */
      }
    });
  }

  /** 不能再拍的选题改为打开交付，否则拿这一条去拍。 t 已有短片且不能再拍就打开那条交付，否则拿它的 id 去拍。 */
  async function produce(t: Topic) {
    const film = projects.find(/** p 用 id 和选题上的短片 id 比对。 */ (p) => p.id === t.studio_project_id);
    if (t.studio_project_id && !topicCanFilm(t, film)) {
      await openDelivery(t.studio_project_id);
      return;
    }
    await filmTopics([t.id]);
  }

  /** 清掉当前短片，回到选题栏。 */
  function goHome() {
    setStudioId("");
    setProject(null);
    setPane("topics");
    setNaming(false);
  }

  /** 勾上或取消这条选题。 id 已在勾选里就去掉，否则加上。 */
  function togglePick(id: string) {
    setPicked(/** prev 复制成集合后，id 已在里面就去掉，否则加上。 */ (prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  /** 把还能拍的已勾选题一起做成片。 */
  async function produceSelected() {
    const rows = topics.filter(/** t 用 id 看是否勾选，并交给 topicCanFilm。 */ (t) => picked.has(t.id) && topicCanFilm(t, projects.find(/** p 用 id 和选题上的短片 id 比对。 */ (p) => p.id === t.studio_project_id)));
    if (!rows.length) {
      toast.error(s.needTopicFirst);
      return;
    }
    await filmTopics(rows.map(/** t 取出 id。 */ (t) => t.id));
  }

  /** 把选题排进队列。重复时只提示已经在队列里。 t 用 id 交给 scheduleTopic。 */
  async function schedule(t: Topic) {
    await run("schedule", async () => {
      const out = (await api.video.scheduleTopic(t.id)) as { duplicate?: boolean };
      await loadTopics();
      toast.success(out?.duplicate ? s.scheduleExists : s.scheduleQueued);
    });
  }

  /** 取消这条选题的排期。 t 用 id 取消排期，然后重读选题。 */
  async function unschedule(t: Topic) {
    await run("unschedule", async () => {
      await api.video.unscheduleTopic(t.id);
      await loadTopics();
    });
  }

  /** 用标题和默认画幅新建短片并打开它。 title 去掉空白，空的就提示并返回，否则作为标题新建短片。 */
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

  /** 选中一条短片。已有成片或自动阶段就去交付，否则去制作。 id 用来找短片、记下选择，并带上标题、状态和门槛去挂副驾。 */
  async function selectProject(id: string) {
    const hit = projects.find(/** row 用 id 和传入的 id 比对。 */ (row) => row.id === id);
    setStudioId(id);
    setPane(hit?.result_hash || hit?.auto_phase ? "publish" : "assemble");
    setDraftPlatform("");
    try {
      await attachCopilot({ id, title: hit?.title, status: hit?.status, gate: hit?.gate });
    } catch {
      /* 副驾会话绑不上也留在这条短片 */
    }
  }

  /** 空标题换成新建短片的文案。 title 去掉空白，空的就用新建短片的文案。 */
  function shortLabel(title?: string) {
    return (title || "").trim() || s.newProject;
  }

  /** 打开删除确认。 p 记成待删除的短片，并关掉切换层。 */
  function askDelete(p: StudioProject) {
    setSwitchOpen(false);
    setKill(p);
  }

  /** 删除这条短片。删的是当前条就回选题。 id 交给 deleteStudio；删的是当前条就回选题。 */
  async function removeShort(id: string) {
    await api.video.deleteStudio(id);
    setKill(null);
    if (studioId === id) goHome();
    await loadProjects();
    props.onRefreshHistory?.();
  }

  /** 把补丁写回当前短片。 patch 展开进当前短片的更新。 */
  async function saveProject(patch: Partial<StudioProject>) {
    if (!current?.id) return;
    await run("save", async () => {
      const next = (await api.video.updateStudio({ id: current.id, ...patch })) as StudioProject;
      setProject(next);
    });
  }

  /** 给当前成片套壳。返回了新哈希就写回短片。 */
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

  /** 在已可发布、已选平台时保存一条发布草稿。 */
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

  /** 用短片或传入的名字建一张人设。 topicId 交给 createPersona，有的话再重读选题。name 去掉空白，空的就用当前短片标题或人设文案。 */
  async function makePersona(topicId = "", name = "") {
    await run("persona", async () => {
      const label = name.trim() || current?.title || s.persona;
      const out = await api.video.createPersona(label, topicId);
      if (topicId) await loadTopics();
      toast.success(out?.title || s.persona);
    });
  }

  /** 有媒体根地址时拼出 /media/ 路径。 hash 为空就返回空字符串，否则在有媒体根时拼出 /media/ 路径。 */
  function mediaSrc(hash?: string) {
    if (!hash) return "";
    return mediaBase ? `${mediaBase}/media/${hash}` : "";
  }

  /** 把热榜平台 id 换成界面名字。 */
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
    publish: publishes.some(/** p 看 status 是不是 posted 或 opened。 */ (p) => p.status === "posted" || p.status === "opened"),
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
  const navPanes = panes.filter(/** p 在首页时只留 radar 和 topics，否则只留 assemble 和 publish。 */ (p) => atHome ? p.id === "radar" || p.id === "topics" : p.id === "assemble" || p.id === "publish");
  const shownTopics = topics.filter(/** t 交给 topicMatchesQuery。 */ (t) => topicMatchesQuery(t, query));
  const shownTrends = trends
    .map(/** g 留下过滤后的 items。 */ (g) => ({ ...g, items: (g.items || []).filter(/** item 交给 trendMatchesQuery。 */ (item) => trendMatchesQuery(item, query)) }))
    .filter(/** g 看 items 还有没有条目。 */ (g) => (g.items || []).length);
  const pickCount = [...picked].filter(/** id 用来找还能拍的选题。 */ (id) => topics.some(/** t 的 id 相同且还能拍才算命中。 */ (t) => t.id === id && topicCanFilm(t, projects.find(/** p 用 id 和选题上的短片 id 比对。 */ (p) => p.id === t.studio_project_id)))).length;

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
                  {projects.map(/** p 用 status、gate 或 furthest_gate 拼说明，用 id 选中或标出当前，用 title 显示名字，删除时把整条交给 askDelete。 */ (p) => {
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
                          onPointerDown={/** e 阻止默认。 */ (e) => e.preventDefault()}
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
              <DropdownMenuItem onSelect={() => { void writeClipboard(current.id).then(/** ok 为真时提示已复制。 */ (ok) => { if (ok) toast.success(s.copiedId); }); }}>
                {s.copyId}
              </DropdownMenuItem>
              {copilotSessionId ? (
                <DropdownMenuItem onSelect={() => { void writeClipboard(copilotSessionId).then(/** ok 为真时提示已复制。 */ (ok) => { if (ok) toast.success(s.copiedId); }); }}>
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
        {navPanes.map(/** p 用 id 当 key 和当前栏，显示 hint 和 label，点击时切到这个 id。i 显示成从 1 开始的序号。 */ (p, i) => (
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
              onChange={/** e 取出 target.value 交给 setQuery。 */ (e) => setQuery(e.target.value)}
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
            onSubmit={/** title 交给 newProject。 */ (title) => void newProject(title)}
          />
        </div>
      ) : null}

      <div className="min-h-0 flex-1 overflow-auto px-4 py-3" data-testid={atHome ? "studio-home" : "studio-project"}>
        {pane === "radar" ? (
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="mr-1 text-[12px] text-muted">{s.platforms}</span>
              {TREND_PLATFORMS.map(/** p 用来判断是否已选，点击时加进或移出，并交给 trendLabel 显示。 */ (p) => {
                const on = platforms.includes(p);
                return (
                  <button
                    key={p}
                    type="button"
                    aria-pressed={on}
                    className={cn("rounded-full px-2 py-0.5 text-[11px]", on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
                    onClick={() => setPlatforms(/** prev 在已选且只剩这一项时不动；已选就去掉 p，未选就把 p 加进去。 */ (prev) => {
                      if (on && prev.length <= 1) return prev;
                      return on ? prev.filter(/** x 和 p 不同才留下。 */ (x) => x !== p) : [...prev, p];
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
              shownTrends.map(/** g 用 platform 或 label 当 key 和标题，不 ok 就显示离线，并列出 items，收藏时把 platform 交给 collect。 */ (g) => (
                <section key={g.platform || g.label} className="overflow-hidden rounded-xl border border-border bg-card">
                  <h2 className="flex items-center gap-2 px-3.5 py-2 text-[12px] font-medium text-muted">
                    {g.label || g.platform}
                    {!g.ok ? <span className="text-danger/80">{s.offline}</span> : null}
                  </h2>
                  <ul className="divide-y divide-border">
                    {(g.items || []).map(/** item 显示 title，有 hot 就显示，收藏时整条交给 collect。i 和平台拼成 key。 */ (item, i) => (
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
              {TOPIC_FILTERS.map(/** st 用来判断选中和按钮 key，点击时记成筛选并交给 loadTopics，空的就显示全部文案。 */ (st) => (
                <button
                  key={st || "all"}
                  type="button"
                  data-testid={`studio-topic-filter-${st || "all"}`}
                  aria-pressed={topicStatus === st}
                  className={cn("rounded-full px-2 py-0.5 text-[11px]", topicStatus === st ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70")}
                  onClick={() => {
                    setTopicStatus(st);
                    void loadTopics(st).catch(/** e 交给 errMessage 后弹出。 */ (e) => toast.error(api.errMessage(e)));
                  }}
                >
                  {st ? statusLabel[st] : s.allTopics}
                </button>
              ))}
            </div>
            <div className="flex gap-2">
              <Input className="h-8" placeholder={s.newTopicPh} value={draftTitle} onChange={/** e 取出 target.value 交给 setDraftTitle。 */ (e) => setDraftTitle(e.target.value)} onKeyDown={/** e 的 key 是 Enter 时保存草稿选题。 */ (e) => e.key === "Enter" && void saveDraftTopic()} />
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
                {shownTopics.map(/** t 交给选题卡片，并参与选中、开拍、改期、补丁、展开、排期、人设和删除。 */ (t) => (
                  <TopicCard
                    key={t.id}
                    topic={t}
                    busy={!!busy}
                    picked={picked.has(t.id)}
                    statusLabel={statusLabel[t.status || "inbox"] || t.status || ""}
                    deleteLabel={copy.video.delete}
                    scoreLabels={scoreLabel}
                    film={projects.find(/** p 用 id 和选题上的短片 id 比对。 */ (p) => p.id === t.studio_project_id)}
                    copy={s}
                    onPick={() => togglePick(t.id)}
                    onProduce={() => void produce(t)}
                    onUnschedule={() => void unschedule(t)}
                    onPatch={/** extra 交给 patchTopic。 */ (extra) => void patchTopic(t, extra)}
                    onExpand={() => void run("expand", async () => {
                      await api.video.expandTopic(t.id);
                      await loadTopics();
                    })}
                    onSchedule={() => void schedule(t)}
                    onPersona={() => void makePersona(t.id, t.title)}
                    onChoose={/** optionId 交给 chooseTopicOption，然后重读选题。 */ (optionId) => void run("choose", async () => {
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
                  onChange={/** next 当成短片写入。 */ (next) => setProject(next as StudioProject)}
                  run={run}
                  saveProject={/** patch 当成短片补丁交给 saveProject。 */ async (patch) => { await saveProject(patch as Partial<StudioProject>); }}
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
            onSelect={/** id 交给 openDelivery。 */ (id) => { void openDelivery(id); }}
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
                {publishes.map(/** row 作为草稿交给卡片，并用 id 当 key、用 status 取状态文案。 */ (row) => (
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

/** 一条发布草稿：导出、打开创作者、记失败或记已发。 props 用 draft 填标题、平台、链接、导出路径和统计初值，statusLabel 接在平台后面，busy 时禁用按钮，导出、打开、记失败和记已发都包在 onRun 里，做完调用 onReload。 */
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
          <Input className="h-8" placeholder={s.postedUrl} value={postedUrl} onChange={/** e 取出 target.value 交给 setPostedUrl。 */ (e) => setPostedUrl(e.target.value)} />
          <Input className="h-8" placeholder={s.analyticsNote} value={note} onChange={/** e 取出 target.value 交给 setNote。 */ (e) => setNote(e.target.value)} />
          <Input className="h-8" placeholder={s.reads} value={reads} onChange={/** e 取出 target.value 交给 setReads。 */ (e) => setReads(e.target.value)} />
          <Input className="h-8" placeholder={s.likes} value={likes} onChange={/** e 取出 target.value 交给 setLikes。 */ (e) => setLikes(e.target.value)} />
          <Input className="h-8 sm:col-span-2" placeholder={s.comments} value={comments} onChange={/** e 取出 target.value 交给 setComments。 */ (e) => setComments(e.target.value)} />
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
