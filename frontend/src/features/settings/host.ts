/**
 * 拥有：设置页拿到的配置、健康和动作回调的形状。
 * 不拥有：具体面板。类型本身不调用 api；patch 等回调会。
 */
import type { AppConfig, Health } from "../../lib/protocol";

/** SettingsHost 是面板能读到的配置，以及保存、探测和更新用的回调。 */
export type SettingsHost = {
  cfg: AppConfig;
  health: Health;
  plugins: any;
  logs: any;
  journal: any;
  doctor: any;
  vault: any;
  sessionId?: string;
  patch: (partial: Partial<AppConfig>) => Promise<void>;
  saveProvider: (next: Partial<AppConfig>, apiKey: string) => Promise<void>;
  refresh?: () => Promise<void>;
  onBrowse: () => Promise<string>;
  onStartMcp: (name: string, command: string, args: string[]) => Promise<void>;
  onStartMcpHttp?: (name: string, endpoint: string) => Promise<void>;
  onStopMcp: (name: string) => Promise<void>;
  onReplaceMcp: (servers: { name: string; command: string; args: string[]; endpoint?: string }[]) => Promise<void>;
  onUnload: (name: string) => Promise<void>;
  onCheckUpdate: () => Promise<any>;
  onUpdateStatus?: () => Promise<any>;
  onApplyUpdate: () => Promise<void>;
  onTestProvider: () => Promise<any>;
  onRevealLogs: () => Promise<void>;
  onRevealJournal?: () => Promise<void>;
  onCopyLogPath?: () => Promise<void>;
  onExportDiagnostics?: () => Promise<void>;
  onDiagnose?: () => Promise<void>;
};
