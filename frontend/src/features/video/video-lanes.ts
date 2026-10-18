import type { VideoProject } from "../../lib/protocol";

export type VideoLane = "create" | "drama" | "canvas";

const KEY = "yoyo-video-lane-threads";

function read(): Partial<Record<VideoLane, string>> {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) || "{}");
    return raw && typeof raw === "object" ? raw : {};
  } catch {
    return {};
  }
}

export function lastLaneThread(lane: VideoLane): string {
  return String(read()[lane] || "").trim();
}

export function rememberLaneThread(lane: VideoLane, id: string) {
  const sessionId = (id || "").trim();
  if (!sessionId) return;
  try {
    localStorage.setItem(KEY, JSON.stringify({ ...read(), [lane]: sessionId }));
  } catch {
    /* ignore */
  }
}

export function sessionLane(
  sessionId: string,
  projects: Pick<VideoProject, "kind" | "sessionId">[],
): VideoLane | "studio" | "" {
  const id = (sessionId || "").trim();
  if (!id) return "";
  const hit = projects.find((p) => (p.sessionId || "").trim() === id);
  if (!hit) return "";
  if (hit.kind === "studio") return "studio";
  if (hit.kind === "drama") return "drama";
  if (hit.kind === "canvas") return "canvas";
  return "";
}

export function laneAcceptsSession(
  lane: VideoLane,
  sessionId: string,
  projects: Pick<VideoProject, "kind" | "sessionId">[],
): boolean {
  const kind = sessionLane(sessionId, projects);
  if (lane === "create") return !kind;
  if (kind) return kind === lane;
  return lastLaneThread(lane) === sessionId;
}
