// @ts-nocheck
import { App, Button, InputNumber } from "antd";
import { SettingsRow } from "@yingce/components/ui/product/settings-row";
import { ArrowLeft, Boxes, Brain, Bug, Cloud, MessageSquareText, RadioTower, SlidersHorizontal, Workflow } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { refreshSystemChannels } from "@yingce/lib/user-session";
import { defaultConfig, useConfigStore } from "@yingce/stores/use-config-store";
import { useUserStore } from "@yingce/stores/use-user-store";
import { PromptPreferencesPane } from "./prompt-preferences-pane";
import DiagnosticsPanel from "./diagnostics-panel";
import AgentMemoryPane from "./agent-memory-pane";
import { RUNNINGHUB_PLUGIN_ID } from "@yingce/lib/plugins/builtin/workflows";
import { usePluginStore } from "@yingce/stores/use-plugin-store";

type ConfigSectionKey = "channels" | "models" | "runninghub" | "preferences" | "prompts" | "agent-memory" | "storage" | "diagnostics";

const HOSTED_SECTIONS = new Set<ConfigSectionKey>(["channels", "models", "runninghub", "storage"]);

function openHostSettings(section: ConfigSectionKey) {
    const to = `/settings?section=${section}`;
    window.dispatchEvent(new CustomEvent("workspace:navigate", { detail: { to }, cancelable: true }));
}

function HostManagedNotice({ section }: { section: ConfigSectionKey }) {
    return (
        <SettingsPane>
            <div className="settings-pane-header">
                <div className="min-w-0">
                    <h2>由 Yoyo 设置管理</h2>
                    <p>密钥、渠道、对象存储和 RunningHub 统一写在宿主 Settings。画布岛内不再保存凭据。</p>
                </div>
            </div>
            <div className="settings-section">
                <Button type="primary" onClick={() => openHostSettings(section)}>打开 Yoyo 设置</Button>
            </div>
        </SettingsPane>
    );
}

const configSections: Array<{ key: ConfigSectionKey; label: string; description: string; icon: ReactNode }> = [
    { key: "channels", label: "个人渠道", description: "模型服务与个人工作流", icon: <RadioTower className="size-4" /> },
    { key: "runninghub", label: "RunningHub 工作流", description: "个人渠道的 RunningHub 工作流配置", icon: <Workflow className="size-4" /> },
    { key: "models", label: "模型选择", description: "按领域选择默认模型", icon: <Boxes className="size-4" /> },
    { key: "preferences", label: "生成偏好", description: "画布生成默认值", icon: <SlidersHorizontal className="size-4" /> },
    { key: "prompts", label: "提示词偏好", description: "按任务定制平台模板", icon: <MessageSquareText className="size-4" /> },
    { key: "agent-memory", label: "Agent 记忆", description: "批准、添加、导出导入、压缩", icon: <Brain className="size-4" /> },
    { key: "storage", label: "我的对象存储", description: "管理个人媒体存储", icon: <Cloud className="size-4" /> },
    { key: "diagnostics", label: "问题诊断", description: "导出日志协助排查", icon: <Bug className="size-4" /> },
];

export function isConfigSection(value: string | null): value is ConfigSectionKey {
    return configSections.some((section) => section.key === value);
}

export default function SettingsPage() {
    const { message } = App.useApp();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const requestedSection = searchParams.get("section");
    const customChannelsEnabled = useUserStore((state) => state.features.customChannelsEnabled);
    const runtimeStatuses = usePluginStore((state) => state.runtimeStatuses);
    const runningHubPluginEnabled = runtimeStatuses[RUNNINGHUB_PLUGIN_ID] === "enabled";
    const requestedSectionEnabled = requestedSection !== "runninghub" || runningHubPluginEnabled;
    const initialSection = isConfigSection(requestedSection) && requestedSectionEnabled ? requestedSection : customChannelsEnabled ? "channels" : "models";
    const [activeTab, setActiveTab] = useState<ConfigSectionKey>(initialSection === "channels" && !customChannelsEnabled ? "models" : initialSection);
    const config = useConfigStore((state) => state.config);
    const updateConfig = useConfigStore((state) => state.updateConfig);
    const shouldPromptContinue = searchParams.get("continue") === "1";
    const userId = useUserStore((state) => state.user?.id);
    const visibleConfigSections = useMemo(() => (customChannelsEnabled ? configSections : configSections.filter((section) => section.key !== "channels"))
        .filter((section) => section.key !== "runninghub" || runningHubPluginEnabled), [customChannelsEnabled, runningHubPluginEnabled]);

    const isVisibleConfigSection = (value: string | null): value is ConfigSectionKey => isConfigSection(value) && visibleConfigSections.some((section) => section.key === value);

    useEffect(() => {
        if (isVisibleConfigSection(requestedSection)) {
            setActiveTab(requestedSection);
            return;
        }
        setActiveTab((current) => visibleConfigSections.some((section) => section.key === current) ? current : customChannelsEnabled ? "channels" : "models");
    }, [customChannelsEnabled, requestedSection, visibleConfigSections]);

    useEffect(() => {
        if (HOSTED_SECTIONS.has(activeTab)) openHostSettings(activeTab);
    }, [activeTab]);

    useEffect(() => {
        if (!userId) return;
        let cancelled = false;
        void refreshSystemChannels().catch((error) => {
            if (!cancelled) message.warning(error instanceof Error ? `系统模型刷新失败：${error.message}` : "系统模型刷新失败，继续使用本地缓存");
        });
        return () => {
            cancelled = true;
        };
    }, [message, userId]);

    const selectSection = (section: ConfigSectionKey) => {
        if (section === "runninghub" && !runningHubPluginEnabled) return;
        setActiveTab(section);
        const next = new URLSearchParams(searchParams);
        next.set("section", section);
        setSearchParams(next, { replace: true });
    };

    const finishConfig = () => {
        message.success("配置已保存，正在返回创作页面");
        navigate(-1);
    };

    const panes: Record<ConfigSectionKey, ReactNode> = {
        channels: <HostManagedNotice section="channels" />,
        models: <HostManagedNotice section="models" />,
        runninghub: <HostManagedNotice section="runninghub" />,
        preferences: (
            <SettingsPane>
                <div className="settings-pane-header">
                    <div className="min-w-0">
                        <h2>生成偏好</h2>
                        <p>设置新建生成任务时使用的初始值，节点内仍可单独覆盖。</p>
                    </div>
                </div>
                <div className="settings-section">
                    <section className="settings-preference-block">
                        <div className="settings-preference-heading">
                            <h3>画布生成</h3>
                            <p>用于新建图片生成任务，节点内仍可单独覆盖。</p>
                        </div>
                        <SettingsRow
                            label="默认生图张数"
                            control={
                                <InputNumber
                                    min={1}
                                    max={15}
                                    precision={0}
                                    className="w-full"
                                    value={Number(config.canvasImageCount)}
                                    onChange={(value) => updateConfig("canvasImageCount", normalizeImageCount(String(value ?? defaultConfig.canvasImageCount)))}
                                />
                            }
                            controlClassName="w-[200px]"
                        />
                    </section>
                </div>
            </SettingsPane>
        ),
        prompts: <SettingsPane fill><PromptPreferencesPane /></SettingsPane>,
        "agent-memory": (
            <SettingsPane>
                <div className="settings-pane-header">
                    <div className="min-w-0">
                        <h2>Agent 记忆</h2>
                        <p>只属于你。Agent 记下的先待批准；手动添加立刻生效。可导入导出，也可用文本模型压缩相近条目。</p>
                    </div>
                </div>
                <div className="settings-section">
                    <AgentMemoryPane />
                </div>
            </SettingsPane>
        ),
        diagnostics: <SettingsPane><DiagnosticsPanel taskId={searchParams.get("taskId") || undefined} projectId={searchParams.get("projectId") || undefined} /></SettingsPane>,
        storage: <HostManagedNotice section="storage" />,
    };

    return (
        <main className="settings-page app-workspace-page app-user-workspace flex h-full min-h-0 flex-col text-foreground">
            {shouldPromptContinue ? (
                <div className="settings-topbar shrink-0">
                    <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                        <Button icon={<ArrowLeft className="size-4" />} onClick={() => navigate(-1)}>返回创作</Button>
                        <Button type="primary" onClick={finishConfig}>保存并返回</Button>
                    </div>
                </div>
            ) : null}
            <div className="settings-library-frame flex min-h-0 flex-1 flex-col md:flex-row">
                <aside className="settings-nav-panel w-full shrink-0 md:w-[200px]">
                    <nav className="thin-scrollbar flex gap-1 overflow-x-auto p-2 md:block md:space-y-1 md:p-2.5" aria-label="配置分类">
                        {visibleConfigSections.map((item) => {
                            const selected = item.key === activeTab;
                            return (
                                <button
                                    key={item.key}
                                    type="button"
                                    className={`settings-nav-item flex h-9 shrink-0 items-center gap-2 rounded-md px-3 text-left text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:h-auto md:w-full md:items-start md:gap-3 md:py-2.5 ${selected ? "is-active" : "text-foreground/58 hover:bg-muted/55 hover:text-foreground"}`}
                                    onClick={() => selectSection(item.key)}
                                    aria-current={selected ? "page" : undefined}
                                >
                                    <span className={`shrink-0 md:mt-0.5 ${selected ? "text-[var(--workspace-accent)]" : ""}`}>{item.icon}</span>
                                    <span className="min-w-0">
                                        <span className="block whitespace-nowrap text-sm font-medium">{item.label}</span>
                                        <span className="mt-1 hidden text-[var(--fs-label)] leading-4 text-current opacity-65 md:block">{item.description}</span>
                                    </span>
                                </button>
                            );
                        })}
                    </nav>
                </aside>
                <section className="settings-content flex min-h-0 min-w-0 flex-1 flex-col">
                    <div className="app-workspace-scroll min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4 md:px-6 md:py-5">
                        <div className={`settings-pane-root ${activeTab === "prompts" ? "h-full w-full" : "mx-auto w-full max-w-none"}`}>
                            {panes[activeTab]}
                        </div>
                    </div>
                </section>
            </div>
        </main>
    );
}

function SettingsPane({ children, fill = false }: { children: ReactNode; fill?: boolean }) {
    return <div className={fill ? "settings-pane h-full" : "settings-pane"}>{children}</div>;
}

function normalizeImageCount(value: string) {
    return String(Math.max(1, Math.min(15, Math.floor(Math.abs(Number(value)) || Number(defaultConfig.canvasImageCount)))));
}
