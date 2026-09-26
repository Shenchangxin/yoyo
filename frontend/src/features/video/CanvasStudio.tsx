import { Component, lazy, Suspense, useEffect, type ErrorInfo, type ReactNode } from "react";
import { useUI } from "../../lib/store";
import { yingcePathForPane } from "../../lib/protocol";
import { useCanvasHost } from "./canvas-host/session";
import { PresenceModuleLoading } from "../presence";

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

function CanvasLoading() {
  useEffect(() => {
    useUI.getState().setModuleLoading(true);
    return () => useUI.getState().setModuleLoading(false);
  }, []);
  return (
    <div className="flex h-full min-h-0 flex-col items-center justify-center" data-testid="canvas-studio-loading" aria-busy="true">
      <span className="sr-only">Opening infinite canvas…</span>
      <PresenceModuleLoading className="min-h-0 h-full" />
    </div>
  );
}

export function CanvasStudio(props: { sessionId?: string; onNeedSession?: () => void; onClose?: () => void }) {
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
      <div className="min-h-0 flex-1">
        <IslandBoundary>
          <Suspense
            fallback={<CanvasLoading />}
          >
            <YingceApp key={start} initialPath={start} />
          </Suspense>
        </IslandBoundary>
      </div>
    </div>
  );
}

export { useCanvasHost };
