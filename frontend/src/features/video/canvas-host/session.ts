/**
 * 本文件拥有托管画布的标题、工程 id 和会话，以及绑定、创建工程。
 * 不拥有画布节点。绑定和创建工程会改服务端。它不是另一条模型循环。
 */
import { create } from "zustand";
import { videoCall } from "../../../lib/client";

type CanvasHostState = {
  title: string;
  projectId: string;
  saving: string;
  sessionId: string;
  ensureSession: () => Promise<string>;
};

/** useCanvasHost 保存画布标题、工程、保存标记和当前会话。 */
export const useCanvasHost = create<CanvasHostState>(() => ({
  title: "",
  projectId: "",
  saving: "",
  sessionId: "",
  /** 还没有会话时返回空字符串。 */
  ensureSession: async () => "",
}));

/** setCanvasHostTitle 只改宿主上的标题。title 空值写成空字符串后写入宿主。 */
export function setCanvasHostTitle(title: string) {
  useCanvasHost.setState({ title: title || "" });
}

/** bindCanvasSession 记下工程 id，并把会话绑到这块画布。sessionId 或 projectId 为空就返回。projectId 写入宿主，并和 sessionId 一起交给 canvas.bind。 */
export async function bindCanvasSession(sessionId: string, projectId: string) {
  if (!sessionId || !projectId) return;
  useCanvasHost.setState({ projectId });
  await videoCall("canvas.bind", { session_id: sessionId, project_id: projectId });
}

/** ensureCanvasProject 没有工程就新建一块，再把会话绑上去。sessionId 在有工程时交给 bindCanvasSession。title 作为新建画布的标题，工程没带回标题时也用它。 */
export async function ensureCanvasProject(sessionId: string, title = "") {
  let id = useCanvasHost.getState().projectId;
  if (!id) {
    const env = await videoCall("canvas.http", { method: "POST", path: "canvas-projects", body: { title } });
    const project = env?.data?.project || env?.project;
    if (project?.id) {
      useCanvasHost.setState({ projectId: project.id, title: project.title || title });
      id = String(project.id);
    }
  }
  if (id && sessionId) await bindCanvasSession(sessionId, id);
  return id;
}
