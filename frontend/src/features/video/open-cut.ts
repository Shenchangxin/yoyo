/**
 * 本文件拥有「送到剪辑」的窗口事件，以及把成片地址抄进媒体库的步骤。
 * 不拥有剪辑台。ingestClipUrl 会把文件写入服务端；其余函数只发事件或拼文案。
 */
import { toast } from "sonner";
import * as api from "../../lib/client";
import type { Copy } from "../../lib/copy";

/** SEND_TO_CUT_EVENT 是送到剪辑时派发的事件名。 */
export const SEND_TO_CUT_EVENT = "yoyo:send-to-cut";

/** SendToCutRequest 是送到剪辑时带上的哈希、资源、地址、标题和会话。 */
export type SendToCutRequest = {
  hash?: string;
  resourceId?: string;
  resource_id?: string;
  storageKey?: string;
  url?: string;
  title?: string;
  sessionId?: string;
};

/** requestSendToCut 在窗口上派发送到剪辑的事件。没有窗口就返回。detail 作为送到剪辑事件的 detail。 */
export function requestSendToCut(detail: SendToCutRequest) {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(SEND_TO_CUT_EVENT, { detail }));
}

/** installSendToCut 接住送到剪辑的事件，失败时弹出错误。返回卸下监听的函数。handler 收到事件里的 detail，失败时用错误弹出提示。 */
export function installSendToCut(handler: (detail: SendToCutRequest) => Promise<void>) {
  /** 交给处理函数，失败时弹出错误。raw 取出 detail 交给 handler。 */
  const on = (raw: Event) => {
    const detail = ((raw as CustomEvent<SendToCutRequest>).detail || {}) as SendToCutRequest;
    void handler(detail).catch(/** e 交给 errMessage 后弹出错误。 */ (e) => toast.error(api.errMessage(e)));
  };
  window.addEventListener(SEND_TO_CUT_EVENT, on);
  return () => window.removeEventListener(SEND_TO_CUT_EVENT, on);
}

/** resourceIdFromSend 从资源字段或地址里取出资源 id。detail 依次看 resourceId、resource_id、storageKey 和 url。 */
export function resourceIdFromSend(detail: SendToCutRequest): string {
  const key = String(detail.resourceId || detail.resource_id || detail.storageKey || "").trim();
  if (key.startsWith("resource:")) return key.slice("resource:".length);
  const url = String(detail.url || "");
  const m = /\/resources?\/([^/?#]+)/.exec(url);
  if (m) return m[1];
  if (key && !/[/:.]/.test(key)) return key;
  return "";
}

/** cutPrompt 把剪辑 id 和哈希填进文案模板。copy 取 conductor.sendPrompt。hash 替换模板里的 {hash}。 */
export function cutPrompt(copy: Copy, id: string, hash: string): string {
  return copy.video.conductor.sendPrompt.replace("{id}", id).replace("{hash}", hash);
}

/** ingestClipUrl 拉取成片地址，小于本文件的大小上限时写入媒体库并返回哈希。url 去掉空白后拉取，空的直接抛错。 */
export async function ingestClipUrl(url: string): Promise<string> {
  const src = String(url || "").trim();
  if (!src) throw new Error("clip url required");
  const res = await fetch(src);
  if (!res.ok) throw new Error(`fetch clip failed (${res.status})`);
  const blob = await res.blob();
  if (blob.size > 64 * 1024 * 1024) {
    throw new Error("clip too large to copy into cut");
  }
  const b64 = await blobToB64(blob);
  const ref = await api.video.putStudioMedia(b64, blob.type || "video/mp4", "video");
  const hash = String(ref?.cas_hash || ref?.hash || "");
  if (!hash) throw new Error("upload did not return a CAS hash");
  return hash;
}

/** 把二进制读成去掉前缀的 base64。blob 交给 FileReader 读成 data URL。 */
function blobToB64(blob: Blob): Promise<string> {
  return new Promise(/** resolve 在读完后交出去掉前缀的 base64。reject 在读失败时交上 FileReader 的错误。 */ (resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const data = String(r.result || "");
      resolve(data.includes(",") ? data.slice(data.indexOf(",") + 1) : data);
    };
    r.onerror = () => reject(r.error);
    r.readAsDataURL(blob);
  });
}
