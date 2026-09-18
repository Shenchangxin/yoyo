import type { SettingsTab } from "../../lib/protocol";

export type SettingsTabDef = {
  key: SettingsTab;
  icon: string;
};

export type SettingsSectionDef = {
  id: string;
  tab: SettingsTab;
  titleKey: keyof import("../../lib/copy").Copy["settings"]["sections"];
};

export const SETTINGS_TABS: readonly SettingsTabDef[] = [
  { key: "general", icon: "sliders" },
  { key: "appearance", icon: "palette" },
  { key: "provider", icon: "key" },
  { key: "policy", icon: "shield" },
  { key: "isolation", icon: "lock" },
  { key: "connectors", icon: "link" },
  { key: "memory", icon: "brain" },
  { key: "automations", icon: "clock" },
  { key: "mcp", icon: "plug" },
  { key: "plugins", icon: "boxes" },
  { key: "shortcuts", icon: "keyboard" },
  { key: "updates", icon: "download" },
  { key: "logs", icon: "scroll" },
  { key: "danger", icon: "alert" },
];

export const SETTINGS_SECTIONS: readonly SettingsSectionDef[] = [
  { id: "general-basics", tab: "general", titleKey: "generalBasics" },
  { id: "general-app", tab: "general", titleKey: "generalApp" },
  { id: "general-desktop", tab: "general", titleKey: "generalDesktop" },
  { id: "appearance-mode", tab: "appearance", titleKey: "appearanceMode" },
  { id: "appearance-language", tab: "appearance", titleKey: "appearanceLanguage" },
  { id: "provider-account", tab: "provider", titleKey: "providerAccount" },
  { id: "provider-models", tab: "provider", titleKey: "providerModels" },
  { id: "policy-gate", tab: "policy", titleKey: "policyGate" },
  { id: "policy-budget", tab: "policy", titleKey: "policyBudget" },
  { id: "isolation-status", tab: "isolation", titleKey: "isolationStatus" },
  { id: "connectors-catalog", tab: "connectors", titleKey: "connectorsCatalog" },
  { id: "memory-items", tab: "memory", titleKey: "memoryItems" },
  { id: "automations-jobs", tab: "automations", titleKey: "automationsJobs" },
  { id: "mcp-servers", tab: "mcp", titleKey: "mcpServers" },
  { id: "mcp-json", tab: "mcp", titleKey: "mcpJson" },
  { id: "plugins-fibers", tab: "plugins", titleKey: "pluginsFibers" },
  { id: "shortcuts-map", tab: "shortcuts", titleKey: "shortcutsMap" },
  { id: "updates-channel", tab: "updates", titleKey: "updatesChannel" },
  { id: "logs-journal", tab: "logs", titleKey: "logsJournal" },
  { id: "logs-doctor", tab: "logs", titleKey: "logsDoctor" },
  { id: "danger-zone", tab: "danger", titleKey: "dangerZone" },
];

export function sectionsFor(tab: SettingsTab) {
  return SETTINGS_SECTIONS.filter((s) => s.tab === tab);
}

export function isSettingsTab(v: string): v is SettingsTab {
  return SETTINGS_TABS.some((t) => t.key === v);
}
