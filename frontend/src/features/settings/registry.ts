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
  { key: "shortcuts", group: "app", icon: "keyboard" },
  { key: "provider", group: "agent", icon: "cpu" },
  { key: "policy", group: "agent", icon: "shield" },
  { key: "extensions", group: "agent", icon: "blocks" },
  { key: "personal", group: "agent", icon: "brain" },
  { key: "advanced", group: "system", icon: "wrench" },
];

export const SETTINGS_SECTIONS: readonly SettingsSectionDef[] = [
  { id: "general-workspace", tab: "general", titleKey: "generalWorkspace", keywords: "workspace folder directory" },
  { id: "general-notifications", tab: "general", titleKey: "generalNotifications", keywords: "notifications toast" },
  { id: "general-desktop", tab: "general", titleKey: "generalDesktop", keywords: "tray login always on top" },
  { id: "appearance-theme", tab: "appearance", titleKey: "appearanceTheme", keywords: "theme dark light color mode" },
  { id: "appearance-palette", tab: "appearance", titleKey: "appearancePalette", keywords: "palette terracotta paper mist ink dim slate neutral warm charcoal" },
  { id: "appearance-display", tab: "appearance", titleKey: "appearanceDisplay", keywords: "language locale scale zoom" },
  { id: "shortcuts-map", tab: "shortcuts", titleKey: "shortcutsMap", keywords: "keymap hotkey binding" },
  { id: "provider-account", tab: "provider", titleKey: "providerAccount", keywords: "api key base url vault" },
  { id: "provider-models", tab: "provider", titleKey: "providerModels", keywords: "model catalog context window" },
  { id: "provider-video", tab: "provider", titleKey: "providerVideo", keywords: "video seedance minimax wan ffmpeg drama" },
  { id: "policy-gate", tab: "policy", titleKey: "policyGate", keywords: "approval gate shell auto allow" },
  { id: "policy-budget", tab: "policy", titleKey: "policyBudget", keywords: "budget usd cost mtok" },
  { id: "policy-isolation", tab: "policy", titleKey: "policyIsolation", keywords: "isolation sandbox vault signature" },
  { id: "extensions-mcp", tab: "extensions", titleKey: "extensionsMcp", keywords: "mcp server stdio http json" },
  { id: "extensions-fibers", tab: "extensions", titleKey: "extensionsFibers", keywords: "plugin fiber wasm" },
  { id: "extensions-connectors", tab: "extensions", titleKey: "extensionsConnectors", keywords: "connector oauth gmail feishu" },
  { id: "personal-memory", tab: "personal", titleKey: "personalMemory", keywords: "memory profile staging" },
  { id: "personal-jobs", tab: "personal", titleKey: "personalJobs", keywords: "automation schedule cron job" },
  { id: "advanced-updates", tab: "advanced", titleKey: "advancedUpdates", keywords: "update channel version release" },
  { id: "advanced-logs", tab: "advanced", titleKey: "advancedLogs", keywords: "logs diagnostics debug support" },
  { id: "advanced-journal", tab: "advanced", titleKey: "advancedJournal", keywords: "journal audit chain" },
  { id: "advanced-doctor", tab: "advanced", titleKey: "advancedDoctor", keywords: "doctor diagnostics health" },
  { id: "advanced-danger", tab: "advanced", titleKey: "advancedDanger", keywords: "danger irreversible apply update" },
];

/** Tabs that existed before the eight-tab regroup. Deep links and menus still use them. */
const LEGACY_TABS: Readonly<Record<string, { tab: SettingsTab; section: string }>> = {
  isolation: { tab: "policy", section: "policy-isolation" },
  connectors: { tab: "extensions", section: "extensions-connectors" },
  memory: { tab: "personal", section: "personal-memory" },
  automations: { tab: "personal", section: "personal-jobs" },
  mcp: { tab: "extensions", section: "extensions-mcp" },
  plugins: { tab: "extensions", section: "extensions-fibers" },
  updates: { tab: "advanced", section: "advanced-updates" },
  logs: { tab: "advanced", section: "advanced-logs" },
  danger: { tab: "advanced", section: "advanced-danger" },
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
  "danger-zone": "advanced-danger",
};

export function isSettingsTab(v: string): v is SettingsTab {
  return SETTINGS_TABS.some((t) => t.key === v);
}

/** Resolves any tab/section pair — current or legacy — onto the eight-tab layout. */
export function resolveSettingsTarget(tab?: string, section?: string): { tab: SettingsTab; section: string } {
  const mappedSection = section ? LEGACY_SECTIONS[section] || section : "";
  if (tab && isSettingsTab(tab)) {
    const valid = mappedSection && SETTINGS_SECTIONS.some((s) => s.id === mappedSection && s.tab === tab);
    return { tab, section: valid ? mappedSection : "" };
  }
  const legacy = tab ? LEGACY_TABS[tab] : undefined;
  if (legacy) {
    const valid = mappedSection && SETTINGS_SECTIONS.some((s) => s.id === mappedSection && s.tab === legacy.tab);
    return { tab: legacy.tab, section: valid ? mappedSection : legacy.section };
  }
  const owner = mappedSection ? SETTINGS_SECTIONS.find((s) => s.id === mappedSection) : undefined;
  if (owner) return { tab: owner.tab, section: owner.id };
  return { tab: "general", section: "" };
}

export function tabsInGroup(group: GroupKey) {
  return SETTINGS_TABS.filter((t) => t.group === group);
}
