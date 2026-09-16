/** models.dev-shaped catalog used to fill context windows, pricing, and capabilities. */

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
  providers: Record<string, Record<string, CatalogEntry>>;
};

export function lookupCatalogModel(
  catalog: ModelsDevCatalog | null | undefined,
  presetId: string,
  modelId: string,
): CatalogEntry | undefined {
  const id = (modelId || "").trim();
  if (!catalog || !id) return undefined;
  const mapped = presetId === "azure" ? "openai" : presetId === "openrouter" ? openrouterPreset(id) : presetId;
  const models = catalog.providers[mapped];
  if (!models) return undefined;
  const exact = models[id] || models[stripOpenrouter(id)];
  if (exact) return exact;
  const undated = id.replace(/-\d{8}$/, "");
  if (undated !== id && models[undated]) return models[undated];
  let best: CatalogEntry | undefined;
  let bestLength = 0;
  const needle = stripOpenrouter(undated);
  for (const [key, entry] of Object.entries(models)) {
    if (key.length > bestLength && (needle.startsWith(key) || key.startsWith(needle))) {
      best = entry;
      bestLength = key.length;
    }
  }
  return best;
}

function stripOpenrouter(id: string): string {
  const i = id.lastIndexOf("/");
  return i >= 0 ? id.slice(i + 1) : id;
}

function openrouterPreset(id: string): string {
  if (id.startsWith("anthropic/")) return "claude";
  if (id.startsWith("openai/") || id.startsWith("openai.")) return "openai";
  if (id.startsWith("google/")) return "gemini";
  if (id.startsWith("deepseek/")) return "deepseek";
  if (id.startsWith("x-ai/") || id.startsWith("xai/")) return "grok";
  if (id.startsWith("qwen/") || id.startsWith("alibaba/")) return "qwen";
  if (id.startsWith("moonshotai/") || id.startsWith("kimi/")) return "kimi";
  if (id.startsWith("z-ai/") || id.startsWith("zai/")) return "zai";
  return "openai";
}

export function modelsForProvider(catalog: ModelsDevCatalog | null | undefined, presetId: string): ModelDefinition[] {
  const mapped = presetId === "azure" ? "openai" : presetId;
  const entries = catalog?.providers[mapped];
  if (!entries) return [];
  const cutoff = new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
  const newest = new Map<string, CatalogEntry>();
  for (const entry of Object.values(entries)) {
    if (entry.releaseDate && entry.releaseDate < cutoff) continue;
    const family = entry.family || entry.model.id;
    const cur = newest.get(family);
    if (!cur || (entry.releaseDate || "") > (cur.releaseDate || "")) newest.set(family, entry);
  }
  return [...newest.values()].map((e) => e.model).sort((a, b) => (a.name || a.id).localeCompare(b.name || b.id));
}

export function mergeModelIds(...lists: Array<string | string[] | undefined | null>): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const list of lists) {
    const items = Array.isArray(list) ? list : list ? [list] : [];
    for (const raw of items) {
      const id = (raw || "").trim();
      if (!id || seen.has(id)) continue;
      seen.add(id);
      out.push(id);
    }
  }
  return out;
}

export function formatTokens(n?: number): string {
  if (n === undefined || n === null || n < 0) return "—";
  if (n === 0) return "0";
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n % 1_000_000 === 0 ? 0 : 1)}M`;
  if (n >= 1000) return `${Math.round(n / 1000)}k`;
  return String(n);
}

export function formatUsd(n?: number): string {
  if (n === undefined || n === null) return "—";
  if (n < 0.01) return `$${n.toFixed(3)}`;
  if (n < 1) return `$${n.toFixed(2)}`;
  return `$${n.toFixed(n >= 10 ? 0 : 1)}`;
}
