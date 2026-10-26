/**
 * 本文件拥有把画布补丁事件收成一次重新读取。不拥有画布节点编辑。
 * 它只把远端工程读进本地画布状态，不在这里写服务端，也不是另一条模型循环。
 */
import { getRemoteCanvasProject } from "@yingce/services/api/user-data";
import { useCanvasStore } from "@yingce/stores/canvas/use-canvas-store";
import { useCanvasHost } from "./session";

/** 取出事件上的条目种类。ev 依次看 item_kind、ItemKind 和 payload.kind。 */
function eventKind(ev: any) {
  return String(ev?.item_kind || ev?.ItemKind || ev?.payload?.kind || "");
}

/** 取出事件载荷，没有就用事件本身。ev 先取 payload 或 Payload，都没有就用自身。 */
function eventPayload(ev: any) {
  return ev?.payload || ev?.Payload || ev || {};
}

/** 尝试拿到 Wails 事件。拿不到就返回空。 */
async function wailsEvents(): Promise<{ On?: (name: string, cb: (e: any) => void) => () => void } | null> {
  try {
    const mod: any = await import("@wailsio/runtime");
    return mod.Events || null;
  } catch {
    return null;
  }
}

/** installCanvasHub 在画布补丁、文本增量或视频任务到来时，40 毫秒后重读当前工程。 */
export function installCanvasHub() {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let off: (() => void) | undefined;
  let alive = true;
  /** 合并短时间内的重读，把远端工程导入本地画布。 */
  const reload = () => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      const id = useCanvasHost.getState().projectId;
      if (!id) return;
      void getRemoteCanvasProject(id)
        .then(/** doc 先取 project，没有就用自身；有 id 才交给 importProject。 */ (doc: any) => {
          const project = doc?.project || doc;
          if (!project?.id) return;
          const store = useCanvasStore.getState();
          store.importProject(project);
        })
        .catch(() => {});
    }, 40);
  };

  void wailsEvents().then(/** Events 没有 On 就返回，否则用 On 听 yoyo:item。 */ (Events) => {
    if (!alive || !Events?.On) return;
    /** 只重读当前画布；还没选工程时先记下事件里的画布 id。 */
    off = Events.On("yoyo:item", /** ev 交给 eventKind 和 eventPayload，用来判断要不要重读当前画布。 */ (ev: any) => {
      const kind = eventKind(ev);
      const payload = eventPayload(ev);
      if (kind === "canvas_patch" || kind === "canvas_text_delta" || kind === "video_job") {
        const canvasId = String(payload.canvas_id || payload.canvasId || payload.project_id || "");
        const current = useCanvasHost.getState().projectId;
        if (canvasId && current && canvasId !== current) return;
        if (canvasId && !current) useCanvasHost.setState({ projectId: canvasId });
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
