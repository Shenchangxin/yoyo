/**
 * 本文件拥有短片和工作区项目之间的会话对照。不拥有聊天记录，也不打开副驾。
 * 它只读传入的列表，不改服务端。
 */
import type { VideoProject } from "../../lib/protocol";

/** studioSessionId 返回这条短片绑定的副驾会话。空字符串表示还要自己的聊天。 */
export function studioSessionId(
  projectId: string,
  projects: Pick<VideoProject, "id" | "kind" | "sessionId">[],
): string {
  const id = (projectId || "").trim();
  if (!id) return "";
  const hit = projects.find((p) => p.id === id && (!p.kind || p.kind === "studio"));
  return (hit?.sessionId || "").trim();
}

/** asStudioProject 把一条短片收成工作区项目，种类固定为 studio。 */
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
