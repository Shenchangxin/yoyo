import { useEffect, useState } from "react";
import { create } from "zustand";
import * as api from "../../lib/client";

export type CutCue = { start?: number; end?: number; text?: string; omit?: boolean };
export type CutClip = {
  id?: string;
  hash?: string;
  start?: number;
  end?: number;
  title?: string;
  scores?: Record<string, number>;
  media_url?: string;
};
export type CutProject = {
  id?: string;
  title?: string;
  source_hash?: string;
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
  setProject: (project: CutProject | null, sessionId?: string) => void;
  noteSession: (sessionId: string) => void;
  adoptServer: (sessionId: string, project: CutProject | null) => void;
  clear: () => void;
};

export const useCutDesk = create<CutDeskState>((set) => ({
  sessionId: "",
  project: null,
  setProject: (project, sessionId) => set((s) => ({
    project,
    sessionId: sessionId ?? s.sessionId,
  })),
  noteSession: (sessionId) => set((s) => (s.sessionId === sessionId ? s : { sessionId, project: null })),
  adoptServer: (sessionId, project) => set((s) => {
    if (s.sessionId === sessionId && s.project?.source_hash && !project?.source_hash) return s;
    if (project?.source_hash) return { project, sessionId };
    if (s.sessionId === sessionId) return { project: null, sessionId };
    return s;
  }),
  clear: () => set({ project: null, sessionId: "" }),
}));

export function cutClips(project: CutProject | null): CutClip[] {
  if (!project) return [];
  return project.clips || project.breakdown?.clips || [];
}

export function cutCues(project: CutProject | null): CutCue[] {
  return project?.captions?.cues || [];
}

export type CutReadiness = { ffmpeg: boolean; whisper: boolean; speech: boolean; loaded: boolean };

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
