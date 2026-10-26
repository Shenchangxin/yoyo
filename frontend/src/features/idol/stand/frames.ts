/**
 * 拥有：立绘画布的帧率闸和设备像素比上限。
 * 不拥有：骨骼、镜头和服装加载。本地节流，不另开模型循环，不改服务端。
 */

/** MAX_FPS 绘制帧率上限。 */
export const MAX_FPS = 60;
/** MAX_PIXEL_RATIO 画布设备像素比上限。 */
export const MAX_PIXEL_RATIO = 2;

const FRAME_MS = 1000 / MAX_FPS;
const SLACK_MS = 1;

/** FrameGate 按帧率上限决定这一时刻画不画。 */
export class FrameGate {
  private due = Number.NEGATIVE_INFINITY;

  /** shouldDraw 未到点则拒绝。到点后把下一次到期推后一帧，已经落后则从现在重算。 */
  shouldDraw(now: number): boolean {
    if (now < this.due - SLACK_MS) return false;
    const next = this.due + FRAME_MS;
    this.due = next < now ? now + FRAME_MS : next;
    return true;
  }
}

/** pixelRatio 在设备像素比无效时用 1，否则不超过上限。 */
export function pixelRatio(devicePixelRatio: number, max = MAX_PIXEL_RATIO): number {
  if (!(devicePixelRatio > 0)) return 1;
  return Math.min(devicePixelRatio, max);
}
