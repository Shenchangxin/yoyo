import { Component, lazy, Suspense, useEffect, type ErrorInfo, type ReactNode } from "react";
import { X } from "lucide-react";
import { Tooltip } from "../../components/ui/tooltip";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { yingcePathForPane } from "../../lib/protocol";
import { useCanvasHost } from "./canvas-host/session";

const YingceApp = lazy(() =>
  import("./canvas-host/YingceApp").catch((err: Error) => ({
    default: function YingceLoadError() {
      return (
        <div className="flex h-full items-center justify-center p-6 text-center text-[13px] text-muted" data-testid="canvas-studio-error">
          {String(err?.message || err)}
        </div>
      );
    },
  })),
);

const iconBtn =
  "inline-flex size-7 items-center justify-center rounded-lg text-muted transition-colors hover:bg-lift hover:text-foreground";

class IslandBoundary extends Component<{ children: ReactNode }, { err: Error | null }> {
  state = { err: null as Error | null };
  static getDerivedStateFromError(err: Error) {
    return { err };
  }
  componentDidCatch(err: Error, info: ErrorInfo) {
    console.error(err, info.componentStack);
  }
  render() {
    if (this.state.err) {
      return (
        <div className="flex h-full items-center justify-center p-6 text-center text-[13px] text-muted" data-testid="canvas-studio-error">
          {this.state.err.message}
        </div>
      );
    }
    return this.props.children;
  }
}

export function CanvasStudio(props: { sessionId?: string; onNeedSession?: () => void; onClose?: () => void }) {
  const copy = useCopy();
  const pane = useUI((s) => s.videoPane);
  const canvasFocus = useUI((s) => s.canvasFocus);
  const projectId = useCanvasHost((s) => s.projectId);
  const start = yingcePathForPane(pane, projectId, canvasFocus);

  useEffect(() => {
    if (typeof window !== "undefined") window.__YOYO_VIDEO_SESSION__ = props.sessionId || "";
    useCanvasHost.setState({ sessionId: props.sessionId || "" });
  }, [props.sessionId]);
  useEffect(() => {
    if (!props.sessionId) props.onNeedSession?.();
  }, [props.sessionId, props.onNeedSession]);

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="canvas-studio-shell" data-pane={pane}>
      {props.onClose ? (
        <header className="flex h-9 shrink-0 items-center border-b border-border/70 px-2">
          <Tooltip content={copy.video.closeBoard}>
            <button type="button" className={cn(iconBtn, "ml-auto")} aria-label={copy.video.closeBoard} onClick={props.onClose}>
              <X className="size-3.5" />
            </button>
          </Tooltip>
        </header>
      ) : null}
      <div className="min-h-0 flex-1">
        <IslandBoundary>
          <Suspense
            fallback={
              <div className="flex h-full items-center justify-center text-[13px] text-muted" data-testid="canvas-studio-loading">
                Opening infinite canvas…
              </div>
            }
          >
            <YingceApp key={start} initialPath={start} />
          </Suspense>
        </IslandBoundary>
      </div>
    </div>
  );
}

export { useCanvasHost };
