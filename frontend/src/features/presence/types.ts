export type VoicePhase = "off" | "listening" | "thinking" | "speaking";

export type PresenceSlotId = "boot" | "home" | "process" | "avatar" | "module";

export type MoodMate = {
  emotionId: string;
  touring?: boolean;
  setEmotion: (id: string) => unknown;
  handleAIMessage: (msg: string | { emotionId: string; tips?: string }) => unknown;
  on: (ev: "change" | "tips" | "error", cb: (e: any) => void) => () => void;
  setGaze: (nx: number, ny: number) => MoodMate;
  clearGaze?: () => MoodMate;
  setStyle: (s: { sketch?: number }) => MoodMate;
  celebrate: (strength?: number) => MoodMate;
  signature: (strength?: number) => boolean;
  spin: (n?: number) => MoodMate;
  burst: (n?: number) => MoodMate;
  bounce: () => MoodMate;
  setActive: (on: boolean) => unknown;
  renderStatic: () => unknown;
  destroy: () => void;
  off?: (ev: string, cb: (e: any) => void) => unknown;
};

export type EmotionBallAPI = {
  create: (el: string | HTMLElement, opts?: MoodMateCreateOpts) => MoodMate;
  version?: string;
  config?: {
    list?: () => unknown[];
    register?: (raw: Record<string, unknown>) => unknown;
  };
};

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
