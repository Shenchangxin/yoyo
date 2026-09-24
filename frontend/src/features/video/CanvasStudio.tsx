import { lazy, Suspense, useEffect } from "react";
import { useCanvasHost } from "./canvas-host/session";

const YingceApp = lazy(() => import("./canvas-host/YingceApp"));

export function CanvasStudio(props: { sessionId?: string; onNeedSession?: () => void }) {
  useEffect(() => {
    if (typeof window !== "undefined") window.__YOYO_VIDEO_SESSION__ = props.sessionId || "";
  }, [props.sessionId]);
  useEffect(() => {
    if (!props.sessionId) props.onNeedSession?.();
  }, [props.sessionId, props.onNeedSession]);

  return (
    <Suspense
      fallback={
        <div className="flex h-full items-center justify-center text-[13px] text-muted" data-testid="canvas-studio-loading">
          Opening infinite canvas…
        </div>
      }
    >
      <YingceApp />
    </Suspense>
  );
}

export { useCanvasHost };
