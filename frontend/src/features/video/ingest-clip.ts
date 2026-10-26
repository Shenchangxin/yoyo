/**
 * 本文件拥有从本机路径判断视频，并据此新建一条剪辑。不拥有剪辑台界面。
 * 写入会把路径登记成媒体并创建剪辑工程，从而改服务端状态。
 */
import * as api from "../../lib/client";
import type { CutProject } from "./cut-desk";

/** nativeFilePath 读取文件对象上的本机路径，没有就返回空字符串。file 读取 path，是字符串就去掉空白，否则返回空。 */
export function nativeFilePath(file: File): string {
  const path = (file as File & { path?: string }).path;
  return typeof path === "string" ? path.trim() : "";
}

/** isVideoPath 看扩展名是不是本文件列出的视频后缀。path 用列出的视频后缀做匹配，空的当空字符串。 */
export function isVideoPath(path: string): boolean {
  return /\.(mp4|mov|mkv|webm|avi|m4v|mpeg|mpg)$/i.test(path || "");
}

/** ingestVideoPath 把本机路径登记成媒体，再用文件名和会话创建剪辑工程。path 登记成媒体，并取出文件名当剪辑标题。sessionId 交给 cutCreate。 */
export async function ingestVideoPath(path: string, sessionId: string): Promise<CutProject> {
  const ref = await api.video.putStudioMediaPath(path);
  const hash = String(ref?.cas_hash || ref?.hash || "");
  if (!hash) throw new Error("upload did not return a CAS hash");
  const base = path.split(/[/\\]/).pop() || "clip";
  return api.video.cutCreate(base.replace(/\.[^.]+$/, ""), hash, sessionId) as Promise<CutProject>;
}
