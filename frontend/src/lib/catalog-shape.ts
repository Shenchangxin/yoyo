/**
 * 拥有：models.dev 目录的类型，以及把线上目录叠到快照上并水合成同一形状。
 * 不拥有：目录的加载和界面状态。
 */

import { asArray, bool, num, pick, str } from "./normalize.ts";

/** ModelDefinition 一条模型。reasoningLevels 是推理档。input 只收 text 和 image。contextWindow 与 maxTokens 来自目录。cost 含 input、output、cacheRead、cacheWrite。 */
export type ModelDefinition = {
  id: string;
  name?: string;
  reasoning?: boolean;
  reasoningLevels?: string[];
  input?: Array<"text" | "image">;
  contextWindow?: number;
  maxTokens?: number;
  cost?: { input: number; output: number; cacheRead: number; cacheWrite: number };
};

/** CatalogEntry 目录里的一条：模型，以及可选的家族和发布日期。 */
export type CatalogEntry = {
  model: ModelDefinition;
  family?: string;
  releaseDate?: string;
};

/** ModelsDevCatalog 一份目录。providers 先按供应商标识，再按模型 id。stale 表示这份目录已过期。 */
export type ModelsDevCatalog = {
  version: number;
  fetchedAt: string;
  source?: string;
  stale?: boolean;
  providers: Record<string, Record<string, CatalogEntry>>;
};

/** overlayCatalog 用线上目录盖住基底。线上没有供应商时原样返回基底。同一供应商整表替换，不是按模型合并。version、fetchedAt、source 在线上为空时留基底的。 */
export function overlayCatalog(
  base: ModelsDevCatalog | null | undefined,
  live: ModelsDevCatalog | null | undefined,
): ModelsDevCatalog {
  const a = base && base.providers ? base : { version: 3, fetchedAt: "", providers: {} as ModelsDevCatalog["providers"] };
  if (!live || !live.providers || !Object.keys(live.providers).length) return a;
  return {
    version: live.version || a.version,
    fetchedAt: live.fetchedAt || a.fetchedAt,
    source: live.source || a.source,
    stale: live.stale,
    providers: { ...a.providers, ...live.providers },
  };
}

/** hydrateCatalog 从松散的 Wails 或 JSON 形状收成目录，再叠到 fallback 上。空 id 的模型丢掉。没有成本对象则不填 cost。version 缺省为 3。 */
export function hydrateCatalog(raw: unknown, fallback?: ModelsDevCatalog | null): ModelsDevCatalog {
  const v = raw as any;
  const providersIn = pick(v, "providers", "Providers") || {};
  const providers: ModelsDevCatalog["providers"] = {};
  if (providersIn && typeof providersIn === "object") {
    for (const [pid, models] of Object.entries(providersIn as Record<string, unknown>)) {
      if (!models || typeof models !== "object") continue;
      const out: Record<string, CatalogEntry> = {};
      for (const [mid, entry] of Object.entries(models as Record<string, unknown>)) {
        const rec = entry as any;
        const model = pick(rec, "model", "Model") || {};
        const id = str(pick(model, "id", "ID"), mid).trim();
        if (!id) continue;
        const costRaw = pick(model, "cost", "Cost");
        const input = asArray(pick(model, "input", "Input"))
          .map((x) => String(x))
          .filter((x): x is "text" | "image" => x === "text" || x === "image");
        const levels = asArray(pick(model, "reasoningLevels", "ReasoningLevels")).map((x) => String(x)).filter(Boolean);
        out[id] = {
          model: {
            id,
            name: str(pick(model, "name", "Name")) || undefined,
            reasoning: bool(pick(model, "reasoning", "Reasoning")),
            reasoningLevels: levels.length ? levels : undefined,
            input: input.length ? input : undefined,
            contextWindow: num(pick(model, "contextWindow", "ContextWindow")) || undefined,
            maxTokens: num(pick(model, "maxTokens", "MaxTokens")) || undefined,
            cost: costRaw
              ? {
                  input: num(pick(costRaw, "input", "Input")),
                  output: num(pick(costRaw, "output", "Output")),
                  cacheRead: num(pick(costRaw, "cacheRead", "CacheRead", "cache_read")),
                  cacheWrite: num(pick(costRaw, "cacheWrite", "CacheWrite", "cache_write")),
                }
              : undefined,
          },
          family: str(pick(rec, "family", "Family")) || undefined,
          releaseDate: str(pick(rec, "releaseDate", "ReleaseDate", "release_date")) || undefined,
        };
      }
      if (Object.keys(out).length) providers[pid] = out;
    }
  }
  const live: ModelsDevCatalog = {
    version: num(pick(v, "version", "Version"), 3) || 3,
    fetchedAt: str(pick(v, "fetchedAt", "FetchedAt", "fetched_at")),
    source: str(pick(v, "source", "Source")) || undefined,
    stale: bool(pick(v, "stale", "Stale")) || undefined,
    providers,
  };
  return overlayCatalog(fallback, live);
}
