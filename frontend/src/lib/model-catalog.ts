import { create } from "zustand";
import { modelsCatalog } from "./client";
import { hydrateCatalog, type ModelsDevCatalog } from "./catalog-shape";
import { MODELS_DEV_SNAPSHOT } from "./models-dev.snapshot";

type ModelCatalogState = {
  catalog: ModelsDevCatalog;
  loading: boolean;
  error: string;
  load: (refresh?: boolean) => Promise<void>;
};

export const useModelCatalog = create<ModelCatalogState>((set, get) => ({
  catalog: MODELS_DEV_SNAPSHOT,
  loading: false,
  error: "",
  load: async (refresh = false) => {
    if (get().loading && !refresh) return;
    set({ loading: true, error: "" });
    try {
      const raw = await modelsCatalog(refresh);
      const next = hydrateCatalog(raw, MODELS_DEV_SNAPSHOT);
      set({ catalog: next, loading: false, error: "" });
    } catch (e: any) {
      set({ loading: false, error: String(e?.message || e || "catalog") });
    }
  },
}));

export function loadModelCatalog(refresh = false): Promise<void> {
  return useModelCatalog.getState().load(refresh);
}
