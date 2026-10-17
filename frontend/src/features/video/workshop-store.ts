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
  title: string;
  pane: string;
  gate: string;
  status: string;
  nextKey: string;
  setStudioId: (id: string) => void;
  setStudioMeta: (meta: Partial<Pick<StudioSelection, "title" | "pane" | "gate" | "status" | "nextKey">>) => void;
};

type GameFocus = "intent" | "build" | "play" | "check" | "";

type GameSelection = {
  gameId: string;
  title: string;
  focus: GameFocus;
  rev: number;
  setGameId: (id: string, title?: string) => void;
  setGameFocus: (focus: GameFocus) => void;
  touch: (id: string) => void;
};

export const useGameSelection = create<GameSelection>((set) => ({
  gameId: read("yoyo-game-id"),
  title: "",
  focus: "",
  rev: 0,
  setGameId: (gameId, title) => {
    persist("yoyo-game-id", gameId);
    set((s) => ({
      gameId,
      title: gameId ? (title !== undefined ? title : (gameId === s.gameId ? s.title : "")) : "",
      focus: gameId && gameId === s.gameId ? s.focus : "",
    }));
  },
  setGameFocus: (focus) => set({ focus }),
  touch: (id) => set((s) => (id && id === s.gameId ? { rev: s.rev + 1 } : s)),
}));

export const useStudioSelection = create<StudioSelection>((set) => ({
  studioId: read("yoyo-studio-id"),
  title: "",
  pane: "radar",
  gate: "",
  status: "",
  nextKey: "",
  setStudioId: (studioId) => {
    persist("yoyo-studio-id", studioId);
    set((s) => (
      studioId
        ? { studioId, title: studioId === s.studioId ? s.title : "", gate: studioId === s.studioId ? s.gate : "", status: studioId === s.studioId ? s.status : "", nextKey: studioId === s.studioId ? s.nextKey : "" }
        : { studioId: "", title: "", pane: "radar", gate: "", status: "", nextKey: "" }
    ));
  },
  setStudioMeta: (meta) => set(meta),
}));
