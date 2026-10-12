import { create } from "zustand";

function read(key: string): string {
  try {
    return localStorage.getItem(key) || "";
  } catch {
    return "";
  }
}

function persist(key: string, v: string) {
  try {
    localStorage.setItem(key, v);
  } catch {
    /* ignore */
  }
}

type DramaSelection = {
  dramaId: string;
  episodeId: string;
  dramaStyle: string;
  setDramaId: (id: string) => void;
  setEpisodeId: (id: string) => void;
  setDramaStyle: (style: string) => void;
};

export const useDramaSelection = create<DramaSelection>((set) => ({
  dramaId: read("yoyo-drama-id"),
  episodeId: read("yoyo-episode-id"),
  dramaStyle: read("yoyo-drama-style") || "3d",
  setDramaId: (dramaId) => {
    persist("yoyo-drama-id", dramaId);
    set({ dramaId });
  },
  setEpisodeId: (episodeId) => {
    persist("yoyo-episode-id", episodeId);
    set({ episodeId });
  },
  setDramaStyle: (dramaStyle) => {
    persist("yoyo-drama-style", dramaStyle || "3d");
    set({ dramaStyle: dramaStyle || "3d" });
  },
}));

export function defaultDramaStyle() {
  return useDramaSelection.getState().dramaStyle || "3d";
}

type StudioSelection = {
  studioId: string;
  setStudioId: (id: string) => void;
};

export const useStudioSelection = create<StudioSelection>((set) => ({
  studioId: read("yoyo-studio-id"),
  setStudioId: (studioId) => {
    persist("yoyo-studio-id", studioId);
    set({ studioId });
  },
}));
