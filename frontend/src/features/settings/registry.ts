import type { SettingsTab } from "../../lib/protocol";

type SectionTitleKey = keyof import("../../lib/copy").Copy["settings"]["sections"];
type GroupKey = keyof import("../../lib/copy").Copy["settings"]["groups"];

export type SettingsTabDef = {
  key: SettingsTab;
  group: GroupKey;
  icon: string;
};

export type SettingsSectionDef = {
  id: string;
  tab: SettingsTab;
  titleKey: SectionTitleKey;
  /** Unlocalised terms so search still finds a row when the UI is in Chinese. */
  keywords?: string;
};

export const SETTINGS_GROUPS: readonly GroupKey[] = ["app", "agent", "system"];

export const SETTINGS_TABS: readonly SettingsTabDef[] = [
  { key: "general", group: "app", icon: "sliders" },
  { key: "appearance", group: "app", icon: "palette" },
  { key: "idol", group: "app", icon: "heart" },
  { key: "provider", group: "agent", icon: "cpu" },
  { key: "generation", group: "agent", icon: "film" },
  { key: "policy", group: "agent", icon: "shield" },
  { key: "extensions", group: "agent", icon: "blocks" },
  { key: "personal", group: "agent", icon: "brain" },
  { key: "advanced", group: "system", icon: "wrench" },
];

export const SETTINGS_SECTIONS: readonly SettingsSectionDef[] = [
  { id: "general-workspace", tab: "general", titleKey: "generalWorkspace", keywords: "workspace folder directory" },
  { id: "general-notifications", tab: "general", titleKey: "generalNotifications", keywords: "notifications toast" },
  { id: "general-desktop", tab: "general", titleKey: "generalDesktop", keywords: "tray login always on top" },
  { id: "shortcuts-map", tab: "general", titleKey: "shortcutsMap", keywords: "keymap hotkey binding shortcuts" },
  { id: "appearance-theme", tab: "appearance", titleKey: "appearanceTheme", keywords: "theme dark light color mode" },
  { id: "appearance-palette", tab: "appearance", titleKey: "appearancePalette", keywords: "palette paper mist ink dim slate neutral stone charcoal blue" },
  { id: "appearance-skins", tab: "appearance", titleKey: "appearanceSkins", keywords: "wallpaper background picture image veil" },
  { id: "appearance-display", tab: "appearance", titleKey: "appearanceDisplay", keywords: "language locale scale zoom thinking reasoning work details compact verbose collapse laboratory lab harness sealed bon hash" },
  { id: "idol-services", tab: "idol", titleKey: "idolServices", keywords: "idol llm asr tts image voice chat speech 偶像 语音 模型 音色 对话" },
  { id: "idol-you", tab: "idol", titleKey: "idolYou", keywords: "address birthday timezone name 称呼 生日" },
  { id: "idol-work", tab: "idol", titleKey: "idolWork", keywords: "agent profile tools work 动手 工作" },
  { id: "provider-account", tab: "provider", titleKey: "providerAccount", keywords: "api key base url vault" },
  { id: "provider-models", tab: "provider", titleKey: "providerModels", keywords: "model catalog context window" },
  { id: "generation-studio", tab: "generation", titleKey: "generationStudio", keywords: "workshop language ffmpeg drama studio shorts 短片 remotion aspect publish 画幅 平台" },
  { id: "generation-canvas", tab: "generation", titleKey: "generationCanvas", keywords: "canvas protocol plugin channel kling veo seedream" },
  { id: "generation-image", tab: "generation", titleKey: "generationImage", keywords: "image still seedream gemini openai gpt-image qwen jimeng kolors cogview hunyuan 即梦 万相 可图" },
  { id: "generation-video", tab: "generation", titleKey: "generationVideo", keywords: "video clip seedance minimax wan hailuo aliyun jimeng vidu hunyuan cogvideo kling 即梦 可灵 混元" },
  { id: "generation-speech", tab: "generation", titleKey: "generationSpeech", keywords: "tts speech voice openai minimax volcengine elevenlabs azure cosyvoice spark 讯飞 腾讯" },
  { id: "generation-music", tab: "generation", titleKey: "generationMusic", keywords: "music bgm soundtrack minimax mureka elevenlabs stability lyria suno 配乐" },
  { id: "generation-stock", tab: "generation", titleKey: "generationStock", keywords: "stock pexels pixabay unsplash giphy footage 库存 素材" },
  { id: "generation-asr", tab: "generation", titleKey: "generationAsr", keywords: "asr transcribe whisper stt groq deepgram assemblyai paraformer spark 转写 讯飞 腾讯" },
  { id: "generation-storage", tab: "generation", titleKey: "generationStorage", keywords: "s3 oss cas bucket object storage 阿里云 腾讯云 cos" },
  { id: "generation-search", tab: "generation", titleKey: "generationSearch", keywords: "search web duckduckgo bocha metaso jina exa 秘塔 搜索" },
  { id: "generation-workflow", tab: "generation", titleKey: "generationWorkflow", keywords: "runninghub workflow comfy" },
  { id: "policy-gate", tab: "policy", titleKey: "policyGate", keywords: "approval gate shell auto allow" },
  { id: "policy-budget", tab: "policy", titleKey: "policyBudget", keywords: "budget usd cost mtok" },
  { id: "policy-shell", tab: "policy", titleKey: "policyShell", keywords: "pwsh powershell 7 runtime sidecar modelscope agent shell" },
  { id: "policy-isolation", tab: "policy", titleKey: "policyIsolation", keywords: "isolation sandbox vault signature" },
  { id: "extensions-mcp", tab: "extensions", titleKey: "extensionsMcp", keywords: "mcp server stdio http json" },
  { id: "extensions-fibers", tab: "extensions", titleKey: "extensionsFibers", keywords: "plugin fiber wasm" },
  { id: "extensions-connectors", tab: "extensions", titleKey: "extensionsConnectors", keywords: "connector oauth gmail feishu" },
  { id: "personal-memory", tab: "personal", titleKey: "personalMemory", keywords: "memory profile staging" },
  { id: "personal-profiles", tab: "personal", titleKey: "personalProfiles", keywords: "profile identity role tools mcp" },
  { id: "personal-jobs", tab: "personal", titleKey: "personalJobs", keywords: "automation schedule cron job" },
  { id: "personal-work", tab: "personal", titleKey: "personalWork", keywords: "task document proposal lease" },
  { id: "personal-ideas", tab: "personal", titleKey: "personalIdeas", keywords: "ideas mail form meet goal" },
  { id: "personal-goals", tab: "personal", titleKey: "personalGoals", keywords: "goal milestone plan" },
  { id: "personal-watch", tab: "personal", titleKey: "personalWatch", keywords: "monitor watch page diff price" },
  { id: "personal-proposals", tab: "personal", titleKey: "personalProposals", keywords: "proposal send approval hash" },
  { id: "advanced-updates", tab: "advanced", titleKey: "advancedUpdates", keywords: "update version release ota modelscope" },
  { id: "advanced-logs", tab: "advanced", titleKey: "advancedLogs", keywords: "logs diagnostics debug support" },
  { id: "advanced-journal", tab: "advanced", titleKey: "advancedJournal", keywords: "journal audit chain" },
  { id: "advanced-doctor", tab: "advanced", titleKey: "advancedDoctor", keywords: "doctor diagnostics health" },
  { id: "advanced-handheld", tab: "advanced", titleKey: "advancedHandheld", keywords: "phone handheld qr pair remote mobile android ios harmony 手机 配对 扫码" },
];

/** Tabs that existed before the current grouping. Deep links and menus still use them. */
const LEGACY_TABS: Readonly<Record<string, { tab: SettingsTab; section: string }>> = {
  shortcuts: { tab: "general", section: "shortcuts-map" },
  isolation: { tab: "policy", section: "policy-isolation" },
  connectors: { tab: "extensions", section: "extensions-connectors" },
  memory: { tab: "personal", section: "personal-memory" },
  automations: { tab: "personal", section: "personal-jobs" },
  mcp: { tab: "extensions", section: "extensions-mcp" },
  plugins: { tab: "extensions", section: "extensions-fibers" },
  updates: { tab: "advanced", section: "advanced-updates" },
  logs: { tab: "advanced", section: "advanced-logs" },
  danger: { tab: "advanced", section: "advanced-updates" },
};

const LEGACY_SECTIONS: Readonly<Record<string, string>> = {
  "general-basics": "general-workspace",
  "general-app": "general-notifications",
  "appearance-mode": "appearance-theme",
  "appearance-language": "appearance-display",
  "isolation-status": "policy-isolation",
  "connectors-catalog": "extensions-connectors",
  "memory-items": "personal-memory",
  "automations-jobs": "personal-jobs",
  "mcp-servers": "extensions-mcp",
  "mcp-json": "extensions-mcp",
  "plugins-fibers": "extensions-fibers",
  "updates-channel": "advanced-updates",
  "logs-journal": "advanced-journal",
  "logs-doctor": "advanced-doctor",
  "danger-zone": "advanced-updates",
  "advanced-danger": "advanced-updates",
  "provider-video": "generation-image",
  "generation-styles": "generation-studio",
};

export function isSettingsTab(v: string): v is SettingsTab {
  return SETTINGS_TABS.some((t) => t.key === v);
}

/** Resolves any tab/section pair — current or legacy — onto the current layout. */
export function resolveSettingsTarget(tab?: string, section?: string): { tab: SettingsTab; section: string } {
  const mappedSection = section ? LEGACY_SECTIONS[section] || section : "";
  const owner = mappedSection ? SETTINGS_SECTIONS.find((s) => s.id === mappedSection) : undefined;
  if (owner) return { tab: owner.tab, section: owner.id };
  if (tab && isSettingsTab(tab)) return { tab, section: "" };
  const legacy = tab ? LEGACY_TABS[tab] : undefined;
  if (legacy) return { tab: legacy.tab, section: legacy.section };
  return { tab: "general", section: "" };
}

export function tabsInGroup(group: GroupKey) {
  return SETTINGS_TABS.filter((t) => t.group === group);
}
