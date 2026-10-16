import { toast } from "sonner";
import * as api from "../../lib/client";
import type { Copy } from "../../lib/copy";

export const SEND_TO_CUT_EVENT = "yoyo:send-to-cut";

export type SendToCutRequest = {
  hash?: string;
  resourceId?: string;
  resource_id?: string;
  storageKey?: string;
  url?: string;
  title?: string;
  sessionId?: string;
};

export function requestSendToCut(detail: SendToCutRequest) {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(SEND_TO_CUT_EVENT, { detail }));
}

export function installSendToCut(handler: (detail: SendToCutRequest) => Promise<void>) {
  const on = (raw: Event) => {
    const detail = ((raw as CustomEvent<SendToCutRequest>).detail || {}) as SendToCutRequest;
    void handler(detail).catch((e) => toast.error(api.errMessage(e)));
  };
  window.addEventListener(SEND_TO_CUT_EVENT, on);
  return () => window.removeEventListener(SEND_TO_CUT_EVENT, on);
}

export function resourceIdFromSend(detail: SendToCutRequest): string {
  const key = String(detail.resourceId || detail.resource_id || detail.storageKey || "").trim();
  if (key.startsWith("resource:")) return key.slice("resource:".length);
  const url = String(detail.url || "");
  const m = /\/resources?\/([^/?#]+)/.exec(url);
  if (m) return m[1];
  if (key && !/[/:.]/.test(key)) return key;
  return "";
}

export function cutPrompt(copy: Copy, id: string, hash: string): string {
  return copy.video.conductor.sendPrompt.replace("{id}", id).replace("{hash}", hash);
}

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

function blobToB64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const data = String(r.result || "");
      resolve(data.includes(",") ? data.slice(data.indexOf(",") + 1) : data);
    };
    r.onerror = () => reject(r.error);
    r.readAsDataURL(blob);
  });
}
