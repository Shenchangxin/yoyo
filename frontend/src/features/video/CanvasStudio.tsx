import { Component, lazy, Suspense, useEffect, useRef, type ErrorInfo, type ReactNode } from "react";
import { useUI } from "../../lib/store";
import { useCopy } from "../../lib/i18n";
import { yingceHref } from "../../lib/protocol";
import { useCanvasHost } from "./canvas-host/session";
import { PresenceModuleLoading } from "../presence";

const YingceApp = lazy(() =>
  import("./canvas-host/YingceApp").catch((err: Error) => ({
    default: function YingceLoadError() {
      return <CanvasCrash message={String(err?.message || err)} />;
    },
  })),
);

function CanvasCrash(props: { message: string }) {
  const copy = useCopy();
  return (
    <div className="flex h-full items-center justify-center p-6 text-center text-[13px] text-muted" data-testid="canvas-studio-error">
      {copy.video.failed}{props.message ? ` · ${props.message}` : ""}
    </div>
  );
}

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
      return <CanvasCrash message={this.state.err.message} />;
    }
    return this.props.children;
  }
}

function CanvasLoading() {
  const copy = useCopy();
  useEffect(() => {
    useUI.getState().setModuleLoading(true);
    return () => useUI.getState().setModuleLoading(false);
  }, []);
  return (
    <div className="flex h-full min-h-0 flex-col items-center justify-center" data-testid="canvas-studio-loading" aria-busy="true">
      <span className="sr-only">{copy.video.canvasOpening}</span>
      <PresenceModuleLoading className="min-h-0 h-full" />
    </div>
  );
}

export function CanvasStudio(props: { sessionId?: string; onNeedSession?: () => void | Promise<string | void>; onClose?: () => void }) {
  const pane = useUI((s) => s.videoPane);
  const canvasFocus = useUI((s) => s.canvasFocus);
  const videoEntry = useUI((s) => s.videoEntry);
  const projectId = useCanvasHost((s) => s.projectId);
  const start = yingceHref(pane, projectId, canvasFocus, videoEntry);
  const needSession = useRef(props.onNeedSession);
  needSession.current = props.onNeedSession;
  const askedSession = useRef(false);

  useEffect(() => {
    if (typeof window !== "undefined") window.__YOYO_VIDEO_SESSION__ = props.sessionId || "";
    useCanvasHost.setState({
      sessionId: props.sessionId || "",
      ensureSession: async () => {
        if (props.sessionId) return props.sessionId;
        const id = await needSession.current?.();
        const next = (typeof id === "string" && id)
          || useCanvasHost.getState().sessionId
          || (typeof window !== "undefined" ? String(window.__YOYO_VIDEO_SESSION__ || "") : "");
        if (next) {
          useCanvasHost.setState({ sessionId: next });
          if (typeof window !== "undefined") window.__YOYO_VIDEO_SESSION__ = next;
        }
        return next;
      },
    });
  }, [props.sessionId]);
  useEffect(() => {
    if (props.sessionId) {
      askedSession.current = false;
      return;
    }
    if (pane === "drama") return;
    if (askedSession.current) return;
    askedSession.current = true;
    void needSession.current?.();
  }, [pane, props.sessionId]);

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="canvas-studio-shell" data-pane={pane}>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
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
