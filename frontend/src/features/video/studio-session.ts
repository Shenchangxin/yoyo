import type { VideoProject } from "../../lib/protocol";

/** Bound copilot session for this short. Empty means it still needs its own chat. */
export function studioSessionId(
  projectId: string,
  projects: Pick<VideoProject, "id" | "kind" | "sessionId">[],
): string {
  const id = (projectId || "").trim();
  if (!id) return "";
  const hit = projects.find((p) => p.id === id && (!p.kind || p.kind === "studio"));
  return (hit?.sessionId || "").trim();
}

export function asStudioProject(p: {
  id: string;
  title?: string;
  sessionId?: string;
  status?: string;
  gate?: string;
}): VideoProject {
  return {
    kind: "studio",
    id: p.id,
    title: p.title || "",
    sessionId: p.sessionId || "",
    status: p.status,
    gate: p.gate,
  };
}
