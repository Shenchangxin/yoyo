import type { AppConfig, Health } from "../../lib/protocol";

export type SettingsHost = {
  cfg: AppConfig;
  health: Health;
  plugins: any;
  logs: any;
  doctor: any;
  vault: any;
  patch: (partial: Partial<AppConfig>) => Promise<void>;
  saveProvider: (next: Partial<AppConfig>, apiKey: string) => Promise<void>;
  onBrowse: () => Promise<string>;
  onStartMcp: (name: string, command: string, args: string[]) => Promise<void>;
  onStartMcpHttp?: (name: string, endpoint: string) => Promise<void>;
  onStopMcp: (name: string) => Promise<void>;
  onReplaceMcp: (servers: { name: string; command: string; args: string[]; endpoint?: string }[]) => Promise<void>;
  onUnload: (name: string) => Promise<void>;
  onCheckUpdate: () => Promise<any>;
  onApplyUpdate: () => Promise<void>;
  onTestProvider: () => Promise<any>;
  onRevealLogs: () => Promise<void>;
};
