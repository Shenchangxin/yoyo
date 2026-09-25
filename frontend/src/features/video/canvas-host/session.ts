import { create } from "zustand";
import { videoCall } from "../../../lib/client";

type CanvasHostState = {
  title: string;
  projectId: string;
  saving: string;
  sessionId: string;
};

export const useCanvasHost = create<CanvasHostState>(() => ({
  title: "Infinite canvas",
  projectId: "",
  saving: "",
  sessionId: "",
}));

export function setCanvasHostTitle(title: string) {
  useCanvasHost.setState({ title: title || "Infinite canvas" });
}

export async function bindCanvasSession(sessionId: string, projectId: string) {
  if (!sessionId || !projectId) return;
  useCanvasHost.setState({ projectId });
  await videoCall("canvas.bind", { session_id: sessionId, project_id: projectId });
}

export async function ensureCanvasProject(sessionId: string, title = "Infinite canvas") {
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
