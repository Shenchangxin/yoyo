import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import { Clapperboard, LayoutGrid, Plus } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import * as api from "../../lib/client";
import { displayTitle } from "../../lib/display-title";
import { useCopy } from "../../lib/i18n";
import { DURATION_SHELL, motionTransition, useMotionReduced } from "../../lib/motion";
import type { VideoProject } from "../../lib/protocol";
import { cn } from "../../lib/utils";
import { bindCanvasSession, useCanvasHost } from "./canvas-host/session";

const YingceApp = lazy(() => import("./canvas-host/YingceApp"));

export function VideoHistory(props: {
  projects: VideoProject[];
  canvasProjectId?: string;
  dramaId?: string;
  sessionId?: string;
  onOpenProject?: (p: VideoProject) => void;
  onRefresh?: () => void;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const tabMotion = motionTransition(reduced, DURATION_SHELL);
  const [tab, setTab] = useState<"projects" | "jobs">("projects");
  const [busy, setBusy] = useState("");
  const dramas = useMemo(() => props.projects.filter((p) => p.kind === "drama"), [props.projects]);
  const canvases = useMemo(() => props.projects.filter((p) => p.kind === "canvas"), [props.projects]);

  useEffect(() => {
    props.onRefresh?.();
  }, [props.onRefresh]);

  async function newCanvas() {
    setBusy("canvas");
    try {
      const env = await api.videoCall("canvas.http", { method: "POST", path: "canvas-projects", body: { title: copy.video.canvas } });
      const project = env?.data?.project || env?.project;
      if (project?.id) {
        useCanvasHost.setState({ projectId: project.id, title: project.title || copy.video.canvas });
        if (props.sessionId) await bindCanvasSession(props.sessionId, project.id);
        props.onRefresh?.();
      }
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function newDrama() {
    setBusy("drama");
    try {
      await api.video.createDrama({ title: copy.video.untitled, style: "3d", aspect_ratio: "16:9" });
      props.onRefresh?.();
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col bg-background" data-testid="video-history">
      <div className="flex shrink-0 items-center gap-2 px-5 py-4">
        <div className="min-w-0 flex-1">
          <h1 className="text-[16px] font-semibold tracking-[-0.03em]">{copy.video.shellTasks}</h1>
          <p className="mt-0.5 text-[13px] text-muted">{copy.video.historyHint}</p>
        </div>
        <div className="relative flex rounded-lg bg-lift p-0.5 text-[13px]">
          {(["projects", "jobs"] as const).map((id) => {
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
                {dramas.map((p) => (
                  <HistoryRow key={`drama-${p.id}`} project={p} active={p.id === props.dramaId} fallback={copy.video.untitled} onSelect={() => props.onOpenProject?.(p)} />
                ))}
              </ul>
            ) : (
              <EmptyState icon={<Clapperboard className="size-5" />} title={copy.video.empty} body={copy.video.emptyHint} />
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
                {canvases.map((p) => (
                  <HistoryRow key={`canvas-${p.id}`} project={p} active={p.id === props.canvasProjectId} fallback={copy.video.canvas} onSelect={() => props.onOpenProject?.(p)} />
                ))}
              </ul>
            ) : (
              <EmptyState icon={<LayoutGrid className="size-5" />} title={copy.video.noCanvas} body={copy.video.historyEmptyCanvas} />
            )}
          </section>
        </div>
      )}
    </div>
  );
}

function HistoryRow(props: { project: VideoProject; active: boolean; fallback: string; onSelect: () => void }) {
  const p = props.project;
  const Icon = p.kind === "drama" ? Clapperboard : LayoutGrid;
  return (
    <li>
      <button
        type="button"
        data-testid={`video-project-${p.kind}-${p.id}`}
        aria-current={props.active ? "page" : undefined}
        className={cn(
          "flex w-full items-center gap-3 px-3.5 py-3 text-left text-[13px]",
          props.active ? "bg-lift text-foreground" : "u-row-hover text-foreground",
        )}
        onClick={props.onSelect}
      >
        <Icon className="size-4 shrink-0 opacity-70" aria-hidden />
        <span className="min-w-0 flex-1 truncate">{displayTitle(p.title, props.fallback)}</span>
        {p.updatedAt ? <span className="shrink-0 text-[11px] text-muted">{p.updatedAt.slice(0, 10)}</span> : null}
      </button>
    </li>
  );
}
