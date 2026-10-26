/**
 * 拥有：模型目录的界面状态。先用快照，再用水合后的线上目录替换。
 * 不拥有：目录的形状和水合规则。
 */

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

/** useModelCatalog 模型目录。初始为快照。 */
export const useModelCatalog = create<ModelCatalogState>((set, get) => ({
  catalog: MODELS_DEV_SNAPSHOT,
  loading: false,
  error: "",
  /** 正在加载且不是强制刷新时跳过。成功则用水合结果替换目录。失败时留下错误文案，目录保持原样。 */
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

/** loadModelCatalog 触发目录加载。refresh 为真时即使正在加载也再请求一次。 */
export function loadModelCatalog(refresh = false): Promise<void> {
  return useModelCatalog.getState().load(refresh);
}
