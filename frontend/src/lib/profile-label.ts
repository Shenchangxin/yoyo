import type { Copy } from "./copy";
import { pick, str } from "./normalize";
import type { Profile } from "./protocol";

/**
 * 拥有：把配置档 id 和后端形状收成界面用的名称、提示和 Profile。
 * 不拥有：配置档的编辑与保存。
 */

const SEED_IDS = ["assistant", "researcher", "secretary"] as const;
type SeedId = (typeof SEED_IDS)[number];

/** 判断 id 是否为内置的助手、研究员或秘书。 */
function isSeed(id: string): id is SeedId {
  return (SEED_IDS as readonly string[]).includes(id);
}

/** profileLabel 内置档用文案表里的名称，其余用给定名称，再没有则用通用芯片文案。 */
export function profileLabel(id: string | undefined, name: string | undefined, copy: Copy): string {
  const key = (id || "assistant").trim() || "assistant";
  if (isSeed(key)) return copy.profile[key];
  const n = (name || "").trim();
  return n || copy.profile.chip;
}

/** profileHint 按内置档返回对应提示，其余用通用提示。 */
export function profileHint(id: string | undefined, copy: Copy): string {
  const key = (id || "assistant").trim() || "assistant";
  if (key === "assistant") return copy.profile.assistantHint;
  if (key === "researcher") return copy.profile.researcherHint;
  if (key === "secretary") return copy.profile.secretaryHint;
  return copy.profile.hint;
}

/** profileOf 从大小写两种字段名里读出 Profile。列表字段空则省略。 */
export function profileOf(v: any): Profile {
  return {
    id: str(pick(v, "id", "ID")),
    name: str(pick(v, "name", "Name")),
    instructions: str(pick(v, "instructions", "Instructions")) || undefined,
    allow_tools: asStringList(pick(v, "allow_tools", "AllowTools")),
    deny_tools: asStringList(pick(v, "deny_tools", "DenyTools")),
    space_ids: asStringList(pick(v, "space_ids", "SpaceIDs")),
    default_space: str(pick(v, "default_space", "DefaultSpace")) || undefined,
    research: !!pick(v, "research", "Research"),
    memory: !!pick(v, "memory", "Memory"),
    mcp_allow: asStringList(pick(v, "mcp_allow", "MCPAllow")),
    speech_connection: str(pick(v, "speech_connection", "SpeechConn")) || undefined,
  };
}

/** 把数组收成去空白后的字符串列表。空数组视为没有。 */
function asStringList(v: unknown): string[] | undefined {
  if (!Array.isArray(v)) return undefined;
  const out = v.map((x) => String(x || "").trim()).filter(Boolean);
  return out.length ? out : undefined;
}

/** resolveProfile 按 id 查找。找不到时用助手档，再没有则取列表第一项。 */
export function resolveProfile(list: Profile[], id?: string): Profile | undefined {
  const key = (id || "assistant").trim() || "assistant";
  return list.find((p) => p.id === key) || list.find((p) => p.id === "assistant") || list[0];
}
