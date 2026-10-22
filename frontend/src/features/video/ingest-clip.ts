import * as api from "../../lib/client";
import type { CutProject } from "./cut-desk";

export function nativeFilePath(file: File): string {
  const path = (file as File & { path?: string }).path;
  return typeof path === "string" ? path.trim() : "";
}

export function isVideoPath(path: string): boolean {
  return /\.(mp4|mov|mkv|webm|avi|m4v|mpeg|mpg)$/i.test(path || "");
}

export async function ingestVideoPath(path: string, sessionId: string): Promise<CutProject> {
  const ref = await api.video.putStudioMediaPath(path);
  const hash = String(ref?.cas_hash || ref?.hash || "");
  if (!hash) throw new Error("upload did not return a CAS hash");
  const base = path.split(/[/\\]/).pop() || "clip";
  return api.video.cutCreate(base.replace(/\.[^.]+$/, ""), hash, sessionId) as Promise<CutProject>;
}
