import type { VideoProject } from "../../lib/protocol";

type GameProjectRef = Pick<VideoProject, "id" | "kind" | "sessionId">;

function isGame(p: GameProjectRef): boolean {
  return !p.kind || p.kind === "game";
}

/** Bound copilot session for this world. Empty means it still needs its own chat. */
export function gameSessionId(
  projectId: string,
  projects: GameProjectRef[],
): string {
  const id = (projectId || "").trim();
  if (!id) return "";
  const hit = projects.find((p) => p.id === id && isGame(p));
  return (hit?.sessionId || "").trim();
}

/** True when this session already belongs to a different world. */
export function sessionBoundToOtherGame(
  sessionId: string,
  projectId: string,
  projects: GameProjectRef[],
): boolean {
  const sid = (sessionId || "").trim();
  const pid = (projectId || "").trim();
  if (!sid || !pid) return false;
  return projects.some((p) => isGame(p) && (p.sessionId || "").trim() === sid && p.id !== pid);
}

/** Keep one session on one world in the local project list. */
export function exclusiveGameProjects(
  projects: VideoProject[],
  project: VideoProject,
  sessionId: string,
): VideoProject[] {
  const row = asGameProject({ ...project, sessionId });
  const sid = (sessionId || "").trim();
  return [
    row,
    ...projects
      .filter((x) => !(x.kind === "game" && x.id === row.id))
      .map((x) => (x.kind === "game" && sid && (x.sessionId || "").trim() === sid ? { ...x, sessionId: "" } : x)),
  ];
}

export function asGameProject(p: {
  id: string;
  title?: string;
  sessionId?: string;
}): VideoProject {
  return {
    kind: "game",
    id: p.id,
    title: p.title || "",
    sessionId: p.sessionId || "",
  };
}
