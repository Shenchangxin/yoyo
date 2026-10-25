import { create } from "zustand";

export type FilmWord = { id?: string; line_id?: string; text?: string; speaker?: string; start?: number; end?: number };
export type FilmNeed = { id?: string; key?: string; kind?: string; prompt?: string; status?: string; cost?: string; error?: string; asset_hash?: string };
export type FilmTask = { id?: string; need_id?: string; status?: string; error?: string };
export type FilmArtifact = { name?: string; hash?: string; media_type?: string };
export type FilmShot = { id?: string; visual?: string; start?: number; end?: number; beat?: boolean };
export type FilmProject = {
  id?: string;
  parent_id?: string;
  title?: string;
  structure?: string;
  note?: string;
  bed?: string;
  brief?: { host?: string; product?: string; language?: string; aspect?: string; hook?: string; price?: string; cta?: string };
  entries?: { id?: string; label?: string }[];
  reference?: { hash?: string; duration?: number };
  words?: FilmWord[];
  shots?: FilmShot[];
  stickers?: { id?: string; text?: string }[];
  needs?: FilmNeed[];
  tasks?: FilmTask[];
  artifacts?: FilmArtifact[];
};

type FilmDeskState = {
  project: FilmProject | null;
  setProject: (project: FilmProject | null) => void;
  clear: () => void;
};

export const useFilmDesk = create<FilmDeskState>((set) => ({
  project: null,
  setProject: (project) => set({ project: project?.id ? project : null }),
  clear: () => set({ project: null }),
}));
