/**
 * 拥有：从目录里解析上下文窗口、列出供应商模型，以及作曲栏里的模型分组。
 * 不拥有：目录的水合。那是 catalog-shape 的事。
 */

import { providerPreset } from "./providers.ts";
import type { CatalogEntry, ModelDefinition, ModelsDevCatalog } from "./catalog-shape.ts";
import type { ChatProvider } from "./protocol.ts";

/** CatalogEntry、ModelDefinition、ModelsDevCatalog 从 catalog-shape 转出。 */
export type { CatalogEntry, ModelDefinition, ModelsDevCatalog } from "./catalog-shape.ts";
/** hydrateCatalog、overlayCatalog 从 catalog-shape 转出。 */
export { hydrateCatalog, overlayCatalog } from "./catalog-shape.ts";

/** UNKNOWN_MODEL_WINDOW 当前模型 id 在 models.dev 里没有条目时，聊天用的默认窗口。 */
export const UNKNOWN_MODEL_WINDOW = 300_000;

/** resolveContextWindow 空 id 返回 0。先精确查找，没有窗口再按折叠后的 id 推断，仍没有则用 UNKNOWN_MODEL_WINDOW。 */
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

/** 把 v 加数字再加点和数字收成 v 加前一个数字，使 AIPC-deepseek-v4.1-flash 对上 deepseek-v4-flash。比较前先小写。 */
function foldModelId(id: string): string {
  return id.toLowerCase().replace(/v(\d+)\.(\d+)/g, "v$1");
}

/** 在全部供应商里找最长的、带 contextWindow 的键。折叠后的键等于去掉最后一个斜杠左边之后的 id，或被该 id 包含。 */
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

/** 按供应商精确查找。azure 映射到 openai，openrouter 按模型前缀映射。再试最后一个斜杠之后的 id，以及去掉结尾八位日期。不做前缀模糊匹配。 */
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

/** lookupCatalogModel 先按 lookupExactCatalogModel 的规则找。没有则取最长的键：查询以前缀开头，或键以查询开头。 */
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

/** 去掉最后一个斜杠及其左边。没有斜杠则原样返回。 */
function stripOpenrouter(id: string): string {
  const i = id.lastIndexOf("/");
  return i >= 0 ? id.slice(i + 1) : id;
}

/** 按 openrouter 的供应前缀映射到目录里的供应商标识。对不上则为 openai。 */
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

/** modelsForProvider azure 映射到 openai。发布日期早于现在减去 365 天的丢掉。每个家族只留发布日期最新的一条。按名称或 id 排序。 */
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

/** modelListedForProvider 精确的目录成员，不做跨供应商的模糊匹配。azure 映射到 openai。去掉结尾八位日期后命中也算。 */
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

/** encodeChatModel 收成供应商标识、两个冒号、模型。任一侧空白则只返回模型。 */
export function encodeChatModel(providerId: string, model: string): string {
  const id = (providerId || "").trim();
  const m = (model || "").trim();
  if (!id || !m) return m;
  return `${id}::${m}`;
}

/** decodeChatModel 在第一个 `::` 处切开。没有 `::`，或它在开头时，供应商为空，模型为整段。 */
export function decodeChatModel(value: string): { providerId: string; model: string } {
  const raw = (value || "").trim();
  const i = raw.indexOf("::");
  if (i <= 0) return { providerId: "", model: raw };
  return { providerId: raw.slice(0, i), model: raw.slice(i + 2) };
}

/** ChatModelGroup 作曲栏里一个供应商及其模型 id。isDefault 来自连接上的默认标记。 */
export type ChatModelGroup = {
  providerId: string;
  providerName: string;
  vendor: string;
  isDefault: boolean;
  models: string[];
};

/** composerModelGroups 只留有 id 且 active 不为假的连接。当前供应商的组把当前模型并进列表。没有模型的组丢掉。 */
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

/** resolveComposerModelValue 供应商和模型都有则直接编码。否则恰好一个组拥有该模型时用它。再否则用默认组，或第一组。 */
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

/** composerModelIds 作曲栏显示当前会话模型和用户配置的列表，并滤到当前供应商。不倒出整份 models.dev 目录。没有供应商或为 custom 时，合并当前模型与配置列表，配置为空则用预设。有供应商时只留当前模型、预设里有的，或该供应商目录里有的；一个都不剩则退回预设。 */
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

/** mergeModelIds 按出现顺序合并，去掉空白和重复。 */
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

/** formatTokens 未给出或为负是破折号。0 显示 0。不少于 1000000 用 M，能整除则不带小数，否则一位。不少于 1000 用四舍五入的 k。 */
export function formatTokens(n?: number): string {
  if (n === undefined || n === null || n < 0) return "—";
  if (n === 0) return "0";
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n % 1_000_000 === 0 ? 0 : 1)}M`;
  if (n >= 1000) return `${Math.round(n / 1000)}k`;
  return String(n);
}

/** formatUsd 未给出是破折号。小于 0.01 保留三位小数，小于 1 保留两位，不小于 10 则不带小数，其余一位。 */
export function formatUsd(n?: number): string {
  if (n === undefined || n === null) return "—";
  if (n < 0.01) return `$${n.toFixed(3)}`;
  if (n < 1) return `$${n.toFixed(2)}`;
  return `$${n.toFixed(n >= 10 ? 0 : 1)}`;
}
