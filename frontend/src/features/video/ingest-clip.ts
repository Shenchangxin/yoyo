/**
 * 本文件拥有从本机路径或拖进来的文件判断视频，并据此新建一条剪辑。不拥有剪辑台界面。
 * 写入会把路径或文件字节登记成媒体并创建剪辑工程，从而改服务端状态。
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

/** DROP_UPLOAD_LIMIT 是拖进来、拿不到本机路径的视频最多走窗口上传的字节数。更大的用「选择视频」按路径入库，不经过窗口。 */
export const DROP_UPLOAD_LIMIT = 150 * 1024 * 1024;

/** CLIP_TOO_LARGE 是拖进来的视频超过上传上限时抛出的错误文字。 */
export const CLIP_TOO_LARGE = "clip too large";

/** isClipTooLarge 判断错误是不是拖进来的视频太大。 */
export function isClipTooLarge(e: unknown): boolean {
  return e instanceof Error && e.message === CLIP_TOO_LARGE;
}

/** ingestVideoFile 把拖进来的视频字节写进媒体库，再用文件名和会话创建剪辑工程。桌面窗口里拖进来的文件常常没有本机路径，就走这条。超过 DROP_UPLOAD_LIMIT 时抛 CLIP_TOO_LARGE。file 读成 base64 上传。sessionId 交给 cutCreate。 */
export async function ingestVideoFile(file: File, sessionId: string): Promise<CutProject> {
  if (file.size > DROP_UPLOAD_LIMIT) throw new Error(CLIP_TOO_LARGE);
  const b64 = await new Promise<string>(/** resolve 交出去掉前缀的 base64，reject 交出读取错误。 */ (resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const data = String(r.result || "");
      resolve(data.includes(",") ? data.slice(data.indexOf(",") + 1) : data);
    };
    r.onerror = () => reject(r.error);
    r.readAsDataURL(file);
  });
  const ref = await api.video.putStudioMedia(b64, file.type || "video/mp4", "video");
  const hash = String(ref?.cas_hash || ref?.hash || "");
  if (!hash) throw new Error("upload did not return a CAS hash");
  return api.video.cutCreate(file.name.replace(/\.[^.]+$/, "") || "clip", hash, sessionId) as Promise<CutProject>;
}
