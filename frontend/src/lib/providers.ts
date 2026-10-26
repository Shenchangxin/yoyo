/**
 * 拥有：OpenAI 兼容聊天供应商的预设地址和默认模型，与 Open-Vetta 的目录对齐。
 * 不拥有：密钥、连接探测，以及当前会话实际选用的供应商。
 */

/** ProviderPreset 一条预设。hint 只在需要网关或自行填写端点时出现。 */
export type ProviderPreset = {
  id: string;
  label: string;
  baseUrl: string;
  model: string;
  models: string[];
  hint?: string;
};

/** PROVIDER_PRESETS 可一键套用的供应商预设。custom 没有基址，套用时要跳过。 */
export const PROVIDER_PRESETS: ProviderPreset[] = [
  {
    id: "openai",
    label: "OpenAI",
    baseUrl: "https://api.openai.com/v1",
    model: "gpt-4.1-mini",
    models: ["gpt-4.1", "gpt-4.1-mini", "o4-mini"],
  },
  {
    id: "claude",
    label: "Claude",
    baseUrl: "https://api.anthropic.com/v1",
    model: "claude-sonnet-4-5",
    models: ["claude-sonnet-4-5", "claude-opus-4-1"],
    hint: "Needs an OpenAI-compatible gateway, or use OpenRouter.",
  },
  {
    id: "gemini",
    label: "Gemini",
    baseUrl: "https://generativelanguage.googleapis.com/v1beta/openai",
    model: "gemini-2.5-flash",
    models: ["gemini-2.5-pro", "gemini-2.5-flash"],
  },
  {
    id: "deepseek",
    label: "DeepSeek",
    baseUrl: "https://api.deepseek.com",
    model: "deepseek-chat",
    models: ["deepseek-chat", "deepseek-reasoner"],
  },
  {
    id: "kimi",
    label: "Kimi",
    baseUrl: "https://api.moonshot.ai/v1",
    model: "kimi-k2-0905",
    models: ["kimi-k2-0905", "moonshot-v1-auto"],
  },
  {
    id: "qwen",
    label: "Qwen",
    baseUrl: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
    model: "qwen-plus",
    models: ["qwen-plus", "qwen-max", "qwq-plus"],
  },
  {
    id: "zai",
    label: "Z.ai (GLM)",
    baseUrl: "https://api.z.ai/api/paas/v4",
    model: "glm-4.5",
    models: ["glm-4.5", "glm-4.5-air"],
  },
  {
    id: "grok",
    label: "Grok",
    baseUrl: "https://api.x.ai/v1",
    model: "grok-4",
    models: ["grok-4", "grok-3"],
  },
  {
    id: "groq",
    label: "Groq",
    baseUrl: "https://api.groq.com/openai/v1",
    model: "llama-3.3-70b-versatile",
    models: ["llama-3.3-70b-versatile", "openai/gpt-oss-120b"],
  },
  {
    id: "openrouter",
    label: "OpenRouter",
    baseUrl: "https://openrouter.ai/api/v1",
    model: "anthropic/claude-sonnet-4.5",
    models: ["anthropic/claude-sonnet-4.5", "openai/gpt-4.1", "google/gemini-2.5-flash"],
  },
  {
    id: "azure",
    label: "Azure OpenAI",
    baseUrl: "https://YOUR_RESOURCE.openai.azure.com/openai/v1",
    model: "gpt-4.1-mini",
    models: ["gpt-4.1", "gpt-4.1-mini"],
    hint: "Replace YOUR_RESOURCE with the Azure resource name.",
  },
  {
    id: "custom",
    label: "Custom",
    baseUrl: "",
    model: "",
    models: [],
    hint: "Any OpenAI-compatible /v1/chat/completions endpoint.",
  },
];

/** providerPreset 按 id 查找预设。 */
export function providerPreset(id: string): ProviderPreset | undefined {
  return PROVIDER_PRESETS.find((p) => p.id === id);
}

/** applyProviderPreset 取出基址、默认模型和模型列表。custom 与未知 id 没有可套用的地址，返回空。 */
export function applyProviderPreset(id: string): Pick<ProviderPreset, "baseUrl" | "model" | "models"> | null {
  const p = providerPreset(id);
  if (!p || p.id === "custom") return null;
  return { baseUrl: p.baseUrl, model: p.model, models: p.models };
}
