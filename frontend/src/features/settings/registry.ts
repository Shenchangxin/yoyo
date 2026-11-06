/**
 * 拥有：设置页的分组、页签和区块目录，以及旧链接落到现行布局的规则。
 * 不拥有：表单。只决定导航，不调用 api。
 */
import type { SettingsTab } from "../../lib/protocol";

type SectionTitleKey = keyof import("../../lib/copy").Copy["settings"]["sections"];
type GroupKey = keyof import("../../lib/copy").Copy["settings"]["groups"];

/** SettingsTabDef 是一个页签的键、分组和图标。hidden 的页签不进侧栏，只从它自己的模块打开。 */
export type SettingsTabDef = {
  key: SettingsTab;
  group: GroupKey;
  icon: string;
  hidden?: boolean;
};

/** SettingsSectionDef 是一个区块的编号、所属页签和标题。 */
export type SettingsSectionDef = {
  id: string;
  tab: SettingsTab;
  titleKey: SectionTitleKey;
  /** 未本地化的检索词，界面是中文时仍能搜到这一行。 */
  keywords?: string;
};

/** SETTINGS_GROUPS 是设置分组的顺序。 */
export const SETTINGS_GROUPS: readonly GroupKey[] = ["app", "agent", "link", "system"];

/** SETTINGS_TABS 是现行页签：八页进侧栏，偶像页从偶像模块打开。 */
export const SETTINGS_TABS: readonly SettingsTabDef[] = [
  { key: "general", group: "app", icon: "sliders" },
  { key: "appearance", group: "app", icon: "palette" },
  { key: "proactive", group: "app", icon: "bell" },
  { key: "provider", group: "agent", icon: "cpu" },
  { key: "generation", group: "agent", icon: "film" },
  { key: "policy", group: "agent", icon: "shield" },
  { key: "connectors", group: "link", icon: "link" },
  { key: "advanced", group: "system", icon: "wrench" },
  { key: "idol", group: "app", icon: "heart", hidden: true },
];

/** SETTINGS_SECTIONS 是现行区块，附带检索词。生成页的服务各自成块，页内用类别筛选。 */
export const SETTINGS_SECTIONS: readonly SettingsSectionDef[] = [
  { id: "general-workspace", tab: "general", titleKey: "generalWorkspace", keywords: "workspace folder directory" },
  { id: "general-desktop", tab: "general", titleKey: "generalDesktop", keywords: "tray login always on top window startup companion 托盘 开机 置顶" },
  { id: "general-language", tab: "general", titleKey: "generalLanguage", keywords: "language locale english chinese 语言 中文 英文" },
  { id: "shortcuts-map", tab: "general", titleKey: "shortcutsMap", keywords: "keymap hotkey binding shortcuts" },
  { id: "appearance-theme", tab: "appearance", titleKey: "appearanceTheme", keywords: "theme dark light color mode palette paper mist ink dim slate neutral 色板 深色 浅色" },
  { id: "appearance-skins", tab: "appearance", titleKey: "appearanceSkins", keywords: "wallpaper background picture image veil" },
  { id: "appearance-display", tab: "appearance", titleKey: "appearanceDisplay", keywords: "scale zoom thinking reasoning work details compact detailed 界面尺寸 思考 工作细节" },
  { id: "proactive-brief", tab: "proactive", titleKey: "proactiveBrief", keywords: "brief morning evening digest 简报 晨间 晚间" },
  { id: "proactive-level", tab: "proactive", titleKey: "proactiveLevel", keywords: "proactive quiet hours weekend mute hold 主动 安静 周末 按住" },
  { id: "proactive-notify", tab: "proactive", titleKey: "proactiveNotify", keywords: "notifications toast system 通知" },
  { id: "proactive-intake", tab: "proactive", titleKey: "proactiveIntake", keywords: "intake admission auto observe off orchestrator 自动接单 观察" },
  { id: "personal-jobs", tab: "proactive", titleKey: "personalJobs", keywords: "automation schedule cron job heartbeat webhook 定时" },
  { id: "personal-goals", tab: "proactive", titleKey: "personalGoals", keywords: "goal milestone plan 目标" },
  { id: "personal-watch", tab: "proactive", titleKey: "personalWatch", keywords: "monitor watch page diff price 盯梢 监视" },
  { id: "personal-ideas", tab: "proactive", titleKey: "personalIdeas", keywords: "ideas mail form meet goal 想法" },
  { id: "personal-work", tab: "proactive", titleKey: "personalWork", keywords: "task document proposal lease 事务" },
  { id: "personal-proposals", tab: "proactive", titleKey: "personalProposals", keywords: "proposal send approval hash 发送审阅" },
  { id: "provider-account", tab: "provider", titleKey: "providerAccount", keywords: "api key base url vault" },
  { id: "provider-models", tab: "provider", titleKey: "providerModels", keywords: "model catalog context window" },
  { id: "generation-studio", tab: "generation", titleKey: "generationStudio", keywords: "workshop language ffmpeg drama studio shorts 短片 remotion aspect publish 画幅 平台" },
  { id: "generation-canvas", tab: "generation", titleKey: "generationCanvas", keywords: "canvas protocol plugin channel kling veo seedream" },
  { id: "generation-image", tab: "generation", titleKey: "generationImage", keywords: "image still seedream gemini openai gpt-image qwen jimeng kolors cogview hunyuan 即梦 万相 可图" },
  { id: "generation-video", tab: "generation", titleKey: "generationVideo", keywords: "video clip seedance minimax wan hailuo aliyun jimeng vidu hunyuan cogvideo kling 即梦 可灵 混元" },
  { id: "generation-speech", tab: "generation", titleKey: "generationSpeech", keywords: "tts speech voice openai minimax volcengine elevenlabs azure cosyvoice spark 讯飞 腾讯" },
  { id: "generation-matting", tab: "generation", titleKey: "generationMatting", keywords: "matting matte cutout remove.bg photoroom saliency 抠像 主体分割" },
  { id: "generation-music", tab: "generation", titleKey: "generationMusic", keywords: "music bgm soundtrack minimax mureka elevenlabs stability lyria suno 配乐" },
  { id: "generation-stock", tab: "generation", titleKey: "generationStock", keywords: "stock pexels pixabay unsplash giphy footage 库存 素材" },
  { id: "generation-asr", tab: "generation", titleKey: "generationAsr", keywords: "asr transcribe whisper stt groq deepgram assemblyai paraformer spark 转写 讯飞 腾讯" },
  { id: "generation-storage", tab: "generation", titleKey: "generationStorage", keywords: "s3 oss cas bucket object storage 阿里云 腾讯云 cos" },
  { id: "generation-search", tab: "generation", titleKey: "generationSearch", keywords: "search web duckduckgo bocha metaso jina exa 秘塔 搜索" },
  { id: "generation-workflow", tab: "generation", titleKey: "generationWorkflow", keywords: "runninghub workflow comfy" },
  { id: "policy-gate", tab: "policy", titleKey: "policyGate", keywords: "approval gate shell auto allow" },
  { id: "policy-budget", tab: "policy", titleKey: "policyBudget", keywords: "budget usd cost mtok" },
  { id: "policy-isolation", tab: "policy", titleKey: "policyIsolation", keywords: "isolation sandbox vault signature pwsh powershell 7 runtime sidecar shell 命令行" },
  { id: "policy-data", tab: "policy", titleKey: "policyData", keywords: "data memory export privacy 数据 记忆 导出 隐私" },
  { id: "extensions-work", tab: "connectors", titleKey: "extensionsWorkSources", keywords: "github feishu source token work 来源 飞书" },
  { id: "extensions-connectors", tab: "connectors", titleKey: "connectorsDirectory", keywords: "connector oauth gmail notion slack github 连接应用 邮件 日历" },
  { id: "messaging-policy", tab: "connectors", titleKey: "messagingPolicy", keywords: "im bot telegram qq discord slack wechat lark dingtalk line whatsapp wake mention 消息 机器人 唤醒" },
  { id: "messaging-accounts", tab: "connectors", titleKey: "messagingAccounts", keywords: "im account bot token webhook 账号 令牌 回调" },
  { id: "extensions-mcp", tab: "connectors", titleKey: "extensionsMcp", keywords: "mcp server stdio http json" },
  { id: "advanced-handheld", tab: "connectors", titleKey: "advancedHandheld", keywords: "phone handheld qr pair remote mobile android ios harmony 手机 配对 扫码" },
  { id: "advanced-updates", tab: "advanced", titleKey: "advancedUpdates", keywords: "update version release ota modelscope" },
  { id: "advanced-logs", tab: "advanced", titleKey: "advancedLogs", keywords: "logs diagnostics debug support journal audit chain doctor health 日志 审计 本机检查" },
  { id: "extensions-fibers", tab: "advanced", titleKey: "extensionsFibers", keywords: "plugin fiber wasm" },
  { id: "advanced-labs", tab: "advanced", titleKey: "advancedLabs", keywords: "laboratory lab harness sealed bon hash verbose collapse 实验室" },
  { id: "idol-services", tab: "idol", titleKey: "idolServices", keywords: "idol llm asr tts image voice chat speech 偶像 语音 模型 音色 对话" },
  { id: "idol-you", tab: "idol", titleKey: "idolYou", keywords: "address birthday timezone name 称呼 生日" },
  { id: "idol-work", tab: "idol", titleKey: "idolWork", keywords: "agent profile tools work 动手 工作" },
];

/** 现行分组之前就存在的页签。深链和菜单仍使用它们。 */
const LEGACY_TABS: Readonly<Record<string, { tab: SettingsTab; section: string }>> = {
  shortcuts: { tab: "general", section: "shortcuts-map" },
  isolation: { tab: "policy", section: "policy-isolation" },
  extensions: { tab: "connectors", section: "extensions-mcp" },
  messaging: { tab: "connectors", section: "messaging-policy" },
  personal: { tab: "proactive", section: "personal-jobs" },
  memory: { tab: "policy", section: "policy-data" },
  automations: { tab: "proactive", section: "personal-jobs" },
  notifications: { tab: "proactive", section: "proactive-notify" },
  mcp: { tab: "connectors", section: "extensions-mcp" },
  plugins: { tab: "advanced", section: "extensions-fibers" },
  updates: { tab: "advanced", section: "advanced-updates" },
  logs: { tab: "advanced", section: "advanced-logs" },
  danger: { tab: "advanced", section: "advanced-updates" },
};

const LEGACY_SECTIONS: Readonly<Record<string, string>> = {
  "general-basics": "general-workspace",
  "general-app": "proactive-notify",
  "general-notifications": "proactive-notify",
  "appearance-mode": "appearance-theme",
  "appearance-palette": "appearance-theme",
  "appearance-language": "general-language",
  "isolation-status": "policy-isolation",
  "policy-shell": "policy-isolation",
  "connectors-catalog": "extensions-connectors",
  "memory-items": "policy-data",
  "personal-memory": "policy-data",
  "automations-jobs": "personal-jobs",
  "mcp-servers": "extensions-mcp",
  "mcp-json": "extensions-mcp",
  "plugins-fibers": "extensions-fibers",
  "updates-channel": "advanced-updates",
  "logs-journal": "advanced-logs",
  "logs-doctor": "advanced-logs",
  "advanced-journal": "advanced-logs",
  "advanced-doctor": "advanced-logs",
  "danger-zone": "advanced-updates",
  "advanced-danger": "advanced-updates",
  "provider-video": "generation-image",
  "generation-styles": "generation-studio",
};

/** isSettingsTab 判断一个字符串是不是现行页签，包括不进侧栏的。v 与 t.key 比较是否相等。 */
export function isSettingsTab(v: string): v is SettingsTab {
  return SETTINGS_TABS.some((t) => t.key === v);
}

/** resolveSettingsTarget 把现行或旧的页签与区块对到现行布局上。区块认得出时以区块为准，其次是现行页签，再次是旧页签。 */
export function resolveSettingsTarget(tab?: string, section?: string): { tab: SettingsTab; section: string } {
  const mappedSection = section ? LEGACY_SECTIONS[section] || section : "";
  const owner = mappedSection ? SETTINGS_SECTIONS.find((s) => s.id === mappedSection) : undefined;
  if (owner) return { tab: owner.tab, section: owner.id };
  if (tab && isSettingsTab(tab)) return { tab, section: "" };
  const legacy = tab ? LEGACY_TABS[tab] : undefined;
  if (legacy) return { tab: legacy.tab, section: legacy.section };
  return { tab: "general", section: "" };
}

/** tabsInGroup 取出某一组里进侧栏的页签。group 与 t.group 比较是否相等。 */
export function tabsInGroup(group: GroupKey) {
  return SETTINGS_TABS.filter((t) => t.group === group && !t.hidden);
}

/** navTabs 是进侧栏的全部页签，按分组顺序。 */
export function navTabs() {
  return SETTINGS_TABS.filter((t) => !t.hidden);
}
