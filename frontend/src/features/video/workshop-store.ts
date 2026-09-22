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
  setDramaId: (id: string) => void;
  setEpisodeId: (id: string) => void;
};

export const useDramaSelection = create<DramaSelection>((set) => ({
  dramaId: read("yoyo-drama-id"),
  episodeId: read("yoyo-episode-id"),
  setDramaId: (dramaId) => {
    persist("yoyo-drama-id", dramaId);
    set({ dramaId });
  },
  setEpisodeId: (episodeId) => {
    persist("yoyo-episode-id", episodeId);
    set({ episodeId });
  },
}));
