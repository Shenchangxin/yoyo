/**
 * 本文件拥有剪辑台的本地工程指针、片段和字幕的读取，以及一次就绪探测。
 * 不拥有时间线界面。探测只读视频状态和语音提供方菜单，不改服务端。
 */
import { useEffect, useState } from "react";
import { create } from "zustand";
import * as api from "../../lib/client";

/** CutCue 是一条字幕，带起止、文本和是否略过。 */
export type CutCue = { start?: number; end?: number; text?: string; omit?: boolean };
/** CutClip 是一个候选片段，带哈希、起止、分数和地址。 */
export type CutClip = {
  id?: string;
  hash?: string;
  start?: number;
  end?: number;
  title?: string;
  scores?: Record<string, number>;
  media_url?: string;
};
/** CutProject 是剪辑工程在界面上用到的字段，含素材、成片、字幕和分轨地址。 */
export type CutProject = {
  id?: string;
  title?: string;
  session_id?: string;
  source_hash?: string;
  kind?: string;
  reference_hash?: string;
  note?: string;
  result_hash?: string;
  duration?: number;
  stale?: boolean;
  media_url?: string;
  source_url?: string;
  result_url?: string;
  poster_url?: string;
  dub_url?: string;
  voice_url?: string;
  music_url?: string;
  updated_at?: string;
  breakdown?: { scenes?: number[]; clips?: CutClip[] };
  edit_list?: { cuts?: CutClip[] };
  clips?: CutClip[];
  captions?: { cues?: CutCue[] };
  stems?: { dub?: string; voice?: string; music?: string };
};

type CutDeskState = {
  sessionId: string;
  project: CutProject | null;
  /** 用户离开剪辑台回到创作入口。不要把已绑定的成片再弹开。 */
  parked: boolean;
  setProject: (project: CutProject | null, sessionId?: string) => void;
  noteSession: (sessionId: string) => void;
  adoptServer: (sessionId: string, project: CutProject | null) => void;
  park: () => void;
  clear: () => void;
};

/** useCutDesk 记住当前剪辑工程、会话，以及用户是否已经离开这张桌子。 */
export const useCutDesk = create<CutDeskState>((set) => ({
  sessionId: "",
  project: null,
  parked: false,
  /** 写入工程并取消停靠；没给会话就沿用原来的。 */
  setProject: (project, sessionId) => set((s) => ({
    project,
    parked: false,
    sessionId: sessionId ?? s.sessionId,
  })),
  /** 会话变了就清掉工程并取消停靠。 */
  noteSession: (sessionId) => set((s) => (
    s.sessionId === sessionId ? s : { sessionId, project: null, parked: false }
  )),
  /** 同一会话上，有素材哈希的工程会打开；停靠中则保持关掉；本地有素材而传来的没有时不覆盖。 */
  adoptServer: (sessionId, project) => set((s) => {
    if (sessionId && s.sessionId && s.sessionId !== sessionId) return s;
    if (s.parked && s.sessionId === sessionId) return { sessionId, project: null, parked: true };
    if (s.sessionId === sessionId && s.project?.source_hash && !project?.source_hash) return s;
    if (project?.source_hash) return { project, sessionId, parked: false };
    if (s.sessionId === sessionId) return { project: null, sessionId, parked: s.parked };
    return s;
  }),
  /** 清掉工程并标记停靠，会话 id 保留。 */
  park: () => set((s) => ({ project: null, parked: true, sessionId: s.sessionId })),
  /** 工程、会话和停靠一起清掉。 */
  clear: () => set({ project: null, sessionId: "", parked: false }),
}));

/** cutClips 优先用工程上的 clips，否则用 breakdown 里的片段。 */
export function cutClips(project: CutProject | null): CutClip[] {
  if (!project) return [];
  return project.clips || project.breakdown?.clips || [];
}

/** cutCues 取出字幕条目，没有就是空数组。 */
export function cutCues(project: CutProject | null): CutCue[] {
  return project?.captions?.cues || [];
}

/** CutReadiness 记下 ffmpeg、whisper、语音提供方是否可用，以及这次探测是否已经返回。 */
export type CutReadiness = { ffmpeg: boolean; whisper: boolean; speech: boolean; loaded: boolean };

/** useCutReadiness 读一次视频状态和语音提供方菜单。 */
export function useCutReadiness(): CutReadiness {
  const [ready, setReady] = useState<CutReadiness>({ ffmpeg: false, whisper: false, speech: false, loaded: false });
  useEffect(() => {
    let alive = true;
    void Promise.all([
      api.video.status().catch(() => ({})),
      api.video.providerMenu("speech").catch(() => []),
    ]).then(([status, speech]) => {
      if (!alive) return;
      setReady({
        ffmpeg: !!status?.ffmpeg,
        whisper: !!status?.whisper,
        speech: Array.isArray(speech) && speech.length > 0,
        loaded: true,
      });
    });
    return () => { alive = false; };
  }, []);
  return ready;
}
