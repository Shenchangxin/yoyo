/**
 * 拥有：情绪球和在场槽位的类型，以及页面上的全局挂载点。
 * 不拥有：绘制。类型本身不另开模型循环，也不改服务端。
 */

/** VoicePhase 语音关掉、在听、在想或在说。 */
export type VoicePhase = "off" | "listening" | "thinking" | "speaking";

/** PresenceSlotId 是情绪球可以落位的槽。 */
export type PresenceSlotId = "boot" | "home" | "module" | "composer";

/** MoodMate 是一只情绪球实例能做的表情、视线和动作。 */
export type MoodMate = {
  emotionId: string;
  touring?: boolean;
  setEmotion: (id: string) => unknown;
  handleAIMessage: (msg: string | { emotionId: string; tips?: string }) => unknown;
  on: (ev: "change" | "tips" | "error", cb: (e: any) => void) => () => void;
  setGaze: (nx: number, ny: number, hold?: boolean) => MoodMate;
  clearGaze?: () => MoodMate;
  setStyle: (s: { sketch?: number }) => MoodMate;
  celebrate: (strength?: number) => MoodMate;
  flourish: () => MoodMate;
  nudge: () => MoodMate;
  signature: (strength?: number) => boolean;
  spin: (n?: number) => MoodMate;
  burst: (n?: number) => MoodMate;
  bounce: () => MoodMate;
  setActive: (on: boolean) => unknown;
  renderStatic: () => unknown;
  destroy: () => void;
  off?: (ev: string, cb: (e: any) => void) => unknown;
};

/** EmotionBallAPI 是页面上用来创建情绪球的入口。 */
export type EmotionBallAPI = {
  create: (el: string | HTMLElement, opts?: MoodMateCreateOpts) => MoodMate;
  version?: string;
  config?: {
    list?: () => unknown[];
    register?: (raw: Record<string, unknown>) => unknown;
  };
};

/** MoodMateCreateOpts 是创建时的表情、闲置和外形。 */
export type MoodMateCreateOpts = {
  emotion?: string;
  idle?: boolean | { standbyAfter?: number; sleepAfter?: number; standbyId?: string; sleepId?: string };
  autostart?: boolean;
  lite?: boolean;
  eyeScale?: number;
  fallbackId?: string;
  color?: string;
  eyeColor?: string;
  shape?: "blob" | "wedge" | "gem";
};

declare global {
  interface Window {
    EmotionBall?: EmotionBallAPI;
    EB_RINGS?: unknown;
    EMOTION_SEED?: unknown[];
    EMOTION_GROUPS?: unknown[];
    __YOYO_VIDEO_SESSION__?: string;
  }
}
