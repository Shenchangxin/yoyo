/**
 * 本文件拥有任务栏里的剪辑、短剧、短片和画布列表。不拥有各工种的编辑器。
 * 新建和删除会改服务端。任务页签只嵌任务路由，不是另一条模型循环。
 */
import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import { Clapperboard, Copy, LayoutGrid, Plus, Scissors, Sparkles, Trash2 } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { writeClipboard } from "../../lib/clipboard";
import * as api from "../../lib/client";
import { displayTitle } from "../../lib/display-title";
import { useCopy } from "../../lib/i18n";
import { DURATION_SHELL, motionTransition, useMotionReduced } from "../../lib/motion";
import type { VideoProject } from "../../lib/protocol";
import { cn } from "../../lib/utils";
import { bindCanvasSession, useCanvasHost } from "./canvas-host/session";
import { CreateShortField } from "./studio-chrome";
import { shownSeriesTitle } from "./drama-lib";
import { ConfirmDialog } from "../ConfirmDialog";
import { defaultDramaStyle, useStudioSelection } from "./workshop-store";
import type { CutProject } from "./cut-desk";

const YingceApp = lazy(() => import("./canvas-host/YingceApp"));

/** VideoHistory 在项目和任务之间切换，并列出剪辑、短剧、短片和画布。 props 用 projects 分成短剧、短片和画布，sessionId 用来绑定画布并带进打开的工程，打开交给 onOpenProject，新建后调用 onRefresh，剪辑行把 id 交给 onOpenCut，dramaId、studioId 和 canvasProjectId 标出当前项。 */
export function VideoHistory(props: {
  projects: VideoProject[];
  canvasProjectId?: string;
  dramaId?: string;
  studioId?: string;
  sessionId?: string;
  onOpenProject?: (p: VideoProject) => void;
  onOpenCut?: (id: string) => void;
  onRefresh?: () => void;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const tabMotion = motionTransition(reduced, DURATION_SHELL);
  const [tab, setTab] = useState<"projects" | "jobs">("projects");
  const [busy, setBusy] = useState("");
  const [kill, setKill] = useState<VideoProject | null>(null);
  const setStudioId = useStudioSelection(/** s 取出 setStudioId。 */ (s) => s.setStudioId);
  const dramas = useMemo(() => props.projects.filter(/** p 看 kind 是不是 drama。 */ (p) => p.kind === "drama"), [props.projects]);
  const canvases = useMemo(() => props.projects.filter(/** p 看 kind 是不是 canvas。 */ (p) => p.kind === "canvas"), [props.projects]);
  const studios = useMemo(() => props.projects.filter(/** p 看 kind 是不是 studio。 */ (p) => p.kind === "studio"), [props.projects]);
  const [cuts, setCuts] = useState<CutProject[]>([]);

  useEffect(() => {
    let alive = true;
    void api.video.cutList().then(/** list 不是数组就当空的，按 updated_at 排序后留下有 id 和 source_hash 的剪辑。 */ (list) => {
      if (!alive) return;
      const rows = (Array.isArray(list) ? list : []) as CutProject[];
      rows.sort(/** a 用 updated_at 参与比较，空的当空字符串。b 用 updated_at 和 a 比较，空的当空字符串。 */ (a, b) => String(b.updated_at || "").localeCompare(String(a.updated_at || "")));
      setCuts(rows.filter(/** row 要有 id 和 source_hash。 */ (row) => row.id && row.source_hash));
    }).catch(() => { if (alive) setCuts([]); });
    return () => { alive = false; };
  }, [props.projects, props.sessionId]);

  /** 新建一块画布，有会话就绑定并打开。 */
  async function newCanvas() {
    setBusy("canvas");
    try {
      const env = await api.videoCall("canvas.http", { method: "POST", path: "canvas-projects", body: { title: copy.video.canvas } });
      const project = env?.data?.project || env?.project;
      if (project?.id) {
        useCanvasHost.setState({ projectId: project.id, title: project.title || copy.video.canvas });
        if (props.sessionId) await bindCanvasSession(props.sessionId, project.id);
        props.onOpenProject?.({
          kind: "canvas",
          id: project.id,
          title: project.title || copy.video.canvas,
          sessionId: props.sessionId,
        });
        props.onRefresh?.();
      }
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  /** 用填写的标题新建短片。画幅只接受 16:9、1:1，否则用 9:16。 title 去掉空白，空的就提示并返回，否则作为标题新建短片。 */
  async function newStudio(title: string) {
    const name = title.trim();
    if (!name) {
      toast.error(copy.video.studio.needNamedShort);
      return;
    }
    setBusy("studio");
    try {
      const cfg = await api.video.settings().catch(() => ({} as Record<string, string>));
      const aspect = String(cfg?.studio_aspect || "9:16");
      const p = await api.video.createStudio({
        title: name,
        aspect: aspect === "16:9" || aspect === "1:1" ? aspect : "9:16",
      });
      if (p?.id) {
        props.onOpenProject?.({
          kind: "studio",
          id: p.id,
          title: p.title || name,
          status: p.status,
          gate: p.gate,
        });
      }
      props.onRefresh?.();
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  /** 按当前风格新建系列，比例是 9:16。 */
  async function newDrama() {
    setBusy("drama");
    try {
      const d = await api.video.createDrama({ title: "", style: defaultDramaStyle(), aspect_ratio: "9:16" });
      if (d?.id) {
        props.onOpenProject?.({
          kind: "drama",
          id: d.id,
          title: d.title || "",
          sessionId: props.sessionId,
        });
      }
      props.onRefresh?.();
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="video-history" data-skin-pane>
      <div className="flex shrink-0 items-center gap-2 px-5 py-4">
        <div className="min-w-0 flex-1">
          <h1 className="text-[16px] font-semibold tracking-[-0.03em]">{copy.video.shellTasks}</h1>
          <p className="mt-0.5 text-[13px] text-muted">{copy.video.historyHint}</p>
        </div>
        <div className="relative flex rounded-lg bg-lift p-0.5 text-[13px]">
          {(["projects", "jobs"] as const).map(/** id 用来判断选中并切换页，projects 显示项目文案，否则显示任务文案。 */ (id) => {
            const on = tab === id;
            return (
              <button
                key={id}
                type="button"
                className="relative rounded-md px-2.5 py-1"
                aria-pressed={on}
                onClick={() => setTab(id)}
              >
                {on ? (
                  <motion.span
                    layoutId="video-history-tab"
                    className="absolute inset-0 rounded-md bg-card"
                    transition={tabMotion}
                    aria-hidden
                  />
                ) : null}
                <span className={cn("relative z-10", on ? "text-foreground" : "text-muted")}>
                  {id === "projects" ? copy.video.historyProjects : copy.video.historyJobs}
                </span>
              </button>
            );
          })}
        </div>
      </div>
      {tab === "jobs" ? (
        <div className="min-h-0 flex-1">
          <Suspense
            fallback={
              <div className="flex h-full flex-col gap-3 p-5" aria-busy="true">
                <span className="sr-only">{copy.video.shellTasks}</span>
                <div className="skeleton-sweep h-10 w-48 rounded-xl" />
                <div className="skeleton-sweep is-soft min-h-0 flex-1 rounded-2xl" />
              </div>
            }
          >
            <YingceApp initialPath="/tasks" />
          </Suspense>
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-auto px-5 pb-6">
          <section className="mb-6">
            <div className="mb-2 flex items-center gap-2">
              <h2 className="text-[12px] font-medium text-muted">{copy.video.conductor.cutTitle}</h2>
            </div>
            {cuts.length ? (
              <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
                {cuts.map(/** row 用 id 当 key，点击时交给 onOpenCut，显示 title，有 duration 就显示秒数。 */ (row) => (
                  <li key={row.id}>
                    <button
                      type="button"
                      className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-[13px] hover:bg-lift"
                      onClick={() => row.id && props.onOpenCut?.(row.id)}
                    >
                      <span className="truncate">{row.title || copy.video.conductor.cutTitle}</span>
                      {row.duration ? <span className="shrink-0 text-[12px] tabular-nums text-muted">{row.duration.toFixed(1)}s</span> : null}
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <EmptyState icon={<Scissors className="size-5" />} title={copy.video.conductor.cutTitle} body={copy.video.historyEmptyCuts} />
            )}
          </section>
          <section className="mb-6">
            <div className="relative mb-2 flex items-center gap-2">
              <h2 className="text-[12px] font-medium text-muted">{copy.video.shellDrama}</h2>
              <Button size="sm" variant="ghost" className="ml-auto h-7 gap-1 text-[12px]" disabled={busy === "drama"} data-testid="video-history-new-drama" onClick={() => void newDrama()}>
                <Plus className="size-3.5" aria-hidden />
                {copy.video.newDrama}
              </Button>
              {busy === "drama" ? <ProgressHairline className="absolute inset-x-0 -bottom-px rounded-none" indeterminate /> : null}
            </div>
            {dramas.length ? (
              <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
                {dramas.map(/** p 用 id 当 key 并和 dramaId 比较是否当前，点击时整条交给 onOpenProject。 */ (p) => (
                  <HistoryRow key={`drama-${p.id}`} project={p} active={p.id === props.dramaId} fallback={copy.video.untitled} onSelect={() => props.onOpenProject?.(p)} />
                ))}
              </ul>
            ) : (
              <EmptyState icon={<Clapperboard className="size-5" />} title={copy.video.empty} body={copy.video.emptyHint} />
            )}
          </section>
          <section className="mb-6">
            <div className="relative mb-2 flex items-center gap-2">
              <h2 className="text-[12px] font-medium text-muted">{copy.video.shellStudio}</h2>
              {busy === "studio" ? <ProgressHairline className="absolute inset-x-0 -bottom-px rounded-none" indeterminate /> : null}
            </div>
            <div className="mb-3">
              <CreateShortField
                disabled={busy === "studio"}
                placeholder={copy.video.studio.createTitle}
                submitLabel={copy.video.studio.create}
                hint={copy.video.studio.createTitleHint}
                testId="video-history-new-studio"
                titleTestId="video-history-studio-title"
                onSubmit={/** title 交给 newStudio。 */ (title) => void newStudio(title)}
              />
            </div>
            {studios.length ? (
              <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
                {studios.map(/** p 用 id 当 key 并和 studioId 比较是否当前，点击时整条交给 onOpenProject，删除时整条交给 setKill。 */ (p) => (
                  <HistoryRow
                    key={`studio-${p.id}`}
                    project={p}
                    active={p.id === props.studioId}
                    fallback={copy.video.studio.newProject}
                    onSelect={() => props.onOpenProject?.(p)}
                    onDelete={() => setKill(p)}
                  />
                ))}
              </ul>
            ) : (
              <EmptyState icon={<Sparkles className="size-5" />} title={copy.video.creative} body={copy.video.historyEmptyStudio} />
            )}
          </section>
          <section>
            <div className="relative mb-2 flex items-center gap-2">
              <h2 className="text-[12px] font-medium text-muted">{copy.video.shellCanvas}</h2>
              <Button size="sm" variant="ghost" className="ml-auto h-7 gap-1 text-[12px]" disabled={busy === "canvas"} data-testid="video-history-new-canvas" onClick={() => void newCanvas()}>
                <Plus className="size-3.5" aria-hidden />
                {copy.video.newCanvas}
              </Button>
              {busy === "canvas" ? <ProgressHairline className="absolute inset-x-0 -bottom-px rounded-none" indeterminate /> : null}
            </div>
            {canvases.length ? (
              <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
                {canvases.map(/** p 用 id 当 key 并和 canvasProjectId 比较是否当前，点击时整条交给 onOpenProject。 */ (p) => (
                  <HistoryRow key={`canvas-${p.id}`} project={p} active={p.id === props.canvasProjectId} fallback={copy.video.canvas} onSelect={() => props.onOpenProject?.(p)} />
                ))}
              </ul>
            ) : (
              <EmptyState icon={<LayoutGrid className="size-5" />} title={copy.video.noCanvas} body={copy.video.historyEmptyCanvas} />
            )}
          </section>
        </div>
      )}
      <ConfirmDialog
        open={!!kill}
        title={copy.video.studio.deleteShort}
        body={copy.video.studio.deleteNamed.replace("{title}", (kill?.title || "").trim() || copy.video.studio.newProject)}
        danger
        confirmLabel={copy.video.studio.deleteShort}
        onCancel={() => setKill(null)}
        onConfirm={async () => {
          if (!kill?.id) return;
          try {
            await api.video.deleteStudio(kill.id);
            if (kill.id === props.studioId) setStudioId("");
            setKill(null);
            props.onRefresh?.();
          } catch (e) {
            toast.error(api.errMessage(e));
          }
        }}
      />
    </div>
  );
}

/** 一行历史。短片可以复制 id，也可以交给删除。 props 用 project 画图标、标题和状态，active 标出当前行，点击交给 onSelect，标题空着就用 fallback，有 onDelete 就显示删除。 */
function HistoryRow(props: { project: VideoProject; active: boolean; fallback: string; onSelect: () => void; onDelete?: () => void }) {
  const copy = useCopy();
  const p = props.project;
  const Icon = p.kind === "drama" ? Clapperboard : p.kind === "studio" ? Sparkles : LayoutGrid;
  const statusLabel: Record<string, string> = {
    inbox: copy.video.studio.inbox,
    ready: copy.video.studio.ready,
    scheduled: copy.video.studio.scheduled,
    in_production: copy.video.studio.inProduction,
    done: copy.video.studio.done,
    draft: copy.video.studio.draft,
    assembled: copy.video.studio.assembled,
    failed: copy.video.studio.failed,
  };
  const gateLabel: Record<string, string> = {
    setup: copy.video.studio.desk.setup,
    script: copy.video.studio.desk.script,
    audio: copy.video.studio.desk.audio,
    visuals: copy.video.studio.desk.visuals,
    compose: copy.video.studio.desk.compose,
    delivery: copy.video.studio.desk.delivery,
  };
  const meta = [
    p.status ? (statusLabel[p.status] || p.status) : "",
    p.gate ? (gateLabel[p.gate] || p.gate) : "",
    p.updatedAt ? p.updatedAt.slice(0, 10) : "",
  ].filter(Boolean);
  return (
    <li className="flex items-stretch">
      <button
        type="button"
        data-testid={`video-project-${p.kind}-${p.id}`}
        aria-current={props.active ? "page" : undefined}
        className={cn(
          "flex min-w-0 flex-1 items-start gap-3 px-3.5 py-3 text-left text-[13px]",
          props.active ? "bg-lift text-foreground" : "u-row-hover text-foreground",
        )}
        onClick={props.onSelect}
      >
        <Icon className="mt-0.5 size-4 shrink-0 opacity-70" aria-hidden />
        <span className="min-w-0 flex-1">
          <span className="block truncate">{p.kind === "drama" ? shownSeriesTitle(p.title, props.fallback) : displayTitle(p.title, props.fallback)}</span>
          {p.kind === "studio" ? (
            <span className="mt-0.5 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-muted">
              {meta.length ? <span>{meta.join(" · ")}</span> : null}
              <span className="truncate font-mono" title={copy.video.studio.inspectHint}>{p.id}</span>
              {p.sessionId ? <span className="truncate font-mono" title={copy.video.studio.sessionId}>{p.sessionId}</span> : null}
            </span>
          ) : p.updatedAt ? (
            <span className="mt-0.5 block text-[11px] text-muted">{p.updatedAt.slice(0, 10)}</span>
          ) : null}
        </span>
      </button>
      {p.kind === "studio" ? (
        <button
          type="button"
          className="inline-flex shrink-0 items-center rounded p-2 text-muted hover:bg-lift hover:text-foreground"
          data-testid={`video-project-copy-${p.id}`}
          aria-label={copy.video.studio.copyId}
          onClick={() => {
            void writeClipboard(p.id).then(/** ok 为真时提示已复制。 */ (ok) => { if (ok) toast.success(copy.video.studio.copiedId); });
          }}
        >
          <Copy className="size-3 opacity-70" aria-hidden />
        </button>
      ) : null}
      {props.onDelete ? (
        <button
          type="button"
          className="grid w-10 shrink-0 place-items-center text-muted hover:bg-danger/10 hover:text-danger"
          data-testid={`video-history-delete-${p.id}`}
          aria-label={copy.video.studio.deleteShort}
          title={copy.video.studio.deleteShort}
          onClick={/** e 阻止默认并停止冒泡，然后交给 onDelete。 */ (e) => {
            e.preventDefault();
            e.stopPropagation();
            props.onDelete?.();
          }}
        >
          <Trash2 className="size-3.5" aria-hidden />
        </button>
      ) : null}
    </li>
  );
}
