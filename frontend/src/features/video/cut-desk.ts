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
  /** User left the desk for the creation entrance. Do not snap the bound clip back open. */
  parked: boolean;
  setProject: (project: CutProject | null, sessionId?: string) => void;
  noteSession: (sessionId: string) => void;
  adoptServer: (sessionId: string, project: CutProject | null) => void;
  park: () => void;
  clear: () => void;
};

export const useCutDesk = create<CutDeskState>((set) => ({
  sessionId: "",
  project: null,
  parked: false,
  setProject: (project, sessionId) => set((s) => ({
    project,
    parked: false,
    sessionId: sessionId ?? s.sessionId,
  })),
  noteSession: (sessionId) => set((s) => (
    s.sessionId === sessionId ? s : { sessionId, project: null, parked: false }
  )),
  adoptServer: (sessionId, project) => set((s) => {
    if (sessionId && s.sessionId && s.sessionId !== sessionId) return s;
    if (s.parked && s.sessionId === sessionId) return { sessionId, project: null, parked: true };
    if (s.sessionId === sessionId && s.project?.source_hash && !project?.source_hash) return s;
    if (project?.source_hash) return { project, sessionId, parked: false };
    if (s.sessionId === sessionId) return { project: null, sessionId, parked: s.parked };
    return s;
  }),
  park: () => set((s) => ({ project: null, parked: true, sessionId: s.sessionId })),
  clear: () => set({ project: null, sessionId: "", parked: false }),
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
