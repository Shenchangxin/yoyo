import { getRemoteCanvasProject } from "@yingce/services/api/user-data";
import { useCanvasStore } from "@yingce/stores/canvas/use-canvas-store";
import { useCanvasHost } from "./session";

function eventKind(ev: any) {
  return String(ev?.item_kind || ev?.ItemKind || ev?.payload?.kind || "");
}

function eventPayload(ev: any) {
  return ev?.payload || ev?.Payload || ev || {};
}

async function wailsEvents(): Promise<{ On?: (name: string, cb: (e: any) => void) => () => void } | null> {
  try {
    const mod: any = await import("@wailsio/runtime");
    return mod.Events || null;
  } catch {
    return null;
  }
}

export function installCanvasHub() {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let off: (() => void) | undefined;
  let alive = true;
  const reload = () => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      const id = useCanvasHost.getState().projectId;
      if (!id) return;
      void getRemoteCanvasProject(id)
        .then((doc: any) => {
          const project = doc?.project || doc;
          if (!project?.id) return;
          const store = useCanvasStore.getState();
          store.importProject(project);
        })
        .catch(() => {});
    }, 40);
  };

  void wailsEvents().then((Events) => {
    if (!alive || !Events?.On) return;
    off = Events.On("yoyo:item", (ev: any) => {
      const kind = eventKind(ev);
      const payload = eventPayload(ev);
      if (kind === "canvas_patch" || kind === "canvas_text_delta" || kind === "video_job") {
        const canvasId = String(payload.canvas_id || payload.canvasId || payload.project_id || "");
        if (canvasId) useCanvasHost.setState({ projectId: canvasId });
        reload();
      }
    });
  });

  return () => {
    alive = false;
    if (timer) clearTimeout(timer);
    off?.();
  };
}
