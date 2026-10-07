import { providerPreset } from "./providers.ts";
import type { CatalogEntry, ModelDefinition, ModelsDevCatalog } from "./catalog-shape.ts";
import type { ChatProvider } from "./protocol.ts";

export type { CatalogEntry, ModelDefinition, ModelsDevCatalog } from "./catalog-shape.ts";
export { hydrateCatalog, overlayCatalog } from "./catalog-shape.ts";

/** Chat default when models.dev has no entry for the current model id. */
export const UNKNOWN_MODEL_WINDOW = 300_000;

export function resolveContextWindow(
  catalog: ModelsDevCatalog | null | undefined,
  presetId: string,
  modelId: string,
): number {
  const id = (modelId || "").trim();
  if (!id) return 0;
  const listed = lookupExactCatalogModel(catalog, (presetId || "").trim(), id);
  if (listed?.model.contextWindow) return listed.model.contextWindow;
  return inferCatalogWindow(catalog, id) || UNKNOWN_MODEL_WINDOW;
}

/** Fold dotted generations so AIPC-deepseek-v4.1-flash hits deepseek-v4-flash. */
function foldModelId(id: string): string {
  return id.toLowerCase().replace(/v(\d+)\.(\d+)/g, "v$1");
}

function inferCatalogWindow(catalog: ModelsDevCatalog | null | undefined, modelId: string): number | undefined {
  if (!catalog) return undefined;
  const needle = foldModelId(stripOpenrouter(modelId));
  let bestLen = 0;
  let best: number | undefined;
  for (const models of Object.values(catalog.providers)) {
    for (const [key, entry] of Object.entries(models)) {
      const k = foldModelId(key);
      const win = entry.model.contextWindow;
      if (!win || k.length <= bestLen) continue;
      if (needle === k || needle.includes(k)) {
        best = win;
        bestLen = k.length;
      }
    }
  }
  return best;
}

function lookupExactCatalogModel(
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
  return undefined;
}

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

/** Exact catalog membership — no fuzzy cross-provider matching. */
export function modelListedForProvider(
  catalog: ModelsDevCatalog | null | undefined,
  presetId: string,
  modelId: string,
): boolean {
  const id = (modelId || "").trim();
  if (!id || !catalog) return false;
  const mapped = presetId === "azure" ? "openai" : presetId;
  const models = catalog.providers[mapped];
  if (!models) return false;
  if (models[id]) return true;
  const undated = id.replace(/-\d{8}$/, "");
  return undated !== id && !!models[undated];
}

export function encodeChatModel(providerId: string, model: string): string {
  const id = (providerId || "").trim();
  const m = (model || "").trim();
  if (!id || !m) return m;
  return `${id}::${m}`;
}

export function decodeChatModel(value: string): { providerId: string; model: string } {
  const raw = (value || "").trim();
  const i = raw.indexOf("::");
  if (i <= 0) return { providerId: "", model: raw };
  return { providerId: raw.slice(0, i), model: raw.slice(i + 2) };
}

export type ChatModelGroup = {
  providerId: string;
  providerName: string;
  vendor: string;
  isDefault: boolean;
  models: string[];
};

export function composerModelGroups(
  providers: ChatProvider[] | undefined,
  currentModel?: string,
  currentProviderId?: string,
): ChatModelGroup[] {
  const now = (currentModel || "").trim();
  const pid = (currentProviderId || "").trim();
  return (providers || [])
    .filter((p) => p && p.id && p.active !== false)
    .map((p) => ({
      providerId: p.id,
      providerName: p.name || p.vendor || p.id,
      vendor: p.vendor || "custom",
      isDefault: !!p.isDefault,
      models: mergeModelIds(p.id === pid ? now : undefined, p.model, p.models),
    }))
    .filter((g) => g.models.length > 0);
}

export function resolveComposerModelValue(
  providers: ChatProvider[] | undefined,
  currentModel?: string,
  currentProviderId?: string,
): string {
  const model = (currentModel || "").trim();
  const pid = (currentProviderId || "").trim();
  if (pid && model) return encodeChatModel(pid, model);
  const groups = composerModelGroups(providers, model, pid);
  const owners = groups.filter((g) => g.models.includes(model));
  if (owners.length === 1) return encodeChatModel(owners[0].providerId, model);
  const def = owners.find((g) => g.isDefault) || groups.find((g) => g.isDefault) || groups[0];
  if (def && model) return encodeChatModel(def.providerId, model);
  if (def?.models[0]) return encodeChatModel(def.providerId, def.models[0]);
  return model;
}

/**
 * Models shown in the composer picker: the current session model plus the
 * user's configured list, filtered to the active provider. Never dumps the
 * full models.dev catalog.
 */
export function composerModelIds(
  catalog: ModelsDevCatalog | null | undefined,
  provider: string | undefined,
  current: string | undefined,
  configured?: string[],
): string[] {
  const preset = providerPreset(provider || "");
  const now = (current || "").trim();
  const seed = (configured || []).map((s) => s.trim()).filter(Boolean);
  const base = seed.length ? seed : (preset?.models || []);
  if (!provider || provider === "custom") return mergeModelIds(now, base);
  const owned = base.filter((id) => id === now || preset?.models.includes(id) || modelListedForProvider(catalog, provider, id));
  return mergeModelIds(now, owned.length ? owned : preset?.models);
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
