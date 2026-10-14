import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import { Clapperboard, Copy, LayoutGrid, Plus, Sparkles, Trash2 } from "lucide-react";
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

const YingceApp = lazy(() => import("./canvas-host/YingceApp"));

export function VideoHistory(props: {
  projects: VideoProject[];
  canvasProjectId?: string;
  dramaId?: string;
  studioId?: string;
  sessionId?: string;
  onOpenProject?: (p: VideoProject) => void;
  onRefresh?: () => void;
}) {
  const copy = useCopy();
  const reduced = useMotionReduced();
  const tabMotion = motionTransition(reduced, DURATION_SHELL);
  const [tab, setTab] = useState<"projects" | "jobs">("projects");
  const [busy, setBusy] = useState("");
  const [kill, setKill] = useState<VideoProject | null>(null);
  const setStudioId = useStudioSelection((s) => s.setStudioId);
  const dramas = useMemo(() => props.projects.filter((p) => p.kind === "drama"), [props.projects]);
  const canvases = useMemo(() => props.projects.filter((p) => p.kind === "canvas"), [props.projects]);
  const studios = useMemo(() => props.projects.filter((p) => p.kind === "studio"), [props.projects]);

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

  async function newDrama() {
    setBusy("drama");
    try {
      await api.video.createDrama({ title: "", style: defaultDramaStyle(), aspect_ratio: "9:16" });
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
                onSubmit={(title) => void newStudio(title)}
              />
            </div>
            {studios.length ? (
              <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
                {studios.map((p) => (
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
              <span className="inline-flex min-w-0 items-center gap-1 font-mono" title={copy.video.studio.inspectHint}>
                <span className="truncate">{p.id}</span>
                <span
                  role="button"
                  tabIndex={0}
                  className="inline-flex shrink-0 rounded p-0.5 hover:bg-background"
                  data-testid={`video-project-copy-${p.id}`}
                  aria-label={copy.video.studio.copyId}
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    void writeClipboard(p.id).then((ok) => { if (ok) toast.success(copy.video.studio.copiedId); });
                  }}
                  onKeyDown={(e) => {
                    if (e.key !== "Enter" && e.key !== " ") return;
                    e.preventDefault();
                    e.stopPropagation();
                    void writeClipboard(p.id).then((ok) => { if (ok) toast.success(copy.video.studio.copiedId); });
                  }}
                >
                  <Copy className="size-3 opacity-70" aria-hidden />
                </span>
              </span>
              {p.sessionId ? <span className="truncate font-mono" title={copy.video.studio.sessionId}>{p.sessionId}</span> : null}
            </span>
          ) : p.updatedAt ? (
            <span className="mt-0.5 block text-[11px] text-muted">{p.updatedAt.slice(0, 10)}</span>
          ) : null}
        </span>
      </button>
      {props.onDelete ? (
        <button
          type="button"
          className="grid w-10 shrink-0 place-items-center text-muted hover:bg-danger/10 hover:text-danger"
          data-testid={`video-history-delete-${p.id}`}
          aria-label={copy.video.studio.deleteShort}
          title={copy.video.studio.deleteShort}
          onClick={(e) => {
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
