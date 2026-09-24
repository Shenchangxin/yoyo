import { video } from "../../../lib/client";

declare global {
  interface Window {
    __YOYO_MEDIA_BASE__?: string;
    __YOYO_VIDEO_SESSION__?: string;
  }
}

export async function primeCanvasMediaBase() {
  try {
    const status = await video.status();
    const base = String(status?.media_base || status?.MediaBase || "").replace(/\/+$/, "");
    if (base && typeof window !== "undefined") window.__YOYO_MEDIA_BASE__ = base;
  } catch {
    /* media server may be down in e2e */
  }
}
