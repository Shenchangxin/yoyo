/**
 * 本文件拥有把视频状态里的媒体根地址写到窗口上。不拥有播放器。
 * 它只读状态，不改服务端。
 */
import { video } from "../../../lib/client";

declare global {
  interface Window {
    __YOYO_MEDIA_BASE__?: string;
    __YOYO_VIDEO_SESSION__?: string;
  }
}

/** primeCanvasMediaBase 读到媒体根地址就去掉结尾斜杠，写到窗口上。 */
export async function primeCanvasMediaBase() {
  try {
    const status = await video.status();
    const base = String(status?.media_base || status?.MediaBase || "").replace(/\/+$/, "");
    if (base && typeof window !== "undefined") window.__YOYO_MEDIA_BASE__ = base;
  } catch {
    /* 端到端里媒体服务可能没起来 */
  }
}
