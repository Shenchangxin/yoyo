export const MAX_FPS = 60;
export const MAX_PIXEL_RATIO = 2;

const FRAME_MS = 1000 / MAX_FPS;
const SLACK_MS = 1;

export class FrameGate {
  private due = Number.NEGATIVE_INFINITY;

  shouldDraw(now: number): boolean {
    if (now < this.due - SLACK_MS) return false;
    const next = this.due + FRAME_MS;
    this.due = next < now ? now + FRAME_MS : next;
    return true;
  }
}

export function pixelRatio(devicePixelRatio: number, max = MAX_PIXEL_RATIO): number {
  if (!(devicePixelRatio > 0)) return 1;
  return Math.min(devicePixelRatio, max);
}
