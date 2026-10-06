import { asArray, bool, num, pick, str } from "./normalize.ts";

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

export type CatalogEntry = {
  model: ModelDefinition;
  family?: string;
  releaseDate?: string;
};

export type ModelsDevCatalog = {
  version: number;
  fetchedAt: string;
  source?: string;
  stale?: boolean;
  providers: Record<string, Record<string, CatalogEntry>>;
};

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
