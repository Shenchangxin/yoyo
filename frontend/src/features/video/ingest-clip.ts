import * as api from "../../lib/client";
import type { CutProject } from "./cut-desk";

const CLIP_CAP = 64 * 1024 * 1024;

export function nativeFilePath(file: File): string {
  const path = (file as File & { path?: string }).path;
  return typeof path === "string" ? path.trim() : "";
}

export function isVideoPath(path: string): boolean {
  return /\.(mp4|mov|mkv|webm|avi|m4v|mpeg|mpg)$/i.test(path || "");
}

export async function ingestVideoFile(file: File, sessionId: string): Promise<CutProject> {
  if (file.size > CLIP_CAP) {
    throw new Error("clip too large");
  }
  const data = await new Promise<string>((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result || ""));
    r.onerror = () => reject(r.error);
    r.readAsDataURL(file);
  });
  const b64 = data.includes(",") ? data.slice(data.indexOf(",") + 1) : data;
  const ref = await api.video.putStudioMedia(b64, file.type || "video/mp4", "video");
  const hash = String(ref?.cas_hash || ref?.hash || "");
  if (!hash) throw new Error("upload did not return a CAS hash");
  return api.video.cutCreate(file.name.replace(/\.[^.]+$/, ""), hash, sessionId) as Promise<CutProject>;
}

export async function ingestVideoPath(path: string, sessionId: string): Promise<CutProject> {
  const ref = await api.video.putStudioMediaPath(path);
  const hash = String(ref?.cas_hash || ref?.hash || "");
  if (!hash) throw new Error("upload did not return a CAS hash");
  const base = path.split(/[/\\]/).pop() || "clip";
  return api.video.cutCreate(base.replace(/\.[^.]+$/, ""), hash, sessionId) as Promise<CutProject>;
}
