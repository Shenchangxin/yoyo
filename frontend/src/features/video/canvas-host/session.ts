import { create } from "zustand";
import { videoCall } from "../../../lib/client";

type CanvasHostState = {
  title: string;
  projectId: string;
  saving: string;
};

export const useCanvasHost = create<CanvasHostState>(() => ({
  title: "Infinite canvas",
  projectId: "",
  saving: "",
}));

export function setCanvasHostTitle(title: string) {
  useCanvasHost.setState({ title: title || "Infinite canvas" });
}

export async function bindCanvasSession(sessionId: string, projectId: string) {
  if (!sessionId || !projectId) return;
  useCanvasHost.setState({ projectId });
  await videoCall("canvas.bind", { session_id: sessionId, project_id: projectId });
}
