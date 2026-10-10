import type { Copy } from "./copy";
import { pick, str } from "./normalize";
import type { Profile } from "./protocol";

const SEED_IDS = ["assistant", "researcher", "secretary"] as const;
type SeedId = (typeof SEED_IDS)[number];

function isSeed(id: string): id is SeedId {
  return (SEED_IDS as readonly string[]).includes(id);
}

export function profileLabel(id: string | undefined, name: string | undefined, copy: Copy): string {
  const key = (id || "assistant").trim() || "assistant";
  if (isSeed(key)) return copy.profile[key];
  const n = (name || "").trim();
  return n || copy.profile.chip;
}

export function profileHint(id: string | undefined, copy: Copy): string {
  const key = (id || "assistant").trim() || "assistant";
  if (key === "assistant") return copy.profile.assistantHint;
  if (key === "researcher") return copy.profile.researcherHint;
  if (key === "secretary") return copy.profile.secretaryHint;
  return copy.profile.hint;
}

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

function asStringList(v: unknown): string[] | undefined {
  if (!Array.isArray(v)) return undefined;
  const out = v.map((x) => String(x || "").trim()).filter(Boolean);
  return out.length ? out : undefined;
}

export function resolveProfile(list: Profile[], id?: string): Profile | undefined {
  const key = (id || "assistant").trim() || "assistant";
  return list.find((p) => p.id === key) || list.find((p) => p.id === "assistant") || list[0];
}
